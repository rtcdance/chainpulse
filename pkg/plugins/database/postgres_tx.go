package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
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
	WHERE indexing_state.last_indexed_block <= EXCLUDED.last_indexed_block
	`

// pgTx implements portcontract.Tx over a single database transaction.
//
// When the transaction is fenced (BeginFencedTx), the lease fields record which
// owner and generation the transaction was opened under. Ownership is verified
// under SELECT ... FOR UPDATE at Begin, so the lease row stays locked for the
// transaction's lifetime and no takeover can interleave.
type pgTx struct {
	tx   *sql.Tx
	done bool

	// Fencing (zero when the transaction is not fenced).
	chainID    string
	ownerToken string
	generation int64
}

var _ portcontract.Tx = (*pgTx)(nil)

// PostgreSQLPluginAdapter implements portcontract.Transactioner.
var _ portcontract.Transactioner = (*PostgreSQLPluginAdapter)(nil)

// BeginTx starts a transaction that commits a batch of events together with the
// checkpoint advance for the chain they belong to.
//
// Committing the events and then advancing the checkpoint as separate
// statements is the split F05 calls out: a crash in between leaves the
// checkpoint ahead of the data (events lost forever) or behind it (the whole
// batch replayed on restart). Grouping them makes the batch the unit of
// progress.
func (a *PostgreSQLPluginAdapter) BeginTx(ctx context.Context) (portcontract.Tx, error) {
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

// StoreEvent implements portcontract.Tx.
func (t *pgTx) StoreEvent(ctx context.Context, event any) error {
	ev, ok := event.(*blockchain.BlockchainEvent)
	if !ok || ev == nil {
		return fmt.Errorf("postgres tx: StoreEvent expects *blockchain.BlockchainEvent, got %T", event)
	}
	if t.chainID != "" && ev.ChainID != t.chainID {
		return fmt.Errorf("postgres tx: event chain does not match lease")
	}
	payload, err := eventPayloadJSON(ev)
	if err != nil {
		return err
	}

	result, err := t.tx.ExecContext(ctx, insertEventSQL,
		eventPK(ev),
		ev.ChainID,
		ev.BlockNumber,
		ev.BlockHash.Hex(),
		ev.TransactionHash.Hex(),
		ev.LogIndex,
		ev.ContractAddress.Hex(),
		ev.EventName,
		payload,
		ev.BlockTimestamp,
		eventStatus(ev),
		nullableString(ev.CorrelationID),
		t.chainID != "",
	)
	if err != nil {
		return fmt.Errorf("failed to store event in transaction: %w", err)
	}
	if t.chainID != "" {
		inserted, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("count inserted event effects: %w", err)
		}
		if inserted == 0 {
			// A previously orphaned branch can become canonical again. Only a
			// fenced replay may restore it; checkpoint/header validation in this
			// same transaction rejects an unscheduled conflicting branch. Keep
			// the original physical identity and immutable event payload.
			if _, err := t.tx.ExecContext(ctx, `UPDATE blockchain_events SET status=$5,
				event_state=jsonb_set(event_state,'{processing}','"processed"')
				WHERE chain_id=$1 AND block_hash=$2 AND transaction_hash=$3 AND log_index=$4 AND status='reorged'`,
				ev.ChainID, ev.BlockHash.Hex(), ev.TransactionHash.Hex(), ev.LogIndex, eventStatus(ev)); err != nil {
				return fmt.Errorf("restore canonical event in fenced transaction: %w", err)
			}
		}
	}
	return nil
}

// SaveCheckpoint implements portcontract.Tx.
func (t *pgTx) SaveCheckpoint(ctx context.Context, chainID string, blockNumber uint64, blockHash string) error {
	if chainID == "" {
		return fmt.Errorf("postgres tx: chainID is required")
	}
	if t.chainID != "" {
		if chainID != t.chainID {
			return fmt.Errorf("postgres tx: checkpoint chain does not match lease")
		}
		var current uint64
		err := t.tx.QueryRowContext(ctx, `SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1 FOR UPDATE`, chainID).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lock checkpoint for chain %s: %w", chainID, err)
		}
		if err == nil {
			if blockNumber > current && blockNumber-current != 1 {
				return fmt.Errorf("non-contiguous checkpoint for chain %s: %d after %d", chainID, blockNumber, current)
			}
			if blockNumber <= current {
				var canonicalHash string
				if err := t.tx.QueryRowContext(ctx, `SELECT hash FROM blocks WHERE chain_id=$1 AND number=$2`, chainID, blockNumber).Scan(&canonicalHash); err != nil {
					return fmt.Errorf("verify replayed block for chain %s: %w", chainID, err)
				}
				if canonicalHash != blockHash {
					return fmt.Errorf("conflicting replayed branch for chain %s block %d", chainID, blockNumber)
				}
				return nil
			}
		}
	}
	// Empty blocks need a durable header identity too. An already observed
	// header retains its parent/timestamp; only its canonical hash is refreshed.
	if _, err := t.tx.ExecContext(ctx, `INSERT INTO blocks(id,chain_id,number,hash,parent_hash,timestamp)
		VALUES($1,$2,$3,$4,'',0) ON CONFLICT(id) DO UPDATE SET hash=EXCLUDED.hash`,
		fmt.Sprintf("%s:%d", chainID, blockNumber), chainID, blockNumber, blockHash); err != nil {
		return fmt.Errorf("save block identity in transaction: %w", err)
	}

	_, err := t.tx.ExecContext(ctx, upsertCheckpointSQL, chainID, blockNumber, blockHash, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to save checkpoint in transaction: %w", err)
	}
	return nil
}

// StoreBlock runs inside the fenced event/checkpoint transaction. Header
// continuity and branch identity are checked before metadata is replaced.
func (t *pgTx) StoreBlock(ctx context.Context, chainID string, block *blockchain.Block) error {
	if block == nil || t.chainID == "" || chainID != t.chainID {
		return fmt.Errorf("complete header requires the matching fenced chain")
	}
	if block.Number > 0 {
		var parentHash string
		err := t.tx.QueryRowContext(ctx, `SELECT hash FROM blocks WHERE chain_id=$1 AND number=$2`, chainID, block.Number-1).Scan(&parentHash)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read complete block parent: %w", err)
		}
		if err == nil && parentHash != block.ParentHash.Hex() {
			return fmt.Errorf("complete block parent conflicts with chain %s canonical block %d", chainID, block.Number-1)
		}
	}
	result, err := t.tx.ExecContext(ctx, `UPDATE blocks SET parent_hash=$3,timestamp=$4
		WHERE chain_id=$1 AND number=$2 AND hash=$5`, chainID, block.Number, block.ParentHash.Hex(), block.Timestamp, block.Hash.Hex())
	if err != nil {
		return fmt.Errorf("store complete header metadata: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count complete header effects: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("complete header does not match one canonical block")
	}
	return nil
}

// Commit implements portcontract.Tx.
func (t *pgTx) Commit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		rollbackErr := t.Rollback(context.Background())
		return errors.Join(fmt.Errorf("cancelled transaction cannot commit: %w", err), rollbackErr)
	}
	if t.done {
		return fmt.Errorf("transaction already finished")
	}
	t.done = true
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback implements portcontract.Tx. It is idempotent so callers can `defer
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
