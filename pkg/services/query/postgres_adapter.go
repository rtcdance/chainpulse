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
	"time"

	"chainpulse/pkg/core"
)

type postgresConnectionProvider interface {
	GetPostgresDB(ctx context.Context) (any, error)
}

// DefaultPostgreSQLAdapter provides PostgreSQL query operations
type DefaultPostgreSQLAdapter struct {
	mu               sync.RWMutex
	dbManager        postgresConnectionProvider
	db               *sql.DB
	logger           core.Logger
	metricsCollector core.MetricsCollector
	initialized      bool
}

var postgresIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

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
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if pa.initialized {
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
	pa.initialized = true

	pa.logger.Info("PostgreSQL adapter initialized", core.LogKeyComponent, "postgres-adapter")

	return nil
}

// Query executes a query against PostgreSQL
func (pa *DefaultPostgreSQLAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if !pa.initialized {
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

	// Build WHERE clause from filter
	whereClause := ""
	args := []any{}
	argIndex := 1

	if req.Filter != nil {
		conditions := []string{}
		for k, v := range req.Filter {
			if !isSafePostgresIdentifier(k) {
				return nil, fmt.Errorf("invalid filter field %q", k)
			}
			conditions = append(conditions, k+" = $"+strconv.Itoa(argIndex))
			args = append(args, v)
			argIndex++
		}
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderClause := ""
	if req.Sort != nil {
		orders := []string{}
		for k, v := range req.Sort {
			if !isSafePostgresIdentifier(k) {
				return nil, fmt.Errorf("invalid sort field %q", k)
			}
			direction := "ASC"
			if v < 0 {
				direction = "DESC"
			}
			orders = append(orders, k+" "+direction)
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
		"SELECT * FROM " + req.Collection,
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
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if !pa.initialized {
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
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if !pa.initialized {
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
