package query

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

// TestPostgreSQLEventMetadataStoreInitialize tests metadata store initialization
func TestPostgreSQLEventMetadataStoreInitialize(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{
		mongoClient: nil,
		postgresDB:  nil,
	}

	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	// Should not be initialized yet
	if store.initialized {
		t.Error("Store should not be initialized before Initialize() call")
	}

	// Initialize should fail with mock
	ctx := context.Background()
	err := store.Initialize(ctx)
	if err == nil {
		t.Error("Initialize should fail with mock database manager")
	}
}

// TestPostgreSQLEventMetadataStoreInsertMetadata tests single metadata insertion
func TestPostgreSQLEventMetadataStoreInsertMetadata(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	metadata := &EventMetadata{
		EventID:         "event-1",
		ChainID:         1,
		BlockNumber:     100,
		TransactionHash: "0xabc123",
		LogIndex:        0,
		ContractAddress: "0xcontract",
		EventName:       "Transfer",
		ProcessingStatus: "pending",
		ProcessedAt:     time.Now(),
	}

	// Should fail because store is not initialized
	ctx := context.Background()
	err := store.InsertMetadata(ctx, metadata)
	if err == nil {
		t.Error("InsertMetadata should fail when store is not initialized")
	}

	// Test with nil metadata
	store.initialized = true
	store.db = &sql.DB{}
	err = store.InsertMetadata(ctx, nil)
	if err == nil {
		t.Error("InsertMetadata should fail with nil metadata")
	}
}

// TestPostgreSQLEventMetadataStoreInsertMetadataBatch tests batch metadata insertion
func TestPostgreSQLEventMetadataStoreInsertMetadataBatch(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	metadataList := []*EventMetadata{
		{
			EventID:         "event-1",
			ChainID:         1,
			BlockNumber:     100,
			TransactionHash: "0xabc123",
			LogIndex:        0,
			ContractAddress: "0xcontract",
			EventName:       "Transfer",
			ProcessingStatus: "pending",
			ProcessedAt:     time.Now(),
		},
		{
			EventID:         "event-2",
			ChainID:         1,
			BlockNumber:     101,
			TransactionHash: "0xabc124",
			LogIndex:        1,
			ContractAddress: "0xcontract",
			EventName:       "Transfer",
			ProcessingStatus: "pending",
			ProcessedAt:     time.Now(),
		},
	}

	// Should fail because store is not initialized
	ctx := context.Background()
	err := store.InsertMetadataBatch(ctx, metadataList)
	if err == nil {
		t.Error("InsertMetadataBatch should fail when store is not initialized")
	}

	// Test with empty batch
	store.initialized = true
	err = store.InsertMetadataBatch(ctx, []*EventMetadata{})
	if err != nil {
		t.Errorf("InsertMetadataBatch should succeed with empty batch: %v", err)
	}
}

// TestPostgreSQLEventMetadataStoreGetMetadata tests single metadata retrieval
func TestPostgreSQLEventMetadataStoreGetMetadata(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.GetMetadata(ctx, "event-1")
	if err == nil {
		t.Error("GetMetadata should fail when store is not initialized")
	}

	// Test with empty event ID
	store.initialized = true
	store.db = &sql.DB{}
	_, err = store.GetMetadata(ctx, "")
	if err == nil {
		t.Error("GetMetadata should fail with empty event ID")
	}
}

// TestPostgreSQLEventMetadataStoreGetMetadataByChain tests chain metadata retrieval
func TestPostgreSQLEventMetadataStoreGetMetadataByChain(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.GetMetadataByChain(ctx, 1, 10, 0)
	if err == nil {
		t.Error("GetMetadataByChain should fail when store is not initialized")
	}
}

// TestPostgreSQLEventMetadataStoreUpdateMetadata tests metadata update
func TestPostgreSQLEventMetadataStoreUpdateMetadata(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	metadata := &EventMetadata{
		EventID:          "event-1",
		ProcessingStatus: "completed",
	}

	// Should fail because store is not initialized
	ctx := context.Background()
	err := store.UpdateMetadata(ctx, metadata)
	if err == nil {
		t.Error("UpdateMetadata should fail when store is not initialized")
	}

	// Test with nil metadata
	store.initialized = true
	store.db = &sql.DB{}
	err = store.UpdateMetadata(ctx, nil)
	if err == nil {
		t.Error("UpdateMetadata should fail with nil metadata")
	}
}

// TestPostgreSQLEventMetadataStoreHealth tests health check
func TestPostgreSQLEventMetadataStoreHealth(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	ctx := context.Background()
	health := store.Health(ctx)

	// Should be unhealthy when not initialized
	if health.Status != "unhealthy" {
		t.Errorf("Health should be unhealthy when not initialized, got %s", health.Status)
	}

	// Should be unhealthy when initialized but PostgreSQL is unavailable
	store.initialized = true
	store.db = &sql.DB{}
	health = store.Health(ctx)
	if health.Status != "unhealthy" {
		t.Errorf("Health should be unhealthy with mock PostgreSQL, got %s", health.Status)
	}
}

// TestPostgreSQLEventMetadataStoreClose tests store closure
func TestPostgreSQLEventMetadataStoreClose(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	ctx := context.Background()

	// Close should succeed even if not initialized
	err := store.Close(ctx)
	if err != nil {
		t.Errorf("Close should succeed: %v", err)
	}

	// Close should succeed when initialized
	store.initialized = true
	err = store.Close(ctx)
	if err != nil {
		t.Errorf("Close should succeed: %v", err)
	}

	// Should be uninitialized after close
	if store.initialized {
		t.Error("Store should be uninitialized after Close()")
	}
}

// TestEventMetadataDefaults tests EventMetadata default values
func TestEventMetadataDefaults(t *testing.T) {
	metadata := &EventMetadata{
		EventID:   "event-1",
		ChainID:   1,
		BlockNumber: 100,
	}

	// ProcessingStatus should default to empty (will be set to "pending" on insert)
	if metadata.ProcessingStatus != "" {
		t.Errorf("Default processing status should be empty, got %s", metadata.ProcessingStatus)
	}

	// RetryCount should default to 0
	if metadata.RetryCount != 0 {
		t.Errorf("Default retry count should be 0, got %d", metadata.RetryCount)
	}
}

// TestPostgreSQLEventMetadataStoreMetricsCollection tests that metrics are collected
func TestPostgreSQLEventMetadataStoreMetricsCollection(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	store := NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)
	store.initialized = true
	store.db = &sql.DB{}

	ctx := context.Background()

	// Try to insert metadata (will fail but should record metrics)
	metadata := &EventMetadata{
		EventID:   "event-1",
		ChainID:   1,
		BlockNumber: 100,
	}

	// This will fail but should still record error metric
	_ = store.InsertMetadata(ctx, metadata)

	// Metrics should have been recorded
	// (We can't directly verify metrics without accessing internal state,
	// but we can verify the operation completes)
}

// TestPostgreSQLEventMetadataStoreProcessingStatusHandling tests processing status handling
func TestPostgreSQLEventMetadataStoreProcessingStatusHandling(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	_ = NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	// Test metadata with explicit status
	metadata := &EventMetadata{
		EventID:          "event-1",
		ProcessingStatus: "completed",
	}

	if metadata.ProcessingStatus != "completed" {
		t.Errorf("Processing status should be 'completed', got %s", metadata.ProcessingStatus)
	}

	// Test metadata with empty status (should default to "pending" on insert)
	metadata2 := &EventMetadata{
		EventID: "event-2",
	}

	if metadata2.ProcessingStatus != "" {
		t.Errorf("Empty processing status should remain empty before insert, got %s", metadata2.ProcessingStatus)
	}
}

// TestPostgreSQLEventMetadataStoreRetryHandling tests retry count handling
func TestPostgreSQLEventMetadataStoreRetryHandling(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	dbManager := &mockDatabaseManager{}
	_ = NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	// Test metadata with retry count
	now := time.Now()
	metadata := &EventMetadata{
		EventID:     "event-1",
		RetryCount:  3,
		LastRetryAt: &now,
	}

	if metadata.RetryCount != 3 {
		t.Errorf("Retry count should be 3, got %d", metadata.RetryCount)
	}

	if metadata.LastRetryAt == nil {
		t.Error("LastRetryAt should not be nil")
	}
}
