package e2e

import (
	"context"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseManager manages database interactions for E2E tests
type DatabaseManager struct {
	pool        *pgxpool.Pool
	connString  string
	initialized bool
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(connString string) *DatabaseManager {
	return &DatabaseManager{
		connString: connString,
	}
}

// Initialize connects to the database
func (dm *DatabaseManager) Initialize(ctx context.Context) error {
	if dm.initialized {
		return fmt.Errorf("database manager already initialized")
	}

	config, err := pgxpool.ParseConfig(dm.connString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	dm.pool = pool
	dm.initialized = true

	return nil
}

// Close closes the database connection
func (dm *DatabaseManager) Close() error {
	if dm.pool != nil {
		dm.pool.Close()
	}
	dm.initialized = false
	return nil
}

// GetPool returns the connection pool
func (dm *DatabaseManager) GetPool() *pgxpool.Pool {
	return dm.pool
}

// GetConnection returns a single connection from the pool
func (dm *DatabaseManager) GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	if !dm.initialized {
		return nil, fmt.Errorf("database manager not initialized")
	}

	conn, err := dm.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	return conn, nil
}

// ExecuteQuery executes a query and returns rows
func (dm *DatabaseManager) ExecuteQuery(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if !dm.initialized {
		return nil, fmt.Errorf("database manager not initialized")
	}

	rows, err := dm.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return rows, nil
}

// ExecuteCommand executes a command (INSERT, UPDATE, DELETE)
func (dm *DatabaseManager) ExecuteCommand(ctx context.Context, query string, args ...any) (int64, error) {
	if !dm.initialized {
		return 0, fmt.Errorf("database manager not initialized")
	}

	result, err := dm.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to execute command: %w", err)
	}

	return result.RowsAffected(), nil
}

// ExecuteRow executes a query that returns a single row
func (dm *DatabaseManager) ExecuteRow(ctx context.Context, query string, args ...any) pgx.Row {
	if !dm.initialized {
		return nil
	}

	return dm.pool.QueryRow(ctx, query, args...)
}

// BeginTransaction begins a transaction
func (dm *DatabaseManager) BeginTransaction(ctx context.Context) (pgx.Tx, error) {
	if !dm.initialized {
		return nil, fmt.Errorf("database manager not initialized")
	}

	tx, err := dm.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

// ClearTable clears all rows from a table
func (dm *DatabaseManager) ClearTable(ctx context.Context, tableName string) error {
	if !dm.initialized {
		return fmt.Errorf("database manager not initialized")
	}

	query := fmt.Sprintf("DELETE FROM %s", tableName)
	_, err := dm.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear table %s: %w", tableName, err)
	}

	return nil
}

// ClearAllTables clears all tables
func (dm *DatabaseManager) ClearAllTables(ctx context.Context, tables []string) error {
	if !dm.initialized {
		return fmt.Errorf("database manager not initialized")
	}

	for _, table := range tables {
		if err := dm.ClearTable(ctx, table); err != nil {
			return err
		}
	}

	return nil
}

// GetTableRowCount returns the number of rows in a table
func (dm *DatabaseManager) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	if !dm.initialized {
		return 0, fmt.Errorf("database manager not initialized")
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	var count int64
	err := dm.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get row count: %w", err)
	}

	return count, nil
}

// IsConnected checks if the database manager is connected
func (dm *DatabaseManager) IsConnected(ctx context.Context) bool {
	if !dm.initialized {
		return false
	}

	err := dm.pool.Ping(ctx)
	return err == nil
}

// WaitForConnection waits for the database to be ready
func (dm *DatabaseManager) WaitForConnection(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database connection: %w", ctx.Err())
		case <-ticker.C:
			if dm.IsConnected(ctx) {
				return nil
			}
		}
	}
}

// CreateTable creates a table with the given schema
func (dm *DatabaseManager) CreateTable(ctx context.Context, schema string) error {
	if !dm.initialized {
		return fmt.Errorf("database manager not initialized")
	}

	_, err := dm.pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// DropTable drops a table
func (dm *DatabaseManager) DropTable(ctx context.Context, tableName string) error {
	if !dm.initialized {
		return fmt.Errorf("database manager not initialized")
	}

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := dm.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	return nil
}
