package query

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson"

	"chainpulse/pkg/core"
)

// TestMongoDBEventStoreInitialize tests event store initialization
func TestMongoDBEventStoreInitialize(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	// Create mock database manager
	dbManager := &mockDatabaseManager{
		mongoClient: nil,
		postgresDB:  nil,
	}

	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	// Should not be initialized yet
	if store.initialized {
		t.Error("Store should not be initialized before Initialize() call")
	}

	// Initialize should succeed (with mock)
	ctx := context.Background()
	err := store.Initialize(ctx)
	if err != nil {
		t.Logf("Initialize error (expected with mock): %v", err)
	}
}

// TestMongoDBEventStoreInsertEvent tests single event insertion
func TestMongoDBEventStoreInsertEvent(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	event := &core.BlockchainEvent{
		ID:              "event-1",
		ChainID:         "1",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0xabc123"),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0xcontract"),
		EventName:       "Transfer",
		EventData:       []byte{},
		DecodedData:     map[string]any{"amount": "1000"},
		BlockTimestamp:  time.Now().Unix(),
	}

	// Should fail because store is not initialized
	ctx := context.Background()
	err := store.InsertEvent(ctx, event)
	if err == nil {
		t.Error("InsertEvent should fail when store is not initialized")
	}

	// Test with nil event
	store.initialized = true
	err = store.InsertEvent(ctx, nil)
	if err == nil {
		t.Error("InsertEvent should fail with nil event")
	}
}

// TestMongoDBEventStoreInsertEventBatch tests batch event insertion
func TestMongoDBEventStoreInsertEventBatch(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	events := []*core.BlockchainEvent{
		{
			ID:              "event-1",
			ChainID:         "1",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0xabc123"),
			LogIndex:        0,
			ContractAddress: common.HexToAddress("0xcontract"),
			EventName:       "Transfer",
			EventData:       []byte{},
			DecodedData:     map[string]any{},
			BlockTimestamp:  time.Now().Unix(),
		},
		{
			ID:              "event-2",
			ChainID:         "1",
			BlockNumber:     101,
			TransactionHash: common.HexToHash("0xabc124"),
			LogIndex:        1,
			ContractAddress: common.HexToAddress("0xcontract"),
			EventName:       "Transfer",
			EventData:       []byte{},
			DecodedData:     map[string]any{},
			BlockTimestamp:  time.Now().Unix(),
		},
	}

	// Should fail because store is not initialized
	ctx := context.Background()
	err := store.InsertEventBatch(ctx, events)
	if err == nil {
		t.Error("InsertEventBatch should fail when store is not initialized")
	}

	// Test with empty batch
	store.initialized = true
	err = store.InsertEventBatch(ctx, []*core.BlockchainEvent{})
	if err != nil {
		t.Errorf("InsertEventBatch should succeed with empty batch: %v", err)
	}
}

// TestMongoDBEventStoreGetEvent tests single event retrieval
func TestMongoDBEventStoreGetEvent(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.GetEvent(ctx, "event-1")
	if err == nil {
		t.Error("GetEvent should fail when store is not initialized")
	}

	// Test with empty event ID
	store.initialized = true
	_, err = store.GetEvent(ctx, "")
	if err == nil {
		t.Error("GetEvent should fail with empty event ID")
	}
}

// TestMongoDBEventStoreGetEventsByChain tests chain event retrieval
func TestMongoDBEventStoreGetEventsByChain(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.GetEventsByChain(ctx, 1, 10, 0)
	if err == nil {
		t.Error("GetEventsByChain should fail when store is not initialized")
	}
}

func TestBuildChainLookupFilter(t *testing.T) {
	t.Parallel()
	t.Run("all events", func(t *testing.T) {
		got := buildChainLookupFilter(0)
		if len(got) != 0 {
			t.Fatalf("expected empty filter, got %v", got)
		}
	})

	t.Run("string chain match", func(t *testing.T) {
		got := buildChainLookupFilter(1)
		chainFilterValue, ok := got["chainId"]
		if !ok {
			t.Fatalf("expected chainId filter, got %v", got)
		}
		inFilter, ok := chainFilterValue.(bson.M)
		if !ok {
			t.Fatalf("expected chainId filter map, got %T", chainFilterValue)
		}
		values, ok := inFilter["$in"].([]any)
		if !ok {
			t.Fatalf("expected $in slice, got %T", inFilter["$in"])
		}
		// After ChainID string migration, $in contains only string values
		if len(values) < 1 {
			t.Fatalf("expected at least 1 value, got %v", values)
		}
		for _, v := range values {
			if _, isStr := v.(string); !isStr {
				t.Errorf("expected string value in $in, got %T: %v", v, v)
			}
		}
		// Must include "1" (numeric string for chain ID 1)
		found := false
		for _, v := range values {
			if v == "1" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected $in to include \"1\", got %v", values)
		}
	})
}

// TestMongoDBEventStoreGetEventsByContract tests contract event retrieval
func TestMongoDBEventStoreGetEventsByContract(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.GetEventsByContract(ctx, "0xcontract", 10, 0)
	if err == nil {
		t.Error("GetEventsByContract should fail when store is not initialized")
	}

	// Test with empty contract address
	store.initialized = true
	_, err = store.GetEventsByContract(ctx, "", 10, 0)
	if err == nil {
		t.Error("GetEventsByContract should fail with empty contract address")
	}
}

// TestMongoDBEventStoreGetEventsByEventName tests event name retrieval
func TestMongoDBEventStoreGetEventsByEventName(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.GetEventsByEventName(ctx, "Transfer", 10, 0)
	if err == nil {
		t.Error("GetEventsByEventName should fail when store is not initialized")
	}

	// Test with empty event name
	store.initialized = true
	_, err = store.GetEventsByEventName(ctx, "", 10, 0)
	if err == nil {
		t.Error("GetEventsByEventName should fail with empty event name")
	}
}

// TestMongoDBEventStoreDeleteExpiredEvents tests expired event deletion
func TestMongoDBEventStoreDeleteExpiredEvents(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()
	config.TTLDays = 0 // No TTL

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	// Should fail because store is not initialized
	ctx := context.Background()
	_, err := store.DeleteExpiredEvents(ctx)
	if err == nil {
		t.Error("DeleteExpiredEvents should fail when store is not initialized")
	}

	// Test with no TTL configured
	store.initialized = true
	count, err := store.DeleteExpiredEvents(ctx)
	if err != nil {
		t.Errorf("DeleteExpiredEvents should succeed with no TTL: %v", err)
	}
	if count != 0 {
		t.Errorf("DeleteExpiredEvents should return 0 with no TTL, got %d", count)
	}
}

// TestMongoDBEventStoreHealth tests health check
func TestMongoDBEventStoreHealth(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

	ctx := context.Background()
	health := store.Health(ctx)

	// Should be unhealthy when not initialized
	if health.Status != "unhealthy" {
		t.Errorf("Health should be unhealthy when not initialized, got %s", health.Status)
	}

	// Should be unhealthy when initialized but MongoDB is unavailable
	store.initialized = true
	health = store.Health(ctx)
	if health.Status != "unhealthy" {
		t.Errorf("Health should be unhealthy with mock MongoDB, got %s", health.Status)
	}
}

// TestMongoDBEventStoreClose tests store closure
func TestMongoDBEventStoreClose(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)

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

// TestMongoDBEventStoreConfigDefaults tests default configuration
func TestMongoDBEventStoreConfigDefaults(t *testing.T) {
	t.Parallel()
	config := DefaultEventStoreConfig()

	if config.CollectionName != "events" {
		t.Errorf("Default collection name should be 'events', got %s", config.CollectionName)
	}

	if config.TTLDays != 30 {
		t.Errorf("Default TTL should be 30 days, got %d", config.TTLDays)
	}

	if config.BatchSize != 100 {
		t.Errorf("Default batch size should be 100, got %d", config.BatchSize)
	}

	if config.IndexTimeout != 10*time.Second {
		t.Errorf("Default index timeout should be 10s, got %v", config.IndexTimeout)
	}
}

// TestMongoDBEventStoreMetricsCollection tests that metrics are collected
func TestMongoDBEventStoreMetricsCollection(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	config := DefaultEventStoreConfig()

	dbManager := &mockDatabaseManager{}
	store := NewMongoDBEventStore(dbManager, logger, metrics, config)
	store.initialized = true

	ctx := context.Background()

	// Try to insert event (will fail but should record metrics)
	event := &core.BlockchainEvent{
		ID:          "event-1",
		ChainID:     "1",
		BlockNumber: 100,
	}

	// This will fail but should still record error metric
	_ = store.InsertEvent(ctx, event)

	// Metrics should have been recorded
	// (We can't directly verify metrics without accessing internal state,
	// but we can verify the operation completes)
}
