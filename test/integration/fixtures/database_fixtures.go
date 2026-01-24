package fixtures

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// DatabaseFixture provides database setup and teardown for integration tests
type DatabaseFixture struct {
	db *sql.DB
	t  *testing.T
}

// NewDatabaseFixture creates a new database fixture
func NewDatabaseFixture(t *testing.T, db *sql.DB) *DatabaseFixture {
	return &DatabaseFixture{
		db: db,
		t:  t,
	}
}

// Setup initializes the database for testing
func (f *DatabaseFixture) Setup(ctx context.Context) error {
	if f.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Verify connection
	if err := f.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create test tables if they don't exist
	if err := f.createTestTables(ctx); err != nil {
		return fmt.Errorf("failed to create test tables: %w", err)
	}

	return nil
}

// Cleanup cleans up the database after testing
func (f *DatabaseFixture) Cleanup(ctx context.Context) error {
	if f.db == nil {
		return nil
	}

	// Truncate all test tables
	if err := f.truncateTestTables(ctx); err != nil {
		return fmt.Errorf("failed to truncate test tables: %w", err)
	}

	return nil
}

// Close closes the database connection
func (f *DatabaseFixture) Close() error {
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}

// createTestTables creates the necessary test tables
func (f *DatabaseFixture) createTestTables(ctx context.Context) error {
	// Create blockchain_events table
	createEventsTable := `
	CREATE TABLE IF NOT EXISTS blockchain_events (
		id VARCHAR(255) PRIMARY KEY,
		chain_id VARCHAR(255) NOT NULL,
		block_number BIGINT NOT NULL,
		transaction_hash VARCHAR(255) NOT NULL,
		event_name VARCHAR(255) NOT NULL,
		event_data JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := f.db.ExecContext(ctx, createEventsTable); err != nil {
		return fmt.Errorf("failed to create blockchain_events table: %w", err)
	}

	// Create index on chain_id and block_number for faster queries
	createIndexSQL := `
	CREATE INDEX IF NOT EXISTS idx_blockchain_events_chain_block 
	ON blockchain_events(chain_id, block_number);
	`

	if _, err := f.db.ExecContext(ctx, createIndexSQL); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// truncateTestTables truncates all test tables
func (f *DatabaseFixture) truncateTestTables(ctx context.Context) error {
	tables := []string{"blockchain_events"}

	for _, table := range tables {
		truncateSQL := fmt.Sprintf("TRUNCATE TABLE %s CASCADE;", table)
		if _, err := f.db.ExecContext(ctx, truncateSQL); err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// InsertEvent inserts a test event into the database
func (f *DatabaseFixture) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	insertSQL := `
	INSERT INTO blockchain_events (id, chain_id, block_number, transaction_hash, event_name)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO NOTHING;
	`

	_, err := f.db.ExecContext(ctx, insertSQL,
		event.ID,
		event.ChainID,
		event.BlockNumber,
		event.TransactionHash.Hex(),
		event.EventName,
	)

	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// InsertEvents inserts multiple test events into the database
func (f *DatabaseFixture) InsertEvents(ctx context.Context, events []*core.BlockchainEvent) error {
	for _, event := range events {
		if err := f.InsertEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// GetEvent retrieves an event from the database
func (f *DatabaseFixture) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	query := `
	SELECT id, chain_id, block_number, transaction_hash, event_name
	FROM blockchain_events
	WHERE id = $1;
	`

	var event core.BlockchainEvent
	err := f.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID,
		&event.ChainID,
		&event.BlockNumber,
		&event.TransactionHash,
		&event.EventName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found: %s", eventID)
		}
		return nil, fmt.Errorf("failed to query event: %w", err)
	}

	return &event, nil
}

// GetEventsByChain retrieves all events for a specific chain
func (f *DatabaseFixture) GetEventsByChain(ctx context.Context, chainID string) ([]*core.BlockchainEvent, error) {
	query := `
	SELECT id, chain_id, block_number, transaction_hash, event_name
	FROM blockchain_events
	WHERE chain_id = $1
	ORDER BY block_number ASC;
	`

	rows, err := f.db.QueryContext(ctx, query, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []*core.BlockchainEvent
	for rows.Next() {
		var event core.BlockchainEvent
		if err := rows.Scan(
			&event.ID,
			&event.ChainID,
			&event.BlockNumber,
			&event.TransactionHash,
			&event.EventName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// GetEventsByBlockRange retrieves events within a block range
func (f *DatabaseFixture) GetEventsByBlockRange(ctx context.Context, chainID string, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	query := `
	SELECT id, chain_id, block_number, transaction_hash, event_name
	FROM blockchain_events
	WHERE chain_id = $1 AND block_number >= $2 AND block_number <= $3
	ORDER BY block_number ASC;
	`

	rows, err := f.db.QueryContext(ctx, query, chainID, fromBlock, toBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []*core.BlockchainEvent
	for rows.Next() {
		var event core.BlockchainEvent
		if err := rows.Scan(
			&event.ID,
			&event.ChainID,
			&event.BlockNumber,
			&event.TransactionHash,
			&event.EventName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// DeleteEvent deletes an event from the database
func (f *DatabaseFixture) DeleteEvent(ctx context.Context, eventID string) error {
	deleteSQL := "DELETE FROM blockchain_events WHERE id = $1;"

	result, err := f.db.ExecContext(ctx, deleteSQL, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

// EventCount returns the total number of events in the database
func (f *DatabaseFixture) EventCount(ctx context.Context) (int64, error) {
	var count int64
	err := f.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blockchain_events;").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

// BeginTx starts a transaction for test isolation
func (f *DatabaseFixture) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// WithTimeout returns a context with timeout
func (f *DatabaseFixture) WithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
