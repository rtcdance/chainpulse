package e2e

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property: Initialization Idempotence
// For any database manager, calling Initialize multiple times should fail on the second call
func TestProperty_DatabaseManager_InitializationIdempotence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		// First initialization should succeed (or fail due to connection)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_ = dm.Initialize(ctx)

		// Second initialization should fail
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()

		err := dm.Initialize(ctx2)
		if err == nil {
			rt.Fatalf("Expected error on second initialization")
		}
	})
}

// Property: State Consistency
// For any database manager, if initialized is true, pool should not be nil
func TestProperty_DatabaseManager_StateConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		// Check initial state
		if dm.initialized && dm.pool != nil {
			rt.Fatalf("Expected initialized to be false initially")
		}

		// After failed initialization attempt, state should be consistent
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_ = dm.Initialize(ctx)

		// If initialized is true, pool should not be nil
		if dm.initialized && dm.pool == nil {
			rt.Fatalf("Expected pool to be set when initialized")
		}

		// If initialized is false, pool should be nil
		if !dm.initialized && dm.pool != nil {
			rt.Fatalf("Expected pool to be nil when not initialized")
		}
	})
}

// Property: Close Idempotence
// For any database manager, calling Close multiple times should not error
func TestProperty_DatabaseManager_CloseIdempotence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")

		// Close multiple times
		for i := 0; i < 3; i++ {
			err := dm.Close()
			if err != nil {
				rt.Fatalf("Close failed on iteration %d: %v", i, err)
			}

			if dm.initialized {
				rt.Fatalf("Expected initialized to be false after Close")
			}
		}
	})
}

// Property: Operations Require Initialization
// For any database manager operation, if not initialized, it should error
func TestProperty_DatabaseManager_OperationsRequireInitialization(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// All operations should fail when not initialized
		_, err := dm.GetConnection(ctx)
		if err == nil {
			rt.Fatalf("Expected GetConnection to fail when not initialized")
		}

		_, err = dm.ExecuteQuery(ctx, "SELECT 1")
		if err == nil {
			rt.Fatalf("Expected ExecuteQuery to fail when not initialized")
		}

		_, err = dm.ExecuteCommand(ctx, "DELETE FROM test")
		if err == nil {
			rt.Fatalf("Expected ExecuteCommand to fail when not initialized")
		}

		_, err = dm.BeginTransaction(ctx)
		if err == nil {
			rt.Fatalf("Expected BeginTransaction to fail when not initialized")
		}

		err = dm.ClearTable(ctx, "test_table")
		if err == nil {
			rt.Fatalf("Expected ClearTable to fail when not initialized")
		}

		err = dm.ClearAllTables(ctx, []string{"table1"})
		if err == nil {
			rt.Fatalf("Expected ClearAllTables to fail when not initialized")
		}

		_, err = dm.GetTableRowCount(ctx, "test_table")
		if err == nil {
			rt.Fatalf("Expected GetTableRowCount to fail when not initialized")
		}

		err = dm.CreateTable(ctx, "CREATE TABLE test (id INT)")
		if err == nil {
			rt.Fatalf("Expected CreateTable to fail when not initialized")
		}

		err = dm.DropTable(ctx, "test_table")
		if err == nil {
			rt.Fatalf("Expected DropTable to fail when not initialized")
		}
	})
}

// Property: GetPool Returns Nil When Not Initialized
// For any database manager, GetPool should return nil when not initialized
func TestProperty_DatabaseManager_GetPoolNilWhenNotInitialized(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		pool := dm.GetPool()
		if pool != nil {
			rt.Fatalf("Expected GetPool to return nil when not initialized")
		}
	})
}

// Property: IsConnected Consistency
// For any database manager, IsConnected should return false when not initialized
func TestProperty_DatabaseManager_IsConnectedConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// When not initialized, IsConnected should return false
		if dm.IsConnected(ctx) {
			rt.Fatalf("Expected IsConnected to return false when not initialized")
		}
	})
}

// Property: WaitForConnection Timeout
// For any database manager, WaitForConnection should timeout if not initialized
func TestProperty_DatabaseManager_WaitForConnectionTimeout(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := dm.WaitForConnection(ctx, 100*time.Millisecond)
		if err == nil {
			rt.Fatalf("Expected WaitForConnection to timeout")
		}
	})
}

// Property: ExecuteRow Returns Nil When Not Initialized
// For any database manager, ExecuteRow should return nil when not initialized
func TestProperty_DatabaseManager_ExecuteRowNilWhenNotInitialized(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		dm := NewDatabaseManager("postgres://user:password@localhost:5432/testdb")
		defer func() { _ = dm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		row := dm.ExecuteRow(ctx, "SELECT 1")
		if row != nil {
			rt.Fatalf("Expected ExecuteRow to return nil when not initialized")
		}
	})
}
