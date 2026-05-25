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

func TestAggregate3_EmptyCalls(t *testing.T) {
	t.Parallel()
	client := NewMulticall3Client("http://localhost:8545", nil)
	results, err := client.Aggregate3(context.Background(), nil)
	if err != nil {
		t.Fatalf("Aggregate3(nil) error: %v", err)
	}
	if results != nil {
		t.Errorf("Aggregate3(nil) = %v, want nil", results)
	}
}

func TestAggregate3_SingleCall(t *testing.T) {
	t.Parallel()
	// Simulate: Aggregate3 returns [(true, 0x000...01)]
	// ABI encode: offset(32) + length(1) + success(1) + returnDataOffset + returnData
	returnHex := encodeTestAggregate3Result([]testCallResult{
		{success: true, returnData: []byte{0x01}},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x` + returnHex + `"}`))
	}))
	defer server.Close()

	client := NewMulticall3Client(server.URL, server.Client())

	calls := []Multicall3Call{
		{
			Target:       common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			AllowFailure: true,
			CallData:     erc20DecimalsSelector,
		},
	}

	results, err := client.Aggregate3(context.Background(), calls)
	if err != nil {
		t.Fatalf("Aggregate3() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Aggregate3() returned %d results, want 1", len(results))
	}
	if !results[0].Success {
		t.Error("result[0].Success = false, want true")
	}
}

func TestAggregate3_RPCError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"contract error"}}`))
	}))
	defer server.Close()

	client := NewMulticall3Client(server.URL, server.Client())
	calls := []Multicall3Call{
		{Target: common.HexToAddress("0x01"), AllowFailure: false, CallData: []byte{0x00}},
	}

	_, err := client.Aggregate3(context.Background(), calls)
	if err == nil {
		t.Fatal("expected error for RPC error response")
	}
	if !strings.Contains(err.Error(), "RPC error") {
		t.Errorf("error should mention RPC error, got: %v", err)
	}
}

func TestBatchERC20Metadata(t *testing.T) {
	t.Parallel()
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48") // USDC
	token2 := common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F") // DAI

	// For 2 tokens, we need 6 results (name, symbol, decimals each)
	// Simulate: token1 name="USD Coin", symbol="USDC", decimals=6
	//          token2 name="Dai Stablecoin", symbol="DAI", decimals=18
	returnHex := encodeTestAggregate3Result([]testCallResult{
		{success: true, returnData: encodeTestString("USD Coin")},
		{success: true, returnData: encodeTestString("USDC")},
		{success: true, returnData: encodeTestUint8(6)},
		{success: true, returnData: encodeTestString("Dai Stablecoin")},
		{success: true, returnData: encodeTestString("DAI")},
		{success: true, returnData: encodeTestUint8(18)},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x` + returnHex + `"}`))
	}))
	defer server.Close()

	client := NewMulticall3Client(server.URL, server.Client())

	metadata, err := client.BatchERC20Metadata(context.Background(), []common.Address{token1, token2})
	if err != nil {
		t.Fatalf("BatchERC20Metadata() error: %v", err)
	}

	if len(metadata) != 2 {
		t.Fatalf("BatchERC20Metadata() returned %d entries, want 2", len(metadata))
	}

	meta1 := metadata[token1]
	if meta1 == nil {
		t.Fatal("token1 metadata is nil")
	}
	if meta1.Symbol != "USDC" {
		t.Errorf("token1 symbol = %q, want %q", meta1.Symbol, "USDC")
	}
	if meta1.Decimals != 6 {
		t.Errorf("token1 decimals = %d, want 6", meta1.Decimals)
	}

	meta2 := metadata[token2]
	if meta2 == nil {
		t.Fatal("token2 metadata is nil")
	}
	if meta2.Symbol != "DAI" {
		t.Errorf("token2 symbol = %q, want %q", meta2.Symbol, "DAI")
	}
	if meta2.Decimals != 18 {
		t.Errorf("token2 decimals = %d, want 18", meta2.Decimals)
	}
}

func TestBatchERC20Metadata_CacheHit(t *testing.T) {
	t.Parallel()
	token := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		callCount++
		returnHex := encodeTestAggregate3Result([]testCallResult{
			{success: true, returnData: encodeTestString("USD Coin")},
			{success: true, returnData: encodeTestString("USDC")},
			{success: true, returnData: encodeTestUint8(6)},
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x` + returnHex + `"}`))
	}))
	defer server.Close()

	client := NewMulticall3Client(server.URL, server.Client())

	// First call: should hit RPC
	_, err := client.BatchERC20Metadata(context.Background(), []common.Address{token})
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("first call: RPC call count = %d, want 1", callCount)
	}

	// Second call: should hit cache
	_, err = client.BatchERC20Metadata(context.Background(), []common.Address{token})
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("second call should use cache, RPC call count = %d, want 1", callCount)
	}
}

func TestBatchERC20Metadata_EmptyInput(t *testing.T) {
	t.Parallel()
	client := NewMulticall3Client("http://localhost:8545", nil)
	result, err := client.BatchERC20Metadata(context.Background(), nil)
	if err != nil {
		t.Fatalf("BatchERC20Metadata(nil) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchERC20Metadata(nil) = %d entries, want 0", len(result))
	}
}

func TestEncodeAggregate3(t *testing.T) {
	t.Parallel()
	calls := []Multicall3Call{
		{
			Target:       common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			AllowFailure: true,
			CallData:     []byte{0x06, 0xfd, 0xde, 0x03},
		},
	}

	encoded := encodeAggregate3(calls)
	if len(encoded) < 4 {
		t.Fatal("encoded data too short")
	}

	// Check Aggregate3 selector
	if !equalBytes(encoded[:4], []byte{0x82, 0xad, 0x56, 0xcb}) {
		t.Errorf("selector = %x, want 82ad56cb", encoded[:4])
	}
}

func TestDecodeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"short", encodeTestString("USDC"), "USDC"},
		{"empty", encodeTestString(""), ""},
		{"longer", encodeTestString("USD Coin"), "USD Coin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeString(tt.data)
			if got != tt.want {
				t.Errorf("decodeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeUint8(t *testing.T) {
	t.Parallel()
	data := encodeTestUint8(18)
	got := decodeUint8(data)
	if got != 18 {
		t.Errorf("decodeUint8() = %d, want 18", got)
	}
}

func TestDecodeUint8TooShort(t *testing.T) {
	t.Parallel()
	got := decodeUint8([]byte{0x01, 0x02, 0x03})
	if got != 0 {
		t.Errorf("decodeUint8(short) = %d, want 0", got)
	}
}

func TestNewMulticall3Client_DefaultClient(t *testing.T) {
	t.Parallel()
	client := NewMulticall3Client("http://localhost:8545", nil)
	if client.client == nil {
		t.Error("NewMulticall3Client with nil client should use http.DefaultClient")
	}
}

// --- Test helpers for ABI encoding ---

type testCallResult struct {
	success    bool
	returnData []byte
}

// encodeTestAggregate3Result builds a hex string representing an Aggregate3 return value.
// ABI layout: offset_to_array(32) + length(32) + [for each tuple: success(32) + returnDataOffset(32)] + dynamic data
// returnDataOffset is relative to the start of each tuple.
func encodeTestAggregate3Result(results []testCallResult) string {
	n := len(results)

	// Each tuple's static part is 2 words (64 bytes): success + returnDataOffset
	// Total static: offset(32) + length(32) + n*64
	staticSize := 32 + 32 + n*64

	// Compute per-tuple returnData offsets (relative to start of each tuple)
	// All dynamic data is appended after the static part.
	var dynamicParts []byte
	var returnDataOffsets []int
	currentDynamicPos := staticSize // absolute position of next dynamic chunk

	tupleStart := 64 // first tuple starts at offset(32) + length(32)
	for i, r := range results {
		// Offset relative to start of this tuple
		relOffset := currentDynamicPos - (tupleStart + i*64)
		returnDataOffsets = append(returnDataOffsets, relOffset)

		paddedLen := ((len(r.returnData) + 31) / 32) * 32
		currentDynamicPos += 32 + paddedLen // length(32) + padded data

		// Build dynamic part: length + data + padding
		var lenBytes [32]byte
		bigLen := new(big.Int).SetUint64(uint64(len(r.returnData)))
		bigLen.FillBytes(lenBytes[:])
		dynamicParts = append(dynamicParts, lenBytes[:]...)
		dynamicParts = append(dynamicParts, r.returnData...)
		if pad := paddedLen - len(r.returnData); pad > 0 {
			dynamicParts = append(dynamicParts, make([]byte, pad)...)
		}
	}

	var buf []byte

	// Offset to array (0x20)
	var offsetBytes [32]byte
	new(big.Int).SetUint64(32).FillBytes(offsetBytes[:])
	buf = append(buf, offsetBytes[:]...)

	// Array length
	var lenBytes [32]byte
	new(big.Int).SetUint64(uint64(n)).FillBytes(lenBytes[:])
	buf = append(buf, lenBytes[:]...)

	// Static part per tuple: success + returnDataOffset
	for i, r := range results {
		var successBytes [32]byte
		if r.success {
			successBytes[31] = 1
		}
		buf = append(buf, successBytes[:]...)

		var rdOffset [32]byte
		new(big.Int).SetUint64(uint64(returnDataOffsets[i])).FillBytes(rdOffset[:])
		buf = append(buf, rdOffset[:]...)
	}

	buf = append(buf, dynamicParts...)

	return hex.EncodeToString(buf)
}

// encodeTestString ABI-encodes a string for test purposes.
func encodeTestString(s string) []byte {
	strBytes := []byte(s)
	var buf []byte

	// Offset (0x20)
	var offset [32]byte
	new(big.Int).SetUint64(32).FillBytes(offset[:])
	buf = append(buf, offset[:]...)

	// Length
	var lenBytes [32]byte
	new(big.Int).SetUint64(uint64(len(strBytes))).FillBytes(lenBytes[:])
	buf = append(buf, lenBytes[:]...)

	// Data + padding
	buf = append(buf, strBytes...)
	if pad := ((len(strBytes) + 31) / 32) * 32; pad > len(strBytes) {
		buf = append(buf, make([]byte, pad-len(strBytes))...)
	}

	return buf
}

// encodeTestUint8 ABI-encodes a uint8 for test purposes.
func encodeTestUint8(v uint8) []byte {
	var b [32]byte
	b[31] = v
	return b[:]
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
