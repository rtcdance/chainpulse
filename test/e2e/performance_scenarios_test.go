package e2e

import (
	"context"
	"testing"
	"time"
)

func TestPerformanceScenario_BlockchainReadThroughput(t *testing.T) {
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

	scenario := PerformanceScenario{
		name:        "BlockchainReadThroughput",
		description: "Test blockchain read operation throughput",
		execute:     executeBlockchainReadThroughput,
	}

	if err := RunPerformanceScenario(ctx, orch, scenario); err != nil {
		t.Errorf("BlockchainReadThroughput scenario failed: %v", err)
	}
}

func TestPerformanceScenario_DatabaseWriteThroughput(t *testing.T) {
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

	scenario := PerformanceScenario{
		name:        "DatabaseWriteThroughput",
		description: "Test database write operation throughput",
		execute:     executeDatabaseWriteThroughput,
	}

	if err := RunPerformanceScenario(ctx, orch, scenario); err != nil {
		t.Errorf("DatabaseWriteThroughput scenario failed: %v", err)
	}
}

func TestPerformanceScenario_ConcurrentBlockchainReads(t *testing.T) {
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

	scenario := PerformanceScenario{
		name:        "ConcurrentBlockchainReads",
		description: "Test concurrent blockchain read operations",
		execute:     executeConcurrentBlockchainReads,
	}

	if err := RunPerformanceScenario(ctx, orch, scenario); err != nil {
		t.Errorf("ConcurrentBlockchainReads scenario failed: %v", err)
	}
}

func TestPerformanceScenario_ConcurrentDatabaseWrites(t *testing.T) {
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

	scenario := PerformanceScenario{
		name:        "ConcurrentDatabaseWrites",
		description: "Test concurrent database write operations",
		execute:     executeConcurrentDatabaseWrites,
	}

	if err := RunPerformanceScenario(ctx, orch, scenario); err != nil {
		t.Errorf("ConcurrentDatabaseWrites scenario failed: %v", err)
	}
}

func TestPerformanceScenario_EndToEndLatency(t *testing.T) {
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

	scenario := PerformanceScenario{
		name:        "EndToEndLatency",
		description: "Test end-to-end operation latency",
		execute:     executeEndToEndLatency,
	}

	if err := RunPerformanceScenario(ctx, orch, scenario); err != nil {
		t.Errorf("EndToEndLatency scenario failed: %v", err)
	}
}

func TestPerformanceScenario_AllScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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

	results := RunAllPerformanceScenarios(ctx, orch)

	failedCount := 0
	for name, err := range results {
		if err != nil {
			t.Errorf("Scenario %s failed: %v", name, err)
			failedCount++
		}
	}

	if failedCount > 0 {
		t.Errorf("%d/%d performance scenarios failed", failedCount, len(results))
	}
}
