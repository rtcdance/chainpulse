package e2e

import (
	"context"
	"testing"
	"time"
)

func TestErrorScenario_InvalidBlockchainAddress(t *testing.T) {
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

	scenario := ErrorScenario{
		name:        "InvalidBlockchainAddress",
		description: "Test handling of invalid blockchain address",
		execute:     executeInvalidBlockchainAddress,
	}

	if err := RunErrorScenario(ctx, orch, scenario); err != nil {
		t.Errorf("InvalidBlockchainAddress scenario failed: %v", err)
	}
}

func TestErrorScenario_DatabaseConnectionFailure(t *testing.T) {
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

	scenario := ErrorScenario{
		name:        "DatabaseConnectionFailure",
		description: "Test handling of database connection failure",
		execute:     executeDatabaseConnectionFailure,
	}

	if err := RunErrorScenario(ctx, orch, scenario); err != nil {
		t.Errorf("DatabaseConnectionFailure scenario failed: %v", err)
	}
}

func TestErrorScenario_InvalidDatabaseQuery(t *testing.T) {
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

	scenario := ErrorScenario{
		name:        "InvalidDatabaseQuery",
		description: "Test handling of invalid database query",
		execute:     executeInvalidDatabaseQuery,
	}

	if err := RunErrorScenario(ctx, orch, scenario); err != nil {
		t.Errorf("InvalidDatabaseQuery scenario failed: %v", err)
	}
}

func TestErrorScenario_APIEndpointNotFound(t *testing.T) {
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

	scenario := ErrorScenario{
		name:        "APIEndpointNotFound",
		description: "Test handling of API endpoint not found",
		execute:     executeAPIEndpointNotFound,
	}

	if err := RunErrorScenario(ctx, orch, scenario); err != nil {
		t.Errorf("APIEndpointNotFound scenario failed: %v", err)
	}
}

func TestErrorScenario_ContextCancellation(t *testing.T) {
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

	scenario := ErrorScenario{
		name:        "ContextCancellation",
		description: "Test handling of context cancellation",
		execute:     executeContextCancellation,
	}

	if err := RunErrorScenario(ctx, orch, scenario); err != nil {
		t.Errorf("ContextCancellation scenario failed: %v", err)
	}
}

func TestErrorScenario_AllScenarios(t *testing.T) {
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

	results := RunAllErrorScenarios(ctx, orch)

	failedCount := 0
	for name, err := range results {
		if err != nil {
			t.Errorf("Scenario %s failed: %v", name, err)
			failedCount++
		}
	}

	if failedCount > 0 {
		t.Errorf("%d/%d error scenarios failed", failedCount, len(results))
	}
}
