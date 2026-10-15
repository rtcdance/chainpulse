package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// ErrDraining is returned when a batch is offered while the process is shutting
// down. The batch was NOT started; the caller must stop feeding batches.
var ErrDraining = errors.New("processor is draining")

// WorkGate admits units of work and can be closed for draining.
// *bootstrap.DrainController satisfies it, without this package depending on the
// application layer.
type WorkGate interface {
	Begin() bool
	End()
}

// FencedStore is the slice of the database port the committer needs: a lease to
// prove chain ownership, and a transaction that may only commit while that
// ownership still holds.
type FencedStore interface {
	AcquireLease(ctx context.Context, chainID, ownerToken string, ttl time.Duration) (int64, error)
	ReleaseLease(ctx context.Context, chainID, ownerToken string) error
	BeginFencedTx(ctx context.Context, chainID, ownerToken string) (portcontract.Tx, error)
}

// BatchCommitter commits one block's events together with the checkpoint
// advance for that block, as a single fenced transaction.
//
// This is the shared commit contract the delivery plan asks for (F05): before
// it, "write events" and "advance the checkpoint" were separate steps, so a
// crash between them lost a batch or replayed it, and nothing stopped a
// rebalanced-away worker from continuing to write.
//
// The three properties it holds:
//
//  1. Atomicity — events and checkpoint commit or roll back together.
//  2. Fencing — a transaction that starts after another worker took the lease
//     is rejected, so a chain has exactly one writer.
//  3. Drainability — once the gate closes, no new batch starts, and the batch
//     already running is allowed to finish (see bootstrap.DrainController).
type BatchCommitter struct {
	store      FencedStore
	gate       WorkGate
	chainID    string
	ownerToken string
	leaseTTL   time.Duration

	mu            sync.Mutex
	generation    int64
	acquiredAt    time.Time
	validateBlock func(context.Context, *blockchain.Block) error
}

// NewBatchCommitter creates a committer for one chain and owner.
//
// leaseTTL must be comfortably longer than a batch takes; the lease is renewed
// lazily once half of it has elapsed.
func NewBatchCommitter(store FencedStore, gate WorkGate, chainID, ownerToken string, leaseTTL time.Duration) (*BatchCommitter, error) {
	if store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if chainID == "" {
		return nil, fmt.Errorf("chainID is required")
	}
	if ownerToken == "" {
		return nil, fmt.Errorf("ownerToken is required")
	}
	if leaseTTL <= 0 {
		return nil, fmt.Errorf("leaseTTL must be positive")
	}
	return &BatchCommitter{
		store:      store,
		gate:       gate,
		chainID:    chainID,
		ownerToken: ownerToken,
		leaseTTL:   leaseTTL,
	}, nil
}

// SetBlockValidator installs the canonical check performed while the fenced
// transaction owns the chain lease, before it writes any effects.
func (c *BatchCommitter) SetBlockValidator(validate func(context.Context, *blockchain.Block) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validateBlock = validate
}

// EnsureLease acquires the chain lease, or renews it once half the TTL has
// elapsed. It is safe to call before every batch.
func (c *BatchCommitter) EnsureLease(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.generation > 0 && time.Since(c.acquiredAt) < c.leaseTTL/2 {
		return nil
	}

	generation, err := c.store.AcquireLease(ctx, c.chainID, c.ownerToken, c.leaseTTL)
	if err != nil {
		return fmt.Errorf("acquire lease for chain %s: %w", c.chainID, err)
	}
	c.generation = generation
	c.acquiredAt = time.Now()
	return nil
}

// Generation returns the current lease generation (0 before the first
// acquisition). Useful for logging which fence a batch committed under.
func (c *BatchCommitter) Generation() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

// Release gives up the lease so the next worker can take over immediately
// instead of waiting for the TTL.
func (c *BatchCommitter) Release(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation = 0
	c.acquiredAt = time.Time{}
	return c.store.ReleaseLease(ctx, c.chainID, c.ownerToken)
}

// RotateOwner runs after the old generation has drained. A distinct token
// prevents a delayed old task from borrowing the new generation's ownership.
func (c *BatchCommitter) RotateOwner(ctx context.Context, ownerToken string) error {
	if ownerToken == "" {
		return fmt.Errorf("new lease owner is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.store.ReleaseLease(ctx, c.chainID, c.ownerToken); err != nil {
		return fmt.Errorf("release previous generation: %w", err)
	}
	c.ownerToken = ownerToken
	c.generation = 0
	c.acquiredAt = time.Time{}
	return nil
}

// CommitBlock atomically persists `events` and advances the chain checkpoint to
// (blockNumber, blockHash).
//
// events may be empty (an empty block) or entirely duplicates; in both cases the
// checkpoint still advances, because otherwise the chain would stall at that
// height forever. Duplicate suppression is the database's job (the natural-key
// unique constraint plus ON CONFLICT DO NOTHING), not a pre-check here.
func (c *BatchCommitter) CommitBlock(ctx context.Context, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent) error {
	return c.commit(ctx, blockNumber, blockHash, events, nil)
}

// CommitCompleteBlock also commits the canonical parent and timestamp in the
// same transaction. A writer that cannot store full headers is rejected.
func (c *BatchCommitter) CommitCompleteBlock(ctx context.Context, block *blockchain.Block, events []*blockchain.BlockchainEvent) error {
	if block == nil {
		return fmt.Errorf("complete block is required")
	}
	for _, event := range events {
		if event == nil || event.ChainID != c.chainID || event.BlockNumber != block.Number || event.BlockHash != block.Hash || event.Removed {
			return fmt.Errorf("event does not belong to complete block for chain %s", c.chainID)
		}
	}
	return c.commit(ctx, block.Number, block.Hash.Hex(), events, block)
}

func (c *BatchCommitter) commit(ctx context.Context, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent, block *blockchain.Block) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.gate != nil {
		if !c.gate.Begin() {
			return ErrDraining
		}
		defer c.gate.End()
	}

	if err := c.EnsureLease(ctx); err != nil {
		return err
	}

	c.mu.Lock()
	owner := c.ownerToken
	c.mu.Unlock()
	tx, err := c.store.BeginFencedTx(ctx, c.chainID, owner)
	if err != nil {
		// A reorg or rebalance may revoke a still-cached lease. Revalidate on
		// the next attempt rather than staying wedged until half its TTL.
		c.mu.Lock()
		c.acquiredAt = time.Time{}
		c.mu.Unlock()
		return fmt.Errorf("begin fenced transaction for chain %s: %w", c.chainID, err)
	}
	// No-op once Commit has run.
	defer func() { _ = tx.Rollback(ctx) }()

	c.mu.Lock()
	validate := c.validateBlock
	c.mu.Unlock()
	if block != nil && validate != nil {
		if err := validate(ctx, block); err != nil {
			return fmt.Errorf("validate fenced canonical block: %w", err)
		}
	}

	for i, event := range events {
		if event == nil {
			continue
		}
		if err := tx.StoreEvent(ctx, event); err != nil {
			return fmt.Errorf("store event %d/%d: %w", i+1, len(events), err)
		}
	}

	if err := tx.SaveCheckpoint(ctx, c.chainID, blockNumber, blockHash); err != nil {
		return fmt.Errorf("save checkpoint at block %d: %w", blockNumber, err)
	}
	if block != nil {
		writer, ok := tx.(interface {
			StoreBlock(context.Context, string, *blockchain.Block) error
		})
		if !ok {
			return fmt.Errorf("transaction cannot persist a complete block header")
		}
		if err := writer.StoreBlock(ctx, c.chainID, block); err != nil {
			return fmt.Errorf("store complete block header: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit block %d: %w", blockNumber, err)
	}
	return nil
}
