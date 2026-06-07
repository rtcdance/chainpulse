package blockchain

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	blockchainmodels "github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestNewRPCTxTypeResolver(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)

	// Pre-populate cache
	hash := common.HexToHash("0xabc123")
	r.addToCache(hash, txTypeCacheEntry{txType: blockchainmodels.TxEIP1559, txStatus: blockchainmodels.TxStatusSuccess})

	typ, _, err := r.ResolveTxType(context.Background(), "0xabc123")
	if err != nil {
		t.Fatalf("ResolveTxType() error = %v", err)
	}
	if typ != blockchainmodels.TxEIP1559 {
		t.Errorf("ResolveTxType() = %d, want %d", typ, blockchainmodels.TxEIP1559)
	}
}

func TestRPCTxTypeResolver_RPCSuccess(t *testing.T) {
	t.Parallel()
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
	if typ != blockchainmodels.TxEIP1559 {
		t.Errorf("ResolveTxType() type = %d, want %d", typ, blockchainmodels.TxEIP1559)
	}
	if status != blockchainmodels.TxStatusSuccess {
		t.Errorf("ResolveTxType() status = %d, want %d", status, blockchainmodels.TxStatusSuccess)
	}

	// Should be cached now
	if r.CacheSize() != 1 {
		t.Errorf("CacheSize() = %d, want 1", r.CacheSize())
	}
}

func TestRPCTxTypeResolver_RPCBlobTx(t *testing.T) {
	t.Parallel()
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
	if typ != blockchainmodels.TxBlob {
		t.Errorf("ResolveTxType() = %d, want %d", typ, blockchainmodels.TxBlob)
	}
}

func TestRPCTxTypeResolver_RPCError(t *testing.T) {
	t.Parallel()
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
	if typ != blockchainmodels.TxLegacy {
		t.Errorf("ResolveTxType() on error = %d, want %d (TxLegacy)", typ, blockchainmodels.TxLegacy)
	}
}

func TestRPCTxTypeResolver_ReceiptNotFound(t *testing.T) {
	t.Parallel()
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
	if typ != blockchainmodels.TxLegacy {
		t.Errorf("ResolveTxType() on null = %d, want %d", typ, blockchainmodels.TxLegacy)
	}
}

func TestRPCTxTypeResolver_ContextCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		<-req.Context().Done()
	}))
	defer server.Close()

	r := NewRPCTxTypeResolver(server.URL, server.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	typ, _, err := r.ResolveTxType(ctx, "0xabc")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if typ != blockchainmodels.TxLegacy {
		t.Errorf("ResolveTxType() on cancelled ctx = %d, want %d", typ, blockchainmodels.TxLegacy)
	}
}

func TestRPCTxTypeResolver_CacheEviction(t *testing.T) {
	t.Parallel()
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)
	r.maxCache = 10 // Small cache for testing

	// Fill cache beyond capacity
	for i := 0; i < 15; i++ {
		hash := common.BigToHash(big.NewInt(int64(i)))
		r.addToCache(hash, txTypeCacheEntry{txType: blockchainmodels.TxEIP1559, txStatus: blockchainmodels.TxStatusSuccess})
	}

	// Cache should not exceed maxCache by much (eviction removes 10%)
	if r.CacheSize() > r.maxCache {
		t.Errorf("CacheSize() = %d, should not exceed %d after eviction", r.CacheSize(), r.maxCache)
	}
}

func TestRPCTxTypeResolver_ClearCache(t *testing.T) {
	t.Parallel()
	r := NewRPCTxTypeResolver("http://localhost:8545", nil)
	r.addToCache(common.HexToHash("0x1"), txTypeCacheEntry{txType: blockchainmodels.TxEIP1559, txStatus: blockchainmodels.TxStatusSuccess})
	r.addToCache(common.HexToHash("0x2"), txTypeCacheEntry{txType: blockchainmodels.TxBlob, txStatus: blockchainmodels.TxStatusSuccess})

	if r.CacheSize() != 2 {
		t.Fatalf("CacheSize() = %d, want 2", r.CacheSize())
	}

	r.ClearCache()

	if r.CacheSize() != 0 {
		t.Errorf("CacheSize() after clear = %d, want 0", r.CacheSize())
	}
}

func TestRPCTxTypeResolver_FailedTransaction(t *testing.T) {
	t.Parallel()
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
	if typ != blockchainmodels.TxEIP1559 {
		t.Errorf("ResolveTxType() type = %d, want %d", typ, blockchainmodels.TxEIP1559)
	}
	if status != blockchainmodels.TxStatusFailed {
		t.Errorf("ResolveTxType() status = %d, want %d (failed)", status, blockchainmodels.TxStatusFailed)
	}
}

func TestParseHexUint8(t *testing.T) {
	t.Parallel()
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
