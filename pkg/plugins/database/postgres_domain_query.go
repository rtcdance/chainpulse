package database

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/rtcdance/chainpulse/pkg/ports"

	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

// QueryDomainEvents serves the deployed API directly from canonical rows.
// Filtering, counting and paging share one repeatable-read database snapshot.
func (a *PostgreSQLPluginAdapter) QueryDomainEvents(ctx context.Context, req *domainquery.Request) (*domainquery.Result, error) {
	if req == nil || req.Limit < 1 || req.Limit > 1000 || req.Offset < 0 || req.Offset > 100000 {
		return nil, fmt.Errorf("query requires limit 1..1000 and offset 0..100000")
	}
	scope := a.ChainScope()
	filter := make(map[string]any, len(req.Filter)+1)
	for k, v := range req.Filter {
		filter[k] = v
	}
	if _, network := filter["network"]; !network {
		if _, chain := filter["chainId"]; !chain {
			if len(scope) != 1 {
				return nil, fmt.Errorf("an explicit chain filter is required")
			}
			filter["chainId"] = scope[0]
		}
	}
	var args []any
	where, err := domainPredicate(filter, &args, 0)
	if err != nil {
		return nil, err
	}
	if len(scope) > 0 {
		args = append(args, pq.Array(scope))
		where += fmt.Sprintf(" AND chain_id=ANY($%d)", len(args))
	}
	if _, explicitStatus := filter["status"]; !explicitStatus {
		where += " AND status<>'reorged' AND event_state->>'canonicality'='canonical'"
	}
	db, err := a.requireDB()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin query snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	base, cursor, err := prepareQueryCursor(ctx, tx, req.Cursor, queryCursorChain(filter, scope), filter, req.Offset)
	if err != nil {
		return nil, err
	}
	cache, cacheKey, err := a.aggregateKey(ctx, tx, "count", filter, scope)
	if err != nil {
		return nil, err
	}
	total, err := a.canonicalCount(ctx, tx, cache, cacheKey, where, args)
	if err != nil {
		return nil, err
	}
	pageWhere := where
	pageArgs := append([]any(nil), args...)
	if cursor != nil {
		anchorArgs := append([]any(nil), args...)
		start := len(anchorArgs) + 1
		anchorArgs = append(anchorArgs, cursor.Block, cursor.Log, cursor.ID)
		position := fmt.Sprintf("(block_number,log_index,id)=($%d,$%d,$%d)", start, start+1, start+2)
		var present bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM blockchain_events WHERE "+where+" AND "+position+")", anchorArgs...).Scan(&present); err != nil {
			return nil, fmt.Errorf("verify cursor anchor: %w", err)
		}
		if !present {
			return nil, &ports.QueryCursorError{Stale: true, Reason: "canonical cursor anchor changed; restart pagination"}
		}
		pageWhere += fmt.Sprintf(" AND (block_number,log_index,id)<($%d,$%d,$%d)", start, start+1, start+2)
		pageArgs = append(pageArgs, cursor.Block, cursor.Log, cursor.ID)
	}
	query := "SELECT " + selectEventColumns + " FROM blockchain_events WHERE " + pageWhere + fmt.Sprintf(" ORDER BY block_number DESC,log_index DESC,id DESC LIMIT $%d OFFSET $%d", len(pageArgs)+1, len(pageArgs)+2)
	if req.Offset > 0 {
		// Skip narrow index entries, fetching payloads only for the selected page.
		ids := "SELECT id,block_number,log_index FROM blockchain_events WHERE " + pageWhere + fmt.Sprintf(" ORDER BY block_number DESC,log_index DESC,id DESC LIMIT $%d OFFSET $%d", len(pageArgs)+1, len(pageArgs)+2)
		columns := "event." + strings.ReplaceAll(selectEventColumns, ", ", ", event.")
		query = "SELECT " + columns + " FROM (" + ids + ") page JOIN blockchain_events event ON event.id=page.id ORDER BY page.block_number DESC,page.log_index DESC,page.id DESC"
	}
	pageArgs = append(pageArgs, req.Limit+1, req.Offset)
	rows, err := tx.QueryContext(ctx, query, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("query canonical page: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := &domainquery.Result{Total: total, Source: "postgres-canonical"}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan canonical page: %w", err)
		}
		result.Events = append(result.Events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read canonical page: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close canonical page: %w", err)
	}
	if err := projectCanonicalFinality(ctx, tx, result.Events); err != nil {
		return nil, err
	}

	hasMore := len(result.Events) > int(req.Limit)
	if hasMore {
		result.Events = result.Events[:req.Limit]
	}
	if base.Chain != "" {
		for i := range result.Events {
			encoded, err := encodeQueryCursor(base, &result.Events[i])
			if err != nil {
				return nil, err
			}
			result.Cursors = append(result.Cursors, encoded)
		}
		if hasMore && len(result.Cursors) > 0 {
			result.NextCursor = result.Cursors[len(result.Cursors)-1]
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("finish query snapshot: %w", err)
	}
	return result, nil
}

func domainPredicate(filter map[string]any, args *[]any, depth int) (string, error) {
	if depth > 3 || len(filter) > 12 {
		return "", fmt.Errorf("filter is too complex")
	}
	keys := make([]string, 0, len(filter))
	for key := range filter {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{"TRUE"}
	for _, key := range keys {
		value := filter[key]
		if key == "$or" {
			options, ok := value.([]any)
			if !ok || len(options) == 0 || len(options) > 8 {
				return "", fmt.Errorf("invalid OR filter")
			}
			var alternatives []string
			for _, option := range options {
				m, ok := option.(map[string]any)
				if !ok {
					return "", fmt.Errorf("invalid OR branch")
				}
				expr, err := domainPredicate(m, args, depth+1)
				if err != nil {
					return "", err
				}
				alternatives = append(alternatives, "("+expr+")")
			}
			parts = append(parts, "("+strings.Join(alternatives, " OR ")+")")
			continue
		}
		column := ""
		switch key {
		case "chainId", "network":
			column = "chain_id"
		case "contractAddress":
			column = "lower(contract_address)"
		case "transactionHash":
			column = "transaction_hash"
		case "eventName":
			column = "event_name"
		case "eventSignature":
			column = "event_data->>'signature'"
		case "status":
			column = "status"
		case "blockNumber":
			column = "block_number"
		case "timestamp":
			column = "timestamp"
		case "logIndex":
			column = "log_index"
		case "correlationId":
			column = "correlation_id"
		case "id", "eventHash":
			column = "id"
		default:
			return "", fmt.Errorf("unsupported filter %q", key)
		}
		bind := func(v any) string { *args = append(*args, v); return fmt.Sprintf("$%d", len(*args)) }
		if ops, ok := value.(map[string]any); ok {
			if regex, ok := ops["$regex"].(string); ok {
				if len(regex) > 256 || key == "blockNumber" || key == "timestamp" {
					return "", fmt.Errorf("invalid search filter")
				}
				parts = append(parts, column+" ~* "+bind(regex))
				continue
			}
			if values, ok := ops["$in"].([]any); ok {
				if len(values) == 0 || len(values) > 50 {
					return "", fmt.Errorf("invalid membership filter")
				}
				// Deduplicate bound values. The chain-alias expansion in
				// event_query_handler emits {chainID, strconv.Itoa(chainID),
				// ResolveChainName(chainID)}, which for an unknown chain are
				// three identical text values: `chain_id IN ('31337','31337',
				// '31337')`. PostgreSQL then refuses the ordered index-only
				// walk on idx_events_canonical_chain_page (multi-valued array
				// on the leading key), sorts every canonical row of the chain
				// instead, and spills it: measured 2026-10-16 on chain 31337
				// at 123,675 rows, external merge Disk=147424kB, 762 ms for
				// the page query versus 6.6 ms with one equality — the cause
				// of T14 attempt 4's list/cursor/contract latency failure. An
				// IN list matches no more than its unique values, so collapsing
				// duplicates (order preserved, args stay deterministic) cannot
				// change any result set.
				seen := make(map[string]struct{}, len(values))
				var placeholders []string
				for _, value := range values {
					text := fmt.Sprint(value)
					if _, dup := seen[text]; dup {
						continue
					}
					seen[text] = struct{}{}
					placeholders = append(placeholders, bind(text))
				}
				if len(placeholders) == 1 {
					parts = append(parts, column+"="+placeholders[0])
				} else {
					parts = append(parts, column+" IN ("+strings.Join(placeholders, ",")+")")
				}
				continue
			}
			for _, op := range []string{"$gte", "$lte", "$lt"} {
				if v, exists := ops[op]; exists {
					operator := ">="
					if op == "$lte" {
						operator = "<="
					}
					if op == "$lt" {
						operator = "<"
					}
					parts = append(parts, column+operator+bind(v))
				}
			}
			if len(ops) == 0 {
				return "", fmt.Errorf("empty filter operators")
			}
			for op := range ops {
				if op != "$gte" && op != "$lte" && op != "$lt" {
					return "", fmt.Errorf("unsupported filter operator %q", op)
				}
			}
		} else {
			if key == "contractAddress" {
				value = strings.ToLower(fmt.Sprint(value))
			}
			if key == "chainId" || key == "network" {
				value = fmt.Sprint(value)
			}
			parts = append(parts, column+"="+bind(value))
		}
	}
	return strings.Join(parts, " AND "), nil
}
