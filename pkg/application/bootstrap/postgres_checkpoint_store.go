package bootstrap

import (
	"context"
	"fmt"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
)

// checkpointStateStore is the slice of the infrastructure checkpoint store the
// adapter needs. An interface keeps the adapter unit-testable without a live
// PostgreSQL.
type checkpointStateStore interface {
	GetLastIndexedBlock(ctx context.Context, chainID string) (uint64, string, error)
	SaveCheckpoint(ctx context.Context, chainID string, blockNumber uint64, blockHash string, cursor string) error
}

// postgresCheckpointStore adapts the infrastructure PostgresCheckpointStore to
// the shared indexing runtime's CheckpointStore port (finding I04). Before
// this, the deployed shared runtime always checkpointed into an in-memory
// map: a restart lost the position, and the API read `indexing_state` — a
// table nothing wrote.
//
// Cursor semantics: Save persists both the block number and the runtime's
// ordering cursor ("chain:block:logIndex"). Load intentionally returns an
// empty Cursor. After a restart the failure journal (in-memory) is empty, so
// a block-level resume is the correct position; a non-empty cursor would make
// the replay comparison filter events against a journal that no longer holds
// them.
type postgresCheckpointStore struct {
	inner checkpointStateStore
}

func newPostgresCheckpointStore(inner checkpointStateStore) *postgresCheckpointStore {
	return &postgresCheckpointStore{inner: inner}
}

func (s *postgresCheckpointStore) Load(ctx context.Context, chainID string) (appindexing.Checkpoint, error) {
	block, hash, err := s.inner.GetLastIndexedBlock(ctx, chainID)
	if err != nil {
		return appindexing.Checkpoint{}, fmt.Errorf("load checkpoint for %s: %w", chainID, err)
	}
	return appindexing.Checkpoint{ChainID: chainID, BlockNumber: block, BlockHash: hash}, nil
}

func (s *postgresCheckpointStore) Save(ctx context.Context, checkpoint appindexing.Checkpoint) error {
	if checkpoint.ChainID == "" {
		return fmt.Errorf("checkpoint chain id is required")
	}
	if err := s.inner.SaveCheckpoint(ctx, checkpoint.ChainID, checkpoint.BlockNumber, checkpoint.BlockHash, checkpoint.Cursor); err != nil {
		return fmt.Errorf("save checkpoint for %s: %w", checkpoint.ChainID, err)
	}
	return nil
}

// resolveCheckpointStore picks the shared runtime's checkpoint store: the
// caller-provided persistent one when set, the in-memory map otherwise (the
// zero-dependency playground keeps working).
func resolveCheckpointStore(configured appindexing.CheckpointStore, chains []string) appindexing.CheckpointStore {
	if configured != nil {
		return configured
	}
	return newMonolithicMemoryCheckpointStore(chains)
}
