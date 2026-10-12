package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// upsertCheckpointSQL advances a chain's checkpoint. It lives next to the event
// insert so the two halves of an atomic commit are visible together.
const upsertCheckpointSQL = `
INSERT INTO indexing_state (chain_id, last_indexed_block, last_indexed_hash, updated_at)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (chain_id) DO UPDATE
	SET last_indexed_block = EXCLUDED.last_indexed_block,
	    last_indexed_hash = EXCLUDED.last_indexed_hash,
	    updated_at = EXCLUDED.updated_at
	`

// pgTx implements core.Tx over a single database transaction.
type pgTx struct {
	tx   *sql.Tx
	done bool
}

var _ core.Tx = (*pgTx)(nil)

// PostgreSQLPluginAdapter implements core.Transactioner.
var _ core.Transactioner = (*PostgreSQLPluginAdapter)(nil)

// BeginTx starts a transaction that commits a batch of events together with the
// checkpoint advance for the chain they belong to.
//
// Committing the events and then advancing the checkpoint as separate
// statements is the split F05 calls out: a crash in between leaves the
// checkpoint ahead of the data (events lost forever) or behind it (the whole
// batch replayed on restart). Grouping them makes the batch the unit of
// progress.
func (a *PostgreSQLPluginAdapter) BeginTx(ctx context.Context) (core.Tx, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	if db == nil {
		a.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	sqlTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		a.RecordError()
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &pgTx{tx: sqlTx}, nil
}

// StoreEvent implements core.Tx.
func (t *pgTx) StoreEvent(ctx context.Context, event any) error {
	ev, ok := event.(*blockchain.BlockchainEvent)
	if !ok || ev == nil {
		return fmt.Errorf("postgres tx: StoreEvent expects *blockchain.BlockchainEvent, got %T", event)
	}

	_, err := t.tx.ExecContext(ctx, insertEventSQL,
		eventPK(ev),
		ev.ChainID,
		ev.BlockNumber,
		ev.BlockHash.Hex(),
		ev.TransactionHash.Hex(),
		ev.LogIndex,
		ev.ContractAddress.Hex(),
		ev.EventName,
		eventDataJSON(ev.EventData),
		ev.BlockTimestamp,
		eventStatus(ev),
		nullableString(ev.CorrelationID),
	)
	if err != nil {
		return fmt.Errorf("failed to store event in transaction: %w", err)
	}
	return nil
}

// SaveCheckpoint implements core.Tx.
func (t *pgTx) SaveCheckpoint(ctx context.Context, chainID string, blockNumber uint64, blockHash string) error {
	if chainID == "" {
		return fmt.Errorf("postgres tx: chainID is required")
	}

	_, err := t.tx.ExecContext(ctx, upsertCheckpointSQL, chainID, blockNumber, blockHash, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to save checkpoint in transaction: %w", err)
	}
	return nil
}

// Commit implements core.Tx.
func (t *pgTx) Commit(_ context.Context) error {
	if t.done {
		return fmt.Errorf("transaction already finished")
	}
	t.done = true
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback implements core.Tx. It is idempotent so callers can `defer
// tx.Rollback(ctx)` without tracking whether the commit already happened.
func (t *pgTx) Rollback(_ context.Context) error {
	if t.done {
		return nil
	}
	t.done = true
	if err := t.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
