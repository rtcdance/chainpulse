package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lib/pq"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

type finalityView struct {
	chain, tag, status               string
	checkedAt                        time.Time
	proof                            *blockchain.FinalityEvidence
	anchorMatches, generationMatches bool
}

// projectEventFinality preserves canonical/processing state and derives only
// finality. Row-level legacy status is deliberately not used as proof.
func projectEventFinality(event *blockchain.BlockchainEvent, views []finalityView, now time.Time) {
	if event.State == nil {
		return
	}
	unknown := blockchain.EventFinality{State: "unknown", Confidence: "unknown", Source: "unknown"}
	event.State.Finality = unknown
	if event.State.Canonicality != "canonical" {
		return
	}
	for _, view := range views {
		if view.chain != event.ChainID {
			continue
		}
		proof := view.proof
		fresh := func(at time.Time) bool {
			return !at.IsZero() && !at.Before(now.Add(-time.Minute)) && !at.After(now.Add(5*time.Second))
		}
		valid := view.status == "known" && (view.tag == "safe" || view.tag == "finalized") && proof != nil &&
			view.anchorMatches && view.generationMatches && fresh(view.checkedAt)
		if proof != nil {
			valid = valid && fresh(proof.ObservedAt) && proof.Hash != (common.Hash{}) && proof.GenesisHash != (common.Hash{})
		}
		if !valid {
			unknown.Confidence = "degraded"
			unknown.Source = "rpc:" + view.tag
			continue
		}
		if event.BlockNumber > proof.Number {
			continue
		}
		if event.State.Finality.State == "finalized" {
			continue
		}
		observedAt, number := proof.ObservedAt, proof.Number
		event.State.Finality = blockchain.EventFinality{State: view.tag, Confidence: view.tag, Source: "rpc:" + view.tag, ObservedAt: &observedAt, HeadNumber: &number, HeadHash: proof.Hash.Hex()}
	}
	if event.State.Finality.State == "unknown" {
		event.State.Finality = unknown
	}
}

// Read within the event page's repeatable-read transaction. At most two proof
// rows per involved chain are loaded, not one query per historical event.
func projectCanonicalFinality(ctx context.Context, tx *sql.Tx, events []blockchain.BlockchainEvent) error {
	if len(events) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	chains := make([]string, 0)
	for _, event := range events {
		if !seen[event.ChainID] {
			seen[event.ChainID] = true
			chains = append(chains, event.ChainID)
		}
	}
	if len(chains) > 50 {
		return fmt.Errorf("finality projection chain scope exceeds 50")
	}
	rows, err := tx.QueryContext(ctx, `SELECT f.chain_id,f.tag,f.status,f.checked_at,f.proof_at,f.head_number,f.head_hash,f.genesis_hash,
 COALESCE(f.generation=COALESCE((SELECT generation FROM reorg_generations g WHERE g.chain_id=f.chain_id),0),false),
 EXISTS(SELECT 1 FROM blocks b WHERE b.chain_id=f.chain_id AND b.number=f.head_number AND b.hash=f.head_hash),CURRENT_TIMESTAMP
 FROM chain_finality_observations f WHERE f.chain_id=ANY($1) ORDER BY f.chain_id,f.tag LIMIT 101`, pq.Array(chains))
	if err != nil {
		return fmt.Errorf("read canonical finality snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()
	views := make([]finalityView, 0, len(chains)*2)
	now := time.Now()
	for rows.Next() {
		var view finalityView
		var at sql.NullTime
		var number sql.NullInt64
		var hash, genesis sql.NullString
		if err := rows.Scan(&view.chain, &view.tag, &view.status, &view.checkedAt, &at, &number, &hash, &genesis, &view.generationMatches, &view.anchorMatches, &now); err != nil {
			return fmt.Errorf("scan finality snapshot: %w", err)
		}
		if at.Valid && number.Valid && number.Int64 >= 0 && hash.Valid && genesis.Valid {
			view.proof = &blockchain.FinalityEvidence{Number: uint64(number.Int64), Hash: common.HexToHash(hash.String), GenesisHash: common.HexToHash(genesis.String), ObservedAt: at.Time}
		}
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read finality snapshot rows: %w", err)
	}
	if len(views) > 100 {
		return fmt.Errorf("finality snapshot exceeds bounded chain scope")
	}
	for i := range events {
		projectEventFinality(&events[i], views, now)
	}
	return nil
}
