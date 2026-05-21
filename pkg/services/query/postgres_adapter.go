package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

type postgresConnectionProvider interface {
	GetPostgresDB(ctx context.Context) (any, error)
}

// DefaultPostgreSQLAdapter provides PostgreSQL query operations
type DefaultPostgreSQLAdapter struct {
	initMu           sync.Mutex
initialized      atomic.Bool
	dbManager        postgresConnectionProvider
	db               *sql.DB
	logger           core.Logger
	metricsCollector core.MetricsCollector
}

var postgresIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// camelCase to snake_case column mapping for the events table
var pgColumnMap = map[string]string{
	"id":               "id",
	"chainId":          "chain_id",
	"chain_id":         "chain_id",
	"blockNumber":      "block_number",
	"block_number":     "block_number",
	"blockHash":        "block_hash",
	"block_hash":       "block_hash",
	"transactionHash":  "transaction_hash",
	"transaction_hash": "transaction_hash",
	"logIndex":         "log_index",
	"log_index":        "log_index",
	"contractAddress":  "contract_address",
	"contract_address": "contract_address",
	"eventName":        "event_name",
	"event_name":       "event_name",
	"eventSignature":   "event_name",
	"eventData":        "event_data",
	"event_data":       "event_data",
	"timestamp":        "timestamp",
	"createdAt":        "created_at",
	"created_at":       "created_at",
	"status":           "status",
	"eventTopic":       "event_name",
	"eventHash":        "id",
}

func resolveColumn(key string) string {
	if col, ok := pgColumnMap[key]; ok {
		return col
	}
	return key
}

// buildPostgresFilter recursively builds a WHERE clause from a filter map,
// supporting MongoDB-style operators ($or, $gte, $lte, $gt, $lt, $in, $regex, $ne).
func buildPostgresFilter(filter map[string]any) (string, []any, error) {
	idx := 1
	clause, args, err := buildPostgresFilterHelper(filter, &idx)
	if err != nil {
		return "", nil, err
	}
	return clause, args, nil
}

// buildPostgresFilterHelper builds filter conditions with shared arg indexing.
// Returns clause with $1..$N placeholders and the corresponding args slice.
// The idx pointer tracks the next available arg number across recursive calls.
func buildPostgresFilterHelper(filter map[string]any, idx *int) (string, []any, error) {
	var conditions []string
	var args []any

	var buildCond func(key string, val any) (string, []any, error)
	buildCond = func(key string, val any) (string, []any, error) {
		col := resolveColumn(key)
		if !isSafePostgresIdentifier(col) {
			return "", nil, fmt.Errorf("invalid filter field %q", key)
		}

		switch v := val.(type) {
		case map[string]any:
			var subConds []string
			var subArgs []any
			for op, opVal := range v {
				switch op {
				case "$gte":
					subConds = append(subConds, col+" >= $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				case "$gt":
					subConds = append(subConds, col+" > $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				case "$lte":
					subConds = append(subConds, col+" <= $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				case "$lt":
					subConds = append(subConds, col+" < $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				case "$in":
					arr, ok := opVal.([]any)
					if !ok {
						return "", nil, fmt.Errorf("$in requires an array for field %q", key)
					}
					placeholders := make([]string, len(arr))
					for i, elem := range arr {
						placeholders[i] = "$" + strconv.Itoa(*idx)
						subArgs = append(subArgs, elem)
						*idx++
					}
					subConds = append(subConds, col+" IN ("+strings.Join(placeholders, ",")+")")
				case "$regex":
					subConds = append(subConds, col+" ~* $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				case "$options":
					continue
				case "$ne":
					subConds = append(subConds, col+" != $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				default:
					subConds = append(subConds, col+" = $"+strconv.Itoa(*idx))
					subArgs = append(subArgs, opVal)
					*idx++
				}
			}
			return strings.Join(subConds, " AND "), subArgs, nil

		case []any:
			return "", nil, fmt.Errorf("unexpected array value for field %q", key)

		default:
			clause := col + " = $" + strconv.Itoa(*idx)
			*idx++
			return clause, []any{val}, nil
		}
	}

	for k, v := range filter {
		switch k {
		case "$or":
			arr, ok := v.([]any)
			if !ok {
				return "", nil, fmt.Errorf("$or requires an array")
			}
			var orConds []string
			for _, elem := range arr {
				elemMap, ok := elem.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("$or element must be a filter object")
				}
				subClause, subArgs, err := buildPostgresFilterHelper(elemMap, idx)
				if err != nil {
					return "", nil, err
				}
				orConds = append(orConds, "("+subClause+")")
				args = append(args, subArgs...)
			}
			conditions = append(conditions, "("+strings.Join(orConds, " OR ")+")")

		case "$and":
			arr, ok := v.([]any)
			if !ok {
				return "", nil, fmt.Errorf("$and requires an array")
			}
			var andConds []string
			for _, elem := range arr {
				elemMap, ok := elem.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("$and element must be a filter object")
				}
				subClause, subArgs, err := buildPostgresFilterHelper(elemMap, idx)
				if err != nil {
					return "", nil, err
				}
				andConds = append(andConds, "("+subClause+")")
				args = append(args, subArgs...)
			}
			conditions = append(conditions, "("+strings.Join(andConds, " AND ")+")")

		default:
			if strings.HasPrefix(k, "$") {
				continue
			}
			if !isSafePostgresIdentifier(k) {
				return "", nil, fmt.Errorf("invalid filter field %q", k)
			}
			cond, subArgs, err := buildCond(k, v)
			if err != nil {
				return "", nil, err
			}
			conditions = append(conditions, cond)
			args = append(args, subArgs...)
		}
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}
	return strings.Join(conditions, " AND "), args, nil
}

// NewPostgreSQLAdapter creates a new PostgreSQL adapter
func NewPostgreSQLAdapter(
	dbManager postgresConnectionProvider,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) PostgreSQLAdapter {
	return &DefaultPostgreSQLAdapter{
		dbManager:        dbManager,
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the PostgreSQL adapter
func (pa *DefaultPostgreSQLAdapter) Initialize(ctx context.Context) error {
	pa.initMu.Lock()
	defer pa.initMu.Unlock()

	if pa.initialized.Load() {
		return fmt.Errorf("PostgreSQL adapter already initialized")
	}

	// Get PostgreSQL connection from database manager
	db, err := pa.dbManager.GetPostgresDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get PostgreSQL connection: %w", err)
	}

	if db == nil {
		return fmt.Errorf("PostgreSQL connection is nil")
	}

	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return fmt.Errorf("expected *sql.DB, got %T", db)
	}
	pa.db = sqlDB
	pa.initialized.Store(true)

	pa.logger.Info("PostgreSQL adapter initialized", core.LogKeyComponent, "postgres-adapter")

	return nil
}

// pgSelectColumns maps PostgreSQL table columns to Scan parameter positions.
// Table: id, chain_id, block_number, block_hash, transaction_hash, log_index,
//        contract_address, event_name, event_data, timestamp, created_at
// Scan:  EventHash, BlockNumber, TransactionHash, LogIndex, ContractAddress,
//        EventTopic, EventData, BlockTimestamp, ChainID
const pgSelectColumns = "id AS event_hash, block_number, transaction_hash, log_index, contract_address, event_name AS event_topic, event_data, timestamp AS block_timestamp, chain_id"

// Query executes a query against PostgreSQL
func (pa *DefaultPostgreSQLAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {

	if !pa.initialized.Load() {
		return nil, fmt.Errorf("PostgreSQL adapter not initialized")
	}

	if pa.db == nil {
		return nil, fmt.Errorf("PostgreSQL database connection is nil")
	}

	if req == nil {
		return nil, fmt.Errorf("query request is required")
	}

	if req.Collection == "" {
		return nil, fmt.Errorf("table name is required")
	}
	if !isSafePostgresIdentifier(req.Collection) {
		return nil, fmt.Errorf("invalid table name %q", req.Collection)
	}

	start := time.Now()

	whereClause := ""
	args := []any{}

	if req.Filter != nil {
		clause, filterArgs, err := buildPostgresFilter(req.Filter)
		if err != nil {
			return nil, fmt.Errorf("PostgreSQL filter build failed: %w", err)
		}
		if clause != "" {
			whereClause = "WHERE " + clause
			args = filterArgs
		}
	}

	// Build ORDER BY clause
	orderClause := ""
	if req.Sort != nil {
		orders := []string{}
		for k, v := range req.Sort {
			col := resolveColumn(k)
			if !isSafePostgresIdentifier(col) {
				return nil, fmt.Errorf("invalid sort field %q", k)
			}
			direction := "ASC"
			if v < 0 {
				direction = "DESC"
			}
			orders = append(orders, col+" "+direction)
		}
		orderClause = "ORDER BY " + strings.Join(orders, ", ")
	}

	// Build LIMIT and OFFSET
	limitClause := ""
	if req.Limit > 0 {
		limitClause = "LIMIT " + strconv.FormatInt(req.Limit, 10)
		if req.Offset > 0 {
			limitClause += " OFFSET " + strconv.FormatInt(req.Offset, 10)
		}
	}

	// Build query
	query := strings.TrimSpace(strings.Join([]string{
		"SELECT " + pgSelectColumns + " FROM " + req.Collection,
		whereClause,
		orderClause,
		limitClause,
	}, " "))

	// Execute query
	rows, err := pa.db.QueryContext(ctx, query, args...)
	if err != nil {
		duration := time.Since(start).Milliseconds()
		pa.metricsCollector.RecordCounter("postgres_query_error", 1, map[string]string{})
		pa.logger.Error("PostgreSQL query failed", "table", req.Collection, core.LogKeyError, err, core.LogKeyDuration, duration)
		return nil, fmt.Errorf("PostgreSQL query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Scan results
	var events []core.BlockchainEvent
	for rows.Next() {
		var event core.BlockchainEvent
		// Note: This is a simplified scan. In production, you'd need to map columns properly
		if err := rows.Scan(
			&event.EventHash,
			&event.BlockNumber,
			&event.TransactionHash,
			&event.LogIndex,
			&event.ContractAddress,
			&event.EventTopic,
			&event.EventData,
			&event.BlockTimestamp,
			&event.ChainID,
		); err != nil {
			pa.logger.Error("Failed to scan PostgreSQL row", core.LogKeyError, err)
			continue
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		duration := time.Since(start).Milliseconds()
		pa.metricsCollector.RecordCounter("postgres_scan_error", 1, map[string]string{})
		pa.logger.Error("PostgreSQL scan error", "table", req.Collection, core.LogKeyError, err, core.LogKeyDuration, duration)
		return nil, fmt.Errorf("PostgreSQL scan failed: %w", err)
	}

	// Get total count
	countQuery := strings.TrimSpace(strings.Join([]string{
		"SELECT COUNT(*) FROM " + req.Collection,
		whereClause,
	}, " "))

	var total int64
	if err := pa.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		pa.logger.Error("Failed to count PostgreSQL rows", "table", req.Collection, core.LogKeyError, err)
		total = int64(len(events))
	}

	duration := time.Since(start).Milliseconds()
	pa.metricsCollector.RecordHistogram("postgres_query_time_ms", float64(duration), map[string]string{})
	pa.metricsCollector.RecordCounter("postgres_query_success", 1, map[string]string{})

	pa.logger.Info("PostgreSQL query successful", "table", req.Collection, core.LogKeyCount, len(events), "total", total, core.LogKeyDuration, duration)

	return &QueryResult{
		Events:       events,
		Total:        total,
		ResponseTime: duration,
		Source:       "postgresql",
	}, nil
}

func isSafePostgresIdentifier(identifier string) bool {
	return postgresIdentifierPattern.MatchString(identifier)
}

// QueryByHash retrieves a single item by hash
func (pa *DefaultPostgreSQLAdapter) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {

	if !pa.initialized.Load() {
		return nil, fmt.Errorf("PostgreSQL adapter not initialized")
	}

	if pa.db == nil {
		return nil, fmt.Errorf("PostgreSQL database connection is nil")
	}

	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	// Build query with prepared statement
	query := "SELECT event_hash, block_number, transaction_hash, log_index, contract_address, event_topic, event_data, block_timestamp, chain_id FROM events WHERE event_hash = $1"

	// Execute query
	var event core.BlockchainEvent
	err := pa.db.QueryRowContext(ctx, query, hash).Scan(
		&event.EventHash,
		&event.BlockNumber,
		&event.TransactionHash,
		&event.LogIndex,
		&event.ContractAddress,
		&event.EventTopic,
		&event.EventData,
		&event.BlockTimestamp,
		&event.ChainID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			duration := time.Since(start).Milliseconds()
			pa.logger.Debug("Event not found in PostgreSQL", core.LogKeyHash, hash, core.LogKeyDuration, duration)
			return nil, nil
		}

		duration := time.Since(start).Milliseconds()
		pa.metricsCollector.RecordCounter("postgres_query_by_hash_error", 1, map[string]string{})
		pa.logger.Error("PostgreSQL query by hash failed", core.LogKeyHash, hash, core.LogKeyError, err, core.LogKeyDuration, duration)
		return nil, fmt.Errorf("PostgreSQL query failed: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	pa.metricsCollector.RecordHistogram("postgres_query_by_hash_time_ms", float64(duration), map[string]string{})
	pa.metricsCollector.RecordCounter("postgres_query_by_hash_success", 1, map[string]string{})

	pa.logger.Info("PostgreSQL query by hash successful", core.LogKeyHash, hash, core.LogKeyDuration, duration)

	return &event, nil
}

// Health returns the health status
func (pa *DefaultPostgreSQLAdapter) Health(ctx context.Context) *core.HealthStatus {

	if !pa.initialized.Load() {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "PostgreSQL adapter not initialized",
		}
	}

	// Ping PostgreSQL
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pa.db.PingContext(pingCtx); err != nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("PostgreSQL ping failed: %v", err),
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "PostgreSQL adapter healthy",
	}
}
