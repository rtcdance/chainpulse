package e2e

import (
	"context"
	"testing"
	"time"
)

func TestMultiChainScenario_MultiChainStateSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://postgres:postgres@localhost:5432/chainpulse_test",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	if err := orch.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize orchestrator: %v", err)
	}

	if err := orch.WaitForAllServices(ctx, 10*time.Second); err != nil {
		t.Skipf("Services not ready: %v", err)
	}

	scenario := MultiChainScenario{
		name:        "MultiChainStateSync",
		description: "Test state synchronization across multiple chains",
		execute:     executeMultiChainStateSync,
	}

	if err := RunMultiChainScenario(ctx, orch, scenario); err != nil {
		t.Errorf("MultiChainStateSync scenario failed: %v", err)
	}
}

func TestMultiChainScenario_CrossChainDataConsistency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://postgres:postgres@localhost:5432/chainpulse_test",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	if err := orch.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize orchestrator: %v", err)
	}

	if err := orch.WaitForAllServices(ctx, 10*time.Second); err != nil {
		t.Skipf("Services not ready: %v", err)
	}

	scenario := MultiChainScenario{
		name:        "CrossChainDataConsistency",
		description: "Test data consistency across chains",
		execute:     executeCrossChainDataConsistency,
	}

	if err := RunMultiChainScenario(ctx, orch, scenario); err != nil {
		t.Errorf("CrossChainDataConsistency scenario failed: %v", err)
	}
}

func TestMultiChainScenario_MultiChainConcurrentOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://postgres:postgres@localhost:5432/chainpulse_test",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	if err := orch.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize orchestrator: %v", err)
	}

	if err := orch.WaitForAllServices(ctx, 10*time.Second); err != nil {
		t.Skipf("Services not ready: %v", err)
	}

	scenario := MultiChainScenario{
		name:        "MultiChainConcurrentOperations",
		description: "Test concurrent operations on multiple chains",
		execute:     executeMultiChainConcurrentOperations,
	}

	if err := RunMultiChainScenario(ctx, orch, scenario); err != nil {
		t.Errorf("MultiChainConcurrentOperations scenario failed: %v", err)
	}
}

func TestMultiChainScenario_MultiChainFailover(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://postgres:postgres@localhost:5432/chainpulse_test",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	if err := orch.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize orchestrator: %v", err)
	}

	if err := orch.WaitForAllServices(ctx, 10*time.Second); err != nil {
		t.Skipf("Services not ready: %v", err)
	}

	scenario := MultiChainScenario{
		name:        "MultiChainFailover",
		description: "Test failover between chains",
		execute:     executeMultiChainFailover,
	}

	if err := RunMultiChainScenario(ctx, orch, scenario); err != nil {
		t.Errorf("MultiChainFailover scenario failed: %v", err)
	}
}

func TestMultiChainScenario_MultiChainDataAggregation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://postgres:postgres@localhost:5432/chainpulse_test",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	if err := orch.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize orchestrator: %v", err)
	}

	if err := orch.WaitForAllServices(ctx, 10*time.Second); err != nil {
		t.Skipf("Services not ready: %v", err)
	}

	scenario := MultiChainScenario{
		name:        "MultiChainDataAggregation",
		description: "Test data aggregation from multiple chains",
		execute:     executeMultiChainDataAggregation,
	}

	if err := RunMultiChainScenario(ctx, orch, scenario); err != nil {
		t.Errorf("MultiChainDataAggregation scenario failed: %v", err)
	}
}

func TestMultiChainScenario_AllScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://postgres:postgres@localhost:5432/chainpulse_test",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	if err := orch.Initialize(ctx); err != nil {
		t.Skipf("Failed to initialize orchestrator: %v", err)
	}

	if err := orch.WaitForAllServices(ctx, 10*time.Second); err != nil {
		t.Skipf("Services not ready: %v", err)
	}

	results := RunAllMultiChainScenarios(ctx, orch)

	failedCount := 0
	for name, err := range results {
		if err != nil {
			t.Errorf("Scenario %s failed: %v", name, err)
			failedCount++
		}
	}

	if failedCount > 0 {
		t.Errorf("%d/%d multi-chain scenarios failed", failedCount, len(results))
	}
}
