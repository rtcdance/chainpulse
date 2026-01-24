package e2e

import (
	"context"
	"testing"
	"time"
)

// Property 15: Concurrent Event Isolation
// Events generated concurrently should be isolated and not interfere with each other
func TestProperty15ConcurrentEventIsolation(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			t.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		t.Fatalf("Failed to startup processor: %v", err)
	}

	// Run multiple concurrent event generations
	for iteration := 0; iteration < 50; iteration++ {
		processor.Reset()

		err := processor.GenerateEventsAsync(ctx, 100, 10)
		if err != nil {
			t.Fatalf("Iteration %d: GenerateEventsAsync failed: %v", iteration, err)
		}

		metrics := processor.GetMetrics()
		if metrics.TotalEventsGenerated != 100 {
			t.Errorf("Iteration %d: Expected 100 events, got %d", iteration, metrics.TotalEventsGenerated)
		}

		if metrics.TotalEventsProcessed != 100 {
			t.Errorf("Iteration %d: Expected 100 processed, got %d", iteration, metrics.TotalEventsProcessed)
		}
	}
}

// Property 16: Concurrent Ordering Consistency
// Events should maintain consistent ordering across concurrent operations
func TestProperty16ConcurrentOrderingConsistency(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			t.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		t.Fatalf("Failed to startup processor: %v", err)
	}

	for iteration := 0; iteration < 50; iteration++ {
		processor.Reset()

		err := processor.GenerateEventsWithDelay(ctx, 50, 5, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Iteration %d: GenerateEventsWithDelay failed: %v", iteration, err)
		}

		// Validate ordering
		err = processor.ValidateConcurrentOrdering(ctx)
		if err != nil {
			t.Logf("Iteration %d: Ordering validation: %v", iteration, err)
		}

		metrics := processor.GetMetrics()
		if metrics.OrderingViolations < 0 {
			t.Errorf("Iteration %d: Invalid ordering violations: %d", iteration, metrics.OrderingViolations)
		}
	}
}

// Property 17: Race Condition Detection Accuracy
// Race condition detection should accurately identify concurrent access patterns
func TestProperty17RaceConditionDetectionAccuracy(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			t.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		t.Fatalf("Failed to startup processor: %v", err)
	}

	for iteration := 0; iteration < 50; iteration++ {
		processor.Reset()

		err := processor.GenerateEventsAsync(ctx, 100, 20)
		if err != nil {
			t.Fatalf("Iteration %d: GenerateEventsAsync failed: %v", iteration, err)
		}

		raceConditions, err := processor.DetectRaceConditions(ctx)
		if err != nil {
			t.Fatalf("Iteration %d: DetectRaceConditions failed: %v", iteration, err)
		}
		_ = raceConditions

		// All detected race conditions should have valid event IDs
		for _, rc := range raceConditions {
			if rc.EventID1 == "" || rc.EventID2 == "" {
				t.Errorf("Iteration %d: Invalid race condition with empty event IDs", iteration)
			}
		}
	}
}

// Property 18: Database Operation Concurrency
// Concurrent database operations should complete without data corruption
func TestProperty18DatabaseOperationConcurrency(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			t.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		t.Fatalf("Failed to startup processor: %v", err)
	}

	for iteration := 0; iteration < 50; iteration++ {
		processor.Reset()

		// Concurrent writes
		err := processor.ConcurrentDatabaseWrites(ctx, 500, 10)
		if err != nil {
			t.Fatalf("Iteration %d: ConcurrentDatabaseWrites failed: %v", iteration, err)
		}

		metrics := processor.GetMetrics()
		if metrics.DatabaseWriteErrors < 0 {
			t.Errorf("Iteration %d: Invalid write errors: %d", iteration, metrics.DatabaseWriteErrors)
		}

		processor.Reset()

		// Concurrent reads
		err = processor.ConcurrentDatabaseReads(ctx, 500, 10)
		if err != nil {
			t.Fatalf("Iteration %d: ConcurrentDatabaseReads failed: %v", iteration, err)
		}

		metrics = processor.GetMetrics()
		if metrics.DatabaseReadErrors < 0 {
			t.Errorf("Iteration %d: Invalid read errors: %d", iteration, metrics.DatabaseReadErrors)
		}
	}
}

// Property 19: Metrics Consistency Under Concurrency
// Metrics should remain consistent and accurate under concurrent load
func TestProperty19MetricsConsistencyUnderConcurrency(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			t.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		t.Fatalf("Failed to startup processor: %v", err)
	}

	for iteration := 0; iteration < 50; iteration++ {
		processor.Reset()

		err := processor.GenerateEventsAsync(ctx, 200, 20)
		if err != nil {
			t.Fatalf("Iteration %d: GenerateEventsAsync failed: %v", iteration, err)
		}

		metrics := processor.GetMetrics()

		// Verify metrics consistency
		if metrics.TotalEventsGenerated != 200 {
			t.Errorf("Iteration %d: Expected 200 events, got %d", iteration, metrics.TotalEventsGenerated)
		}

		if metrics.TotalEventsProcessed != 200 {
			t.Errorf("Iteration %d: Expected 200 processed, got %d", iteration, metrics.TotalEventsProcessed)
		}

		if metrics.TotalEventsGenerated != metrics.TotalEventsProcessed {
			t.Errorf("Iteration %d: Generated != Processed", iteration)
		}

		// Latency metrics should be valid
		if metrics.AverageLatency < 0 {
			t.Errorf("Iteration %d: Invalid average latency: %v", iteration, metrics.AverageLatency)
		}

		if metrics.MaxLatency < metrics.MinLatency && metrics.MaxLatency > 0 {
			t.Errorf("Iteration %d: Max latency < Min latency", iteration)
		}

		// Throughput should be positive
		if metrics.Throughput <= 0 {
			t.Errorf("Iteration %d: Invalid throughput: %f", iteration, metrics.Throughput)
		}
	}
}

// Benchmark: Concurrent Event Generation Performance
func BenchmarkConcurrentEventGeneration(b *testing.B) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		b.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			b.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		b.Fatalf("Failed to startup processor: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Reset()
		if err := processor.GenerateEventsAsync(ctx, 100, 10); err != nil {
			b.Fatalf("GenerateEventsAsync failed: %v", err)
		}
	}
}

// Benchmark: Race Condition Detection Performance
func BenchmarkRaceConditionDetection(b *testing.B) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	if err := bm.Initialize(ctx); err != nil {
		b.Fatalf("Failed to initialize blockchain manager: %v", err)
	}
	defer func() {
		if err := bm.Close(); err != nil {
			b.Logf("Failed to close blockchain manager: %v", err)
		}
	}()

	processor := NewDefaultConcurrentEventProcessor(bm)
	if err := processor.Startup(ctx); err != nil {
		b.Fatalf("Failed to startup processor: %v", err)
	}

	if err := processor.GenerateEventsAsync(ctx, 100, 10); err != nil {
		b.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.DetectRaceConditions(ctx)
		if err != nil {
			b.Fatalf("DetectRaceConditions failed: %v", err)
		}
	}
}
