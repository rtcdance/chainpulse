package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// Chain-scoped accessors (invariant I01).
//
// The core.DatabasePlugin port addresses blocks and events WITHOUT a chain key:
// GetBlock(number), GetLatestBlock(), GetEventsByBlockRange(from, to),
// MarkEventsAsReorged(from, to), GetAllBlocks(), GetAllEvents(). Those
// signatures are kept for compatibility, but they are only safe in a
// single-chain deployment — with two chains at the same height they mix data,
// and a rollback on one chain rewrites rows belonging to the other.
//
// The methods below take the chain key explicitly. Multi-chain code MUST use
// them; the unscoped port methods are for explicitly single-chain contexts.

// GetEventsByBlockRangeForChain returns one chain's events in a block range.
func (a *PostgreSQLPluginAdapter) GetEventsByBlockRangeForChain(ctx context.Context, chainID string, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error) {
	if chainID == "" {
		return nil, fmt.Errorf("chainID is required")
	}

	db, err := a.requireDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ` + selectEventColumns + `
		FROM blockchain_events
		WHERE chain_id = $1 AND block_number >= $2 AND block_number <= $3
		ORDER BY block_number ASC, log_index ASC
	`

	rows, err := db.QueryContext(ctx, query, chainID, fromBlock, toBlock)
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to query events for chain %s: %w", chainID, err)
	}
	defer func() { _ = rows.Close() }()

	var events []*blockchain.BlockchainEvent
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			a.RecordError()
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}
	return events, nil
}

// GetBlockForChain returns one chain's block at a height.
func (a *PostgreSQLPluginAdapter) GetBlockForChain(ctx context.Context, chainID string, blockNumber uint64) (*blockchain.Block, error) {
	if chainID == "" {
		return nil, fmt.Errorf("chainID is required")
	}

	db, err := a.requireDB()
	if err != nil {
		return nil, err
	}

	block := &blockchain.Block{}
	var hash, parentHash string
	err = db.QueryRowContext(ctx, `
SELECT number, hash, parent_hash, timestamp
	FROM blocks
	WHERE chain_id = $1 AND number = $2`, chainID, blockNumber).Scan(
		&block.Number, &hash, &parentHash, &block.Timestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to query block for chain %s: %w", chainID, err)
	}

	block.Hash = common.HexToHash(hash)
	block.ParentHash = common.HexToHash(parentHash)
	return block, nil
}

// GetLatestBlockForChain returns one chain's highest indexed block number.
func (a *PostgreSQLPluginAdapter) GetLatestBlockForChain(ctx context.Context, chainID string) (uint64, error) {
	if chainID == "" {
		return 0, fmt.Errorf("chainID is required")
	}

	db, err := a.requireDB()
	if err != nil {
		return 0, err
	}

	var number uint64
	err = db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(number), 0) FROM blocks WHERE chain_id = $1", chainID).Scan(&number)
	if err != nil {
		a.RecordError()
		return 0, fmt.Errorf("failed to query latest block for chain %s: %w", chainID, err)
	}
	return number, nil
}

// GetAllBlocksForChain returns one chain's blocks, ascending.
func (a *PostgreSQLPluginAdapter) GetAllBlocksForChain(ctx context.Context, chainID string) ([]*blockchain.Block, error) {
	if chainID == "" {
		return nil, fmt.Errorf("chainID is required")
	}

	db, err := a.requireDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
SELECT number, hash, parent_hash, timestamp
	FROM blocks
	WHERE chain_id = $1
	ORDER BY number ASC`, chainID)
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to query blocks for chain %s: %w", chainID, err)
	}
	defer func() { _ = rows.Close() }()

	var blocks []*blockchain.Block
	for rows.Next() {
		block := &blockchain.Block{}
		var hash, parentHash string
		if err := rows.Scan(&block.Number, &hash, &parentHash, &block.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}
		block.Hash = common.HexToHash(hash)
		block.ParentHash = common.HexToHash(parentHash)
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blocks: %w", err)
	}
	return blocks, nil
}

// MarkEventsAsReorgedForChain soft-deletes one chain's events in a range.
//
// The unscoped MarkEventsAsReorged has no chain filter, so using it with two
// chains would mark the other chain's events as reorged — the exact cross-chain
// leak I01 forbids.
func (a *PostgreSQLPluginAdapter) MarkEventsAsReorgedForChain(ctx context.Context, chainID string, fromBlock, toBlock uint64) (int64, error) {
	if chainID == "" {
		return 0, fmt.Errorf("chainID is required")
	}

	db, err := a.requireDB()
	if err != nil {
		return 0, err
	}

	res, err := db.ExecContext(ctx, `
UPDATE blockchain_events
	SET status = 'reorged'
	WHERE chain_id = $1 AND block_number >= $2 AND block_number <= $3`,
		chainID, fromBlock, toBlock)
	if err != nil {
		a.RecordError()
		return 0, fmt.Errorf("failed to mark events reorged for chain %s: %w", chainID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read affected rows: %w", err)
	}
	return affected, nil
}

// GetCheckpointForChain returns one chain's checkpoint (0, "" when unset).
func (a *PostgreSQLPluginAdapter) GetCheckpointForChain(ctx context.Context, chainID string) (uint64, string, error) {
	if chainID == "" {
		return 0, "", fmt.Errorf("chainID is required")
	}

	db, err := a.requireDB()
	if err != nil {
		return 0, "", err
	}

	var block uint64
	var hash sql.NullString
	err = db.QueryRowContext(ctx,
		"SELECT last_indexed_block, last_indexed_hash FROM indexing_state WHERE chain_id = $1", chainID,
	).Scan(&block, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", nil
	}
	if err != nil {
		a.RecordError()
		return 0, "", fmt.Errorf("failed to read checkpoint for chain %s: %w", chainID, err)
	}
	return block, hash.String, nil
}

// requireDB returns the connection or a clear error, so each accessor does not
// repeat the nil check.
func (a *PostgreSQLPluginAdapter) requireDB() (*sql.DB, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		a.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}
	return db, nil
}
