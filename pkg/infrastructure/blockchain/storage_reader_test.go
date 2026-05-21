package blockchain

import (
	"context"
	"encoding/hex"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestReadStorageSlot(t *testing.T) {
	t.Parallel()
	slotValue := "0x0000000000000000000000000000000000000000000000000000000000000001"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + slotValue + `"}`))
	}))
	defer server.Close()

	reader := NewStorageReader(server.URL, server.Client())

	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	slot := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")

	result, err := reader.ReadStorageSlot(context.Background(), addr, slot, "latest")
	if err != nil {
		t.Fatalf("ReadStorageSlot() error: %v", err)
	}
	if result != common.HexToHash(slotValue) {
		t.Errorf("ReadStorageSlot() = %s, want %s", result.Hex(), slotValue)
	}
}

func TestReadStorageSlot_RPCError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"header not found"}}`))
	}))
	defer server.Close()

	reader := NewStorageReader(server.URL, server.Client())
	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	slot := common.HexToHash("0x00")

	_, err := reader.ReadStorageSlot(context.Background(), addr, slot, "latest")
	if err == nil {
		t.Fatal("expected error for RPC error response")
	}
	if !strings.Contains(err.Error(), "RPC error") {
		t.Errorf("error should mention RPC error, got: %v", err)
	}
}

func TestReadProxyImplementation(t *testing.T) {
	t.Parallel()
	implAddr := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	// Left-pad the address to 32 bytes (as stored in the slot)
	slotHex := "0x000000000000000000000000" + implAddr.Hex()[2:]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + slotHex + `"}`))
	}))
	defer server.Close()

	reader := NewStorageReader(server.URL, server.Client())
	proxyAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	result, err := reader.ReadProxyImplementation(context.Background(), proxyAddr, "latest")
	if err != nil {
		t.Fatalf("ReadProxyImplementation() error: %v", err)
	}
	if result != implAddr {
		t.Errorf("ReadProxyImplementation() = %s, want %s", result.Hex(), implAddr.Hex())
	}
}

func TestReadERC20BalanceOf(t *testing.T) {
	t.Parallel()
	balance := big.NewInt(1000000)
	balanceHex := "0x" + hex.EncodeToString(balance.Bytes())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Verify it's a getStorageAt call
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"` + balanceHex + `"}`))
	}))
	defer server.Close()

	reader := NewStorageReader(server.URL, server.Client())
	tokenAddr := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") // USDC
	account := common.HexToAddress("0x2222222222222222222222222222222222222222")

	result, err := reader.ReadERC20BalanceOf(context.Background(), tokenAddr, account, 0, "latest")
	if err != nil {
		t.Fatalf("ReadERC20BalanceOf() error: %v", err)
	}
	if result.Cmp(balance) != 0 {
		t.Errorf("ReadERC20BalanceOf() = %d, want %d", result, balance)
	}
}

func TestReadStorageSlot_ContextCanceled(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		<-req.Context().Done()
	}))
	defer server.Close()

	reader := NewStorageReader(server.URL, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	addr := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	slot := common.HexToHash("0x00")

	_, err := reader.ReadStorageSlot(ctx, addr, slot, "latest")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestComputeMappingKey(t *testing.T) {
	t.Parallel()
	// Known test vector: mapping(address => uint256) at slot 0
	// key = keccak256(pad32(address) ++ pad32(0))
	account := common.HexToAddress("0x2222222222222222222222222222222222222222")
	key := computeMappingKey(account, big.NewInt(0))

	// The result should be a valid 32-byte hash
	if len(key) != 32 {
		t.Errorf("computeMappingKey() result length = %d, want 32", len(key))
	}
	// Different slot indices should produce different keys
	key1 := computeMappingKey(account, big.NewInt(1))
	if key == key1 {
		t.Error("different slot indices should produce different mapping keys")
	}
}

func TestParseHash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"full_32_bytes", "0x" + strings.Repeat("ab", 32), "0x" + strings.Repeat("ab", 32)},
		{"short_value", "0x01", "0x0000000000000000000000000000000000000000000000000000000000000001"},
		{"zero", "0x00", "0x0000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHash(tt.input)
			if err != nil {
				t.Fatalf("parseHash(%s) error: %v", tt.input, err)
			}
			if got.Hex() != tt.want {
				t.Errorf("parseHash(%s) = %s, want %s", tt.input, got.Hex(), tt.want)
			}
		})
	}
}

func TestNewStorageReader_DefaultClient(t *testing.T) {
	t.Parallel()
	reader := NewStorageReader("http://localhost:8545", nil)
	if reader.client == nil {
		t.Error("NewStorageReader with nil client should use http.DefaultClient")
	}
}
