package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lib/pq" // PostgreSQL driver and parameterized arrays
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/logkeys"
)

// insertEventSQL is the single INSERT shared by WriteEvent / WriteEvents /
// WriteBatch, so the column list cannot drift between the three paths.
const insertEventSQL = `
INSERT INTO blockchain_events (id, chain_id, block_number, block_hash, transaction_hash, log_index, contract_address, event_name, event_data, timestamp, status, correlation_id, event_state)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::text, $12,
	 jsonb_build_object('canonicality',CASE WHEN $11::text='reorged' THEN 'orphaned' ELSE 'canonical' END,
	 'processing',CASE WHEN $13::boolean THEN 'processed' WHEN $11::text IN ('confirmed','finalized') THEN 'processed' WHEN $11::text='failed' THEN 'failed' ELSE 'pending' END,
	 'finality',jsonb_build_object('state','unknown','confidence','unknown','source','unknown')))
	ON CONFLICT (chain_id, block_hash, transaction_hash, log_index) DO NOTHING
	`

// selectEventColumns is the column list every event read path selects, in the
// order the scan code expects.
const selectEventColumns = `id, chain_id, block_number, block_hash, transaction_hash, log_index, contract_address, event_name, event_data, timestamp, status, correlation_id, event_state`

// eventStatus returns the value for the NOT NULL `status` column. An unset
// status maps to "pending", matching the column default.
func eventStatus(event *blockchain.BlockchainEvent) string {
	if event.Status == "" {
		return string(blockchain.EventStatusPending)
	}
	return string(event.Status)
}

// nullableString maps "" to SQL NULL.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// eventScanner is satisfied by *sql.Row and *sql.Rows.
type eventScanner interface {
	Scan(dest ...any) error
}

// scanEvent scans one row selected with selectEventColumns into an event,
// applying the same conversions on every read path.
func scanEvent(s eventScanner) (*blockchain.BlockchainEvent, error) {
	var (
		event                              blockchain.BlockchainEvent
		blockHash, txHash, contractAddress string
		eventData                          []byte
		eventState                         []byte
		status                             string
		correlationID                      sql.NullString
	)
	if err := s.Scan(
		&event.ID, &event.ChainID, &event.BlockNumber, &blockHash, &txHash,
		&event.LogIndex, &contractAddress, &event.EventName, &eventData,
		&event.BlockTimestamp, &status, &correlationID, &eventState,
	); err != nil {
		return nil, err
	}
	event.EventHash = event.ID
	event.BlockHash = common.HexToHash(blockHash)
	event.TransactionHash = common.HexToHash(txHash)
	event.ContractAddress = common.HexToAddress(contractAddress)
	if err := restoreEventPayload(eventData, &event); err != nil {
		return nil, err
	}
	event.Status = blockchain.EventStatus(status)
	if err := json.Unmarshal(eventState, &event.State); err != nil {
		return nil, fmt.Errorf("decode independent event state: %w", err)
	}
	if correlationID.Valid {
		event.CorrelationID = correlationID.String
	}
	return &event, nil
}

// eventPK returns the value stored in the `id` column.
//
// The caller-provided ID wins when set (the core.DatabasePlugin port addresses
// events by ID via GetEvent(ctx, id)); otherwise the canonical event hash is
// used. Production leaves ID empty and relies on EventHash, while some callers
// (and the contract suite) set only ID.
func eventPK(event *blockchain.BlockchainEvent) string {
	if event.ID != "" {
		return event.ID
	}
	return event.EventHash
}

// eventDataFromJSON reads legacy string-only rows during schema upgrade.
func eventDataFromJSON(stored []byte) []byte {
	if len(stored) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(stored, &s); err != nil {
		// Not JSON (legacy row) — return the raw bytes unchanged.
		return stored
	}
	return []byte(s)
}

const (
	defaultPostgresMaxConnections  = 25
	defaultPostgresQueryTimeout    = 30 * time.Second
	defaultPostgresConnMaxLifetime = time.Hour
)

// PostgreSQLDatabase implements DatabasePlugin for PostgreSQL
type PostgreSQLDatabase struct {
	*BaseDatabasePlugin
	db             *sql.DB
	maxConnections int
	queryTimeout   time.Duration
	mu             sync.RWMutex
}

// NewPostgreSQLDatabase creates a new PostgreSQL database plugin
func NewPostgreSQLDatabase(logger portcontract.Logger, metricsCollector core.MetricsCollector) *PostgreSQLDatabase {
	return &PostgreSQLDatabase{
		BaseDatabasePlugin: NewBaseDatabasePlugin(logger, metricsCollector),
		maxConnections:     defaultPostgresMaxConnections,
		queryTimeout:       defaultPostgresQueryTimeout,
	}
}

// Initialize initializes the PostgreSQL database plugin
func (p *PostgreSQLDatabase) Initialize(ctx context.Context, config core.Config) error {
	if err := p.BaseDatabasePlugin.Initialize(ctx, config); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Extract PostgreSQL connection string from config
	connStr := config.DatabaseURL
	if connStr == "" {
		// Build connection string from individual components
		host := orDefault(config.PostgresHost, "localhost")
		port := orDefault(config.PostgresPort, "5432")
		user := orDefault(config.PostgresUser, "postgres")
		password := config.PostgresPassword
		dbname := orDefault(config.PostgresDB, "chainpulse")
		sslmode := orDefault(config.SslMode, "require")

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password.Value(), dbname, sslmode)
	}

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to open PostgreSQL connection", logkeys.LogKeyError, err)
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(p.maxConnections)
	db.SetMaxIdleConns(p.maxConnections / 2)
	db.SetConnMaxLifetime(defaultPostgresConnMaxLifetime)

	// Test connection
	pingCtx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		p.RecordError()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Store the handle. Without this assignment p.db stays nil, so every
	// method reports "database not initialized" and Start() nil-panics — the
	// adapter was unusable even against a correct schema.
	p.db = db

	return nil
}

// Start starts the PostgreSQL database plugin
func (p *PostgreSQLDatabase) Start(ctx context.Context) error {
	if err := p.BaseDatabasePlugin.Start(ctx); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Verify connection is still active
	pingCtx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	if err := p.db.PingContext(pingCtx); err != nil {
		p.RecordError()
		return fmt.Errorf("failed to verify PostgreSQL connection: %w", err)
	}

	p.logger.Info("PostgreSQL database started", logkeys.LogKeyComponent, "postgres_database")

	return nil
}

// Stop stops the PostgreSQL database plugin
func (p *PostgreSQLDatabase) Stop(ctx context.Context) error {
	if err := p.BaseDatabasePlugin.Stop(ctx); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db != nil {
		if err := p.db.Close(); err != nil {
			p.RecordError()
			p.logger.Error("Failed to close PostgreSQL connection", logkeys.LogKeyError, err)
			return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
		}
	}

	p.logger.Info("PostgreSQL database stopped", logkeys.LogKeyComponent, "postgres_database")

	return nil
}

// WriteEvent writes a blockchain event to the database
func (p *PostgreSQLDatabase) WriteEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
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

	ctx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	insertSQL := insertEventSQL
	payload, err := eventPayloadJSON(event)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(
		ctx, insertSQL,
		eventPK(event),
		event.ChainID,
		event.BlockNumber,
		event.BlockHash.Hex(),
		event.TransactionHash.Hex(),
		event.LogIndex,
		event.ContractAddress.Hex(),
		event.EventName,
		payload,
		event.BlockTimestamp,
		eventStatus(event),
		nullableString(event.CorrelationID),
		false,
	)

	duration := time.Since(start).Milliseconds()

	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to write event to PostgreSQL", logkeys.LogKeyError, err, logkeys.LogKeyHash, event.EventHash)
		return fmt.Errorf("failed to write event: %w", err)
	}

	p.RecordWrite(duration)

	// Update event count
	p.updateEventCount()

	return nil
}

// WriteBlock upserts a canonical block.
//
// The `id` column is derived from (chain_id, number), so invariant I03 — at
// most one canonical block per chain per height — is enforced by the primary
// key itself. A reorg that replaces the block at a height updates the hash in
// place rather than inserting a second row.
func (p *PostgreSQLDatabase) WriteBlock(ctx context.Context, chainID string, block *blockchain.Block) error {
	if block == nil {
		return fmt.Errorf("block is required")
	}
	if chainID == "" {
		return fmt.Errorf("chainID is required")
	}

	start := time.Now()

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()
	if db == nil {
		p.RecordError()
		return fmt.Errorf("database not initialized")
	}

	ctx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	const upsertBlockSQL = `
INSERT INTO blocks (id, chain_id, number, hash, parent_hash, timestamp)
	VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (id) DO UPDATE
	SET hash = EXCLUDED.hash, parent_hash = EXCLUDED.parent_hash, timestamp = EXCLUDED.timestamp
	`

	id := fmt.Sprintf("%s:%d", chainID, block.Number)
	_, err := db.ExecContext(ctx, upsertBlockSQL, id, chainID, block.Number,
		block.Hash.Hex(), block.ParentHash.Hex(), block.Timestamp)
	if err != nil {
		p.RecordError()
		p.logger.Error("Failed to write block to PostgreSQL", logkeys.LogKeyError, err)
		return fmt.Errorf("failed to write block: %w", err)
	}

	p.RecordWrite(time.Since(start).Milliseconds())
	return nil
}

// WriteEvents writes multiple blockchain events to the database (batch)
func (p *PostgreSQLDatabase) WriteEvents(ctx context.Context, events []blockchain.BlockchainEvent) error {
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

	ctx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		p.RecordError()
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	insertSQL := insertEventSQL

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.Error("failed to rollback transaction", logkeys.LogKeyError, rbErr)
		}
		p.RecordError()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			p.logger.Error("failed to close statement", logkeys.LogKeyError, err)
		}
	}()

	for _, event := range events {
		payload, err := eventPayloadJSON(&event)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		_, err = stmt.ExecContext(
			ctx,
			eventPK(&event),
			event.ChainID,
			event.BlockNumber,
			event.BlockHash.Hex(),
			event.TransactionHash.Hex(),
			event.LogIndex,
			event.ContractAddress.Hex(),
			event.EventName,
			payload,
			event.BlockTimestamp,
			eventStatus(&event),
			nullableString(event.CorrelationID),
			false,
		)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				p.logger.Error("failed to rollback transaction", logkeys.LogKeyError, rbErr)
			}
			p.RecordError()
			p.logger.Error("Failed to write event in batch", logkeys.LogKeyError, err, logkeys.LogKeyHash, event.EventHash)
			return fmt.Errorf("failed to write event in batch: %w", err)
		}

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

// WriteBatch writes multiple blockchain events atomically using a single DB transaction.
func (p *PostgreSQLDatabase) WriteBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error {
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

	ctx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		p.RecordError()
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	insertSQL := insertEventSQL

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.Error("failed to rollback transaction", logkeys.LogKeyError, rbErr)
		}
		p.RecordError()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			p.logger.Error("failed to close statement", logkeys.LogKeyError, closeErr)
		}
	}()

	for _, event := range events {
		if event == nil {
			continue
		}
		payload, err := eventPayloadJSON(event)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		_, err = stmt.ExecContext(
			ctx,
			eventPK(event),
			event.ChainID,
			event.BlockNumber,
			event.BlockHash.Hex(),
			event.TransactionHash.Hex(),
			event.LogIndex,
			event.ContractAddress.Hex(),
			event.EventName,
			payload,
			event.BlockTimestamp,
			eventStatus(event),
			nullableString(event.CorrelationID),
			false,
		)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				p.logger.Error("failed to rollback transaction", logkeys.LogKeyError, rbErr)
			}
			p.RecordError()
			p.logger.Error("Failed to write event in batch", logkeys.LogKeyError, err, logkeys.LogKeyHash, event.EventHash)
			return fmt.Errorf("failed to write event in batch: %w", err)
		}

	}

	if err := tx.Commit(); err != nil {
		p.RecordError()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	p.RecordWrite(duration)
	p.updateEventCount()

	return nil
}

// QueryEvents queries events from the database
func (p *PostgreSQLDatabase) QueryEvents(filter *core.EventFilter) (*portcontract.QueryResult, error) {
	return p.QueryEventsContext(context.Background(), filter)
}

// QueryEventsContext preserves cancellation across the database boundary.
func (p *PostgreSQLDatabase) QueryEventsContext(ctx context.Context, filter *core.EventFilter) (*portcontract.QueryResult, error) {
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

	ctx, cancel := NewContextWithTimeout(ctx, p.queryTimeout)
	defer cancel()

	// Build query
	query := "SELECT " + selectEventColumns + " FROM blockchain_events WHERE 1=1"
	args := []any{}
	argIndex := 1
	if filter.Network != "" {
		query += fmt.Sprintf(" AND chain_id = $%d", argIndex)
		args = append(args, filter.Network)
		argIndex++
	}
	if len(filter.Status) == 0 {
		query += " AND status <> 'reorged'"
	} else {
		query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
		args = append(args, pq.Array(filter.Status))
		argIndex++
	}

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
	query += " ORDER BY block_number DESC, log_index DESC, id DESC"
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
		p.logger.Error("Failed to query events from PostgreSQL", logkeys.LogKeyError, err)
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			p.logger.Error("failed to close rows", logkeys.LogKeyError, err)
		}
	}()

	events := []blockchain.BlockchainEvent{}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			p.RecordError()
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, *event)
	}

	if err := rows.Err(); err != nil {
		p.RecordError()
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	p.RecordRead(duration)

	result := &portcontract.QueryResult{
		Events:       events,
		Total:        int64(len(events)),
		ResponseTime: duration,
	}

	return result, nil
}

// GetEventByHash retrieves an event by its hash
func (p *PostgreSQLDatabase) GetEventByHash(hash string) (*blockchain.BlockchainEvent, error) {
	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		p.RecordError()
		return nil, fmt.Errorf("database not initialized")
	}

	ctx, cancel := NewContextWithTimeout(context.Background(), p.queryTimeout)
	defer cancel()

	query := "SELECT " + selectEventColumns + " FROM blockchain_events WHERE id = $1"

	event, err := scanEvent(db.QueryRowContext(ctx, query, hash))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			duration := time.Since(start).Milliseconds()
			p.RecordRead(duration)
			return nil, nil
		}
		p.RecordError()
		p.logger.Error("Failed to get event by hash from PostgreSQL", logkeys.LogKeyError, err, logkeys.LogKeyHash, hash)
		return nil, fmt.Errorf("failed to get event by hash: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	p.RecordRead(duration)

	// Update in-memory cache

	return event, nil
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
		p.logger.Error("Failed to delete event from PostgreSQL", logkeys.LogKeyError, err, logkeys.LogKeyEventID, eventID)
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

	ctx, cancel := NewContextWithTimeout(context.Background(), p.queryTimeout)
	defer cancel()

	var count int64
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blockchain_events").Scan(&count)
	if err != nil {
		p.logger.Error("Failed to update event count", logkeys.LogKeyError, err)
		return
	}

	p.UpdateEventCount(count)
}

// NewContextWithTimeout creates a context with timeout derived from the parent context
func NewContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// GetAllEvents retrieves all events from the database
func (p *PostgreSQLDatabase) GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT ` + selectEventColumns + `
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
			p.logger.Error("failed to close rows", logkeys.LogKeyError, err)
		}
	}()

	var events []*blockchain.BlockchainEvent
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
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
func (p *PostgreSQLDatabase) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT number, hash, parent_hash, timestamp
		FROM blocks
		ORDER BY number ASC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query blocks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var blocks []*blockchain.Block
	for rows.Next() {
		block := &blockchain.Block{}
		var hash, parentHash string
		if err := rows.Scan(
			&block.Number, &hash, &parentHash, &block.Timestamp,
		); err != nil {
			p.RecordError()
			return nil, fmt.Errorf("failed to scan block: %w", err)
		}
		block.Hash = common.HexToHash(hash)
		block.ParentHash = common.HexToHash(parentHash)
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
func (p *PostgreSQLDatabase) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*blockchain.BlockchainEvent, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT ` + selectEventColumns + `
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

	var events []*blockchain.BlockchainEvent
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
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
func (p *PostgreSQLDatabase) GetBlock(ctx context.Context, blockNumber uint64) (*blockchain.Block, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT number, hash, parent_hash, timestamp
		FROM blocks
		WHERE number = $1
	`

	block := &blockchain.Block{}
	var hash, parentHash string
	err := db.QueryRowContext(ctx, query, blockNumber).Scan(
		&block.Number, &hash, &parentHash, &block.Timestamp,
	)
	if err == nil {
		block.Hash = common.HexToHash(hash)
		block.ParentHash = common.HexToHash(parentHash)
	}

	if errors.Is(err, sql.ErrNoRows) {
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

// MarkEventsAsReorged marks events within a block range as reorged (soft delete)
func (p *PostgreSQLDatabase) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	p.mu.RLock()
	db := p.db
	p.mu.RUnlock()

	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	query := `UPDATE blockchain_events SET status = 'reorged' WHERE block_number >= $1 AND block_number <= $2`

	result, err := db.ExecContext(ctx, query, fromBlock, toBlock)
	if err != nil {
		p.RecordError()
		return 0, fmt.Errorf("failed to mark events as reorged: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		p.RecordError()
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	p.RecordWrite(rowsAffected)
	return rowsAffected, nil
}

// GetReorgStats retrieves reorg statistics
func (p *PostgreSQLDatabase) GetReorgStats(ctx context.Context) (*portcontract.ReorgStats, error) {
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

	stats := &portcontract.ReorgStats{}
	err := db.QueryRowContext(ctx, query).Scan(
		&stats.TotalReorgsDetected, &stats.TotalBlocksRolledBack, &stats.AverageReorgSize,
		&stats.LastReorgTime, &stats.LastReorgBlock,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return &portcontract.ReorgStats{}, nil
	}
	if err != nil {
		p.RecordError()
		return nil, fmt.Errorf("failed to query reorg stats: %w", err)
	}

	p.RecordRead(0)
	return stats, nil
}

// orDefault returns s if non-empty, otherwise def.
func orDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}
