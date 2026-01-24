package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"chainpulse/pkg/core"
)

// PostgreSQLDatabase implements DatabasePlugin for PostgreSQL
type PostgreSQLDatabase struct {
	*BaseDatabasePlugin
	db              *sql.DB
	connectionPool  *sql.DB
	maxConnections  int
	queryTimeout    time.Duration
	mu              sync.RWMutex
	events          map[string]*core.BlockchainEvent // in-memory cache for testing
	eventsMu        sync.RWMutex
}

// NewPostgreSQLDatabase creates a new PostgreSQL database plugin
func NewPostgreSQLDatabase(logger core.Logger, metricsCollector core.MetricsCollector) *PostgreSQLDatabase {
	return &PostgreSQLDatabase{
		BaseDatabasePlugin: NewBaseDatabasePlugin(logger, metricsCollector),
		maxConnections:     25,
		queryTimeout:       30 * time.Second,
		events:             make(map[string]*core.BlockchainEvent),
	}
}

// Initialize initializes the PostgreSQL database plugin
func (p *PostgreSQLDatabase) Initialize(config *core.Config) error {
	if err := p.BaseDatabasePlugin.Initialize(config); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Extract PostgreSQL connection string from config
	connStr := config.GetString("POSTGRES_CONNECTION_STRING", "")
	if connStr == "" {
		// Build connection string from individual components
		host := config.GetString("POSTGRES_HOST", "localhost")
		port := config.GetString("POSTGRES_PORT", "5432")
		user := config.GetString("POSTGRES_USER", "postgres")
		password := config.GetString("POSTGRES_PASSWORD", "")
		dbname := config.GetString("POSTGRES_DB", "chainpulse")

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to open PostgreSQL connection", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(p.maxConnections)
	db.SetMaxIdleConns(p.maxConnections / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		p.RecordError()
		p.logger.Error("Failed to ping PostgreSQL", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	p.db = db
	p.connectionPool = db

	// Create tables if they don't exist
	if err := p.createTables(); err != nil {
		p.RecordError()
		return fmt.Errorf("failed to create tables: %w", err)
	}

	p.logger.Info("PostgreSQL database initialized", map[string]interface{}{
		"component": "postgres_database",
	})

	return nil
}

// createTables creates the necessary database tables
func (p *PostgreSQLDatabase) createTables() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS blockchain_events (
		id SERIAL PRIMARY KEY,
		event_hash VARCHAR(255) UNIQUE NOT NULL,
		block_number BIGINT NOT NULL,
		transaction_hash VARCHAR(255) NOT NULL,
		log_index BIGINT NOT NULL,
		contract_address VARCHAR(255) NOT NULL,
		event_name VARCHAR(255),
		event_data BYTEA,
		timestamp BIGINT NOT NULL,
		processed_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_event_hash ON blockchain_events(event_hash);
	CREATE INDEX IF NOT EXISTS idx_block_number ON blockchain_events(block_number);
	CREATE INDEX IF NOT EXISTS idx_contract_address ON blockchain_events(contract_address);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON blockchain_events(timestamp);
	`

	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	_, err := p.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// Start starts the PostgreSQL database plugin
func (p *PostgreSQLDatabase) Start() error {
	if err := p.BaseDatabasePlugin.Start(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Verify connection is still active
	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	if err := p.db.PingContext(ctx); err != nil {
		p.RecordError()
		return fmt.Errorf("failed to verify PostgreSQL connection: %w", err)
	}

	p.logger.Info("PostgreSQL database started", map[string]interface{}{
		"component": "postgres_database",
	})

	return nil
}

// Stop stops the PostgreSQL database plugin
func (p *PostgreSQLDatabase) Stop() error {
	if err := p.BaseDatabasePlugin.Stop(); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db != nil {
		if err := p.db.Close(); err != nil {
			p.RecordError()
			p.logger.Error("Failed to close PostgreSQL connection", map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
		}
	}

	p.logger.Info("PostgreSQL database stopped", map[string]interface{}{
		"component": "postgres_database",
	})

	return nil
}

// WriteEvent writes a blockchain event to the database
func (p *PostgreSQLDatabase) WriteEvent(event *core.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}

	start := time.Now()

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.RecordError()
		return fmt.Errorf("database not initialized")
	}

	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	insertSQL := `
	INSERT INTO blockchain_events (event_hash, block_number, transaction_hash, log_index, contract_address, event_name, event_data, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (event_hash) DO UPDATE SET processed_at = CURRENT_TIMESTAMP
	`

	_, err := db.ExecContext(ctx, insertSQL,
		event.EventHash,
		event.BlockNumber,
		event.TransactionHash,
		event.LogIndex,
		event.ContractAddress,
		event.EventName,
		event.EventData,
		event.BlockTimestamp,
	)

	duration := time.Since(start).Milliseconds()

	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to write event to PostgreSQL", map[string]interface{}{
			"error": err.Error(),
			"hash":  event.EventHash,
		})
		return fmt.Errorf("failed to write event: %w", err)
	}

	p.RecordWrite(duration)

	// Update in-memory cache
	p.eventsMu.Lock()
	p.events[event.EventHash] = event
	p.eventsMu.Unlock()

	// Update event count
	p.updateEventCount()

	return nil
}

// WriteEvents writes multiple blockchain events to the database (batch)
func (p *PostgreSQLDatabase) WriteEvents(events []core.BlockchainEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.RecordError()
		return fmt.Errorf("database not initialized")
	}

	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		p.RecordError()
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	insertSQL := `
	INSERT INTO blockchain_events (event_hash, block_number, transaction_hash, log_index, contract_address, event_name, event_data, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (event_hash) DO UPDATE SET processed_at = CURRENT_TIMESTAMP
	`

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.Error("failed to rollback transaction", map[string]interface{}{"error": rbErr})
		}
		p.RecordError()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			p.logger.Error("failed to close statement", map[string]interface{}{"error": err})
		}
	}()

	for _, event := range events {
		_, err := stmt.ExecContext(ctx,
			event.EventHash,
			event.BlockNumber,
			event.TransactionHash,
			event.LogIndex,
			event.ContractAddress,
			event.EventName,
			event.EventData,
			event.BlockTimestamp,
		)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				p.logger.Error("failed to rollback transaction", map[string]interface{}{"error": rbErr})
			}
			p.RecordError()
			p.logger.Error("Failed to write event in batch", map[string]interface{}{
				"error": err.Error(),
				"hash":  event.EventHash,
			})
			return fmt.Errorf("failed to write event in batch: %w", err)
		}

		// Update in-memory cache
		p.eventsMu.Lock()
		p.events[event.EventHash] = &event
		p.eventsMu.Unlock()
	}

	if err := tx.Commit(); err != nil {
		p.RecordError()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	p.RecordWrite(duration)

	// Update event count
	p.updateEventCount()

	return nil
}

// QueryEvents queries events from the database
func (p *PostgreSQLDatabase) QueryEvents(filter *core.EventFilter) (*core.QueryResult, error) {
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}

	start := time.Now()

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	// Build query
	query := "SELECT event_hash, block_number, transaction_hash, log_index, contract_address, event_name, event_data, timestamp FROM blockchain_events WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if len(filter.ContractAddress) > 0 {
		query += fmt.Sprintf(" AND contract_address = $%d", argIndex)
		args = append(args, filter.ContractAddress[0].Hex())
		argIndex++
	}

	if filter.FromBlock > 0 {
		query += fmt.Sprintf(" AND block_number >= $%d", argIndex)
		args = append(args, filter.FromBlock)
		argIndex++
	}

	if filter.ToBlock > 0 {
		query += fmt.Sprintf(" AND block_number <= $%d", argIndex)
		args = append(args, filter.ToBlock)
		argIndex++
	}

	// Add pagination
	query += " ORDER BY block_number DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to query events from PostgreSQL", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.Error("failed to close rows", map[string]interface{}{"error": err})
		}
	}()

	events := []core.BlockchainEvent{}
	for rows.Next() {
		var event core.BlockchainEvent
		err := rows.Scan(
			&event.EventHash,
			&event.BlockNumber,
			&event.TransactionHash,
			&event.LogIndex,
			&event.ContractAddress,
			&event.EventName,
			&event.EventData,
			&event.BlockTimestamp,
		)
		if err != nil {
			p.RecordError()
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		p.RecordError()
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	p.RecordRead(duration)

	result := &core.QueryResult{
		Events:       events,
		Total:        int64(len(events)),
		ResponseTime: duration,
	}

	return result, nil
}

// GetEventByHash retrieves an event by its hash
func (p *PostgreSQLDatabase) GetEventByHash(hash string) (*core.BlockchainEvent, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	// Check in-memory cache first
	p.eventsMu.RLock()
	if event, exists := p.events[hash]; exists {
		p.eventsMu.RUnlock()
		duration := time.Since(start).Milliseconds()
		p.RecordRead(duration)
		return event, nil
	}
	p.eventsMu.RUnlock()

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	query := "SELECT event_hash, block_number, transaction_hash, log_index, contract_address, event_name, event_data, timestamp FROM blockchain_events WHERE event_hash = $1"

	var event core.BlockchainEvent
	err := db.QueryRowContext(ctx, query, hash).Scan(
		&event.EventHash,
		&event.BlockNumber,
		&event.TransactionHash,
		&event.LogIndex,
		&event.ContractAddress,
		&event.EventName,
		&event.EventData,
		&event.BlockTimestamp,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			duration := time.Since(start).Milliseconds()
			p.RecordRead(duration)
			return nil, nil
		}
		p.RecordError()
		p.logger.Error("Failed to get event by hash from PostgreSQL", map[string]interface{}{
			"error": err.Error(),
			"hash":  hash,
		})
		return nil, fmt.Errorf("failed to get event by hash: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	p.RecordRead(duration)

	// Update in-memory cache
	p.eventsMu.Lock()
	p.events[hash] = &event
	p.eventsMu.Unlock()

	return &event, nil
}

// DeleteEvent deletes an event from the database by ID
func (p *PostgreSQLDatabase) DeleteEvent(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event ID is required")
	}

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.RecordError()
		return fmt.Errorf("database not initialized")
	}

	query := "DELETE FROM blockchain_events WHERE id = $1"

	result, err := db.ExecContext(ctx, query, eventID)
	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to delete event from PostgreSQL", map[string]interface{}{
			"error":    err.Error(),
			"event_id": eventID,
		})
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		p.RecordError()
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil
	}

	p.RecordDelete()
	p.updateEventCount()

	return nil
}

// GetStats returns database statistics
func (p *PostgreSQLDatabase) GetStats() *DatabaseStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgWriteTime := 0.0
	if p.writeCount > 0 {
		avgWriteTime = float64(p.totalWriteTime) / float64(p.writeCount)
	}

	avgReadTime := 0.0
	if p.readCount > 0 {
		avgReadTime = float64(p.totalReadTime) / float64(p.readCount)
	}

	return &DatabaseStats{
		WriteCount:     p.writeCount,
		ReadCount:      p.readCount,
		DeleteCount:    p.deleteCount,
		ErrorCount:     p.errorCount,
		TotalSize:      p.totalSize,
		EventCount:     p.eventCount,
		AvgWriteTimeMs: avgWriteTime,
		AvgReadTimeMs:  avgReadTime,
		LastWriteTime:  p.lastWriteTime,
		LastReadTime:   p.lastReadTime,
	}
}

// updateEventCount updates the event count from the database
func (p *PostgreSQLDatabase) updateEventCount() {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return
	}

	ctx, cancel := NewContextWithTimeout(p.queryTimeout)
	defer cancel()

	var count int64
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blockchain_events").Scan(&count)
	if err != nil {
		p.logger.Error("Failed to update event count", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	p.UpdateEventCount(count)
}

// NewContextWithTimeout creates a context with timeout
func NewContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// GetAllEvents retrieves all events from the database
func (p *PostgreSQLDatabase) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, block_number, transaction_hash, log_index, event_signature, 
		       contract_address, event_name, event_hash, data, indexed_params, 
		       created_at
		FROM blockchain_events
		ORDER BY block_number ASC, log_index ASC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.Error("failed to close rows", map[string]interface{}{"error": err})
		}
	}()

	var events []*core.BlockchainEvent
	for rows.Next() {
		event := &core.BlockchainEvent{}
		if err := rows.Scan(
			&event.ID, &event.BlockNumber, &event.TransactionHash, &event.LogIndex,
			&event.EventSignature, &event.ContractAddress, &event.EventName,
			&event.EventHash, &event.EventData, &event.EventTopic, &event.CreatedAt,
		); err != nil {
			p.RecordError()
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		p.RecordError()
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	p.RecordRead(0)
	return events, nil
}

// GetAllBlocks retrieves all blocks from the database
func (p *PostgreSQLDatabase) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT number, hash, parent_hash, timestamp, miner, difficulty, gas_used, gas_limit
		FROM blocks
		ORDER BY number ASC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var blocks []*core.Block
	for rows.Next() {
		block := &core.Block{}
		if err := rows.Scan(
			&block.Number, &block.Hash, &block.ParentHash, &block.Timestamp,
			&block.Miner, &block.Difficulty, &block.GasUsed, &block.GasLimit,
		); err != nil {
			p.RecordError()
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}
		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		p.RecordError()
		return nil, fmt.Errorf("error iterating blocks: %w", err)
	}

	p.RecordRead(0)
	return blocks, nil
}

// GetEventsByBlockRange retrieves events within a block range
func (p *PostgreSQLDatabase) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT id, block_number, transaction_hash, log_index, event_signature, 
		       contract_address, event_name, event_hash, data, indexed_params, 
		       created_at
		FROM blockchain_events
		WHERE block_number >= $1 AND block_number <= $2
		ORDER BY block_number ASC, log_index ASC
	`

	rows, err := db.QueryContext(ctx, query, fromBlock, toBlock)
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []*core.BlockchainEvent
	for rows.Next() {
		event := &core.BlockchainEvent{}
		if err := rows.Scan(
			&event.ID, &event.BlockNumber, &event.TransactionHash, &event.LogIndex,
			&event.EventSignature, &event.ContractAddress, &event.EventName,
			&event.EventHash, &event.EventData, &event.EventTopic, &event.CreatedAt,
		); err != nil {
			p.RecordError()
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		p.RecordError()
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	p.RecordRead(0)
	return events, nil
}

// GetBlock retrieves a specific block by number
func (p *PostgreSQLDatabase) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT number, hash, parent_hash, timestamp, miner, difficulty, gas_used, gas_limit
		FROM blocks
		WHERE number = $1
	`

	block := &core.Block{}
	err := db.QueryRowContext(ctx, query, blockNumber).Scan(
		&block.Number, &block.Hash, &block.ParentHash, &block.Timestamp,
		&block.Miner, &block.Difficulty, &block.GasUsed, &block.GasLimit,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query block: %w", err)
	}

	p.RecordRead(0)
	return block, nil
}

// GetLatestBlock retrieves the latest block number
func (p *PostgreSQLDatabase) GetLatestBlock(ctx context.Context) (uint64, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	query := `SELECT COALESCE(MAX(number), 0) FROM blocks`

	var blockNumber uint64
	err := db.QueryRowContext(ctx, query).Scan(&blockNumber)
	if err != nil {
		p.RecordError()
		return 0, fmt.Errorf("failed to query latest block: %w", err)
	}

	p.RecordRead(0)
	return blockNumber, nil
}

// DeleteEventsByBlockRange deletes events within a block range
func (p *PostgreSQLDatabase) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	query := `DELETE FROM blockchain_events WHERE block_number >= $1 AND block_number <= $2`

	result, err := db.ExecContext(ctx, query, fromBlock, toBlock)
	if err != nil {
		p.RecordError()
		return 0, fmt.Errorf("failed to delete events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		p.RecordError()
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	p.RecordDelete()
	return rowsAffected, nil
}

// GetReorgStats retrieves reorg statistics
func (p *PostgreSQLDatabase) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT COALESCE(total_reorgs, 0), COALESCE(total_blocks_rolled_back, 0), 
		       COALESCE(average_reorg_size, 0), COALESCE(last_reorg_time, NOW()), 
		       COALESCE(last_reorg_block, 0)
		FROM reorg_stats
		LIMIT 1
	`

	stats := &core.ReorgStats{}
	err := db.QueryRowContext(ctx, query).Scan(
		&stats.TotalReorgsDetected, &stats.TotalBlocksRolledBack, &stats.AverageReorgSize,
		&stats.LastReorgTime, &stats.LastReorgBlock,
	)

	if err == sql.ErrNoRows {
		return &core.ReorgStats{}, nil
	}
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query reorg stats: %w", err)
	}

	p.RecordRead(0)
	return stats, nil
}
