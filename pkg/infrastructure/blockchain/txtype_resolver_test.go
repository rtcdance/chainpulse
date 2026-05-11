package blockchain

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	core "chainpulse/pkg/core"
)

func TestNewRPCTxTypeResolver(t *testing.T) {
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)
	if r == nil {
		t.Fatal("NewRPCTxTypeResolver returned nil")
	}
	if r.rpcURL != "http://localhost:8545" {
		t.Errorf("rpcURL = %q, want %q", r.rpcURL, "http://localhost:8545")
	}
	if r.maxCache != 10000 {
		t.Errorf("maxCache = %d, want 10000", r.maxCache)
	}
}

func TestRPCTxTypeResolver_CacheHit(t *testing.T) {
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)

	// Pre-populate cache
	hash := common.HexToHash("0xabc123")
	r.addToCache(hash, txTypeCacheEntry{txType: core.TxEIP1559, txStatus: core.TxStatusSuccess})

	typ, _, err := r.ResolveTxType(context.Background(), "0xabc123")
	if err != nil {
		t.Fatalf("ResolveTxType() error = %v", err)
	}
	if typ != core.TxEIP1559 {
		t.Errorf("ResolveTxType() = %d, want %d", typ, core.TxEIP1559)
	}
}

func TestRPCTxTypeResolver_RPCSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"type":"0x2","status":"0x1"}}`))
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	typ, status, err := r.ResolveTxType(context.Background(), "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err != nil {
		t.Fatalf("ResolveTxType() error = %v", err)
	}
	if typ != core.TxEIP1559 {
		t.Errorf("ResolveTxType() type = %d, want %d", typ, core.TxEIP1559)
	}
	if status != core.TxStatusSuccess {
		t.Errorf("ResolveTxType() status = %d, want %d", status, core.TxStatusSuccess)
	}

	// Should be cached now
	if r.CacheSize() != 1 {
		t.Errorf("CacheSize() = %d, want 1", r.CacheSize())
	}
}

func TestRPCTxTypeResolver_RPCBlobTx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"type":"0x3","status":"0x1"}}`))
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	typ, _, err := r.ResolveTxType(context.Background(), "0xabc")
	if err != nil {
		t.Fatalf("ResolveTxType() error = %v", err)
	}
	if typ != core.TxBlob {
		t.Errorf("ResolveTxType() = %d, want %d", typ, core.TxBlob)
	}
}

func TestRPCTxTypeResolver_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"internal error"}}`))
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	typ, _, err := r.ResolveTxType(context.Background(), "0xabc")
	if err == nil {
		t.Fatal("expected error for RPC error response")
	}
	// Should fall back to TxLegacy
	if typ != core.TxLegacy {
		t.Errorf("ResolveTxType() on error = %d, want %d (TxLegacy)", typ, core.TxLegacy)
	}
}

func TestRPCTxTypeResolver_ReceiptNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`))
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	typ, _, err := r.ResolveTxType(context.Background(), "0xabc")
	if err == nil {
		t.Fatal("expected error for null result")
	}
	if typ != core.TxLegacy {
		t.Errorf("ResolveTxType() on null = %d, want %d", typ, core.TxLegacy)
	}
}

func TestRPCTxTypeResolver_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Never respond
		select {}
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	typ, _, err := r.ResolveTxType(ctx, "0xabc")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if typ != core.TxLegacy {
		t.Errorf("ResolveTxType() on cancelled ctx = %d, want %d", typ, core.TxLegacy)
	}
}

func TestRPCTxTypeResolver_CacheEviction(t *testing.T) {
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)
	r.maxCache = 10 // Small cache for testing

	// Fill cache beyond capacity
	for i := 0; i < 15; i++ {
		hash := common.BigToHash(big.NewInt(int64(i)))
		r.addToCache(hash, txTypeCacheEntry{txType: core.TxEIP1559, txStatus: core.TxStatusSuccess})
	}

	// Cache should not exceed maxCache by much (eviction removes 10%)
	if r.CacheSize() > r.maxCache {
		t.Errorf("CacheSize() = %d, should not exceed %d after eviction", r.CacheSize(), r.maxCache)
	}
}

func TestRPCTxTypeResolver_ClearCache(t *testing.T) {
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)
	r.addToCache(common.HexToHash("0x1"), txTypeCacheEntry{txType: core.TxEIP1559, txStatus: core.TxStatusSuccess})
	r.addToCache(common.HexToHash("0x2"), txTypeCacheEntry{txType: core.TxBlob, txStatus: core.TxStatusSuccess})

	if r.CacheSize() != 2 {
		t.Fatalf("CacheSize() = %d, want 2", r.CacheSize())
	}

	r.ClearCache()

	if r.CacheSize() != 0 {
		t.Errorf("CacheSize() after clear = %d, want 0", r.CacheSize())
	}
}

func TestRPCTxTypeResolver_FailedTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"type":"0x2","status":"0x0"}}`))
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	typ, status, err := r.ResolveTxType(context.Background(), "0xdeadbeef")
	if err != nil {
		t.Fatalf("ResolveTxType() error = %v", err)
	}
	if typ != core.TxEIP1559 {
		t.Errorf("ResolveTxType() type = %d, want %d", typ, core.TxEIP1559)
	}
	if status != core.TxStatusFailed {
		t.Errorf("ResolveTxType() status = %d, want %d (failed)", status, core.TxStatusFailed)
	}
}

func TestParseHexUint8(t *testing.T) {
	tests := []struct {
		input   string
		want    uint8
		wantErr bool
	}{
		{"0x0", 0, false},
		{"0x1", 1, false},
		{"0x2", 2, false},
		{"0x3", 3, false},
		{"0xff", 255, false},
		{"0", 0, false},  // decimal
		{"2", 2, false},  // decimal
		{"0x", 0, true},  // incomplete hex
		{"abc", 0, true}, // not a number
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseHexUint8(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHexUint8(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseHexUint8(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
