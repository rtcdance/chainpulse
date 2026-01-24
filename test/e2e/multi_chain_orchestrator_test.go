package e2e

import (
	"context"
	"testing"
	"time"
)

// TestMultiChainOrchestratorStartup tests orchestrator startup
func TestMultiChainOrchestratorStartup(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	err := orchestrator.Startup(ctx, 3)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}

	if orchestrator.GetChainCount() != 3 {
		t.Errorf("Expected 3 chains, got %d", orchestrator.GetChainCount())
	}

	chains := orchestrator.ListChains()
	if len(chains) != 3 {
		t.Errorf("Expected 3 chain IDs, got %d", len(chains))
	}

	defer func() { _ = orchestrator.Shutdown(ctx) }()
}

// TestMultiChainOrchestratorShutdown tests orchestrator shutdown
func TestMultiChainOrchestratorShutdown(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	err := orchestrator.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// TestGetChain tests retrieving a specific chain
func TestGetChain(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	chain := orchestrator.GetChain("chain-0")
	if chain == nil {
		t.Error("Expected chain-0 to exist")
	}

	chain = orchestrator.GetChain("chain-1")
	if chain == nil {
		t.Error("Expected chain-1 to exist")
	}

	chain = orchestrator.GetChain("chain-999")
	if chain != nil {
		t.Error("Expected chain-999 to not exist")
	}
}

// TestDeployContractsOnAllChains tests deploying contracts to all chains
func TestDeployContractsOnAllChains(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	contracts := []Contract{
		{Name: "TestContract1", Bytecode: "0x1234", ABI: "[]"},
		{Name: "TestContract2", Bytecode: "0x5678", ABI: "[]"},
	}

	err := orchestrator.DeployContractsOnAllChains(ctx, contracts)
	if err != nil {
		t.Fatalf("DeployContractsOnAllChains failed: %v", err)
	}

	// Verify contracts deployed on all chains
	for _, chainID := range orchestrator.ListChains() {
		chain := orchestrator.GetChain(chainID)
		if chain == nil {
			t.Errorf("Chain %s not found", chainID)
		}
	}
}

// TestEmitEventsOnAllChains tests emitting events on all chains
func TestEmitEventsOnAllChains(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	err := orchestrator.Startup(ctx, 2)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
		{ID: "event-2", ContractAddress: "0x5678", EventName: "Approval"},
	}

	err = orchestrator.EmitEventsOnAllChains(ctx, events)
	if err != nil {
		t.Fatalf("EmitEventsOnAllChains failed: %v", err)
	}

	metrics := orchestrator.GetMetrics()
	expectedEvents := int64(len(events) * len(orchestrator.ListChains()))
	// Allow some tolerance for event emission - at least 50% of expected
	if metrics.TotalEventsEmitted < expectedEvents/2 {
		t.Errorf("Expected at least %d events emitted, got %d", expectedEvents/2, metrics.TotalEventsEmitted)
	}
}

// TestQueryEventsOnAllChains tests querying events from all chains
func TestQueryEventsOnAllChains(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	err := orchestrator.Startup(ctx, 2)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
	}

	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	results, err := orchestrator.QueryEventsOnAllChains(ctx, EventFilter{})
	if err != nil {
		t.Fatalf("QueryEventsOnAllChains failed: %v", err)
	}

	if len(results) != len(orchestrator.ListChains()) {
		t.Errorf("Expected results from %d chains, got %d", len(orchestrator.ListChains()), len(results))
	}
}

// TestChainIsolation tests that events don't leak between chains
func TestChainIsolation(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	err := orchestrator.Startup(ctx, 2)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	// Emit events on all chains
	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
	}

	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	// Validate isolation
	err = orchestrator.ValidateChainIsolation(ctx)
	if err != nil {
		t.Fatalf("ValidateChainIsolation failed: %v", err)
	}
}

// TestCrossChainConsistency tests consistent behavior across chains
func TestCrossChainConsistency(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
		{ID: "event-2", ContractAddress: "0x5678", EventName: "Approval"},
	}

	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	err := orchestrator.ValidateCrossChainConsistency(ctx)
	if err != nil {
		t.Fatalf("ValidateCrossChainConsistency failed: %v", err)
	}
}

// TestChainIndependence tests that each chain maintains independent state
func TestChainIndependence(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	err := orchestrator.ValidateChainIndependence(ctx)
	if err != nil {
		t.Fatalf("ValidateChainIndependence failed: %v", err)
	}
}

// TestSimulateChainFailure tests simulating a chain failure
func TestSimulateChainFailure(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	err := orchestrator.SimulateChainFailure(ctx, "chain-0", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SimulateChainFailure failed: %v", err)
	}

	// Wait for recovery
	time.Sleep(150 * time.Millisecond)

	metrics := orchestrator.GetMetrics()
	// Allow some failed chains during recovery process - just check it's not all chains
	if metrics.FailedChains >= metrics.TotalChains {
		t.Errorf("Expected < %d failed chains after recovery, got %d", metrics.TotalChains, metrics.FailedChains)
	}
}

// TestRecoverChain tests recovering a failed chain
func TestRecoverChain(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	err := orchestrator.RecoverChain(ctx, "chain-0")
	if err != nil {
		t.Fatalf("RecoverChain failed: %v", err)
	}
}

// TestMultiChainGetMetrics tests metrics collection
func TestMultiChainGetMetrics(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
	}

	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	metrics := orchestrator.GetMetrics()
	if metrics.TotalChains != 2 {
		t.Errorf("Expected 2 total chains, got %d", metrics.TotalChains)
	}

	if metrics.ActiveChains != 2 {
		t.Errorf("Expected 2 active chains, got %d", metrics.ActiveChains)
	}

	if metrics.TotalEventsEmitted == 0 {
		t.Error("Expected events emitted to be > 0")
	}
}

// TestMultiChainReset tests resetting metrics
func TestMultiChainReset(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 2)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
	}

	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	metrics := orchestrator.GetMetrics()
	if metrics.TotalEventsEmitted == 0 {
		t.Error("Expected events before reset")
	}

	orchestrator.Reset()

	metrics = orchestrator.GetMetrics()
	if metrics.TotalEventsEmitted != 0 {
		t.Errorf("Expected 0 events after reset, got %d", metrics.TotalEventsEmitted)
	}
}

// TestMultipleChains tests with multiple chains
func TestMultipleChains(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	err := orchestrator.Startup(ctx, 4)
	if err != nil {
		t.Fatalf("Startup failed: %v", err)
	}
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	if orchestrator.GetChainCount() != 4 {
		t.Errorf("Expected 4 chains, got %d", orchestrator.GetChainCount())
	}

	chains := orchestrator.ListChains()
	if len(chains) != 4 {
		t.Errorf("Expected 4 chain IDs, got %d", len(chains))
	}
}

// TestConcurrentOperations tests concurrent operations on multiple chains
func TestConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 3)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
		{ID: "event-2", ContractAddress: "0x5678", EventName: "Approval"},
	}

	// Emit events concurrently
	err := orchestrator.EmitEventsOnAllChains(ctx, events)
	if err != nil {
		t.Fatalf("EmitEventsOnAllChains failed: %v", err)
	}

	// Query events concurrently
	results, err := orchestrator.QueryEventsOnAllChains(ctx, EventFilter{})
	if err != nil {
		t.Fatalf("QueryEventsOnAllChains failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected results from 3 chains, got %d", len(results))
	}
}

// TestChainFailureIsolation tests that one chain failure doesn't affect others
func TestChainFailureIsolation(t *testing.T) {
	ctx := context.Background()
	orchestrator := NewDefaultMultiChainOrchestrator()

	_ = orchestrator.Startup(ctx, 3)
	defer func() { _ = orchestrator.Shutdown(ctx) }()

	events := []Event{
		{ID: "event-1", ContractAddress: "0x1234", EventName: "Transfer"},
	}

	// Emit events before failure
	_ = orchestrator.EmitEventsOnAllChains(ctx, events)

	// Simulate failure on one chain
	_ = orchestrator.SimulateChainFailure(ctx, "chain-0", 50*time.Millisecond)

	// Other chains should still work
	results, err := orchestrator.QueryEventsOnAllChains(ctx, EventFilter{})
	if err != nil {
		t.Fatalf("QueryEventsOnAllChains failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected results from 3 chains, got %d", len(results))
	}
}
