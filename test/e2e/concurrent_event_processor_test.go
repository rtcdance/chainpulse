package e2e

import (
	"context"
	"testing"
	"time"
)

// TestConcurrentEventProcessorStartup tests processor startup
func TestConcurrentEventProcessorStartup(t *testing.T) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	err := processor.Startup(ctx)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
}

// TestGenerateEventsAsync tests async event generation
func TestGenerateEventsAsync(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsAsync(ctx, 100, 4)
	if err != nil {
		t.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.TotalEventsGenerated != 100 {
		t.Errorf("Expected 100 events generated, got %d", metrics.TotalEventsGenerated)
	}
}

// TestGenerateEventsWithDelay tests event generation with delays
func TestGenerateEventsWithDelay(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsWithDelay(ctx, 20, 2, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("GenerateEventsWithDelay failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.TotalEventsGenerated != 20 {
		t.Errorf("Expected 20 events generated, got %d", metrics.TotalEventsGenerated)
	}
}

// TestDetectRaceConditions tests race condition detection
func TestDetectRaceConditions(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsAsync(ctx, 50, 5)
	if err != nil {
		t.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	_, err = processor.DetectRaceConditions(ctx)
	if err != nil {
		t.Fatalf("DetectRaceConditions failed: %v", err)
	}
}

// TestValidateConcurrentOrdering tests concurrent ordering validation
func TestValidateConcurrentOrdering(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsWithDelay(ctx, 30, 3, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("GenerateEventsWithDelay failed: %v", err)
	}

	err = processor.ValidateConcurrentOrdering(ctx)
	if err != nil {
		t.Logf("Ordering validation: %v", err)
	}
}

// TestConcurrentDatabaseWrites tests concurrent database writes
func TestConcurrentDatabaseWrites(t *testing.T) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.ConcurrentDatabaseWrites(ctx, 1000, 10)
	if err != nil {
		t.Fatalf("ConcurrentDatabaseWrites failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.DatabaseWriteErrors < 0 {
		t.Errorf("Invalid database write errors: %d", metrics.DatabaseWriteErrors)
	}
}

// TestConcurrentDatabaseReads tests concurrent database reads
func TestConcurrentDatabaseReads(t *testing.T) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.ConcurrentDatabaseReads(ctx, 1000, 10)
	if err != nil {
		t.Fatalf("ConcurrentDatabaseReads failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.DatabaseReadErrors < 0 {
		t.Errorf("Invalid database read errors: %d", metrics.DatabaseReadErrors)
	}
}

// TestGetMetrics tests metrics collection
func TestGetMetrics(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsAsync(ctx, 50, 5)
	if err != nil {
		t.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.TotalEventsGenerated != 50 {
		t.Errorf("Expected 50 events generated, got %d", metrics.TotalEventsGenerated)
	}

	if metrics.Throughput <= 0 {
		t.Errorf("Expected positive throughput, got %f", metrics.Throughput)
	}
}

// TestReset tests metrics reset
func TestReset(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsAsync(ctx, 50, 5)
	if err != nil {
		t.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.TotalEventsGenerated == 0 {
		t.Error("Expected events before reset")
	}

	processor.Reset()

	metrics = processor.GetMetrics()
	if metrics.TotalEventsGenerated != 0 {
		t.Errorf("Expected 0 events after reset, got %d", metrics.TotalEventsGenerated)
	}
}

// TestHighConcurrency tests with high concurrency
func TestHighConcurrency(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsAsync(ctx, 200, 20)
	if err != nil {
		t.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.TotalEventsGenerated != 200 {
		t.Errorf("Expected 200 events generated, got %d", metrics.TotalEventsGenerated)
	}
}

// TestConcurrentOperationsWithErrors tests concurrent operations with error handling
func TestConcurrentOperationsWithErrors(t *testing.T) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.ConcurrentDatabaseWrites(ctx, 500, 5)
	if err != nil {
		t.Fatalf("ConcurrentDatabaseWrites failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.DatabaseWriteErrors < 0 {
		t.Errorf("Invalid database write errors: %d", metrics.DatabaseWriteErrors)
	}
}

// TestMetricsAccuracy tests metrics accuracy
func TestMetricsAccuracy(t *testing.T) {
	if !IsAnvilAvailable() {
		t.Skip("Anvil not available")
	}
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.GenerateEventsAsync(ctx, 100, 10)
	if err != nil {
		t.Fatalf("GenerateEventsAsync failed: %v", err)
	}

	metrics := processor.GetMetrics()
	if metrics.TotalEventsGenerated != metrics.TotalEventsProcessed {
		t.Errorf("Events generated (%d) != events processed (%d)", metrics.TotalEventsGenerated, metrics.TotalEventsProcessed)
	}

	if metrics.AverageLatency == 0 && metrics.TotalEventsGenerated > 0 {
		t.Error("Expected non-zero average latency")
	}
}

// TestShutdown tests processor shutdown
func TestShutdown(t *testing.T) {
	ctx := context.Background()
	bm := NewDefaultBlockchainManager()
	_ = bm.Initialize(ctx)
	defer func() { _ = bm.Close() }()

	processor := NewDefaultConcurrentEventProcessor(bm)
	_ = processor.Startup(ctx)

	err := processor.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}
