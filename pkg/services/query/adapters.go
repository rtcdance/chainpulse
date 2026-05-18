package query

import (
	"context"
	"time"

	"chainpulse/pkg/core"
)

// MongoDBAdapter defines the interface for MongoDB query operations
type MongoDBAdapter interface {
	// Initialize initializes the adapter
	Initialize(ctx context.Context) error

	// Query executes a query against MongoDB
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)

	// QueryByHash retrieves a single item by hash
	QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error)

	// Health returns the health status
	Health(ctx context.Context) *core.HealthStatus
}

// PostgreSQLAdapter defines the interface for PostgreSQL query operations
type PostgreSQLAdapter interface {
	// Initialize initializes the adapter
	Initialize(ctx context.Context) error

	// Query executes a query against PostgreSQL
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)

	// QueryByHash retrieves a single item by hash
	QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error)

	// Health returns the health status
	Health(ctx context.Context) *core.HealthStatus
}

// CacheService defines the interface for cache operations
type CacheService interface {
	// Initialize initializes the cache service
	Initialize(ctx context.Context) error

	// Start starts the cache service
	Start(ctx context.Context) error

	// Stop stops the cache service
	Stop(ctx context.Context) error

	// Get retrieves a cached value
	Get(ctx context.Context, key string) ([]core.BlockchainEvent, error)

	// GetSingle retrieves a single cached value
	GetSingle(ctx context.Context, key string) (*core.BlockchainEvent, error)

	// Set sets a cached value
	Set(ctx context.Context, key string, value []core.BlockchainEvent, ttl time.Duration) error

	// SetSingle sets a single cached value
	SetSingle(ctx context.Context, key string, value *core.BlockchainEvent, ttl time.Duration) error

	// SetQueryResult caches a query result with total count
	SetQueryResult(ctx context.Context, key string, events []core.BlockchainEvent, total int64, ttl time.Duration) error

	// GetQueryResult retrieves a cached query result with total count
	GetQueryResult(ctx context.Context, key string) ([]core.BlockchainEvent, int64, error)

	// Delete deletes a cached value
	Delete(ctx context.Context, key string) error

	// Health returns the health status
	Health(ctx context.Context) *core.HealthStatus
}
