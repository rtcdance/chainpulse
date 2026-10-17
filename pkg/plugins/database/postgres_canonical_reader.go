package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

// GetCanonicalEvent serves physical IDs within declared chains. Transaction
// lookup requires a single chain because an EVM tx can appear on two chains.
func (a *PostgreSQLPluginAdapter) GetCanonicalEvent(ctx context.Context, id string) (*blockchain.BlockchainEvent, error) {
	scope := a.ChainScope()
	if id == "" || len(scope) == 0 {
		return nil, fmt.Errorf("event identity and declared chain scope are required")
	}
	filter := map[string]any{"id": id}
	if len(scope) == 1 {
		filter = map[string]any{"chainId": scope[0], "$or": []any{map[string]any{"id": id}, map[string]any{"transactionHash": id}}}
	} else {
		if strings.HasPrefix(id, "0x") {
			return nil, fmt.Errorf("transaction lookup requires an explicit single chain")
		}
		ids := make([]any, len(scope))
		for i, chain := range scope {
			ids[i] = chain
		}
		filter["chainId"] = map[string]any{"$in": ids}
	}
	result, err := a.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("query canonical event identity: %w", err)
	}
	if len(result.Events) == 0 {
		return nil, nil
	}
	return &result.Events[0], nil
}

// CanonicalEventStats returns all dashboard counts from one bounded snapshot.
// It never materializes event rows or exposes chains outside declared scope.
func (a *PostgreSQLPluginAdapter) CanonicalEventStats(ctx context.Context) (int64, map[string]int64, map[string]int64, int64, error) {
	scope := a.ChainScope()
	if len(scope) == 0 || len(scope) > 50 {
		return 0, nil, nil, 0, fmt.Errorf("bounded declared chain scope is required for stats")
	}
	db, err := a.requireDB()
	if err != nil {
		return 0, nil, nil, 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return 0, nil, nil, 0, fmt.Errorf("begin canonical stats snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	cache, cacheKey, err := a.aggregateKey(ctx, tx, "dashboard", nil, scope)
	if err != nil {
		return 0, nil, nil, 0, err
	}
	stats, err := a.canonicalStatsFromCache(ctx, tx, cache, cacheKey, scope)
	if err != nil {
		return 0, nil, nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("finish canonical stats snapshot: %w", err)
	}
	return stats.Total, stats.ByChain, stats.ByName, stats.Reorged, nil
}

func readCanonicalGroupedStats(ctx context.Context, tx *sql.Tx, scope []string) (canonicalStats, error) {
	stats := canonicalStats{ByChain: make(map[string]int64), ByName: make(map[string]int64)}
	// The partial-index predicate stays in WHERE. GROUPING SETS computes the
	// total and both dimensions in one canonical scan, including an empty total.
	rows, err := tx.QueryContext(ctx, `SELECT chain_id,event_name,GROUPING(chain_id,event_name),count(*)
FROM blockchain_events WHERE chain_id=ANY($1) AND status<>'reorged' AND event_state->>'canonicality'='canonical'
GROUP BY GROUPING SETS ((chain_id),(event_name),()) LIMIT 1076`, pq.Array(scope))
	if err != nil {
		return stats, fmt.Errorf("aggregate canonical stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var chain, name sql.NullString
		var grouping int
		var count int64
		if err := rows.Scan(&chain, &name, &grouping, &count); err != nil {
			return stats, fmt.Errorf("scan canonical stats: %w", err)
		}
		switch grouping {
		case 1:
			if !chain.Valid {
				return stats, fmt.Errorf("canonical stats chain is NULL")
			}
			stats.ByChain[chain.String] = count
		case 2:
			if !name.Valid {
				return stats, fmt.Errorf("canonical stats event name is NULL")
			}
			stats.ByName[name.String] = count
		case 3:
			stats.Total = count
		default:
			return stats, fmt.Errorf("unexpected canonical stats grouping")
		}
		if len(stats.ByName) > 1024 || len(stats.ByChain) > 50 {
			return stats, fmt.Errorf("canonical stats group cardinality exceeds bound")
		}
	}
	if err := rows.Err(); err != nil {
		return stats, fmt.Errorf("read canonical stats: %w", err)
	}
	if err := rows.Close(); err != nil {
		return stats, fmt.Errorf("close canonical stats: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id=ANY($1) AND status='reorged'`, pq.Array(scope)).Scan(&stats.Reorged); err != nil {
		return stats, fmt.Errorf("count reorged stats: %w", err)
	}
	if !stats.valid(scope) {
		return stats, fmt.Errorf("canonical stats aggregates are inconsistent")
	}
	return stats, nil
}

func (a *PostgreSQLPluginAdapter) CountCanonicalEvents(ctx context.Context) (int64, error) {
	scope := a.ChainScope()
	if len(scope) == 0 || len(scope) > 50 {
		return 0, fmt.Errorf("bounded declared chain scope is required")
	}
	db, err := a.requireDB()
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int64
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM blockchain_events WHERE chain_id=ANY($1) AND status<>'reorged' AND event_state->>'canonicality'='canonical'`, pq.Array(scope)).Scan(&count); err != nil {
		return 0, fmt.Errorf("count canonical events: %w", err)
	}
	return count, nil
}

func (a *PostgreSQLPluginAdapter) GetCanonicalEventsPaginated(ctx context.Context, cursor string, limit int) ([]*blockchain.BlockchainEvent, bool, error) {
	if len(a.ChainScope()) != 1 {
		return nil, false, fmt.Errorf("cursor page requires a declared single chain")
	}
	filter := map[string]any{}
	offset := int64(0)
	if cursor != "" {
		if page, ok := domainquery.DecodePageCursor(cursor); ok {
			anchor, err := a.GetCanonicalEvent(ctx, page.EventID)
			if err != nil {
				return nil, false, err
			}
			if anchor == nil || anchor.BlockNumber != page.BlockNumber || anchor.LogIndex != page.LogIndex {
				return nil, false, fmt.Errorf("cursor no longer identifies a canonical event")
			}
			filter["$or"] = []any{map[string]any{"blockNumber": map[string]any{"$lt": page.BlockNumber}}, map[string]any{"blockNumber": page.BlockNumber, "logIndex": map[string]any{"$lt": page.LogIndex}}, map[string]any{"blockNumber": page.BlockNumber, "logIndex": page.LogIndex, "id": map[string]any{"$lt": page.EventID}}}
		} else {
			var err error
			offset, err = strconv.ParseInt(cursor, 10, 64)
			if err != nil {
				return nil, false, fmt.Errorf("invalid legacy cursor: %w", err)
			}
		}
	}
	result, err := a.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: int64(limit), Offset: offset})
	if err != nil {
		return nil, false, err
	}
	events := make([]*blockchain.BlockchainEvent, len(result.Events))
	for i := range result.Events {
		events[i] = &result.Events[i]
	}
	return events, result.Total > offset+int64(len(events)), nil
}
