package e2e

import (
	"context"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Property: Initialization Idempotence
// For any blockchain manager, calling Initialize multiple times should fail on the second call
func TestProperty_BlockchainManager_InitializationIdempotence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")
		defer func() { _ = bm.Close() }()

		// First initialization should succeed (or fail due to connection)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_ = bm.Initialize(ctx)

		// Second initialization should fail
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()

		err := bm.Initialize(ctx2)
		if err == nil {
			rt.Fatalf("Expected error on second initialization")
		}
	})
}

// Property: State Consistency
// For any blockchain manager, if initialized is true, client should not be nil
func TestProperty_BlockchainManager_StateConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")
		defer func() { _ = bm.Close() }()

		// Check initial state
		if bm.initialized && bm.client != nil {
			rt.Fatalf("Expected initialized to be false initially")
		}

		// After failed initialization attempt, state should be consistent
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		_ = bm.Initialize(ctx)

		// If initialized is true, client should not be nil
		if bm.initialized && bm.client == nil {
			rt.Fatalf("Expected client to be set when initialized")
		}

		// If initialized is false, client should be nil
		if !bm.initialized && bm.client != nil {
			rt.Fatalf("Expected client to be nil when not initialized")
		}
	})
}

// Property: Close Idempotence
// For any blockchain manager, calling Close multiple times should not error
func TestProperty_BlockchainManager_CloseIdempotence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")

		// Close multiple times
		for i := 0; i < 3; i++ {
			err := bm.Close()
			if err != nil {
				rt.Fatalf("Close failed on iteration %d: %v", i, err)
			}

			if bm.initialized {
				rt.Fatalf("Expected initialized to be false after Close")
			}
		}
	})
}

// Property: Operations Require Initialization
// For any blockchain manager operation, if not initialized, it should error
func TestProperty_BlockchainManager_OperationsRequireInitialization(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")
		defer func() { _ = bm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// All operations should fail when not initialized
		_, err := bm.GetBalance(ctx, [20]byte{})
		if err == nil {
			rt.Fatalf("Expected GetBalance to fail when not initialized")
		}

		_, err = bm.GetBlockNumber(ctx)
		if err == nil {
			rt.Fatalf("Expected GetBlockNumber to fail when not initialized")
		}

		_, _, err = bm.GetTransaction(ctx, [32]byte{})
		if err == nil {
			rt.Fatalf("Expected GetTransaction to fail when not initialized")
		}

		_, err = bm.GetTransactionReceipt(ctx, [32]byte{})
		if err == nil {
			rt.Fatalf("Expected GetTransactionReceipt to fail when not initialized")
		}

		err = bm.SendTransaction(ctx, nil)
		if err == nil {
			rt.Fatalf("Expected SendTransaction to fail when not initialized")
		}

		_, err = bm.GetTransactionOpts(ctx, [20]byte{})
		if err == nil {
			rt.Fatalf("Expected GetTransactionOpts to fail when not initialized")
		}
	})
}

// Property: GetCallOpts Always Returns Valid Opts
// For any blockchain manager, GetCallOpts should always return a valid CallOpts
func TestProperty_BlockchainManager_GetCallOptsAlwaysValid(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")
		defer func() { _ = bm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		opts := bm.GetCallOpts(ctx)
		if opts == nil {
			rt.Fatalf("Expected GetCallOpts to return valid opts")
		}

		if opts != nil && opts.Context != ctx {
			rt.Fatalf("Expected context to be set in opts")
		}
	})
}

// Property: IsConnected Consistency
// For any blockchain manager, IsConnected should return false when not initialized
func TestProperty_BlockchainManager_IsConnectedConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")
		defer func() { _ = bm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// When not initialized, IsConnected should return false
		if bm.IsConnected(ctx) {
			rt.Fatalf("Expected IsConnected to return false when not initialized")
		}
	})
}

// Property: WaitForConnection Timeout
// For any blockchain manager, WaitForConnection should timeout if not initialized
func TestProperty_BlockchainManager_WaitForConnectionTimeout(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		bm := NewBlockchainManager("http://localhost:8545")
		defer func() { _ = bm.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := bm.WaitForConnection(ctx, 100*time.Millisecond)
		if err == nil {
			rt.Fatalf("Expected WaitForConnection to timeout")
		}
	})
}
