package database

import (
	"context"
	"fmt"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// PostgresCheckpointStore implements core.CheckpointStore using PostgreSQL
type PostgresCheckpointStore struct {
	manager *DefaultDatabaseManager
}

// NewPostgresCheckpointStore creates a new checkpoint store backed by PostgreSQL
func NewPostgresCheckpointStore(manager *DefaultDatabaseManager) *PostgresCheckpointStore {
	return &PostgresCheckpointStore{manager: manager}
}

// GetLastIndexedBlock returns the last indexed block number and hash for a chain
func (s *PostgresCheckpointStore) GetLastIndexedBlock(ctx context.Context, chainID string) (uint64, string, error) {
	db, err := s.manager.GetPostgresDB(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get postgres db: %w", err)
	}

	sqlDB, ok := db.(interface {
		QueryRowContext(context.Context, string, ...any) interface {
			Scan(...any) error
		}
	})
	if !ok {
		return 0, "", fmt.Errorf("unexpected postgres db type")
	}

	var blockNum uint64
	var blockHash string
	row := sqlDB.QueryRowContext(ctx,
		"SELECT last_indexed_block, COALESCE(last_indexed_hash, '') FROM indexing_state WHERE chain_id = $1",
		chainID,
	)
	if err := row.Scan(&blockNum, &blockHash); err != nil {
		return 0, "", nil // no row found is not an error
	}
	return blockNum, blockHash, nil
}

// SaveLastIndexedBlock persists the last indexed block for a chain
func (s *PostgresCheckpointStore) SaveLastIndexedBlock(ctx context.Context, chainID string, blockNumber uint64, blockHash string) error {
	db, err := s.manager.GetPostgresDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get postgres db: %w", err)
	}

	sqlDB, ok := db.(interface {
		ExecContext(context.Context, string, ...any) (any, error)
	})
	if !ok {
		return fmt.Errorf("unexpected postgres db type")
	}

	_, err = sqlDB.ExecContext(ctx,
		`INSERT INTO indexing_state (chain_id, last_indexed_block, last_indexed_hash, updated_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (chain_id) DO UPDATE SET last_indexed_block = $2, last_indexed_hash = $3, updated_at = $4`,
		chainID, blockNumber, blockHash, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

// GetBlockHash returns the block hash for a given chain and block number
func (s *PostgresCheckpointStore) GetBlockHash(ctx context.Context, chainID string, blockNumber uint64) (string, error) {
	db, err := s.manager.GetPostgresDB(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get postgres db: %w", err)
	}

	sqlDB, ok := db.(interface {
		QueryRowContext(context.Context, string, ...any) interface {
			Scan(...any) error
		}
	})
	if !ok {
		return "", fmt.Errorf("unexpected postgres db type")
	}

	var hash string
	row := sqlDB.QueryRowContext(ctx,
		"SELECT hash FROM blocks WHERE chain_id = $1 AND number = $2",
		chainID, blockNumber,
	)
	if err := row.Scan(&hash); err != nil {
		return "", nil // not found is not an error
	}
	return hash, nil
}

// Compile-time check
var _ core.CheckpointStore = (*PostgresCheckpointStore)(nil)
