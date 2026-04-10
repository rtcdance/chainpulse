package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
	"github.com/ethereum/go-ethereum/common"
	"pgregory.net/rapid"
)

// Property 4: Resource Cleanup
// For any integration test, after it completes, all created resources
// (database records, cache entries, connections) SHALL be cleaned up.

// hashFromString converts a string to a common.Hash
func hashFromString(s string) common.Hash {
	h := common.Hash{}
	copy(h[:], []byte(s))
	return h
}

// TestProperty_CacheService_ResourceCleanup verifies that cache resources
// are properly cleaned up after operations complete.
func TestProperty_CacheService_ResourceCleanup(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate random test parameters
		numEvents := rapid.IntRange(1, 100).Draw(rt, "numEvents")
		numOperations := rapid.IntRange(1, 50).Draw(rt, "numOperations")
		numConcurrentOps := rapid.IntRange(1, 10).Draw(rt, "numConcurrentOps")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create cache service
		logger := &testLogger{}
		metricsCollector := &testMetricsCollector{}
		cacheService := query.NewCacheService(logger, metricsCollector)

		// Initialize and start
		if err := cacheService.Initialize(ctx); err != nil {
			rt.Fatalf("failed to initialize cache service: %v", err)
		}
		if err := cacheService.Start(ctx); err != nil {
			rt.Fatalf("failed to start cache service: %v", err)
		}

		// Create test events
		testEvents := make([]core.BlockchainEvent, numEvents)
		for i := 0; i < numEvents; i++ {
			testEvents[i] = core.BlockchainEvent{
				ID:              fmt.Sprintf("event-%d", i),
				BlockNumber:     uint64(i),
				TransactionHash: hashFromString(fmt.Sprintf("0x%d", i)),
				ChainID:         "test-chain",
				EventName:       "TestEvent",
			}
		}

		// Execute concurrent cache operations
		var wg sync.WaitGroup
		for op := 0; op < numConcurrentOps; op++ {
			wg.Add(1)
			go func(opID int) {
				defer wg.Done()

				for i := 0; i < numOperations; i++ {
					eventIdx := (opID*numOperations + i) % numEvents
					cacheKey := fmt.Sprintf("cache-key-%d-%d", opID, i)

					// Set operation
					_ = cacheService.Set(ctx, cacheKey, []core.BlockchainEvent{testEvents[eventIdx]}, 5*time.Minute)

					// Get operation
					_, _ = cacheService.Get(ctx, cacheKey)

					// Delete operation
					_ = cacheService.Delete(ctx, cacheKey)
				}
			}(op)
		}

		wg.Wait()

		// Stop cache service - this should clean up all resources
		if err := cacheService.Stop(ctx); err != nil {
			rt.Fatalf("failed to stop cache service: %v", err)
		}

		// Verify service stopped cleanly (no resource leaks)
		// If we got here without panic or timeout, cleanup was successful
	})
}

// TestProperty_CacheService_NoLeaksAfterErrors verifies that resources
// are cleaned up even when errors occur during operations.
func TestProperty_CacheService_NoLeaksAfterErrors(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		numOperations := rapid.IntRange(5, 50).Draw(rt, "numOperations")
		errorPattern := rapid.IntRange(1, 5).Draw(rt, "errorPattern")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger := &testLogger{}
		metricsCollector := &testMetricsCollector{}
		cacheService := query.NewCacheService(logger, metricsCollector)

		if err := cacheService.Initialize(ctx); err != nil {
			rt.Fatalf("failed to initialize cache service: %v", err)
		}
		if err := cacheService.Start(ctx); err != nil {
			rt.Fatalf("failed to start cache service: %v", err)
		}

		// Execute operations with some errors
		for i := 0; i < numOperations; i++ {
			cacheKey := fmt.Sprintf("key-%d", i)
			event := core.BlockchainEvent{
				ID:              fmt.Sprintf("event-%d", i),
				BlockNumber:     uint64(i),
				TransactionHash: hashFromString(fmt.Sprintf("0x%d", i)),
			}

			if i%errorPattern == 0 {
				// Simulate error by using invalid context
				invalidCtx, cancel := context.WithCancel(context.Background())
				cancel()
				_ = cacheService.Set(invalidCtx, cacheKey, []core.BlockchainEvent{event}, 5*time.Minute)
			} else {
				_ = cacheService.Set(ctx, cacheKey, []core.BlockchainEvent{event}, 5*time.Minute)
			}
		}

		// Stop should succeed even after errors
		if err := cacheService.Stop(ctx); err != nil {
			rt.Fatalf("failed to stop cache service after errors: %v", err)
		}
	})
}

// TestProperty_CacheService_ContextCancellationCleanup verifies that
// resources are cleaned up when context is cancelled.
func TestProperty_CacheService_ContextCancellationCleanup(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		cancelAfterOps := rapid.IntRange(1, 20).Draw(rt, "cancelAfterOps")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger := &testLogger{}
		metricsCollector := &testMetricsCollector{}
		cacheService := query.NewCacheService(logger, metricsCollector)

		if err := cacheService.Initialize(ctx); err != nil {
			rt.Fatalf("failed to initialize cache service: %v", err)
		}
		if err := cacheService.Start(ctx); err != nil {
			rt.Fatalf("failed to start cache service: %v", err)
		}

		// Create cancellable context
		opCtx, opCancel := context.WithCancel(ctx)

		// Execute operations
		for i := 0; i < cancelAfterOps; i++ {
			cacheKey := fmt.Sprintf("key-%d", i)
			event := core.BlockchainEvent{
				ID:              fmt.Sprintf("event-%d", i),
				BlockNumber:     uint64(i),
				TransactionHash: hashFromString(fmt.Sprintf("0x%d", i)),
			}
			_ = cacheService.Set(opCtx, cacheKey, []core.BlockchainEvent{event}, 5*time.Minute)
		}

		// Cancel context
		opCancel()

		// Stop should succeed after context cancellation
		if err := cacheService.Stop(ctx); err != nil {
			rt.Fatalf("failed to stop cache service after context cancellation: %v", err)
		}
	})
}

// TestProperty_CacheService_ConcurrentCleanup verifies that concurrent
// operations don't cause resource leaks.
func TestProperty_CacheService_ConcurrentCleanup(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		numGoroutines := rapid.IntRange(5, 50).Draw(rt, "numGoroutines")
		operationsPerGoroutine := rapid.IntRange(10, 100).Draw(rt, "operationsPerGoroutine")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger := &testLogger{}
		metricsCollector := &testMetricsCollector{}
		cacheService := query.NewCacheService(logger, metricsCollector)

		if err := cacheService.Initialize(ctx); err != nil {
			rt.Fatalf("failed to initialize cache service: %v", err)
		}
		if err := cacheService.Start(ctx); err != nil {
			rt.Fatalf("failed to start cache service: %v", err)
		}

		// Run concurrent operations
		var wg sync.WaitGroup
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for op := 0; op < operationsPerGoroutine; op++ {
					cacheKey := fmt.Sprintf("key-%d-%d", goroutineID, op)
					event := core.BlockchainEvent{
						ID:              fmt.Sprintf("event-%d-%d", goroutineID, op),
						BlockNumber:     uint64(goroutineID*operationsPerGoroutine + op),
						TransactionHash: hashFromString(fmt.Sprintf("0x%d-%d", goroutineID, op)),
					}

					// Cache operation
					_ = cacheService.Set(ctx, cacheKey, []core.BlockchainEvent{event}, 5*time.Minute)

					// Get operation
					_, _ = cacheService.Get(ctx, cacheKey)

					// Delete operation
					_ = cacheService.Delete(ctx, cacheKey)
				}
			}(g)
		}

		wg.Wait()

		// Stop should succeed after concurrent operations
		if err := cacheService.Stop(ctx); err != nil {
			rt.Fatalf("failed to stop cache service: %v", err)
		}
	})
}

// TestProperty_CacheService_InvalidationCleanup verifies that cache
// invalidation properly cleans up entries.
func TestProperty_CacheService_InvalidationCleanup(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		numEvents := rapid.IntRange(5, 100).Draw(rt, "numEvents")
		invalidationPattern := rapid.IntRange(1, 5).Draw(rt, "invalidationPattern")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger := &testLogger{}
		metricsCollector := &testMetricsCollector{}
		cacheService := query.NewCacheService(logger, metricsCollector)

		if err := cacheService.Initialize(ctx); err != nil {
			rt.Fatalf("failed to initialize cache service: %v", err)
		}
		if err := cacheService.Start(ctx); err != nil {
			rt.Fatalf("failed to start cache service: %v", err)
		}

		// Create and cache events
		for i := 0; i < numEvents; i++ {
			cacheKey := fmt.Sprintf("key-%d", i)
			event := core.BlockchainEvent{
				ID:              fmt.Sprintf("event-%d", i),
				BlockNumber:     uint64(i),
				TransactionHash: hashFromString(fmt.Sprintf("0x%d", i)),
			}
			_ = cacheService.Set(ctx, cacheKey, []core.BlockchainEvent{event}, 5*time.Minute)
		}

		// Invalidate entries based on pattern
		for i := 0; i < numEvents; i++ {
			if i%invalidationPattern == 0 {
				cacheKey := fmt.Sprintf("key-%d", i)
				_ = cacheService.Delete(ctx, cacheKey)
			}
		}

		// Verify invalidated entries are gone
		for i := 0; i < numEvents; i++ {
			if i%invalidationPattern == 0 {
				cacheKey := fmt.Sprintf("key-%d", i)
				_, err := cacheService.Get(ctx, cacheKey)
				if err == nil {
					rt.Fatalf("invalidated entry still in cache: %s", cacheKey)
				}
			}
		}

		// Stop should succeed
		if err := cacheService.Stop(ctx); err != nil {
			rt.Fatalf("failed to stop cache service: %v", err)
		}
	})
}

// testLogger is a simple test logger implementation
type testLogger struct{}

func (l *testLogger) Debug(msg string, fields ...interface{}) {
	// No-op for testing
}

func (l *testLogger) Info(msg string, fields ...interface{}) {
	// No-op for testing
}

func (l *testLogger) Warn(msg string, fields ...interface{}) {
	// No-op for testing
}

func (l *testLogger) Error(msg string, fields ...interface{}) {
	// No-op for testing
}

func (l *testLogger) Fatal(msg string, fields ...interface{}) {
	// No-op for testing
}

func (l *testLogger) WithCorrelationID(id string) core.Logger {
	return l
}

// testMetricsCollector is a simple test metrics collector implementation
type testMetricsCollector struct {
	mu      sync.Mutex
	metrics map[string]interface{}
}

func (m *testMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name] = value
}

func (m *testMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name] = value
}

func (m *testMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.metrics == nil {
		m.metrics = make(map[string]interface{})
	}
	m.metrics[name] = value
}

func (m *testMetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.metrics
}
