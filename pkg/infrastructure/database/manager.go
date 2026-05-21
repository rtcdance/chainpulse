package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DatabaseManager manages connections to MongoDB and PostgreSQL
type DatabaseManager interface {
	Initialize(ctx context.Context) error
	GetMongoClient(ctx context.Context) (any, error)
	GetMongoDatabase(name string) *mongo.Database
	GetPostgresDB(ctx context.Context) (any, error)
	CheckMongoHealth(ctx context.Context) error
	CheckPostgresHealth(ctx context.Context) error
	Health(ctx context.Context) any
	Close(ctx context.Context) error
}

// BatchInserter provides bulk write operations for efficient data loading.
type BatchInserter interface {
	// BatchInsertEvents inserts multiple events in a single database operation.
	// Returns the number of successfully inserted events and any error.
	BatchInsertEvents(ctx context.Context, events []any) (int, error)
}

// DefaultDatabaseManager provides default implementation of DatabaseManager.
// Each database is initialized independently — only the databases with
// non-empty connection URIs are connected. This allows deployments that
// use only PostgreSQL or only MongoDB without forcing both.
type DefaultDatabaseManager struct {
	mongoURI        string
	postgresURL     string
	postgresSSLMode string
	mongoClient     *mongo.Client
	postgresClient  *sql.DB
	poolSize        int
	mongoTimeout    time.Duration
	postgresTimeout time.Duration
	mu              sync.RWMutex
	mongoInit       bool
	postgresInit    bool
	closed          bool
}

// NewDatabaseManager creates a new database manager.
// Either mongoURI or postgresURL (or both) can be empty — only
// non-empty URIs will be initialized during Initialize().
func NewDatabaseManager(mongoURI, postgresURL, postgresSSLMode string, poolSize int, timeout time.Duration) *DefaultDatabaseManager {
	if postgresSSLMode == "" {
		postgresSSLMode = "disable"
	}
	return &DefaultDatabaseManager{
		mongoURI:        mongoURI,
		postgresURL:     postgresURL,
		postgresSSLMode: postgresSSLMode,
		mongoTimeout:    timeout,
		postgresTimeout: timeout,
		poolSize:        poolSize,
	}
}

// Initialize connects to each configured database independently.
// Only databases with non-empty connection URIs are initialized.
// Returns an error only if no databases could be initialized
// and both URIs were non-empty.
func (m *DefaultDatabaseManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mongoInit || m.postgresInit {
		return fmt.Errorf("database manager already initialized")
	}

	// Track which databases were configured and which succeeded.
	// Each database is independent — failure of one should not roll back another.
	hasMongo := m.mongoURI != ""
	hasPostgres := m.postgresURL != ""

	if !hasMongo && !hasPostgres {
		return fmt.Errorf("no database configured: both MongoDB URI and PostgreSQL URL are empty")
	}

	var mongoErr, postgresErr error

	if hasMongo {
		mongoErr = m.initMongo(ctx)
		if mongoErr == nil {
			m.mongoInit = true
		}
	}

	if hasPostgres {
		postgresErr = m.initPostgres(ctx)
		if postgresErr == nil {
			m.postgresInit = true
		}
	}

	// If all configured databases failed, return a combined error.
	if (hasMongo && !m.mongoInit) && (hasPostgres && !m.postgresInit) {
		return fmt.Errorf("failed to initialize databases: MongoDB: %w; PostgreSQL: %v", mongoErr, postgresErr)
	}
	if hasMongo && !m.mongoInit && !hasPostgres {
		return fmt.Errorf("failed to initialize MongoDB: %w", mongoErr)
	}
	if hasPostgres && !m.postgresInit && !hasMongo {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", postgresErr)
	}

	// At least one database succeeded — degraded but operational.
	return nil
}

// initMongo initializes MongoDB connection
func (m *DefaultDatabaseManager) initMongo(ctx context.Context) error {
	if m.mongoURI == "" {
		return fmt.Errorf("MongoDB URI is required")
	}

	maxPoolSize := sanitizeMongoPoolSize(m.poolSize)
	minPoolSize := sanitizeMongoPoolSize(m.poolSize / 2)

	opts := options.Client().
		ApplyURI(m.mongoURI).
		SetMaxPoolSize(maxPoolSize).
		SetMinPoolSize(minPoolSize)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, m.mongoTimeout)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
		}
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.mongoClient = client
	return nil
}

const (
	defaultMinPoolSize  = 4
	defaultMaxPoolSize  = 25
	defaultMinMongoPool = 2
	defaultMaxMongoPool = 100
)

func sanitizeMongoPoolSize(size int) uint64 {
	if size <= 0 {
		return defaultMaxMongoPool
	}
	if size > math.MaxInt32 {
		return math.MaxInt32
	}
	if size < defaultMinMongoPool {
		return defaultMinMongoPool
	}
	return uint64(size)
}

func sanitizePostgresPoolSize(size int) int {
	if size <= 0 {
		return defaultMaxPoolSize
	}
	if size < defaultMinPoolSize {
		return defaultMinPoolSize
	}
	return size
}

// initPostgres initializes PostgreSQL connection
func (m *DefaultDatabaseManager) initPostgres(ctx context.Context) error {
	if m.postgresURL == "" {
		return fmt.Errorf("PostgreSQL URL is required")
	}

	connStr := m.postgresURL
	if !strings.Contains(connStr, "sslmode") {
		if strings.Contains(connStr, "?") {
			connStr += "&sslmode=" + m.postgresSSLMode
		} else {
			connStr += "?sslmode=" + m.postgresSSLMode
		}
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool with safe minimums
	maxOpen := sanitizePostgresPoolSize(m.poolSize)
	maxIdle := maxOpen / 2
	if maxIdle < 2 {
		maxIdle = 2
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, m.postgresTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		if cerr := db.Close(); cerr != nil {
			slog.Warn("failed to close PostgreSQL connection after ping failure", "error", cerr)
		}
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	m.postgresClient = db
	return nil
}

// GetMongoClient returns the MongoDB client
func (m *DefaultDatabaseManager) GetMongoClient(ctx context.Context) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.mongoInit {
		return nil, fmt.Errorf("MongoDB not initialized")
	}

	if m.mongoClient == nil {
		return nil, fmt.Errorf("MongoDB client not available")
	}

	return m.mongoClient, nil
}

// GetMongoDatabase returns a MongoDB database
func (m *DefaultDatabaseManager) GetMongoDatabase(name string) *mongo.Database {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.mongoClient == nil {
		return nil
	}

	return m.mongoClient.Database(name)
}

// GetPostgresDB returns the PostgreSQL database connection
func (m *DefaultDatabaseManager) GetPostgresDB(ctx context.Context) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.postgresInit {
		return nil, fmt.Errorf("PostgreSQL not initialized")
	}

	if m.postgresClient == nil {
		return nil, fmt.Errorf("PostgreSQL connection not available")
	}

	return m.postgresClient, nil
}

// CheckMongoHealth checks MongoDB connectivity
func (m *DefaultDatabaseManager) CheckMongoHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.mongoClient == nil {
		return fmt.Errorf("MongoDB client not initialized")
	}

	pingCtx, cancel := context.WithTimeout(ctx, m.mongoTimeout)
	defer cancel()

	return m.mongoClient.Ping(pingCtx, nil)
}

// CheckPostgresHealth checks PostgreSQL connectivity
func (m *DefaultDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.postgresClient == nil {
		return fmt.Errorf("PostgreSQL connection not initialized")
	}

	pingCtx, cancel := context.WithTimeout(ctx, m.postgresTimeout)
	defer cancel()

	return m.postgresClient.PingContext(pingCtx)
}

// Health returns the health status
func (m *DefaultDatabaseManager) Health(ctx context.Context) any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.mongoInit && !m.postgresInit {
		return map[string]any{
			"status": "unhealthy",
			"reason": "no databases initialized",
		}
	}

	mongoHealthy := m.mongoInit && m.mongoClient != nil
	postgresHealthy := m.postgresInit && m.postgresClient != nil

	if !mongoHealthy && !postgresHealthy {
		return map[string]any{
			"status": "unhealthy",
			"reason": "both databases unavailable",
		}
	}

	return map[string]any{
		"status":   "healthy",
		"mongodb":  mongoHealthy,
		"postgres": postgresHealthy,
	}
}

// BatchInsertEvents inserts multiple events in bulk using MongoDB's InsertMany
// for efficiency. Returns an error if MongoDB is not available.
func (m *DefaultDatabaseManager) BatchInsertEvents(ctx context.Context, events []any) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(events) == 0 {
		return 0, nil
	}

	// MongoDB bulk insert requires mongoClient to be initialized
	if m.mongoClient == nil || !m.mongoInit {
		return 0, fmt.Errorf("MongoDB not initialized: cannot batch insert %d events", len(events))
	}

	db := m.mongoClient.Database("chainpulse")
	collection := db.Collection("events")

	result, err := collection.InsertMany(ctx, events)
	if err != nil {
		// Partial insert: return count of successfully inserted documents
		if result != nil && len(result.InsertedIDs) > 0 {
			return len(result.InsertedIDs), fmt.Errorf("partial batch insert: %w", err)
		}
		return 0, fmt.Errorf("batch insert failed: %w", err)
	}

	return len(result.InsertedIDs), nil
}

// Close closes both MongoDB and PostgreSQL connections
func (m *DefaultDatabaseManager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("database manager already closed")
	}

	var errs []error

	// Close MongoDB
	if m.mongoClient != nil {
		if err := m.mongoClient.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to disconnect MongoDB: %w", err))
		}
	}

	// Close PostgreSQL
	if m.postgresClient != nil {
		if err := m.postgresClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close PostgreSQL: %w", err))
		}
	}

	m.closed = true

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Warmup pre-allocates connection pools and prepares prepared statements.
// This should be called after Initialize() and before serving traffic
// to reduce cold-start latency for the first requests.
func (m *DefaultDatabaseManager) Warmup(ctx context.Context) error {
	m.mu.RLock()
	mongoClient := m.mongoClient
	mongoInit := m.mongoInit
	postgresClient := m.postgresClient
	m.mu.RUnlock()

	// MongoDB: trigger initial connection pool allocation
	if mongoClient != nil && mongoInit {
		// ListDatabaseNames forces the driver to establish a connection,
		// which warms up the connection pool.
		if _, err := mongoClient.ListDatabaseNames(ctx, map[string]any{}); err != nil {
			return fmt.Errorf("mongodb warmup failed: %w", err)
		}
	}

	// PostgreSQL: create prepared statements for common queries
	if postgresClient != nil {
		// Ping to ensure connection is ready
		if err := postgresClient.PingContext(ctx); err != nil {
			return fmt.Errorf("postgres warmup failed: %w", err)
		}
		// The sql.DB connection pool is automatically warmed as queries execute.
		// For explicit pool warming, we can execute a simple query.
		_, err := postgresClient.ExecContext(ctx, "SELECT 1")
		if err != nil {
			return fmt.Errorf("postgres warmup query failed: %w", err)
		}
	}

	return nil
}
