package finality

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// mockLogger implements core.Logger for testing
type mockLogger struct {
	warns  []string
	errors []string
}

func (m *mockLogger) Debug(msg string, fields ...any) {}
func (m *mockLogger) Info(msg string, fields ...any)  {}
func (m *mockLogger) Warn(msg string, fields ...any) {
	m.warns = append(m.warns, msg)
}

func (m *mockLogger) Error(msg string, fields ...any) {
	m.errors = append(m.errors, msg)
}
func (m *mockLogger) Fatal(msg string, fields ...any)         {}
func (m *mockLogger) WithCorrelationID(id string) core.Logger { return m }

// rpcResponse builds a JSON-RPC response for eth_getBlockByNumber
func rpcResponse(blockNumber string, errMsg string) string {
	if errMsg != "" {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"message":"%s"}}`, errMsg)
	}
	if blockNumber == "" {
		return `{"jsonrpc":"2.0","id":1,"result":null}`
	}
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":{"number":"%s"}}`, blockNumber)
}

// --- L1 "finalized" tag ---

func TestGetFinalizedBlockNumber_L1Finalized(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request uses "finalized" tag
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		params, _ := req["params"].([]any)
		tag, _ := params[0].(string)

		if tag != "finalized" {
			t.Errorf("L1 chain should use 'finalized' tag, got %q", tag)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x3e8", "")) // block 1000
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL) // Ethereum mainnet = L1

	blockNum, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blockNum != 1000 {
		t.Errorf("block number = %d, want 1000", blockNum)
	}
}

// --- L2 "safe" tag ---

func TestGetFinalizedBlockNumber_L2Safe(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		params, _ := req["params"].([]any)
		tag, _ := params[0].(string)

		if tag != "safe" {
			t.Errorf("L2 chain should use 'safe' tag, got %q", tag)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x2710", "")) // block 10000
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("42161", server.URL) // Arbitrum One = L2, FinalityBlocks=960

	blockNum, err := fc.GetFinalizedBlockNumber(context.Background(), "42161")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 10000 - 960 (Arbitrum FinalityBlocks) = 9040
	if blockNum != 9040 {
		t.Errorf("block number = %d, want 9040", blockNum)
	}
}

func TestGetFinalizedBlockNumber_L2FinalityDiscountUnderflow(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x1f4", "")) // block 500
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("42161", server.URL) // Arbitrum One, FinalityBlocks=960

	blockNum, err := fc.GetFinalizedBlockNumber(context.Background(), "42161")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 500 < 960 FinalityBlocks → should clamp to 0
	if blockNum != 0 {
		t.Errorf("block number = %d, want 0 (underflow protection)", blockNum)
	}
}

// --- Fallback: finalized fails → safe → latest ---

func TestGetFinalizedBlockNumber_FallbackToLatest(t *testing.T) {
	t.Parallel()
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		params, _ := req["params"].([]any)
		tag, _ := params[0].(string)

		atomic.AddInt32(&callCount, 1)

		w.Header().Set("Content-Type", "application/json")
		switch tag {
		case "finalized":
			// "finalized" tag not supported — return error
			fmt.Fprint(w, rpcResponse("", "finalized block not found"))
		case "latest":
			// Fallback succeeds
			fmt.Fprint(w, rpcResponse("0x7b0", "")) // block 1968
		default:
			t.Errorf("unexpected tag: %q", tag)
			fmt.Fprint(w, rpcResponse("", "unexpected tag"))
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	fc := NewRPCFinalityChecker(logger)
	fc.RegisterChain("1", server.URL)

	blockNum, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if blockNum != 1968 {
		t.Errorf("block number = %d, want 1968", blockNum)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 RPC calls (finalized + latest), got %d", callCount)
	}
	if len(logger.errors) == 0 {
		t.Error("expected an error log for fallback")
	}
}

// --- Fallback: L2 safe fails → latest ---

func TestGetFinalizedBlockNumber_L2FallbackToLatest(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		params, _ := req["params"].([]any)
		tag, _ := params[0].(string)

		w.Header().Set("Content-Type", "application/json")
		switch tag {
		case "safe":
			fmt.Fprint(w, rpcResponse("", "safe tag not supported"))
		case "latest":
			fmt.Fprint(w, rpcResponse("0x2710", "")) // block 10000
		default:
			fmt.Fprint(w, rpcResponse("", "unexpected tag"))
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	fc := NewRPCFinalityChecker(logger)
	fc.RegisterChain("42161", server.URL) // Arbitrum

	blockNum, err := fc.GetFinalizedBlockNumber(context.Background(), "42161")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 10000 - 960 (Arbitrum FinalityBlocks) = 9040
	if blockNum != 9040 {
		t.Errorf("block number = %d, want 9040", blockNum)
	}
}

// --- All fallbacks fail ---

func TestGetFinalizedBlockNumber_AllFallbacksFail(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("", "node unavailable"))
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	_, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error when all RPC calls fail")
	}
}

// --- Unregistered chain ---

func TestGetFinalizedBlockNumber_UnregisteredChain(t *testing.T) {
	t.Parallel()
	fc := NewRPCFinalityChecker(&mockLogger{})
	// Don't register any chain

	_, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error for unregistered chain")
	}
}

// --- Cache hit ---

func TestGetFinalizedBlockNumber_CacheHit(t *testing.T) {
	t.Parallel()
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x64", "")) // block 100
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	// First call — should hit RPC
	block1, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err != nil || block1 != 100 {
		t.Fatalf("first call: block=%d, err=%v", block1, err)
	}

	// Second call — should hit cache (no new RPC request)
	block2, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err != nil || block2 != 100 {
		t.Fatalf("second call: block=%d, err=%v", block2, err)
	}

	if count := atomic.LoadInt32(&callCount); count != 1 {
		t.Errorf("expected 1 RPC call (second should be cached), got %d", count)
	}
}

// --- Cache expiry ---

func TestGetFinalizedBlockNumber_CacheExpiry(t *testing.T) {
	t.Parallel()
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		// Return different block numbers for each call
		blockNum := 100 + (count-1)*10
		resp := rpcResponse(fmt.Sprintf("0x%x", blockNum), "")
		fmt.Fprint(w, resp)
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.cacheTTL = 50 * time.Millisecond // Short TTL for test
	fc.RegisterChain("1", server.URL)

	// First call
	block1, _ := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if block1 != 100 {
		t.Fatalf("first call: block=%d, want 100", block1)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second call — should hit RPC again
	block2, _ := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if block2 != 110 {
		t.Fatalf("after cache expiry: block=%d, want 110", block2)
	}

	if count := atomic.LoadInt32(&callCount); count != 2 {
		t.Errorf("expected 2 RPC calls (cache expired), got %d", count)
	}
}

// --- RPC error response ---

func TestGetFinalizedBlockNumber_RPCErrorResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("", "method not found"))
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	_, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error for RPC error response")
	}
}

// --- Null result (block tag returns null) ---

func TestGetFinalizedBlockNumber_NullResult(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	_, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error for null block result")
	}
}

// --- Invalid JSON response ---

func TestGetFinalizedBlockNumber_InvalidJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not json at all")
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	_, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

// --- IsBlockFinalized ---

func TestIsBlockFinalized_True(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x3e8", "")) // block 1000
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	finalized, err := fc.IsBlockFinalized(context.Background(), "1", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !finalized {
		t.Error("block 500 should be finalized when finalized block is 1000")
	}
}

func TestIsBlockFinalized_False(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x3e8", "")) // block 1000
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	finalized, err := fc.IsBlockFinalized(context.Background(), "1", 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finalized {
		t.Error("block 2000 should NOT be finalized when finalized block is 1000")
	}
}

// --- Context cancellation ---

func TestGetFinalizedBlockNumber_ContextCancelled(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Slow response
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x64", ""))
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := fc.GetFinalizedBlockNumber(ctx, "1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- Non-200 HTTP response ---

func TestGetFinalizedBlockNumber_Non200Response(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	_, err := fc.GetFinalizedBlockNumber(context.Background(), "1")
	if err == nil {
		t.Fatal("expected error for HTTP 500 response")
	}
}

// --- IsBlockFinalizedWithStatus ---

func TestIsBlockFinalizedWithStatus_Degraded(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// First call: "finalized" tag fails, triggers fallback
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"not found"}}`)
		} else {
			// Second call: "latest" succeeds
			fmt.Fprint(w, rpcResponse("0x3e8", "")) // block 1000
		}
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	result, err := fc.IsBlockFinalizedWithStatus(context.Background(), "1", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsFinalized {
		t.Error("block 500 should be finalized when finalized block is 1000")
	}
	if !result.Degraded {
		t.Error("result should be marked as degraded when finality came from 'latest' fallback")
	}
}

func TestIsBlockFinalizedWithStatus_NotDegraded(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, rpcResponse("0x3e8", "")) // block 1000
	}))
	defer server.Close()

	fc := NewRPCFinalityChecker(&mockLogger{})
	fc.RegisterChain("1", server.URL)

	result, err := fc.IsBlockFinalizedWithStatus(context.Background(), "1", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsFinalized {
		t.Error("block 500 should be finalized when finalized block is 1000")
	}
	if result.Degraded {
		t.Error("result should NOT be marked as degraded when finality tag succeeded")
	}
}
