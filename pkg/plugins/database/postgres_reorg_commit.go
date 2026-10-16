package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/core/topics"
)

// CommitReorg atomically revokes orphaned effects, rewinds durable progress,
// fences the writer, advances the cache generation and records notifications.
// No notification may describe a rollback that was only partly applied.
func (a *PostgreSQLPluginAdapter) CommitReorg(ctx context.Context, chainID string, from, to uint64) (int64, error) {
	return a.commitReorg(ctx, chainID, from, to, "")
}

// CommitReorgAtCheckpoint rejects a stale supervisor decision after a writer
// advanced or another supervisor already repaired the same height.
func (a *PostgreSQLPluginAdapter) CommitReorgAtCheckpoint(ctx context.Context, chainID string, from, to uint64, expectedHash string) (int64, error) {
	if expectedHash == "" {
		return 0, fmt.Errorf("expected checkpoint hash is required")
	}
	return a.commitReorg(ctx, chainID, from, to, expectedHash)
}

func (a *PostgreSQLPluginAdapter) commitReorg(ctx context.Context, chainID string, from, to uint64, expectedHash string) (int64, error) {
	if chainID == "" || from == 0 || to < from {
		return 0, fmt.Errorf("chain and valid non-genesis reorg range are required")
	}
	db, err := a.requireDB()
	if err != nil {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin reorg: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// Lock the same row as the fenced committer before touching its progress.
	if _, err := tx.ExecContext(ctx, `UPDATE chain_leases SET generation=generation+1, expires_at=now() WHERE chain_id=$1`, chainID); err != nil {
		return 0, fmt.Errorf("fence reorg writer: %w", err)
	}
	var height uint64
	var oldHash string
	err = tx.QueryRowContext(ctx, `SELECT last_indexed_block, COALESCE(last_indexed_hash,'') FROM indexing_state WHERE chain_id=$1 FOR UPDATE`, chainID).Scan(&height, &oldHash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("chain %s has no durable checkpoint", chainID)
	}
	if err != nil {
		return 0, fmt.Errorf("load reorg checkpoint: %w", err)
	}
	if height < from {
		return 0, nil // Retried after the committed rewind; deferred rollback undoes fencing.
	}
	if height > to {
		return 0, fmt.Errorf("checkpoint advanced to %d beyond reorg range ending %d", height, to)
	}
	if expectedHash != "" && (height != to || oldHash != expectedHash) {
		return 0, fmt.Errorf("reorg checkpoint changed; retry canonical inspection")
	}

	var protected bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chain_finality_observations WHERE chain_id=$1 AND tag='finalized' AND head_number >= $2)`, chainID, from).Scan(&protected); err != nil {
		return 0, fmt.Errorf("read verified finalized boundary: %w", err)
	}
	if protected {
		return 0, fmt.Errorf("reorg crosses verified finalized boundary; manual recovery is required")
	}
	var finalized bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM blockchain_events
		WHERE chain_id=$1 AND block_number BETWEEN $2 AND $3 AND (status='finalized' OR event_state#>>'{finality,state}'='finalized'))`, chainID, from, to).Scan(&finalized); err != nil {
		return 0, fmt.Errorf("check finalized reorg boundary: %w", err)
	}
	if finalized {
		return 0, fmt.Errorf("reorg crosses finalized effects; manual recovery is required")
	}
	var generation int64
	err = tx.QueryRowContext(ctx, `INSERT INTO reorg_generations(chain_id,generation) VALUES($1,1)
		ON CONFLICT(chain_id) DO UPDATE SET generation=reorg_generations.generation+1, updated_at=now() RETURNING generation`, chainID).Scan(&generation)
	if err != nil {
		return 0, fmt.Errorf("advance reorg generation: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE blockchain_events SET status='reorged' WHERE chain_id=$1 AND block_number BETWEEN $2 AND $3 AND status<>'reorged'`, chainID, from, to)
	if err != nil {
		return 0, fmt.Errorf("revoke orphaned events: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count orphaned events: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM blocks WHERE chain_id=$1 AND number BETWEEN $2 AND $3`, chainID, from, to); err != nil {
		return 0, fmt.Errorf("revoke orphaned block snapshots: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE indexing_state SET last_indexed_block=$2,
		last_indexed_hash=COALESCE((SELECT hash FROM blocks WHERE chain_id=$1 AND number=$2),''),
		last_cursor=NULL,updated_at=now() WHERE chain_id=$1`, chainID, from-1); err != nil {
		return 0, fmt.Errorf("rewind reorg checkpoint: %w", err)
	}
	jobID := fmt.Sprintf("%s:%d:%d:%s", chainID, from, to, oldHash)
	if _, err := tx.ExecContext(ctx, `INSERT INTO reorg_jobs(id,chain_id,generation,from_block,to_block,status)
		VALUES($1,$2,$3,$4,$5,'replaying') ON CONFLICT(id) DO NOTHING`, jobID, chainID, generation, from, to); err != nil {
		return 0, fmt.Errorf("record applied reorg: %w", err)
	}
	payload, err := json.Marshal(portcontract.ReorgRollbackEvent{ChainID: chainID, FromBlock: from, ToBlock: to, DetectedAt: time.Now().UTC()})
	if err != nil {
		return 0, fmt.Errorf("encode reorg notice: %w", err)
	}
	detected, err := json.Marshal(struct {
		DetectedAt       time.Time
		ReorgBlock       uint64
		BlocksAffected   uint64
		EventsRolledBack int64
	}{time.Now().UTC(), from, to - from + 1, count})
	if err != nil {
		return 0, fmt.Errorf("encode reorg detection: %w", err)
	}
	for _, topic := range []string{topics.TopicReorgDetected, topics.TopicReorgRollback} {
		notice := payload
		if topic == topics.TopicReorgDetected {
			notice = detected
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO reorg_outbox(job_id,topic,payload) VALUES($1,$2,$3)`, jobID, topic, string(notice)); err != nil {
			return 0, fmt.Errorf("record reorg notification: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit reorg: %w", err)
	}
	return count, nil
}

// CompleteReorgsThrough records replay completion only after the canonical
// writer has durably caught up through the revoked range.
func (a *PostgreSQLPluginAdapter) CompleteReorgsThrough(ctx context.Context, chainID string, height uint64) error {
	db, err := a.requireDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE reorg_jobs SET status='done',updated_at=now()
		WHERE chain_id=$1 AND status='replaying' AND to_block<=$2
		AND EXISTS(SELECT 1 FROM indexing_state WHERE chain_id=$1 AND last_indexed_block>=$2)`, chainID, height)
	if err != nil {
		return fmt.Errorf("record canonical replay completion: %w", err)
	}
	return nil
}
