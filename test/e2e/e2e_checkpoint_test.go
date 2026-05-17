package e2e

import (
	"context"
	"testing"
)

// TestCheckpointBasicFunctionality tests basic E2E functionality
func TestCheckpointBasicFunctionality(t *testing.T) {
	ctx := context.Background()

	// Test 1: Orchestrator creation
	orchestrator := NewTestOrchestrator()
	if orchestrator == nil {
		t.Fatal("Failed to create orchestrator")
	}

	// Test 2: Multi-chain orchestrator
	multiChainOrch := NewDefaultMultiChainOrchestrator()
	if multiChainOrch == nil {
		t.Fatal("Failed to create multi-chain orchestrator")
	}

	// Test 3: Startup multi-chain
	err := multiChainOrch.Startup(ctx, 2)
	if err != nil {
		t.Fatalf("Failed to startup multi-chain orchestrator: %v", err)
	}

	// Test 4: Get chain count
	count := multiChainOrch.GetChainCount()
	if count != 2 {
		t.Fatalf("Expected 2 chains, got %d", count)
	}

	// Test 5: Get metrics
	metrics := multiChainOrch.GetMetrics()
	if metrics.TotalChains != 2 {
		t.Fatalf("Expected 2 total chains in metrics, got %d", metrics.TotalChains)
	}

	// Test 6: Shutdown
	err = multiChainOrch.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Failed to shutdown multi-chain orchestrator: %v", err)
	}

	// Test 7: Metrics collector
	mc := NewMetricsCollector("test_checkpoint")
	mc.RecordSuccess()
	mc.RecordSuccess()
	mc.RecordError()

	successRate := mc.GetSuccessRate()
	if successRate < 60 || successRate > 70 {
		t.Fatalf("Expected success rate around 66%%, got %.2f%%", successRate)
	}

	// Test 8: Performance manager
	pm := NewPerformanceManager()
	pm.RecordOperation("test_op", 100, true)
	pm.RecordOperation("test_op", 200, true)

	metrics_map := pm.CalculateMetrics()
	if _, exists := metrics_map["test_op"]; !exists {
		t.Fatal("Expected test_op metrics to exist")
	}

	t.Logf("✓ All checkpoint tests passed")
}

// TestCheckpointScenarioFramework tests the scenario framework
func TestCheckpointScenarioFramework(t *testing.T) {
	ctx := context.Background()

	// Create scenario executor
	logger := &SimpleLogger{}
	metrics := NewMetricsCollector("scenario_test")
	executor := NewScenarioExecutor(logger, metrics)

	// Create a simple scenario
	scenario := NewScenarioBuilder("test_scenario", ScenarioTypeHappyPath).
		WithDescription("Test scenario").
		WithSetup(func(ctx context.Context) error {
			return nil
		}).
		AddStep(ScenarioStep{
			Name:        "step1",
			Description: "First step",
			Action: func(ctx context.Context) error {
				return nil
			},
			Timeout: 10,
		}).
		WithTeardown(func(ctx context.Context) error {
			return nil
		}).
		Build()

	// Register scenario
	err := executor.RegisterScenario(scenario)
	if err != nil {
		t.Fatalf("Failed to register scenario: %v", err)
	}

	// Execute scenario
	result, err := executor.ExecuteScenario(ctx, "test_scenario")
	if err != nil {
		t.Fatalf("Failed to execute scenario: %v", err)
	}

	if result.Status != "PASSED" {
		t.Fatalf("Expected scenario to pass, got status: %s", result.Status)
	}

	t.Logf("✓ Scenario framework tests passed")
}

// SimpleLogger is a simple logger implementation for testing
type SimpleLogger struct{}

func (sl *SimpleLogger) Infof(format string, args ...any) {
	// No-op for testing
}

func (sl *SimpleLogger) Warnf(format string, args ...any) {
	// No-op for testing
}

func (sl *SimpleLogger) Errorf(format string, args ...any) {
	// No-op for testing
}

func (sl *SimpleLogger) Debugf(format string, args ...any) {
	// No-op for testing
}
