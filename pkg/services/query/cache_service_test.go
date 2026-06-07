package query

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// TestCacheServiceInitialization tests cache service initialization
func TestCacheServiceInitialization(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := cs.Initialize(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize cache service: %v", err)
	}

	// Double initialization should fail
	err = cs.Initialize(ctx)
	if err == nil {
		t.Error("Expected error for double initialization")
	}
}

// TestCacheServiceStartStop tests cache service start and stop
func TestCacheServiceStartStop(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize first
	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
}

// TestCacheServiceSetAndGet tests setting and getting cache values
func TestCacheServiceSetAndGet(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Create test events
	events := []blockchain.BlockchainEvent{
		{
			EventHash:   "0x123",
			BlockNumber: 1000,
			ChainID:     "1",
		},
		{
			EventHash:   "0x456",
			BlockNumber: 1001,
			ChainID:     "1",
		},
	}

	// Set cache
	err := cs.Set(ctx, "test-key", events, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get cache
	retrieved, err := cs.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if len(retrieved) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(retrieved))
	}

	if retrieved[0].EventHash != events[0].EventHash {
		t.Errorf("Expected hash %s, got %s", events[0].EventHash, retrieved[0].EventHash)
	}
}

// TestCacheServiceSetAndGetSingle tests setting and getting single cache values
func TestCacheServiceSetAndGetSingle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Create test event
	event := &blockchain.BlockchainEvent{
		EventHash:   "0x789",
		BlockNumber: 2000,
		ChainID:     "1",
	}

	// Set cache
	err := cs.SetSingle(ctx, "single-key", event, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get cache
	retrieved, err := cs.GetSingle(ctx, "single-key")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if retrieved.EventHash != event.EventHash {
		t.Errorf("Expected hash %s, got %s", event.EventHash, retrieved.EventHash)
	}

	if retrieved.BlockNumber != event.BlockNumber {
		t.Errorf("Expected block number %d, got %d", event.BlockNumber, retrieved.BlockNumber)
	}
}

// TestCacheServiceDelete tests cache deletion
func TestCacheServiceDelete(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Set cache
	events := []blockchain.BlockchainEvent{{EventHash: "0xabc"}}
	if err := cs.Set(ctx, "delete-key", events, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Delete cache
	if err := cs.Delete(ctx, "delete-key"); err != nil {
		t.Fatalf("Failed to delete cache: %v", err)
	}

	// Get should fail
	_, err := cs.Get(ctx, "delete-key")
	if err == nil {
		t.Error("Expected error for deleted cache key")
	}
}

// TestCacheServiceExpiration tests cache expiration
func TestCacheServiceExpiration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Set cache with short TTL
	events := []blockchain.BlockchainEvent{{EventHash: "0xdef"}}
	if err := cs.Set(ctx, "expire-key", events, 100*time.Millisecond); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Get immediately should work
	_, err := cs.Get(ctx, "expire-key")
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get should fail
	_, err = cs.Get(ctx, "expire-key")
	if err == nil {
		t.Error("Expected error for expired cache key")
	}
}

// TestCacheServiceMiss tests cache miss
func TestCacheServiceMiss(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Get non-existent key
	_, err := cs.Get(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent cache key")
	}
}

// TestCacheServiceNotRunning tests operations when not running
func TestCacheServiceNotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Don't start, try to use
	events := []blockchain.BlockchainEvent{{EventHash: "0x111"}}
	err := cs.Set(ctx, "key", events, 1*time.Hour)
	if err == nil {
		t.Error("Expected error when cache service not running")
	}

	_, err = cs.Get(ctx, "key")
	if err == nil {
		t.Error("Expected error when cache service not running")
	}
}

// TestCacheServiceHealth tests health check
func TestCacheServiceHealth(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Not initialized
	health := cs.Health(ctx)
	if health.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status, got %s", health.Status)
	}

	// Initialize
	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Not running
	health = cs.Health(ctx)
	if health.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status, got %s", health.Status)
	}

	// Start
	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Running
	health = cs.Health(ctx)
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
}

// TestCacheServiceEmptyKey tests operations with empty key
func TestCacheServiceEmptyKey(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Set with empty key
	events := []blockchain.BlockchainEvent{{EventHash: "0x222"}}
	err := cs.Set(ctx, "", events, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for empty key")
	}

	// Get with empty key
	_, err = cs.Get(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key")
	}

	// Delete with empty key
	err = cs.Delete(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

// TestCacheServiceNilValue tests operations with nil value
func TestCacheServiceNilValue(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	// Set with nil value
	err := cs.Set(ctx, "key", nil, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for nil value")
	}

	// SetSingle with nil value
	err = cs.SetSingle(ctx, "key", nil, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for nil value")
	}
}

func TestCacheServiceSetQueryResultAndGet(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	events := []blockchain.BlockchainEvent{
		{EventHash: "0xqr1", BlockNumber: 3000, ChainID: "1"},
		{EventHash: "0xqr2", BlockNumber: 3001, ChainID: "1"},
	}
	total := int64(42)

	err := cs.SetQueryResult(ctx, "qr-key", events, total, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set query result: %v", err)
	}

	retrieved, retrievedTotal, err := cs.GetQueryResult(ctx, "qr-key")
	if err != nil {
		t.Fatalf("Failed to get query result: %v", err)
	}

	if len(retrieved) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(retrieved))
	}
	if retrievedTotal != total {
		t.Errorf("Expected total %d, got %d", total, retrievedTotal)
	}
	if retrieved[0].EventHash != events[0].EventHash {
		t.Errorf("Expected hash %s, got %s", events[0].EventHash, retrieved[0].EventHash)
	}
}

func TestCacheServiceQueryResultExpiration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	events := []blockchain.BlockchainEvent{{EventHash: "0xqr3"}}
	if err := cs.SetQueryResult(ctx, "qr-expire-key", events, 1, 100*time.Millisecond); err != nil {
		t.Fatalf("Failed to set query result: %v", err)
	}

	_, _, err := cs.GetQueryResult(ctx, "qr-expire-key")
	if err != nil {
		t.Fatalf("Failed to get query result: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	_, _, err = cs.GetQueryResult(ctx, "qr-expire-key")
	if err == nil {
		t.Error("Expected error for expired query result")
	}
}

func TestCacheServiceQueryResultNotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	events := []blockchain.BlockchainEvent{{EventHash: "0xqr4"}}
	err := cs.SetQueryResult(ctx, "key", events, 1, 1*time.Hour)
	if err == nil {
		t.Error("Expected error when cache service not running")
	}

	_, _, err = cs.GetQueryResult(ctx, "key")
	if err == nil {
		t.Error("Expected error when cache service not running")
	}
}

func TestCacheServiceQueryResultEmptyKey(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	events := []blockchain.BlockchainEvent{{EventHash: "0xqr5"}}
	err := cs.SetQueryResult(ctx, "", events, 1, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for empty key")
	}

	_, _, err = cs.GetQueryResult(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

func TestCacheServiceQueryResultMiss(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := NewCacheService(logger, metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	_, _, err := cs.GetQueryResult(ctx, "non-existent-qr")
	if err == nil {
		t.Error("Expected error for non-existent query result key")
	}
}

func TestCacheServiceEvictOldest(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelDebug)
	metrics := core.NewDefaultMetricsCollector()

	cs := &DefaultCacheService{
		cache:            make(map[string]cacheEntry),
		maxEntries:       3,
		logger:           logger,
		metricsCollector: metrics,
		done:             make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	if err := cs.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer func() {
		_ = cs.Stop(ctx)
	}()

	events := []blockchain.BlockchainEvent{{EventHash: "0xev1"}}
	if err := cs.Set(ctx, "key-1", events, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set key-1: %v", err)
	}

	events2 := []blockchain.BlockchainEvent{{EventHash: "0xev2"}}
	if err := cs.Set(ctx, "key-2", events2, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set key-2: %v", err)
	}

	events3 := []blockchain.BlockchainEvent{{EventHash: "0xev3"}}
	if err := cs.Set(ctx, "key-3", events3, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set key-3: %v", err)
	}

	events4 := []blockchain.BlockchainEvent{{EventHash: "0xev4"}}
	if err := cs.Set(ctx, "key-4", events4, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set key-4: %v", err)
	}

	_, err := cs.Get(ctx, "key-1")
	if err == nil {
		t.Error("Expected key-1 to be evicted")
	}

	retrieved, err := cs.Get(ctx, "key-2")
	if err != nil {
		t.Errorf("Expected key-2 to exist: %v", err)
	}
	if retrieved[0].EventHash != "0xev2" {
		t.Errorf("Expected hash 0xev2, got %s", retrieved[0].EventHash)
	}

	retrieved, err = cs.Get(ctx, "key-3")
	if err != nil {
		t.Errorf("Expected key-3 to exist: %v", err)
	}
	if retrieved[0].EventHash != "0xev3" {
		t.Errorf("Expected hash 0xev3, got %s", retrieved[0].EventHash)
	}

	retrieved, err = cs.Get(ctx, "key-4")
	if err != nil {
		t.Errorf("Expected key-4 to exist: %v", err)
	}
	if retrieved[0].EventHash != "0xev4" {
		t.Errorf("Expected hash 0xev4, got %s", retrieved[0].EventHash)
	}
}
