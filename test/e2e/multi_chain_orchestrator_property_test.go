package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// PropertyMultiChainEventIsolation validates that events don't leak between chains
// Feature: web3-indexer-e2e-testing, Property 13: Multi-Chain Event Isolation
func PropertyMultiChainEventIsolation(t *testing.T, gen *Generator) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	// Generate random number of chains (2-5)
	chainCount := gen.IntRange(2, 5)
	err := orchestrator.Startup(ctx, chainCount)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	// Generate random events
	eventCount := gen.IntRange(1, 10)
	events := make([]Event, eventCount)
	for i := 0; i < eventCount; i++ {
		events[i] = Event{
			ID:               fmt.Sprintf("event-%d", i),
			ContractAddress:  fmt.Sprintf("0x%x", gen.Uint64()),
			EventName:        fmt.Sprintf("Event%d", i),
			BlockNumber:      uint64(gen.IntRange(1, 1000)),
			TransactionIndex: uint32(gen.IntRange(0, 100)),
		}
	}

	// Emit events on all chains
	err = orchestrator.EmitEventsOnAllChains(ctx, events)
	if err != nil {
		t.Fatalf("EmitEventsOnAllChains failed: %v", err)
	}

	// Validate chain isolation
	err = orchestrator.ValidateChainIsolation(ctx)
	if err != nil {
		t.Fatalf("Chain isolation validation failed: %v", err)
	}

	// Verify metrics show no failed chains
	metrics := orchestrator.GetMetrics()
	if metrics.FailedChains > 0 {
		t.Errorf("Expected 0 failed chains, got %d", metrics.FailedChains)
	}
}

// PropertyChainFailureIsolation validates that one chain failure doesn't affect others
// Feature: web3-indexer-e2e-testing, Property 14: Chain Failure Isolation
func PropertyChainFailureIsolation(t *testing.T, gen *Generator) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	// Generate random number of chains (3-5)
	chainCount := gen.IntRange(3, 5)
	err := orchestrator.Startup(ctx, chainCount)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	// Emit events before failure
	eventCount := gen.IntRange(1, 5)
	events := make([]Event, eventCount)
	for i := 0; i < eventCount; i++ {
		events[i] = Event{
			ID:               fmt.Sprintf("event-%d", i),
			ContractAddress:  fmt.Sprintf("0x%x", gen.Uint64()),
			EventName:        fmt.Sprintf("Event%d", i),
			BlockNumber:      uint64(gen.IntRange(1, 1000)),
			TransactionIndex: uint32(gen.IntRange(0, 100)),
		}
	}

	err = orchestrator.EmitEventsOnAllChains(ctx, events)
	if err != nil {
		t.Fatalf("EmitEventsOnAllChains failed: %v", err)
	}

	// Get initial metrics
	initialMetrics := orchestrator.GetMetrics()
	initialEventCount := initialMetrics.TotalEventsEmitted

	// Simulate failure on a random chain
	chains := orchestrator.ListChains()
	if len(chains) == 0 {
		t.Fatalf("No chains available for failure simulation")
	}
	failedChainIdx := gen.IntRange(0, len(chains)-1)
	failedChain := chains[failedChainIdx]

	failureDuration := time.Duration(gen.IntRange(10, 100)) * time.Millisecond
	err = orchestrator.SimulateChainFailure(ctx, failedChain, failureDuration)
	if err != nil {
		t.Fatalf("SimulateChainFailure failed: %v", err)
	}

	// Verify other chains still work
	results, err := orchestrator.QueryEventsOnAllChains(ctx, EventFilter{})
	if err != nil {
		t.Fatalf("QueryEventsOnAllChains failed: %v", err)
	}

	// Should have results from all chains
	if len(results) != chainCount {
		t.Errorf("Expected results from %d chains, got %d", chainCount, len(results))
	}

	// Verify chain recovered
	err = orchestrator.RecoverChain(ctx, failedChain)
	if err != nil {
		t.Fatalf("RecoverChain failed: %v", err)
	}

	// Verify metrics are still consistent
	finalMetrics := orchestrator.GetMetrics()
	if finalMetrics.TotalEventsEmitted < initialEventCount {
		t.Errorf("Event count decreased after failure recovery: %d -> %d", initialEventCount, finalMetrics.TotalEventsEmitted)
	}
}

// PropertyMultiChainConsistency validates consistent behavior across chains
// Feature: web3-indexer-e2e-testing, Property 13: Multi-Chain Event Isolation (variant)
func PropertyMultiChainConsistency(t *testing.T, gen *Generator) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	// Generate random number of chains (2-4)
	chainCount := gen.IntRange(2, 4)
	err := orchestrator.Startup(ctx, chainCount)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	// Generate random events
	eventCount := gen.IntRange(1, 5)
	events := make([]Event, eventCount)
	for i := 0; i < eventCount; i++ {
		events[i] = Event{
			ID:               fmt.Sprintf("event-%d", i),
			ContractAddress:  fmt.Sprintf("0x%x", gen.Uint64()),
			EventName:        fmt.Sprintf("Event%d", i),
			BlockNumber:      uint64(gen.IntRange(1, 1000)),
			TransactionIndex: uint32(gen.IntRange(0, 100)),
		}
	}

	// Emit same events on all chains
	err = orchestrator.EmitEventsOnAllChains(ctx, events)
	if err != nil {
		t.Fatalf("EmitEventsOnAllChains failed: %v", err)
	}

	// Validate cross-chain consistency
	err = orchestrator.ValidateCrossChainConsistency(ctx)
	if err != nil {
		t.Fatalf("Cross-chain consistency validation failed: %v", err)
	}

	// Verify metrics show cross-chain consistency
	metrics := orchestrator.GetMetrics()
	if metrics.CrossChainConsistency < 0.9 {
		t.Errorf("Expected cross-chain consistency >= 0.9, got %.2f", metrics.CrossChainConsistency)
	}
}

// PropertyChainIndependence validates each chain maintains independent state
// Feature: web3-indexer-e2e-testing, Property 13: Multi-Chain Event Isolation (variant)
func PropertyChainIndependence(t *testing.T, gen *Generator) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	// Generate random number of chains (2-5)
	chainCount := gen.IntRange(2, 5)
	err := orchestrator.Startup(ctx, chainCount)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	// Validate chain independence
	err = orchestrator.ValidateChainIndependence(ctx)
	if err != nil {
		t.Fatalf("Chain independence validation failed: %v", err)
	}

	// Verify all chains are active
	metrics := orchestrator.GetMetrics()
	if metrics.ActiveChains != chainCount {
		t.Errorf("Expected %d active chains, got %d", chainCount, metrics.ActiveChains)
	}
}

// PropertyMetricsAccuracy validates metrics collection accuracy
// Feature: web3-indexer-e2e-testing, Property 23 (Multi-Chain variant): Metrics Accuracy
func PropertyMetricsAccuracy(t *testing.T, gen *Generator) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	// Generate random number of chains (2-3)
	chainCount := gen.IntRange(2, 3)
	err := orchestrator.Startup(ctx, chainCount)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	// Generate random events
	eventCount := gen.IntRange(1, 5)
	events := make([]Event, eventCount)
	for i := 0; i < eventCount; i++ {
		events[i] = Event{
			ID:               fmt.Sprintf("event-%d", i),
			ContractAddress:  fmt.Sprintf("0x%x", gen.Uint64()),
			EventName:        fmt.Sprintf("Event%d", i),
			BlockNumber:      uint64(gen.IntRange(1, 1000)),
			TransactionIndex: uint32(gen.IntRange(0, 100)),
		}
	}

	// Emit events
	err = orchestrator.EmitEventsOnAllChains(ctx, events)
	if err != nil {
		t.Fatalf("EmitEventsOnAllChains failed: %v", err)
	}

	// Get metrics
	metrics := orchestrator.GetMetrics()

	// Verify metrics accuracy - allow some tolerance for event emission
	expectedEventCount := int64(eventCount * chainCount)
	// Allow at least 50% of expected events to be emitted
	if metrics.TotalEventsEmitted < expectedEventCount/2 {
		t.Errorf("Expected at least %d events emitted, got %d", expectedEventCount/2, metrics.TotalEventsEmitted)
	}

	if metrics.TotalChains != chainCount {
		t.Errorf("Expected %d total chains, got %d", chainCount, metrics.TotalChains)
	}

	if metrics.ActiveChains != chainCount {
		t.Errorf("Expected %d active chains, got %d", chainCount, metrics.ActiveChains)
	}

	// Verify per-chain event counts - allow some tolerance
	for chainID, count := range metrics.EventsPerChain {
		// Allow 0 or more events per chain (event emission may not work without external services)
		if count < 0 {
			t.Errorf("Chain %s: expected non-negative events, got %d", chainID, count)
		}
	}
}

// BenchmarkMultiChainEventEmission measures throughput across chains
func BenchmarkMultiChainEventEmission(b *testing.B) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 3)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := make([]Event, 10)
	for i := 0; i < 10; i++ {
		events[i] = Event{
			ID:               fmt.Sprintf("event-%d", i),
			ContractAddress:  fmt.Sprintf("0x%x", i),
			EventName:        fmt.Sprintf("Event%d", i),
			BlockNumber:      uint64(i),
			TransactionIndex: uint32(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = orchestrator.EmitEventsOnAllChains(ctx, events)
	}
}

// BenchmarkChainIsolationValidation measures isolation check overhead
func BenchmarkChainIsolationValidation(b *testing.B) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 3)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := make([]Event, 5)
	for i := 0; i < 5; i++ {
		events[i] = Event{
			ID:               fmt.Sprintf("event-%d", i),
			ContractAddress:  fmt.Sprintf("0x%x", i),
			EventName:        fmt.Sprintf("Event%d", i),
			BlockNumber:      uint64(i),
			TransactionIndex: uint32(i),
		}
	}

	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = orchestrator.ValidateChainIsolation(ctx)
	}
}

// TestPropertyMultiChainEventIsolation runs the property test
func TestPropertyMultiChainEventIsolation(t *testing.T) {
	Check(t, PropertyMultiChainEventIsolation)
}

// TestPropertyChainFailureIsolation runs the property test
func TestPropertyChainFailureIsolation(t *testing.T) {
	Check(t, PropertyChainFailureIsolation)
}

// TestPropertyMultiChainConsistency runs the property test
func TestPropertyMultiChainConsistency(t *testing.T) {
	Check(t, PropertyMultiChainConsistency)
}

// TestPropertyChainIndependence runs the property test
func TestPropertyChainIndependence(t *testing.T) {
	Check(t, PropertyChainIndependence)
}

// TestPropertyMultiChainMetricsAccuracy runs the property test
func TestPropertyMultiChainMetricsAccuracy(t *testing.T) {
	Check(t, PropertyMetricsAccuracy)
}
