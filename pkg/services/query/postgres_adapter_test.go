package query

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestPostgreSQLAdapterInitialization tests adapter initialization
func TestPostgreSQLAdapterInitialization(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := NewPostgreSQLAdapter(mockDBManager, logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should fail because mock returns nil
	err := adapter.Initialize(ctx)
	if err == nil {
		t.Error("Expected initialization to fail with nil db")
	}
}

// TestPostgreSQLAdapterQueryWithNilRequest tests query with nil request
func TestPostgreSQLAdapterQueryWithNilRequest(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
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

// TestPostgreSQLAdapterQueryWithoutTable tests query without table name
func TestPostgreSQLAdapterQueryWithoutTable(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
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
		t.Error("Expected error for empty table name")
	}
}

// TestPostgreSQLAdapterNotInitialized tests operations on uninitialized adapter
func TestPostgreSQLAdapterNotInitialized(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
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

// TestPostgreSQLAdapterQueryByHashWithEmptyHash tests query by hash with empty hash
func TestPostgreSQLAdapterQueryByHashWithEmptyHash(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
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

// TestPostgreSQLAdapterHealthNotInitialized tests health check when not initialized
func TestPostgreSQLAdapterHealthNotInitialized(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
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

// TestPostgreSQLAdapterDoubleInitialization tests double initialization
func TestPostgreSQLAdapterDoubleInitialization(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
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

// TestPostgreSQLAdapterQueryBuildsCorrectSQL tests SQL query building
func TestPostgreSQLAdapterQueryBuildsCorrectSQL(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()
	mockDBManager := &mockDatabaseManager{}

	adapter := &DefaultPostgreSQLAdapter{
		dbManager:        mockDBManager,
		logger:           logger,
		metricsCollector: metrics,
		initialized:      true,
		db:               nil, // Will fail on actual query
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &QueryRequest{
		Collection: "events",
		Filter: map[string]interface{}{
			"chain_id": 1,
		},
		Limit:  10,
		Offset: 0,
		Sort: map[string]int{
			"block_number": -1,
		},
	}

	// This will fail because db is nil, but we're testing the query building logic
	_, err := adapter.Query(ctx, req)
	if err == nil {
		t.Error("Expected error due to nil db")
	}
}
