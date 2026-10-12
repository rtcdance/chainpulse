package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
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
	BeginFencedTx(ctx context.Context, chainID, ownerToken string) (core.Tx, error)
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

	mu         sync.Mutex
	generation int64
	acquiredAt time.Time
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
	c.generation = 0
	c.mu.Unlock()
	return c.store.ReleaseLease(ctx, c.chainID, c.ownerToken)
}

// CommitBlock atomically persists `events` and advances the chain checkpoint to
// (blockNumber, blockHash).
//
// events may be empty (an empty block) or entirely duplicates; in both cases the
// checkpoint still advances, because otherwise the chain would stall at that
// height forever. Duplicate suppression is the database's job (the natural-key
// unique constraint plus ON CONFLICT DO NOTHING), not a pre-check here.
func (c *BatchCommitter) CommitBlock(ctx context.Context, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent) error {
	if c.gate != nil {
		if !c.gate.Begin() {
			return ErrDraining
		}
		defer c.gate.End()
	}

	if err := c.EnsureLease(ctx); err != nil {
		return err
	}

	tx, err := c.store.BeginFencedTx(ctx, c.chainID, c.ownerToken)
	if err != nil {
		return fmt.Errorf("begin fenced transaction for chain %s: %w", c.chainID, err)
	}
	// No-op once Commit has run.
	defer func() { _ = tx.Rollback(ctx) }()

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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit block %d: %w", blockNumber, err)
	}
	return nil
}
