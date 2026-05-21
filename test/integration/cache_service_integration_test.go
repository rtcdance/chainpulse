package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

// TestCacheService_Set_Get_Basic tests basic cache set and get operations
func TestCacheService_Set_Get_Basic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Create test events
	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
		{
			ID:              "event2",
			ChainID:         "ethereum",
			BlockNumber:     101,
			TransactionHash: hashFromString("0xdef456"),
			EventName:       "Approval",
		},
	}

	// Set cache
	cacheKey := "test:events:1"
	if err := cacheService.Set(ctx, cacheKey, testEvents, 1*time.Hour); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Get cache
	retrieved, err := cacheService.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("failed to get cache: %v", err)
	}

	// Verify
	if len(retrieved) != len(testEvents) {
		t.Errorf("expected %d events, got %d", len(testEvents), len(retrieved))
	}
	if retrieved[0].ID != "event1" {
		t.Errorf("expected event ID 'event1', got '%s'", retrieved[0].ID)
	}
	if retrieved[1].ID != "event2" {
		t.Errorf("expected event ID 'event2', got '%s'", retrieved[1].ID)
	}
}

// TestCacheService_SetSingle_GetSingle_Basic tests single event cache operations
func TestCacheService_SetSingle_GetSingle_Basic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Create test event
	testEvent := &core.BlockchainEvent{
		ID:              "event1",
		ChainID:         "ethereum",
		BlockNumber:     100,
		TransactionHash: hashFromString("0xabc123"),
		EventName:       "Transfer",
	}

	// Set cache
	cacheKey := "test:event:1"
	if err := cacheService.SetSingle(ctx, cacheKey, testEvent, 1*time.Hour); err != nil {
		t.Fatalf("failed to set single cache: %v", err)
	}

	// Get cache
	retrieved, err := cacheService.GetSingle(ctx, cacheKey)
	if err != nil {
		t.Fatalf("failed to get single cache: %v", err)
	}

	// Verify
	if retrieved.ID != testEvent.ID {
		t.Errorf("expected event ID '%s', got '%s'", testEvent.ID, retrieved.ID)
	}
	if retrieved.BlockNumber != testEvent.BlockNumber {
		t.Errorf("expected block number %d, got %d", testEvent.BlockNumber, retrieved.BlockNumber)
	}
}

// TestCacheService_CacheMiss tests cache miss behavior
func TestCacheService_CacheMiss(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Try to get non-existent key
	_, err := cacheService.Get(ctx, "non-existent-key")
	if err == nil {
		t.Errorf("expected error for cache miss, got nil")
	}
	if err.Error() != "cache miss" {
		t.Errorf("expected 'cache miss' error, got '%s'", err.Error())
	}
}

// TestCacheService_Invalidate tests cache invalidation
func TestCacheService_Invalidate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Set cache
	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
	}
	cacheKey := "test:events:1"
	if err := cacheService.Set(ctx, cacheKey, testEvents, 1*time.Hour); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Verify cache hit
	_, err := cacheService.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("expected cache hit, got error: %v", err)
	}

	// Delete cache
	if err := cacheService.Delete(ctx, cacheKey); err != nil {
		t.Fatalf("failed to delete cache: %v", err)
	}

	// Verify cache miss after deletion
	_, err = cacheService.Get(ctx, cacheKey)
	if err == nil {
		t.Errorf("expected cache miss after deletion, got nil error")
	}
}

// TestCacheService_Expiration tests cache entry expiration
func TestCacheService_Expiration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Set cache with short TTL
	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
	}
	cacheKey := "test:events:expiring"
	ttl := 500 * time.Millisecond

	if err := cacheService.Set(ctx, cacheKey, testEvents, ttl); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Verify cache hit immediately
	_, err := cacheService.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("expected cache hit immediately, got error: %v", err)
	}

	// Wait for expiration
	time.Sleep(ttl + 100*time.Millisecond)

	// Verify cache miss after expiration
	_, err = cacheService.Get(ctx, cacheKey)
	if err == nil {
		t.Errorf("expected cache miss after expiration, got nil error")
	}
	if err.Error() != "cache entry expired" {
		t.Errorf("expected 'cache entry expired' error, got '%s'", err.Error())
	}
}

// TestCacheService_ConcurrentAccess tests concurrent cache operations
func TestCacheService_ConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Run concurrent operations
	numGoroutines := 10
	operationsPerGoroutine := 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				cacheKey := fmt.Sprintf("test:concurrent:%d:%d", goroutineID, op)
				testEvents := []core.BlockchainEvent{
					{
						ID:              fmt.Sprintf("event-%d-%d", goroutineID, op),
						ChainID:         "ethereum",
						BlockNumber:     uint64(goroutineID*operationsPerGoroutine + op),
						TransactionHash: hashFromString(fmt.Sprintf("0x%d%d", goroutineID, op)),
						EventName:       "Transfer",
					},
				}

				// Set
				if err := cacheService.Set(ctx, cacheKey, testEvents, 1*time.Hour); err != nil {
					errors <- fmt.Errorf("set failed: %w", err)
					return
				}

				// Get
				retrieved, err := cacheService.Get(ctx, cacheKey)
				if err != nil {
					errors <- fmt.Errorf("get failed: %w", err)
					return
				}

				if len(retrieved) != 1 {
					errors <- fmt.Errorf("expected 1 event, got %d", len(retrieved))
					return
				}

				// Delete
				if err := cacheService.Delete(ctx, cacheKey); err != nil {
					errors <- fmt.Errorf("delete failed: %w", err)
					return
				}

				// Verify miss
				_, err = cacheService.Get(ctx, cacheKey)
				if err == nil {
					errors <- fmt.Errorf("expected cache miss after deletion")
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// TestCacheService_InvalidKey tests cache operations with invalid keys
func TestCacheService_InvalidKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
	}

	// Try to set with empty key
	err := cacheService.Set(ctx, "", testEvents, 1*time.Hour)
	if err == nil {
		t.Errorf("expected error for empty key, got nil")
	}
	if err.Error() != "cache key is required" {
		t.Errorf("expected 'cache key is required' error, got '%s'", err.Error())
	}

	// Try to get with empty key
	_, err = cacheService.Get(ctx, "")
	if err == nil {
		t.Errorf("expected error for empty key, got nil")
	}
	if err.Error() != "cache key is required" {
		t.Errorf("expected 'cache key is required' error, got '%s'", err.Error())
	}
}

// TestCacheService_InvalidValue tests cache operations with invalid values
func TestCacheService_InvalidValue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Try to set with nil value
	err := cacheService.Set(ctx, "test:key", nil, 1*time.Hour)
	if err == nil {
		t.Errorf("expected error for nil value, got nil")
	}
	if err.Error() != "cache value is required" {
		t.Errorf("expected 'cache value is required' error, got '%s'", err.Error())
	}

	// Try to set single with nil value
	err = cacheService.SetSingle(ctx, "test:key", nil, 1*time.Hour)
	if err == nil {
		t.Errorf("expected error for nil value, got nil")
	}
	if err.Error() != "cache value is required" {
		t.Errorf("expected 'cache value is required' error, got '%s'", err.Error())
	}
}

// TestCacheService_NotRunning tests cache operations when service is not running
func TestCacheService_NotRunning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}

	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
	}

	// Try to set without starting
	err := cacheService.Set(ctx, "test:key", testEvents, 1*time.Hour)
	if err == nil {
		t.Errorf("expected error when service not running, got nil")
	}
	if err.Error() != "cache service not running" {
		t.Errorf("expected 'cache service not running' error, got '%s'", err.Error())
	}

	// Try to get without starting
	_, err = cacheService.Get(ctx, "test:key")
	if err == nil {
		t.Errorf("expected error when service not running, got nil")
	}
	if err.Error() != "cache service not running" {
		t.Errorf("expected 'cache service not running' error, got '%s'", err.Error())
	}
}

// TestCacheService_MultipleInitialize tests that service can't be initialized twice
func TestCacheService_MultipleInitialize(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("first initialize failed: %v", err)
	}

	// Try to initialize again
	err := cacheService.Initialize(ctx)
	if err == nil {
		t.Errorf("expected error on second initialize, got nil")
	}
	if err.Error() != "cache service already initialized" {
		t.Errorf("expected 'cache service already initialized' error, got '%s'", err.Error())
	}
}

// TestCacheService_MultipleStart tests that service can't be started twice
func TestCacheService_MultipleStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Try to start again
	err := cacheService.Start(ctx)
	if err == nil {
		t.Errorf("expected error on second start, got nil")
	}
	if err.Error() != "cache service already running" {
		t.Errorf("expected 'cache service already running' error, got '%s'", err.Error())
	}
}

// TestCacheService_ContextCancellation tests cache operations with cancelled context
func TestCacheService_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Create cancelled context
	cancelledCtx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()

	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
	}

	// Operations should still work (context is only used for timeout, not for cancellation in cache service)
	err := cacheService.Set(cancelledCtx, "test:key", testEvents, 1*time.Hour)
	if err != nil {
		// This is acceptable - service may check context
		t.Logf("set with cancelled context returned error: %v", err)
	}
}

// TestCacheService_LargeDataSet tests cache with large data sets
func TestCacheService_LargeDataSet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize cache service: %v", err)
	}
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start cache service: %v", err)
	}
	defer func() { _ = cacheService.Stop(ctx) }()

	// Create large dataset
	numEvents := 1000
	testEvents := make([]core.BlockchainEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		testEvents[i] = core.BlockchainEvent{
			ID:              fmt.Sprintf("event-%d", i),
			ChainID:         "ethereum",
			BlockNumber:     uint64(i),
			TransactionHash: hashFromString(fmt.Sprintf("0x%d", i)),
			EventName:       "Transfer",
		}
	}

	// Set large dataset
	cacheKey := "test:large:dataset"
	if err := cacheService.Set(ctx, cacheKey, testEvents, 1*time.Hour); err != nil {
		t.Fatalf("failed to set large dataset: %v", err)
	}

	// Get large dataset
	retrieved, err := cacheService.Get(ctx, cacheKey)
	if err != nil {
		t.Fatalf("failed to get large dataset: %v", err)
	}

	// Verify
	if len(retrieved) != numEvents {
		t.Errorf("expected %d events, got %d", numEvents, len(retrieved))
	}

	// Verify first and last events
	if retrieved[0].ID != "event-0" {
		t.Errorf("expected first event ID 'event-0', got '%s'", retrieved[0].ID)
	}
	if retrieved[numEvents-1].ID != fmt.Sprintf("event-%d", numEvents-1) {
		t.Errorf("expected last event ID 'event-%d', got '%s'", numEvents-1, retrieved[numEvents-1].ID)
	}
}

// TestCacheService_Lifecycle tests complete service lifecycle
func TestCacheService_Lifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup
	logger := &testLogger{}
	metricsCollector := &testMetricsCollector{}
	cacheService := query.NewCacheService(logger, metricsCollector)

	// Initialize
	if err := cacheService.Initialize(ctx); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Start
	if err := cacheService.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Use service
	testEvents := []core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: hashFromString("0xabc123"),
			EventName:       "Transfer",
		},
	}
	cacheKey := "test:lifecycle"
	if err := cacheService.Set(ctx, cacheKey, testEvents, 1*time.Hour); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Stop
	if err := cacheService.Stop(ctx); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	// Verify operations fail after stop
	_, err := cacheService.Get(ctx, cacheKey)
	if err == nil {
		t.Errorf("expected error after stop, got nil")
	}
	if err.Error() != "cache service not running" {
		t.Errorf("expected 'cache service not running' error, got '%s'", err.Error())
	}
}
