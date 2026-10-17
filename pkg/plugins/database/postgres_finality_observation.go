package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// RecordFinalityObservation retains a verified high-water proof when a poll
// degrades. A failure never erases the finalized rollback boundary.
func (a *PostgreSQLPluginAdapter) RecordFinalityObservation(ctx context.Context, chain, tag string, proof *blockchain.FinalityEvidence) error {
	allowed := false
	for _, declared := range a.ChainScope() {
		if declared == chain {
			allowed = true
		}
	}
	if !allowed || (tag != "safe" && tag != "finalized") {
		return fmt.Errorf("finality observation requires declared chain and safe/finalized tag")
	}
	db, err := a.requireDB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin finality observation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize both tags and observer replicas, including the first insert.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "chainpulse:finality:"+chain); err != nil {
		return fmt.Errorf("lock finality observation: %w", err)
	}
	if proof == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_finality_observations(chain_id,tag,status) VALUES($1,$2,'degraded')
   ON CONFLICT(chain_id,tag) DO UPDATE SET status='degraded',checked_at=CURRENT_TIMESTAMP`, chain, tag)
		if err != nil {
			return fmt.Errorf("record degraded finality: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit degraded finality: %w", err)
		}
		return nil
	}
	if proof.Number > math.MaxInt64 || proof.Hash == (common.Hash{}) || proof.GenesisHash == (common.Hash{}) || proof.ObservedAt.IsZero() {
		return fmt.Errorf("finality proof requires bounded height, hashes and observation time")
	}
	var checkpoint uint64
	var now time.Time
	err = tx.QueryRowContext(ctx, `SELECT last_indexed_block,CURRENT_TIMESTAMP FROM indexing_state WHERE chain_id=$1 FOR SHARE`, chain).Scan(&checkpoint, &now)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("finality proof requires a committed chain checkpoint")
	}
	if err != nil {
		return fmt.Errorf("lock finality checkpoint: %w", err)
	}
	if proof.ObservedAt.Before(now.Add(-time.Minute)) || proof.ObservedAt.After(now.Add(5*time.Second)) {
		return fmt.Errorf("finality proof is stale or from the future")
	}
	if proof.Number > checkpoint {
		return fmt.Errorf("finality head is ahead of committed checkpoint")
	}
	var hash string
	if err := tx.QueryRowContext(ctx, `SELECT hash FROM blocks WHERE chain_id=$1 AND number=$2`, chain, proof.Number).Scan(&hash); err != nil {
		return fmt.Errorf("read finality canonical anchor: %w", err)
	}
	if hash != proof.Hash.Hex() {
		return fmt.Errorf("finality header conflicts with committed canonical branch")
	}
	var mismatched bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM chain_finality_observations WHERE chain_id=$1 AND genesis_hash IS NOT NULL AND genesis_hash<>$2)`, chain, proof.GenesisHash.Hex()).Scan(&mismatched); err != nil {
		return fmt.Errorf("read pinned finality identity: %w", err)
	}
	if mismatched {
		return fmt.Errorf("finality provider genesis identity changed")
	}
	var generation int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT generation FROM reorg_generations WHERE chain_id=$1),0)`, chain).Scan(&generation); err != nil {
		return fmt.Errorf("read finality generation: %w", err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO chain_finality_observations(chain_id,tag,status,checked_at,proof_at,head_number,head_hash,genesis_hash,generation)
 VALUES($1,$2,'known',CURRENT_TIMESTAMP,$3,$4,$5,$6,$7)
 ON CONFLICT(chain_id,tag) DO UPDATE SET status='known',checked_at=CURRENT_TIMESTAMP,proof_at=EXCLUDED.proof_at,
 head_number=EXCLUDED.head_number,head_hash=EXCLUDED.head_hash,genesis_hash=EXCLUDED.genesis_hash,generation=EXCLUDED.generation
 WHERE chain_finality_observations.head_number IS NULL OR
 ((chain_finality_observations.head_number<=EXCLUDED.head_number OR (EXCLUDED.tag='safe' AND chain_finality_observations.generation<>EXCLUDED.generation))
 AND (chain_finality_observations.head_number<>EXCLUDED.head_number OR chain_finality_observations.head_hash=EXCLUDED.head_hash OR (EXCLUDED.tag='safe' AND chain_finality_observations.generation<>EXCLUDED.generation)))`, chain, tag, proof.ObservedAt, proof.Number, proof.Hash.Hex(), proof.GenesisHash.Hex(), generation)
	if err != nil {
		return fmt.Errorf("persist verified finality proof: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read finality update result: %w", err)
	}
	if changed != 1 {
		return fmt.Errorf("finality proof regressed or changed an existing anchor")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finality proof: %w", err)
	}
	return nil
}
