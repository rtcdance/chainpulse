package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
)

// PostgreSQLEventMetadataStore implements EventMetadataStore for PostgreSQL
type PostgreSQLEventMetadataStore struct {
	dbManager   database.DatabaseManager
	logger      core.Logger
	metrics     core.MetricsCollector
	db          *sql.DB
	initialized bool
}

// NewPostgreSQLEventMetadataStore creates a new PostgreSQL event metadata store
func NewPostgreSQLEventMetadataStore(
	dbManager database.DatabaseManager,
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
		s.logger.Error("Failed to get PostgreSQL connection", "error", err.Error())
		return fmt.Errorf("failed to get PostgreSQL connection: %w", err)
	}

	if db == nil {
		return fmt.Errorf("PostgreSQL connection is nil")
	}

	s.db = db.(*sql.DB)

	// Create table if it doesn't exist
	if err := s.createTable(ctx); err != nil {
		s.logger.Error("Failed to create table", "error", err.Error())
		return fmt.Errorf("failed to create table: %w", err)
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

// InsertMetadata inserts a single event metadata record
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

	query := `
	INSERT INTO events_metadata (
		event_id, chain_id, block_number, transaction_hash, log_index,
		contract_address, event_name, processing_status, processed_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id
	`

	now := time.Now()
	status := metadata.ProcessingStatus
	if status == "" {
		status = "pending"
	}

	// Wrap database operation with panic recovery
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("database operation panicked: %v", r)
			}
		}()
		err = s.db.QueryRowContext(
			ctx, query,
			metadata.EventID, metadata.ChainID, metadata.BlockNumber, metadata.TransactionHash,
			metadata.LogIndex, metadata.ContractAddress, metadata.EventName, status,
			metadata.ProcessedAt, now, now,
		).Scan(&metadata.ID)
	}()

	if err != nil {
		s.logger.Error("Failed to insert metadata", "eventId", metadata.EventID, "error", err.Error())
		s.metrics.RecordCounter("postgres_metadata_insert_error", int64(1), nil)
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	s.metrics.RecordCounter("postgres_metadata_insert_success", int64(1), nil)
	return nil
}

// InsertMetadataBatch inserts multiple metadata records in a batch operation
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

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err.Error())
		s.metrics.RecordCounter("postgres_metadata_batch_insert_error", int64(1), nil)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
	INSERT INTO events_metadata (
		event_id, chain_id, block_number, transaction_hash, log_index,
		contract_address, event_name, processing_status, processed_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id
	`

	now := time.Now()
	for _, metadata := range metadataList {
		if metadata == nil {
			continue
		}

		status := metadata.ProcessingStatus
		if status == "" {
			status = "pending"
		}

		err := tx.QueryRowContext(
			ctx, query,
			metadata.EventID, metadata.ChainID, metadata.BlockNumber, metadata.TransactionHash,
			metadata.LogIndex, metadata.ContractAddress, metadata.EventName, status,
			metadata.ProcessedAt, now, now,
		).Scan(&metadata.ID)
		if err != nil {
			s.logger.Error("Failed to insert metadata in batch", "eventId", metadata.EventID, "error", err.Error())
			s.metrics.RecordCounter("postgres_metadata_batch_insert_error", int64(1), nil)
			return fmt.Errorf("failed to insert metadata in batch: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", "error", err.Error())
		s.metrics.RecordCounter("postgres_metadata_batch_insert_error", int64(1), nil)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.metrics.RecordHistogram("postgres_metadata_batch_insert_success", float64(len(metadataList)), nil)
	return nil
}

// GetMetadata retrieves metadata for a single event
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

	query := `
	SELECT id, event_id, chain_id, block_number, transaction_hash, log_index,
	       contract_address, event_name, processing_status, processing_error,
	       retry_count, last_retry_at, processed_at, created_at, updated_at
	FROM events_metadata
	WHERE event_id = $1
	`

	metadata := &EventMetadata{}
	err := s.db.QueryRowContext(ctx, query, eventID).Scan(
		&metadata.ID, &metadata.EventID, &metadata.ChainID, &metadata.BlockNumber,
		&metadata.TransactionHash, &metadata.LogIndex, &metadata.ContractAddress,
		&metadata.EventName, &metadata.ProcessingStatus, &metadata.ProcessingError,
		&metadata.RetryCount, &metadata.LastRetryAt, &metadata.ProcessedAt,
		&metadata.CreatedAt, &metadata.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		s.logger.Error("Failed to get metadata", "eventId", eventID, "error", err.Error())
		s.metrics.RecordCounter("postgres_metadata_get_error", int64(1), nil)
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	s.metrics.RecordCounter("postgres_metadata_get_success", int64(1), nil)
	return metadata, nil
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
		s.logger.Error("Failed to query metadata by chain", "chainId", chainID, "error", err.Error())
		s.metrics.RecordCounter("postgres_metadata_query_chain_error", int64(1), nil)
		return nil, fmt.Errorf("failed to query metadata by chain: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var metadataList []*EventMetadata
	for rows.Next() {
		metadata := &EventMetadata{}
		err := rows.Scan(
			&metadata.ID, &metadata.EventID, &metadata.ChainID, &metadata.BlockNumber,
			&metadata.TransactionHash, &metadata.LogIndex, &metadata.ContractAddress,
			&metadata.EventName, &metadata.ProcessingStatus, &metadata.ProcessingError,
			&metadata.RetryCount, &metadata.LastRetryAt, &metadata.ProcessedAt,
			&metadata.CreatedAt, &metadata.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan metadata", "error", err.Error())
			return nil, fmt.Errorf("failed to scan metadata: %w", err)
		}
		metadataList = append(metadataList, metadata)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating metadata rows", "error", err.Error())
		return nil, fmt.Errorf("error iterating metadata rows: %w", err)
	}

	s.metrics.RecordHistogram("postgres_metadata_query_chain_success", float64(len(metadataList)), nil)
	return metadataList, nil
}

// UpdateMetadata updates metadata for an event
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

	query := `
	UPDATE events_metadata
	SET processing_status = $1, processing_error = $2, retry_count = $3,
	    last_retry_at = $4, updated_at = $5
	WHERE event_id = $6
	`

	now := time.Now()
	result, err := s.db.ExecContext(
		ctx, query,
		metadata.ProcessingStatus, metadata.ProcessingError, metadata.RetryCount,
		metadata.LastRetryAt, now, metadata.EventID,
	)
	if err != nil {
		s.logger.Error("Failed to update metadata", "eventId", metadata.EventID, "error", err.Error())
		s.metrics.RecordCounter("postgres_metadata_update_error", int64(1), nil)
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected", "error", err.Error())
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

// Close closes the metadata store
func (s *PostgreSQLEventMetadataStore) Close(ctx context.Context) error {
	if !s.initialized {
		return nil
	}

	s.initialized = false
	return nil
}
