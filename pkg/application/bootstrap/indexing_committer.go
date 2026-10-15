package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"

	"github.com/rtcdance/chainpulse/pkg/services/processor"
)

const defaultBatchCommitterLeaseTTL = time.Minute

// MonolithicBatchCommitter routes atomic block commits to one processor.BatchCommitter
// per chain. The AtomicBatchCommitter port carries no chain ID, so the chain is
// derived from the events being committed.
type MonolithicBatchCommitter struct {
	committers map[string]*processor.BatchCommitter
}

// CommitBlock persists the block's events and advances the checkpoint atomically,
// via the committer for the chain the events belong to.
func (m *MonolithicBatchCommitter) CommitBlock(ctx context.Context, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent) error {
	chainID, err := routeChainID(events)
	if err != nil {
		return err
	}
	return m.CommitChainBlock(ctx, chainID, blockNumber, blockHash, events)
}

// CommitChainBlock carries the chain explicitly, including for zero-log blocks.
func (m *MonolithicBatchCommitter) CommitChainBlock(ctx context.Context, chainID string, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent) error {
	for _, event := range events {
		if event == nil || event.ChainID != chainID || event.BlockNumber != blockNumber {
			return fmt.Errorf("invalid event identity for chain %s block %d", chainID, blockNumber)
		}
	}
	committer, ok := m.committers[chainID]
	if !ok {
		return fmt.Errorf("no batch committer for chain %q", chainID)
	}
	if err := committer.CommitBlock(ctx, blockNumber, blockHash, events); err != nil {
		return fmt.Errorf("commit block %d for chain %s: %w", blockNumber, chainID, err)
	}
	return nil
}

// CommitChainCompleteBlock is shared by native single-process and Kafka
// consumer wiring; both use the same processor and PostgreSQL transaction.
func (m *MonolithicBatchCommitter) CommitChainCompleteBlock(ctx context.Context, chainID string, block *blockchain.Block, events []*blockchain.BlockchainEvent) error {
	committer, ok := m.committers[chainID]
	if !ok {
		return fmt.Errorf("no complete block committer for chain %q", chainID)
	}
	if err := committer.CommitCompleteBlock(ctx, block, events); err != nil {
		return fmt.Errorf("commit complete block for chain %s: %w", chainID, err)
	}
	return nil
}

// SetBlockValidator configures the same canonical fence for every chain writer.
func (m *MonolithicBatchCommitter) SetBlockValidator(validate func(context.Context, string, *blockchain.Block) error) {
	for chain, committer := range m.committers {
		committer.SetBlockValidator(func(ctx context.Context, block *blockchain.Block) error { return validate(ctx, chain, block) })
	}
}

// RotateOwnership binds all chain writers to the active broker generation.
// Call only after the previous generation's handlers have exited.
func (m *MonolithicBatchCommitter) RotateOwnership(ctx context.Context, owner string) error {
	for chain, committer := range m.committers {
		if err := committer.RotateOwner(ctx, owner); err != nil {
			return fmt.Errorf("rotate chain %s ownership: %w", chain, err)
		}
	}
	return nil
}

// ReleaseAll gives up every chain lease so the next worker takes over immediately
// instead of waiting for the TTL to expire.
func (m *MonolithicBatchCommitter) ReleaseAll(ctx context.Context) error {
	var errs []error
	for chainID, committer := range m.committers {
		if err := committer.Release(ctx); err != nil {
			errs = append(errs, fmt.Errorf("release lease for chain %s: %w", chainID, err))
		}
	}
	return errors.Join(errs...)
}

// BuildMonolithicBatchCommitter constructs the atomic committer for the
// monolithic path (D2b). The database is unwrapped and asserted against
// processor.FencedStore; when it cannot fence (e.g. the in-memory store), the
// factory returns (nil, nil) and the caller keeps the four-step commit path.
func BuildMonolithicBatchCommitter(
	database portcontract.DatabasePlugin,
	gate processor.WorkGate,
	chains []string,
) (*MonolithicBatchCommitter, error) {
	if database == nil {
		return nil, fmt.Errorf("database is required")
	}
	if len(chains) == 0 {
		return nil, fmt.Errorf("at least one chain ID is required")
	}

	store := unwrapFencedStore(database)
	if store == nil {
		return nil, nil
	}

	ownerToken, err := newLeaseOwnerToken()
	if err != nil {
		return nil, fmt.Errorf("generate lease owner token: %w", err)
	}

	committers := make(map[string]*processor.BatchCommitter, len(chains))
	for _, chainID := range chains {
		committer, err := processor.NewBatchCommitter(store, gate, chainID, ownerToken, defaultBatchCommitterLeaseTTL)
		if err != nil {
			return nil, fmt.Errorf("new batch committer for chain %s: %w", chainID, err)
		}
		committers[chainID] = committer
	}
	return &MonolithicBatchCommitter{committers: committers}, nil
}

func unwrapFencedStore(database portcontract.DatabasePlugin) processor.FencedStore {
	current := database
	for range 5 {
		if store, ok := current.(processor.FencedStore); ok {
			return store
		}
		unwrapper, ok := current.(interface {
			Unwrap() portcontract.DatabasePlugin
		})
		if !ok {
			return nil
		}
		next := unwrapper.Unwrap()
		if next == nil || next == current {
			return nil
		}
		current = next
	}
	return nil
}

func newLeaseOwnerToken() (string, error) {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("chainpulse-monolithic-%s-%d-%s", hostname, os.Getpid(), hex.EncodeToString(random[:])), nil
}

func routeChainID(events []*blockchain.BlockchainEvent) (string, error) {
	for _, event := range events {
		if event != nil && event.ChainID != "" {
			return event.ChainID, nil
		}
	}
	return "", fmt.Errorf("cannot route commit: no event carries a chain ID")
}
