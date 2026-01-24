package e2e

import (
	"context"
	"testing"
	"time"
)

func TestOrchestrator_Initialize(t *testing.T) {
	// This test requires all services to be running
	// Skip if services are not available
	t.Skip("Requires Anvil, PostgreSQL, and API to be running")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)
	defer func() { _ = orch.Close() }()

	err := orch.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !orch.initialized {
		t.Error("Expected initialized to be true")
	}
}

func TestOrchestrator_Initialize_AlreadyInitialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)
	orch.initialized = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Initialize(ctx)
	if err == nil {
		t.Error("Expected error when already initialized")
	}
}

func TestOrchestrator_Close(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)
	orch.initialized = true

	err := orch.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if orch.initialized {
		t.Error("Expected initialized to be false")
	}
}

func TestOrchestrator_Close_NotInitialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	err := orch.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestOrchestrator_GetBlockchainManager(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	bm := orch.GetBlockchainManager()
	if bm == nil {
		t.Error("Expected blockchain manager to be returned")
	}

	if bm != orch.blockchain {
		t.Error("Expected returned blockchain manager to be the same instance")
	}
}

func TestOrchestrator_GetDatabaseManager(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	dm := orch.GetDatabaseManager()
	if dm == nil {
		t.Error("Expected database manager to be returned")
	}

	if dm != orch.database {
		t.Error("Expected returned database manager to be the same instance")
	}
}

func TestOrchestrator_GetAPIManager(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	am := orch.GetAPIManager()
	if am == nil {
		t.Error("Expected API manager to be returned")
	}

	if am != orch.api {
		t.Error("Expected returned API manager to be the same instance")
	}
}

func TestOrchestrator_WaitForAllServices_NotInitialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := orch.WaitForAllServices(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestOrchestrator_IsReady_NotInitialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if orch.IsReady(ctx) {
		t.Error("Expected IsReady to return false when not initialized")
	}
}

func TestOrchestrator_Reset_NotInitialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := orch.Reset(ctx, []string{"table1", "table2"})
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestOrchestrator_GetStatus_NotInitialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := orch.GetStatus(ctx)
	if status["initialized"] {
		t.Error("Expected initialized to be false")
	}
}

func TestOrchestrator_GetStatus_Initialized(t *testing.T) {
	orch := NewOrchestrator(
		"http://localhost:8545",
		"postgres://user:password@localhost:5432/testdb",
		"http://localhost:8080",
	)
	orch.initialized = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := orch.GetStatus(ctx)
	if !status["initialized"] {
		t.Error("Expected initialized to be true")
	}

	// Other services should be false since they're not actually initialized
	if status["blockchain"] {
		t.Error("Expected blockchain to be false")
	}

	if status["database"] {
		t.Error("Expected database to be false")
	}

	if status["api"] {
		t.Error("Expected api to be false")
	}
}
