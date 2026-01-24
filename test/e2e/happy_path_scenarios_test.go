package e2e

import (
	"context"
	"testing"
	"time"
)

func TestHappyPath_BlockchainTransaction(t *testing.T) {
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

	scenario := HappyPathScenario{
		name:        "BlockchainTransaction",
		description: "Test successful blockchain transaction execution",
		execute:     executeBlockchainTransaction,
	}

	if err := RunHappyPathScenario(ctx, orch, scenario); err != nil {
		t.Errorf("BlockchainTransaction scenario failed: %v", err)
	}
}

func TestHappyPath_DatabaseInsertAndQuery(t *testing.T) {
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

	scenario := HappyPathScenario{
		name:        "DatabaseInsertAndQuery",
		description: "Test successful database insert and query",
		execute:     executeDatabaseInsertAndQuery,
	}

	if err := RunHappyPathScenario(ctx, orch, scenario); err != nil {
		t.Errorf("DatabaseInsertAndQuery scenario failed: %v", err)
	}
}

func TestHappyPath_APIEndpointCall(t *testing.T) {
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

	scenario := HappyPathScenario{
		name:        "APIEndpointCall",
		description: "Test successful API endpoint call",
		execute:     executeAPIEndpointCall,
	}

	if err := RunHappyPathScenario(ctx, orch, scenario); err != nil {
		t.Errorf("APIEndpointCall scenario failed: %v", err)
	}
}

func TestHappyPath_CompleteWorkflow(t *testing.T) {
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

	scenario := HappyPathScenario{
		name:        "CompleteWorkflow",
		description: "Test complete workflow across all services",
		execute:     executeCompleteWorkflow,
	}

	if err := RunHappyPathScenario(ctx, orch, scenario); err != nil {
		t.Errorf("CompleteWorkflow scenario failed: %v", err)
	}
}

func TestHappyPath_DataConsistency(t *testing.T) {
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

	scenario := HappyPathScenario{
		name:        "DataConsistency",
		description: "Test data consistency across blockchain and database",
		execute:     executeDataConsistency,
	}

	if err := RunHappyPathScenario(ctx, orch, scenario); err != nil {
		t.Errorf("DataConsistency scenario failed: %v", err)
	}
}

func TestHappyPath_AllScenarios(t *testing.T) {
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

	results := RunAllHappyPathScenarios(ctx, orch)

	failedCount := 0
	for name, err := range results {
		if err != nil {
			t.Errorf("Scenario %s failed: %v", name, err)
			failedCount++
		}
	}

	if failedCount > 0 {
		t.Errorf("%d/%d happy path scenarios failed", failedCount, len(results))
	}
}
