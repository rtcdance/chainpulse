package query

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestMongoDBAdapterInitialization tests adapter initialization
func TestMongoDBAdapterInitialization(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	// Create a mock database manager
	mockDBManager := &mockDatabaseManager{
		mongoClient: nil,
	}

	adapter := NewMongoDBAdapter(mockDBManager, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should fail because mock returns nil
	err := adapter.Initialize(ctx)
	if err == nil {
		t.Error("Expected initialization to fail with nil client")
	}
}

// TestMongoDBAdapterQueryWithNilRequest tests query with nil request
func TestMongoDBAdapterQueryWithNilRequest(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultMongoDBAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := adapter.Query(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}
}

// TestMongoDBAdapterQueryWithoutCollection tests query without collection name
func TestMongoDBAdapterQueryWithoutCollection(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultMongoDBAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &QueryRequest{
		Collection: "",
		Filter:     map[string]interface{}{},
	}

	_, err := adapter.Query(ctx, req)
	if err == nil {
		t.Error("Expected error for empty collection name")
	}
}

// TestMongoDBAdapterNotInitialized tests operations on uninitialized adapter
func TestMongoDBAdapterNotInitialized(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultMongoDBAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &QueryRequest{
		Collection: "events",
		Filter:     map[string]interface{}{},
	}

	_, err := adapter.Query(ctx, req)
	if err == nil {
		t.Error("Expected error for uninitialized adapter")
	}

	_, err = adapter.QueryByHash(ctx, "test-hash")
	if err == nil {
		t.Error("Expected error for uninitialized adapter")
	}
}

// TestMongoDBAdapterQueryByHashWithEmptyHash tests query by hash with empty hash
func TestMongoDBAdapterQueryByHashWithEmptyHash(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultMongoDBAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := adapter.QueryByHash(ctx, "")
	if err == nil {
		t.Error("Expected error for empty hash")
	}
}

// TestMongoDBAdapterHealthNotInitialized tests health check when not initialized
func TestMongoDBAdapterHealthNotInitialized(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultMongoDBAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := adapter.Health(ctx)
	if health.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status, got %s", health.Status)
	}
}

// TestMongoDBAdapterDoubleInitialization tests double initialization
func TestMongoDBAdapterDoubleInitialization(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultMongoDBAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := adapter.Initialize(ctx)
	if err == nil {
		t.Error("Expected error for double initialization")
	}
}


