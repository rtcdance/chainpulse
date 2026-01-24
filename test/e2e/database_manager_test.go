package e2e

import (
	"context"
	"testing"
	"time"
)

func TestDatabaseManager_Initialize(t *testing.T) {
	// This test requires PostgreSQL to be running
	// Skip if PostgreSQL is not available
	t.Skip("Requires PostgreSQL to be running")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connString := "postgres://user:password@localhost:5432/testdb"
	dm := NewDatabaseManager(connString)
	defer func() { _ = dm.Close() }()

	err := dm.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !dm.initialized {
		t.Error("Expected initialized to be true")
	}

	if dm.pool == nil {
		t.Error("Expected pool to be set")
	}
}

func TestDatabaseManager_Initialize_AlreadyInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
	dm.initialized = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dm.Initialize(ctx)
	if err == nil {
		t.Error("Expected error when already initialized")
	}
}

func TestDatabaseManager_Close(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
	dm.initialized = true

	err := dm.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if dm.initialized {
		t.Error("Expected initialized to be false")
	}
}

func TestDatabaseManager_GetPool_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	pool := dm.GetPool()
	if pool != nil {
		t.Error("Expected pool to be nil when not initialized")
	}
}

func TestDatabaseManager_GetConnection_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dm.GetConnection(ctx)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_ExecuteQuery_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dm.ExecuteQuery(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_ExecuteCommand_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dm.ExecuteCommand(ctx, "DELETE FROM test_table")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_ExecuteRow_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := dm.ExecuteRow(ctx, "SELECT 1")
	if row != nil {
		t.Error("Expected row to be nil when not initialized")
	}
}

func TestDatabaseManager_BeginTransaction_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dm.BeginTransaction(ctx)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_ClearTable_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dm.ClearTable(ctx, "test_table")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_ClearAllTables_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dm.ClearAllTables(ctx, []string{"table1", "table2"})
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_GetTableRowCount_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dm.GetTableRowCount(ctx, "test_table")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_IsConnected_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if dm.IsConnected(ctx) {
		t.Error("Expected IsConnected to return false when not initialized")
	}
}

func TestDatabaseManager_WaitForConnection_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := dm.WaitForConnection(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_CreateTable_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dm.CreateTable(ctx, "CREATE TABLE test (id INT)")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestDatabaseManager_DropTable_NotInitialized(t *testing.T) {
	dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := dm.DropTable(ctx, "test_table")
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}
