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
	var total, reorged int64
	err = tx.QueryRowContext(ctx, `SELECT count(*) FILTER (WHERE status<>'reorged' AND event_state->>'canonicality'='canonical'),count(*) FILTER (WHERE status='reorged') FROM blockchain_events WHERE chain_id=ANY($1)`, pq.Array(scope)).Scan(&total, &reorged)
	if err != nil {
		return 0, nil, nil, 0, fmt.Errorf("count canonical stats: %w", err)
	}
	groups := make([]map[string]int64, 0, 2)
	for _, column := range []string{"chain_id", "event_name"} {
		// Column names come from this fixed list, never from request input.
		rows, err := tx.QueryContext(ctx, `SELECT `+column+`,count(*) FROM blockchain_events WHERE chain_id=ANY($1) AND status<>'reorged' AND event_state->>'canonicality'='canonical' GROUP BY `+column+` LIMIT 1025`, pq.Array(scope))
		if err != nil {
			return 0, nil, nil, 0, fmt.Errorf("aggregate canonical stats: %w", err)
		}
		group := map[string]int64{}
		for rows.Next() {
			var key string
			var count int64
			if err := rows.Scan(&key, &count); err != nil {
				_ = rows.Close()
				return 0, nil, nil, 0, fmt.Errorf("scan canonical stats: %w", err)
			}
			group[key] = count
		}
		err = rows.Err()
		closeErr := rows.Close()
		if err != nil {
			return 0, nil, nil, 0, fmt.Errorf("read canonical stats: %w", err)
		}
		if closeErr != nil {
			return 0, nil, nil, 0, fmt.Errorf("close canonical stats: %w", closeErr)
		}
		if len(group) > 1024 {
			return 0, nil, nil, 0, fmt.Errorf("canonical stats group cardinality exceeds bound")
		}
		groups = append(groups, group)
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, nil, 0, fmt.Errorf("finish canonical stats snapshot: %w", err)
	}
	return total, groups[0], groups[1], reorged, nil
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
