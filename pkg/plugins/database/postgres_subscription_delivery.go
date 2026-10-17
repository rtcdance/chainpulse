package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rtcdance/chainpulse/pkg/ports"
)

func (a *PostgreSQLPluginAdapter) subscriptionChain(chain string) error {
	scope := a.ChainScope()
	if len(scope) == 0 || len(scope) > 50 || !slices.Contains(scope, chain) {
		return fmt.Errorf("subscription reads require an explicitly declared chain")
	}
	return nil
}

func (a *PostgreSQLPluginAdapter) ReadSubscriptionBlock(ctx context.Context, chain string, height uint64, hash string) (ports.SubscriptionBlock, error) {
	var result ports.SubscriptionBlock
	if err := a.subscriptionChain(chain); err != nil {
		return result, err
	}
	if height > math.MaxInt64 || len(hash) != 66 {
		return result, fmt.Errorf("invalid subscription block identity")
	}
	db, err := a.requireDB()
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return result, fmt.Errorf("begin subscription block snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var checkpoint uint64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT last_indexed_block FROM indexing_state WHERE chain_id=$1),0),
COALESCE((SELECT generation FROM reorg_generations WHERE chain_id=$1),0)`, chain).Scan(&checkpoint, &result.Generation); err != nil {
		return result, fmt.Errorf("read subscription checkpoint and generation: %w", err)
	}
	if result.Generation < 0 {
		return result, fmt.Errorf("invalid persisted subscription generation")
	}
	var actual string
	err = tx.QueryRowContext(ctx, `SELECT hash FROM blocks WHERE chain_id=$1 AND number=$2`, chain, height).Scan(&actual)
	if errors.Is(err, sql.ErrNoRows) {
		if height > checkpoint {
			var rewound bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reorg_jobs WHERE chain_id=$1 AND from_block<=$2 AND to_block>=$2)`, chain, height).Scan(&rewound); err != nil {
				return result, fmt.Errorf("read subscription rewind authority: %w", err)
			}
			if !rewound {
				return result, fmt.Errorf("subscription notification precedes durable checkpoint")
			}
		}
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("read subscription canonical header: %w", err)
	}
	if !strings.EqualFold(actual, hash) {
		return result, nil
	}
	if height > checkpoint {
		return result, fmt.Errorf("subscription header exceeds durable checkpoint")
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+selectEventColumns+` FROM blockchain_events
WHERE chain_id=$1 AND block_number=$2 AND block_hash=$3 AND status<>'reorged'
AND event_state->>'canonicality'='canonical' ORDER BY log_index ASC,id ASC LIMIT 10001`, chain, height, actual)
	if err != nil {
		return result, fmt.Errorf("read subscription canonical events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return result, fmt.Errorf("scan subscription event: %w", err)
		}
		result.Events = append(result.Events, event)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("read subscription event rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return result, fmt.Errorf("close subscription event rows: %w", err)
	}
	if len(result.Events) > 10000 {
		return result, fmt.Errorf("subscription block exceeds the log count bound")
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("finish subscription snapshot: %w", err)
	}
	result.Canonical = true
	return result, nil
}

func (a *PostgreSQLPluginAdapter) ReadSubscriptionReorg(ctx context.Context, chain string, id int64) (ports.SubscriptionReorg, error) {
	var result ports.SubscriptionReorg
	if err := a.subscriptionChain(chain); err != nil {
		return result, err
	}
	if id <= 0 {
		return result, fmt.Errorf("subscription outbox identity must be positive")
	}
	db, err := a.requireDB()
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = db.QueryRowContext(ctx, `SELECT j.chain_id,j.id,o.topic,j.generation,COALESCE(g.generation,0),o.payload::text
FROM reorg_outbox o JOIN reorg_jobs j ON j.id=o.job_id
LEFT JOIN reorg_generations g ON g.chain_id=j.chain_id WHERE o.id=$1 AND j.chain_id=$2`, id, chain).
		Scan(&result.ChainID, &result.JobID, &result.Kind, &result.Generation, &result.CurrentGeneration, &result.Payload)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("read subscription durable outbox: %w", err)
	}
	result.Found = true
	return result, nil
}

func (a *PostgreSQLPluginAdapter) QuarantineSubscription(ctx context.Context, message ports.SubscriptionDeliveryMessage, reason string) error {
	if len(message.Payload) > 8<<20 || message.Topic == "" || len(message.Topic) > 249 || reason == "" || len(reason) > 512 {
		return fmt.Errorf("subscription quarantine exceeds its input bounds")
	}
	db, err := a.requireDB()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode subscription quarantine: %w", err)
	}
	digest := sha256.Sum256(payload)
	identity := hex.EncodeToString(digest[:])
	chain := message.PartitionKey
	if a.subscriptionChain(chain) != nil {
		chain = "unknown"
	}
	original := message.ID
	if len(original) > 255 || !utf8.ValidString(original) {
		original = identity
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = db.ExecContext(ctx, `INSERT INTO dlq_events(id,chain_id,original_event_id,error_message,status,payload)
VALUES($1,$2,$3,$4,'subscription_pending',$5) ON CONFLICT(id) DO NOTHING`, "subscription:"+identity, chain, original, reason, string(payload))
	if err != nil {
		return fmt.Errorf("persist subscription quarantine record: %w", err)
	}
	return nil
}
