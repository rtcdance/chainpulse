package query

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// mockDatabaseManager is a mock implementation of DatabaseManager for testing
type mockDatabaseManager struct {
	mu          sync.RWMutex
	mongoClient any
	postgresDB  any
	mongoDb     *mongo.Database
}

func (m *mockDatabaseManager) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockDatabaseManager) Close(ctx context.Context) error {
	return nil
}

func (m *mockDatabaseManager) GetMongoClient(ctx context.Context) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mongoClient, nil
}

func (m *mockDatabaseManager) GetPostgresDB(ctx context.Context) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.postgresDB, nil
}

func (m *mockDatabaseManager) GetMongoDatabase(name string) *mongo.Database {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mongoDb
}

func (m *mockDatabaseManager) CheckMongoHealth(ctx context.Context) error {
	return nil
}

func (m *mockDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	return nil
}

func (m *mockDatabaseManager) Health(ctx context.Context) any {
	return map[string]any{"status": "healthy"}
}

// MockMongoDBAdapter is a mock implementation of MongoDB adapter for testing
type MockMongoDBAdapter struct {
	mu      sync.RWMutex
	healthy bool
}

func (m *MockMongoDBAdapter) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockMongoDBAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	return nil, nil
}

func (m *MockMongoDBAdapter) QueryByHash(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *MockMongoDBAdapter) Health(ctx context.Context) *core.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := "healthy"
	if !m.healthy {
		status = "unhealthy"
	}
	return &core.HealthStatus{
		Status: status,
	}
}

// MockPostgreSQLAdapter is a mock implementation of PostgreSQL adapter for testing
type MockPostgreSQLAdapter struct {
	mu      sync.RWMutex
	healthy bool
}

func (m *MockPostgreSQLAdapter) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockPostgreSQLAdapter) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	return nil, nil
}

func (m *MockPostgreSQLAdapter) QueryByHash(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (m *MockPostgreSQLAdapter) Health(ctx context.Context) *core.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := "healthy"
	if !m.healthy {
		status = "unhealthy"
	}
	return &core.HealthStatus{
		Status: status,
	}
}
