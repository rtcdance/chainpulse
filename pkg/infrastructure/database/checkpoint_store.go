package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// PostgresCheckpointStore persists per-chain indexing checkpoints in the
// `indexing_state` table.
//
// Before 2026-10-13 this type was dead code: nothing constructed it, and its
// ExecContext type assertion `(any, error)` could never match *sql.DB (which
// returns sql.Result), so even a constructed store failed on every write.
// Both are fixed: the write paths now assert *sql.DB directly, and the
// bootstrap wires this store into the shared indexing runtime when the
// deployment asks for PostgreSQL (finding I04).
type PostgresCheckpointStore struct {
	manager *DefaultDatabaseManager
}

// NewPostgresCheckpointStore creates a new checkpoint store backed by PostgreSQL
func NewPostgresCheckpointStore(manager *DefaultDatabaseManager) *PostgresCheckpointStore {
	return &PostgresCheckpointStore{manager: manager}
}

// GetLastIndexedBlock returns the last indexed block number and hash for a chain
func (s *PostgresCheckpointStore) GetLastIndexedBlock(ctx context.Context, chainID string) (uint64, string, error) {
	db, err := s.postgres(ctx)
	if err != nil {
		return 0, "", err
	}

	var blockNum uint64
	var blockHash string
	row := db.QueryRowContext(
		ctx,
		"SELECT last_indexed_block, COALESCE(last_indexed_hash, '') FROM indexing_state WHERE chain_id = $1",
		chainID,
	)
	if err := row.Scan(&blockNum, &blockHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("failed to scan checkpoint: %w", err)
	}
	return blockNum, blockHash, nil
}

// SaveLastIndexedBlock persists the last indexed block for a chain
func (s *PostgresCheckpointStore) SaveLastIndexedBlock(ctx context.Context, chainID string, blockNumber uint64, blockHash string) error {
	return s.SaveCheckpoint(ctx, chainID, blockNumber, blockHash, "")
}

// SaveCheckpoint persists block position AND the runtime's ordering cursor
// (I04). Unlike SaveLastIndexedBlock it records last_cursor, the
// "<chain>:<block>:<log-index>" marker the shared indexing runtime resumes
// from; last_indexed_hash stays reserved for block hashes.
func (s *PostgresCheckpointStore) SaveCheckpoint(ctx context.Context, chainID string, blockNumber uint64, blockHash string, cursor string) error {
	db, err := s.postgres(ctx)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(
		ctx,
		`INSERT INTO indexing_state (chain_id, last_indexed_block, last_indexed_hash, last_cursor, updated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (chain_id) DO UPDATE SET
		   last_indexed_block = $2, last_indexed_hash = $3, last_cursor = $4, updated_at = $5`,
		chainID, blockNumber, blockHash, cursor, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}
	return nil
}

// GetLastCursor returns the runtime cursor persisted by SaveCheckpoint, or ""
// when no checkpoint exists for the chain.
func (s *PostgresCheckpointStore) GetLastCursor(ctx context.Context, chainID string) (string, error) {
	db, err := s.postgres(ctx)
	if err != nil {
		return "", err
	}

	var cursor string
	row := db.QueryRowContext(
		ctx,
		"SELECT COALESCE(last_cursor, '') FROM indexing_state WHERE chain_id = $1",
		chainID,
	)
	if err := row.Scan(&cursor); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to scan cursor: %w", err)
	}
	return cursor, nil
}

// GetBlockHash returns the block hash for a given chain and block number
func (s *PostgresCheckpointStore) GetBlockHash(ctx context.Context, chainID string, blockNumber uint64) (string, error) {
	db, err := s.postgres(ctx)
	if err != nil {
		return "", err
	}

	var hash string
	row := db.QueryRowContext(
		ctx,
		"SELECT hash FROM blocks WHERE chain_id = $1 AND number = $2",
		chainID, blockNumber,
	)
	if err := row.Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to scan block hash: %w", err)
	}
	return hash, nil
}

// postgres returns the live *sql.DB handle. The manager stores it as `any`,
// but it is always the manager's own *sql.DB — anything else is a wiring bug
// and must fail loudly rather than fall into an always-false capability probe.
func (s *PostgresCheckpointStore) postgres(ctx context.Context) (*sql.DB, error) {
	db, err := s.manager.GetPostgresDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get postgres db: %w", err)
	}
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("unexpected postgres db type %T", db)
	}
	return sqlDB, nil
}

// Compile-time check
var _ core.CheckpointStore = (*PostgresCheckpointStore)(nil)
