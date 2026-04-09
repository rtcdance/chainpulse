package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DatabaseManager manages connections to MongoDB and PostgreSQL
type DatabaseManager interface {
	// Initialize initializes the database manager
	Initialize(ctx context.Context) error

	// MongoDB operations
	GetMongoClient(ctx context.Context) (interface{}, error)
	GetMongoDatabase(name string) *mongo.Database

	// PostgreSQL operations
	GetPostgresDB(ctx context.Context) (interface{}, error)

	// Health checks
	CheckMongoHealth(ctx context.Context) error
	CheckPostgresHealth(ctx context.Context) error

	// Health returns the health status
	Health(ctx context.Context) interface{}

	// Lifecycle
	Close(ctx context.Context) error
}

// DefaultDatabaseManager provides default implementation of DatabaseManager
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
	initialized     bool
	closed          bool
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(mongoURI, postgresURL string, poolSize int, timeout time.Duration) *DefaultDatabaseManager {
	return &DefaultDatabaseManager{
		mongoURI:        mongoURI,
		postgresURL:     postgresURL,
		postgresSSLMode: "disable",
		mongoTimeout:    timeout,
		postgresTimeout: timeout,
		poolSize:        poolSize,
	}
}

// Initialize initializes both MongoDB and PostgreSQL connections
func (m *DefaultDatabaseManager) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("database manager already initialized")
	}

	// Initialize MongoDB
	if err := m.initMongo(ctx); err != nil {
		return fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	// Initialize PostgreSQL
	if err := m.initPostgres(ctx); err != nil {
		// Close MongoDB if PostgreSQL fails
		_ = m.mongoClient.Disconnect(context.Background())
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	m.initialized = true
	return nil
}

// initMongo initializes MongoDB connection
func (m *DefaultDatabaseManager) initMongo(ctx context.Context) error {
	if m.mongoURI == "" {
		return fmt.Errorf("MongoDB URI is required")
	}

	opts := options.Client().
		ApplyURI(m.mongoURI).
		SetMaxPoolSize(uint64(m.poolSize)).
		SetMinPoolSize(uint64(m.poolSize / 2))

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, m.mongoTimeout)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.mongoClient = client
	return nil
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

	// Configure connection pool
	db.SetMaxOpenConns(m.poolSize)
	db.SetMaxIdleConns(m.poolSize / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, m.postgresTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	m.postgresClient = db
	return nil
}

// GetMongoClient returns the MongoDB client
func (m *DefaultDatabaseManager) GetMongoClient(ctx context.Context) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("database manager not initialized")
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
func (m *DefaultDatabaseManager) GetPostgresDB(ctx context.Context) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("database manager not initialized")
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
func (m *DefaultDatabaseManager) Health(ctx context.Context) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return map[string]interface{}{
			"status": "unhealthy",
			"reason": "not initialized",
		}
	}

	mongoHealthy := m.mongoClient != nil
	postgresHealthy := m.postgresClient != nil

	if !mongoHealthy && !postgresHealthy {
		return map[string]interface{}{
			"status": "unhealthy",
			"reason": "both databases unavailable",
		}
	}

	return map[string]interface{}{
		"status":   "healthy",
		"mongodb":  mongoHealthy,
		"postgres": postgresHealthy,
	}
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
		return fmt.Errorf("errors closing database connections: %v", errs)
	}

	return nil
}
