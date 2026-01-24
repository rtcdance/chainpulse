package e2e

import (
	"context"
	"testing"
	"time"
)

func TestBlockchainManager_Initialize(t *testing.T) {
	// This test requires Anvil to be running
	// Skip if Anvil is not available
	t.Skip("Requires Anvil to be running")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bm := NewBlockchainManager("http://localhost:8545")
	defer func() { _ = bm.Close() }()

	err := bm.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !bm.initialized {
		t.Error("Expected initialized to be true")
	}

	if bm.client == nil {
		t.Error("Expected client to be set")
	}

	if bm.chainID == nil {
		t.Error("Expected chainID to be set")
	}
}

func TestBlockchainManager_Initialize_AlreadyInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")
	bm.initialized = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bm.Initialize(ctx)
	if err == nil {
		t.Error("Expected error when already initialized")
	}
}

func TestBlockchainManager_Close(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")
	bm.initialized = true

	err := bm.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if bm.initialized {
		t.Error("Expected initialized to be false")
	}
}

func TestBlockchainManager_GetChainID(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	chainID := bm.GetChainID()
	if chainID != nil {
		t.Error("Expected chainID to be nil before initialization")
	}
}

func TestBlockchainManager_IsConnected_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if bm.IsConnected(ctx) {
		t.Error("Expected IsConnected to return false when not initialized")
	}
}

func TestBlockchainManager_GetBalance_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := bm.GetBalance(ctx, [20]byte{})
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestBlockchainManager_GetBlockNumber_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := bm.GetBlockNumber(ctx)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestBlockchainManager_GetTransaction_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := bm.GetTransaction(ctx, [32]byte{})
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestBlockchainManager_GetTransactionReceipt_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := bm.GetTransactionReceipt(ctx, [32]byte{})
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestBlockchainManager_SendTransaction_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bm.SendTransaction(ctx, nil)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestBlockchainManager_GetTransactionOpts_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := bm.GetTransactionOpts(ctx, [20]byte{})
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}

func TestBlockchainManager_GetCallOpts(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := bm.GetCallOpts(ctx)
	if opts == nil {
		t.Error("Expected call opts to be returned")
	}
}

func TestBlockchainManager_WaitForConnection_NotInitialized(t *testing.T) {
	bm := NewBlockchainManager("http://localhost:8545")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := bm.WaitForConnection(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected error when not initialized")
	}
}
