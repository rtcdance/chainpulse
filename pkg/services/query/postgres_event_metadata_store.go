package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// PostgreSQLEventMetadataStore implements EventMetadataStore for PostgreSQL
type PostgreSQLEventMetadataStore struct {
	dbManager   postgresConnectionProvider
	logger      core.Logger
	metrics     core.MetricsCollector
	db          *sql.DB
	initialized bool
	// Prepared statements for repeated queries
	insertStmt *sql.Stmt
	updateStmt *sql.Stmt
	getStmt    *sql.Stmt
}

// NewPostgreSQLEventMetadataStore creates a new PostgreSQL event metadata store
func NewPostgreSQLEventMetadataStore(
	dbManager postgresConnectionProvider,
	logger core.Logger,
	metrics core.MetricsCollector,
) *PostgreSQLEventMetadataStore {
	return &PostgreSQLEventMetadataStore{
		dbManager:   dbManager,
		logger:      logger,
		metrics:     metrics,
		initialized: false,
	}
}

// Initialize initializes the PostgreSQL event metadata store
func (s *PostgreSQLEventMetadataStore) Initialize(ctx context.Context) error {
	if s.initialized {
		return nil
	}

	// Get PostgreSQL connection
	db, err := s.dbManager.GetPostgresDB(ctx)
	if err != nil {
		s.logger.Error("Failed to get PostgreSQL connection", "error", err)
		return fmt.Errorf("failed to get PostgreSQL connection: %w", err)
	}

	if db == nil {
		return fmt.Errorf("PostgreSQL connection is nil")
	}

	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return fmt.Errorf("expected *sql.DB, got %T", db)
	}
	s.db = sqlDB

	// Create table if it doesn't exist
	if err := s.createTable(ctx); err != nil {
		s.logger.Error("Failed to create table", "error", err)
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Prepare frequently-used statements for performance
	if err := s.prepareStatements(ctx); err != nil {
		s.logger.Error("Failed to prepare statements", "error", err)
		return fmt.Errorf("failed to prepare statements: %w", err)
	}

	s.initialized = true
	s.logger.Info("PostgreSQL event metadata store initialized")
	return nil
}

// createTable creates the events_metadata table if it doesn't exist
func (s *PostgreSQLEventMetadataStore) createTable(ctx context.Context) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS events_metadata (
		id BIGSERIAL PRIMARY KEY,
		event_id VARCHAR(255) UNIQUE NOT NULL,
		chain_id INTEGER NOT NULL,
		block_number BIGINT NOT NULL,
		transaction_hash VARCHAR(255) NOT NULL,
		log_index INTEGER NOT NULL,
		contract_address VARCHAR(255) NOT NULL,
		event_name VARCHAR(255) NOT NULL,
		processing_status VARCHAR(50) NOT NULL DEFAULT 'pending',
		processing_error TEXT,
		retry_count INTEGER NOT NULL DEFAULT 0,
		last_retry_at TIMESTAMP,
		processed_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := s.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		if isIgnorablePostgresSchemaConflict(err) {
			// Concurrent schema creation can still surface transient duplicate catalog
			// conflicts even when IF NOT EXISTS is used. Treat those as success.
			return nil
		}
		return fmt.Errorf("failed to create events_metadata table: %w", err)
	}

	// Create indexes
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_events_metadata_chain_block
	ON events_metadata(chain_id, block_number DESC);

	CREATE INDEX IF NOT EXISTS idx_events_metadata_contract_event
	ON events_metadata(contract_address, event_name);

	CREATE INDEX IF NOT EXISTS idx_events_metadata_processed_at
	ON events_metadata(processed_at DESC);

	CREATE INDEX IF NOT EXISTS idx_events_metadata_status
	ON events_metadata(processing_status);
	`

	_, err = s.db.ExecContext(ctx, createIndexSQL)
	if err != nil {
		if isIgnorablePostgresSchemaConflict(err) {
			return nil
		}
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Add unique constraint on natural key to prevent duplicate indexing
	alterSQL := `
	ALTER TABLE events_metadata
	    ADD CONSTRAINT uq_events_metadata_natural_key
	    UNIQUE (chain_id, block_number, transaction_hash, log_index);
	`
	_, err = s.db.ExecContext(ctx, alterSQL)
	if err != nil {
		if isIgnorablePostgresSchemaConflict(err) || strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to add natural key constraint: %w", err)
	}

	return nil
}

func isIgnorablePostgresSchemaConflict(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "pg_type_typname_nsp_index") ||
		strings.Contains(message, "pg_class_relname_nsp_index")
}

// prepareStatements creates prepared statements for frequently-used queries.
// Prepared statements reduce per-query parsing overhead in PostgreSQL.
func (s *PostgreSQLEventMetadataStore) prepareStatements(ctx context.Context) error {
	var err error

	s.insertStmt, err = s.db.PrepareContext(ctx, `
		INSERT INTO events_metadata (
			event_id, chain_id, block_number, transaction_hash, log_index,
			contract_address, event_name, processing_status, processed_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}

	s.updateStmt, err = s.db.PrepareContext(ctx, `
		UPDATE events_metadata
		SET processing_status = $1, processing_error = $2, retry_count = $3,
		    last_retry_at = $4, updated_at = $5
		WHERE event_id = $6`)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}

	s.getStmt, err = s.db.PrepareContext(ctx, `
		SELECT id, event_id, chain_id, block_number, transaction_hash, log_index,
		       contract_address, event_name, processing_status, processing_error,
		       retry_count, last_retry_at, processed_at, created_at, updated_at
		FROM events_metadata
		WHERE event_id = $1`)
	if err != nil {
		return fmt.Errorf("failed to prepare get statement: %w", err)
	}

	return nil
}

// insertColumnsCount is the number of columns in the INSERT statement (11 fields)
const insertColumnsCount = 11

// InsertMetadata inserts a single event metadata record using a prepared statement
func (s *PostgreSQLEventMetadataStore) InsertMetadata(ctx context.Context, metadata *EventMetadata) error {
	if !s.initialized {
		return fmt.Errorf("metadata store not initialized")
	}

	if s.db == nil {
		return fmt.Errorf("PostgreSQL database connection is nil")
	}

	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("postgres_metadata_insert_time_ms", float64(duration), nil)
	}()

	now := time.Now()
	status := metadata.ProcessingStatus
	if status == "" {
		status = "pending"
	}

	// Use prepared statement if available, fallback to inline query
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("database operation panicked: %v", r)
			}
		}()
		if s.insertStmt != nil {
			err = s.insertStmt.QueryRowContext(
				ctx,
				metadata.EventID, metadata.ChainID, metadata.BlockNumber, metadata.TransactionHash,
				metadata.LogIndex, metadata.ContractAddress, metadata.EventName, status,
				metadata.ProcessedAt, now, now,
			).Scan(&metadata.ID)
		} else {
			err = s.db.QueryRowContext(
				ctx, `
				INSERT INTO events_metadata (
					event_id, chain_id, block_number, transaction_hash, log_index,
					contract_address, event_name, processing_status, processed_at, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
				RETURNING id`,
				metadata.EventID, metadata.ChainID, metadata.BlockNumber, metadata.TransactionHash,
				metadata.LogIndex, metadata.ContractAddress, metadata.EventName, status,
				metadata.ProcessedAt, now, now,
			).Scan(&metadata.ID)
		}
	}()

	if err != nil {
		s.logger.Error("Failed to insert metadata", "eventId", metadata.EventID, "error", err)
		s.metrics.RecordCounter("postgres_metadata_insert_error", int64(1), nil)
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	s.metrics.RecordCounter("postgres_metadata_insert_success", int64(1), nil)
	return nil
}

// InsertMetadataBatch inserts multiple metadata records using multi-value INSERT
// for significantly better throughput than row-by-row insertion.
// Records are batched into chunks of batchSize (100) to avoid oversized SQL.
func (s *PostgreSQLEventMetadataStore) InsertMetadataBatch(ctx context.Context, metadataList []*EventMetadata) error {
	if !s.initialized {
		return fmt.Errorf("metadata store not initialized")
	}

	if len(metadataList) == 0 {
		return nil
	}

	if s.db == nil {
		return fmt.Errorf("PostgreSQL database connection is nil")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("postgres_metadata_batch_insert_time_ms", float64(duration), nil)
	}()

	const batchSize = 100

	for i := 0; i < len(metadataList); i += batchSize {
		end := i + batchSize
		if end > len(metadataList) {
			end = len(metadataList)
		}
		chunk := metadataList[i:end]
		if err := s.insertBatchChunk(ctx, chunk); err != nil {
			s.logger.Error("Failed to insert batch chunk", "chunk_start", i, "chunk_size", len(chunk), "error", err)
			s.metrics.RecordCounter("postgres_metadata_batch_insert_error", int64(1), nil)
			return fmt.Errorf("failed to insert batch chunk at offset %d: %w", i, err)
		}
	}

	s.metrics.RecordHistogram("postgres_metadata_batch_insert_success", float64(len(metadataList)), nil)
	return nil
}

// insertBatchChunk inserts a chunk of metadata records using a single multi-value INSERT.
// This is 5-10x faster than individual INSERT statements for bulk data.
func (s *PostgreSQLEventMetadataStore) insertBatchChunk(ctx context.Context, batch []*EventMetadata) error {
	// Build multi-value INSERT: VALUES ($1,$2,...,$11), ($12,...,$22), ...
	valueClauses := make([]string, 0, len(batch))
	args := make([]any, 0, len(batch)*insertColumnsCount)
	paramIdx := 1
	now := time.Now()

	for _, m := range batch {
		if m == nil {
			continue
		}

		status := m.ProcessingStatus
		if status == "" {
			status = "pending"
		}

		clause := fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			paramIdx, paramIdx+1, paramIdx+2, paramIdx+3, paramIdx+4,
			paramIdx+5, paramIdx+6, paramIdx+7, paramIdx+8, paramIdx+9, paramIdx+10)
		valueClauses = append(valueClauses, clause)
		paramIdx += insertColumnsCount

		args = append(args,
			m.EventID, m.ChainID, m.BlockNumber, m.TransactionHash,
			m.LogIndex, m.ContractAddress, m.EventName, status,
			m.ProcessedAt, now, now,
		)
	}

	if len(valueClauses) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		INSERT INTO events_metadata (
			event_id, chain_id, block_number, transaction_hash, log_index,
			contract_address, event_name, processing_status, processed_at, created_at, updated_at
		) VALUES %s
		RETURNING id`, strings.Join(valueClauses, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("multi-value INSERT failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Scan returned IDs back into metadata objects in order
	rowIdx := 0
	for rows.Next() {
		if rowIdx >= len(batch) {
			break
		}
		m := batch[rowIdx]
		if m == nil {
			rowIdx++
			continue
		}
		if err := rows.Scan(&m.ID); err != nil {
			return fmt.Errorf("failed to scan returned ID at row %d: %w", rowIdx, err)
		}
		rowIdx++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating returned IDs: %w", err)
	}

	return nil
}

// GetMetadata retrieves metadata for a single event using a prepared statement
func (s *PostgreSQLEventMetadataStore) GetMetadata(ctx context.Context, eventID string) (*EventMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("metadata store not initialized")
	}

	if s.db == nil {
		return nil, fmt.Errorf("PostgreSQL database connection is nil")
	}

	if eventID == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("postgres_metadata_get_time_ms", float64(duration), nil)
	}()

	var row *sql.Row
	if s.getStmt != nil {
		row = s.getStmt.QueryRowContext(ctx, eventID)
	} else {
		row = s.db.QueryRowContext(ctx, `
			SELECT id, event_id, chain_id, block_number, transaction_hash, log_index,
			       contract_address, event_name, processing_status, processing_error,
			       retry_count, last_retry_at, processed_at, created_at, updated_at
			FROM events_metadata
			WHERE event_id = $1`, eventID)
	}

	metadata, err := scanEventMetadataRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		s.logger.Error("Failed to get metadata", "eventId", eventID, "error", err)
		s.metrics.RecordCounter("postgres_metadata_get_error", int64(1), nil)
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	s.metrics.RecordCounter("postgres_metadata_get_success", int64(1), nil)
	return metadata, nil
}

// GetMetadataBatch retrieves metadata for multiple events in a single query.
// Uses WHERE event_id IN ($1, $2, ..., $n) to avoid N+1 query problem.
func (s *PostgreSQLEventMetadataStore) GetMetadataBatch(ctx context.Context, eventIDs []string) (map[string]*EventMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("metadata store not initialized")
	}

	if s.db == nil {
		return nil, fmt.Errorf("PostgreSQL database connection is nil")
	}

	if len(eventIDs) == 0 {
		return map[string]*EventMetadata{}, nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("postgres_metadata_batch_get_time_ms", float64(duration), nil)
	}()

	// Build WHERE event_id IN ($1, $2, ..., $n)
	placeholders := make([]string, len(eventIDs))
	args := make([]any, len(eventIDs))
	for i, id := range eventIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, event_id, chain_id, block_number, transaction_hash, log_index,
		       contract_address, event_name, processing_status, processing_error,
		       retry_count, last_retry_at, processed_at, created_at, updated_at
		FROM events_metadata
		WHERE event_id IN (%s)`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to batch query metadata", "count", len(eventIDs), "error", err)
		s.metrics.RecordCounter("postgres_metadata_batch_get_error", int64(1), nil)
		return nil, fmt.Errorf("failed to batch query metadata: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]*EventMetadata, len(eventIDs))
	for rows.Next() {
		metadata, err := scanEventMetadataRow(rows)
		if err != nil {
			s.logger.Error("Failed to scan metadata row", "error", err)
			return nil, fmt.Errorf("failed to scan metadata row: %w", err)
		}
		result[metadata.EventID] = metadata
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating batch metadata rows", "error", err)
		return nil, fmt.Errorf("error iterating batch metadata rows: %w", err)
	}

	s.metrics.RecordHistogram("postgres_metadata_batch_get_success", float64(len(result)), nil)
	return result, nil
}

// GetMetadataByChain retrieves metadata for events in a specific chain
func (s *PostgreSQLEventMetadataStore) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*EventMetadata, error) {
	if !s.initialized {
		return nil, fmt.Errorf("metadata store not initialized")
	}

	if s.db == nil {
		return nil, fmt.Errorf("PostgreSQL database connection is nil")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("postgres_metadata_query_chain_time_ms", float64(duration), nil)
	}()

	query := `
	SELECT id, event_id, chain_id, block_number, transaction_hash, log_index,
	       contract_address, event_name, processing_status, processing_error,
	       retry_count, last_retry_at, processed_at, created_at, updated_at
	FROM events_metadata
	WHERE chain_id = $1
	ORDER BY block_number DESC
	LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, chainID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to query metadata by chain", "chainId", chainID, "error", err)
		s.metrics.RecordCounter("postgres_metadata_query_chain_error", int64(1), nil)
		return nil, fmt.Errorf("failed to query metadata by chain: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var metadataList []*EventMetadata
	for rows.Next() {
		metadata, err := scanEventMetadataRow(rows)
		if err != nil {
			s.logger.Error("Failed to scan metadata", "error", err)
			return nil, fmt.Errorf("failed to scan metadata: %w", err)
		}
		metadataList = append(metadataList, metadata)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating metadata rows", "error", err)
		return nil, fmt.Errorf("error iterating metadata rows: %w", err)
	}

	s.metrics.RecordHistogram("postgres_metadata_query_chain_success", float64(len(metadataList)), nil)
	return metadataList, nil
}

type metadataRowScanner interface {
	Scan(dest ...any) error
}

func scanEventMetadataRow(scanner metadataRowScanner) (*EventMetadata, error) {
	metadata := &EventMetadata{}
	var processingError sql.NullString
	var lastRetryAt sql.NullTime

	err := scanner.Scan(
		&metadata.ID, &metadata.EventID, &metadata.ChainID, &metadata.BlockNumber,
		&metadata.TransactionHash, &metadata.LogIndex, &metadata.ContractAddress,
		&metadata.EventName, &metadata.ProcessingStatus, &processingError,
		&metadata.RetryCount, &lastRetryAt, &metadata.ProcessedAt,
		&metadata.CreatedAt, &metadata.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan metadata row: %w", err)
	}
	if processingError.Valid {
		metadata.ProcessingError = processingError.String
	}
	if lastRetryAt.Valid {
		retryAt := lastRetryAt.Time
		metadata.LastRetryAt = &retryAt
	}

	return metadata, nil
}

// UpdateMetadata updates metadata for an event using a prepared statement
func (s *PostgreSQLEventMetadataStore) UpdateMetadata(ctx context.Context, metadata *EventMetadata) error {
	if !s.initialized {
		return fmt.Errorf("metadata store not initialized")
	}

	if s.db == nil {
		return fmt.Errorf("PostgreSQL database connection is nil")
	}

	if metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		s.metrics.RecordHistogram("postgres_metadata_update_time_ms", float64(duration), nil)
	}()

	now := time.Now()

	var result sql.Result
	var err error
	if s.updateStmt != nil {
		result, err = s.updateStmt.ExecContext(
			ctx,
			metadata.ProcessingStatus, metadata.ProcessingError, metadata.RetryCount,
			metadata.LastRetryAt, now, metadata.EventID,
		)
	} else {
		result, err = s.db.ExecContext(
			ctx, `
			UPDATE events_metadata
			SET processing_status = $1, processing_error = $2, retry_count = $3,
			    last_retry_at = $4, updated_at = $5
			WHERE event_id = $6`,
			metadata.ProcessingStatus, metadata.ProcessingError, metadata.RetryCount,
			metadata.LastRetryAt, now, metadata.EventID,
		)
	}

	if err != nil {
		s.logger.Error("Failed to update metadata", "eventId", metadata.EventID, "error", err)
		s.metrics.RecordCounter("postgres_metadata_update_error", int64(1), nil)
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected", "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		s.logger.Warn("No metadata found to update", "eventId", metadata.EventID)
		return fmt.Errorf("no metadata found to update")
	}

	s.metrics.RecordCounter("postgres_metadata_update_success", int64(1), nil)
	return nil
}

// Health returns the health status of the metadata store
func (s *PostgreSQLEventMetadataStore) Health(ctx context.Context) *core.HealthStatus {
	if !s.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "metadata store not initialized",
		}
	}

	if s.db == nil {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "PostgreSQL database connection is nil",
		}
	}

	// Try to ping the database with panic recovery
	healthy := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				// If panic occurs, database is unhealthy
				healthy = false
			}
		}()
		err := s.db.PingContext(ctx)
		healthy = (err == nil)
	}()

	if healthy {
		return &core.HealthStatus{
			Status:  "healthy",
			Message: "metadata store is healthy",
		}
	}

	return &core.HealthStatus{
		Status:  "unhealthy",
		Message: "PostgreSQL database is unavailable",
	}
}

// Close closes the metadata store and releases prepared statements
func (s *PostgreSQLEventMetadataStore) Close(ctx context.Context) error {
	if !s.initialized {
		return nil
	}

	// Close prepared statements
	if s.insertStmt != nil {
		_ = s.insertStmt.Close()
	}
	if s.updateStmt != nil {
		_ = s.updateStmt.Close()
	}
	if s.getStmt != nil {
		_ = s.getStmt.Close()
	}

	s.initialized = false
	return nil
}
