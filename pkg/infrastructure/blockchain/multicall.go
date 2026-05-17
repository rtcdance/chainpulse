package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Multicall3Call represents a single call in a Multicall3 Aggregate3 batch.
type Multicall3Call struct {
	Target       common.Address `json:"target"`
	AllowFailure bool           `json:"allowFailure"`
	CallData     []byte         `json:"callData"`
}

// Multicall3Result represents the result of a single call from Aggregate3.
type Multicall3Result struct {
	Success    bool   `json:"success"`
	ReturnData []byte `json:"returnData"`
}

// ERC20Metadata holds the name, symbol, and decimals of an ERC-20 token.
type ERC20Metadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

// Multicall3Address is the canonical Multicall3 contract address deployed across many chains.
var Multicall3Address = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// ERC-20 function selectors (first 4 bytes of keccak256 hash).
var (
	erc20NameSelector     = []byte{0x06, 0xfd, 0xde, 0x03} // name()
	erc20SymbolSelector   = []byte{0x95, 0xd8, 0x9b, 0x41} // symbol()
	erc20DecimalsSelector = []byte{0x31, 0x3c, 0xe5, 0x67} // decimals()
)

// Multicall3Client batches multiple eth_call requests via the Multicall3 contract.
type Multicall3Client struct {
	client   *http.Client
	rpcURL   string
	cache    map[common.Address]*ERC20Metadata
	cacheMu  sync.RWMutex
	maxCache int
}

// NewMulticall3Client creates a new Multicall3 batch client.
// If client is nil, http.DefaultClient is used.
func NewMulticall3Client(rpcURL string, client *http.Client) *Multicall3Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Multicall3Client{
		client:   client,
		rpcURL:   rpcURL,
		cache:    make(map[common.Address]*ERC20Metadata),
		maxCache: 10000,
	}
}

// Aggregate3 executes a batch of calls via the Multicall3 Aggregate3 function.
// It encodes the call data, sends a single eth_call to the Multicall3 contract,
// and decodes the results.
func (m *Multicall3Client) Aggregate3(ctx context.Context, calls []Multicall3Call) ([]Multicall3Result, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	// Encode Aggregate3(tuple(address,bool,bytes)[])
	// Function selector: 0x82ad56cb
	callData := encodeAggregate3(calls)

	// Make eth_call to Multicall3 contract
	returnData, err := m.ethCall(ctx, Multicall3Address, callData, "latest")
	if err != nil {
		return nil, fmt.Errorf("multicall3 Aggregate3 eth_call: %w", err)
	}

	// Decode results
	results, err := decodeAggregate3Result(returnData)
	if err != nil {
		return nil, fmt.Errorf("decode Aggregate3 result: %w", err)
	}

	if len(results) != len(calls) {
		return nil, fmt.Errorf("result count %d != call count %d", len(results), len(calls))
	}

	return results, nil
}

// BatchERC20Metadata fetches name, symbol, and decimals for multiple ERC-20 tokens
// in a single Multicall3 batch.
func (m *Multicall3Client) BatchERC20Metadata(ctx context.Context, tokenAddresses []common.Address) (map[common.Address]*ERC20Metadata, error) {
	if len(tokenAddresses) == 0 {
		return map[common.Address]*ERC20Metadata{}, nil
	}

	// Check cache first
	result := make(map[common.Address]*ERC20Metadata, len(tokenAddresses))
	var uncached []common.Address

	m.cacheMu.RLock()
	for _, addr := range tokenAddresses {
		if meta, ok := m.cache[addr]; ok {
			result[addr] = meta
		} else {
			uncached = append(uncached, addr)
		}
	}
	m.cacheMu.RUnlock()

	if len(uncached) == 0 {
		return result, nil
	}

	// Build 3 calls per token: name(), symbol(), decimals()
	var calls []Multicall3Call
	for _, addr := range uncached {
		calls = append(calls,
			Multicall3Call{Target: addr, AllowFailure: true, CallData: erc20NameSelector},
			Multicall3Call{Target: addr, AllowFailure: true, CallData: erc20SymbolSelector},
			Multicall3Call{Target: addr, AllowFailure: true, CallData: erc20DecimalsSelector},
		)
	}

	multiResults, err := m.Aggregate3(ctx, calls)
	if err != nil {
		return nil, fmt.Errorf("batch ERC-20 metadata: %w", err)
	}

	// Parse results: 3 results per token
	for i, addr := range uncached {
		base := i * 3
		if base+2 >= len(multiResults) {
			continue
		}

		meta := &ERC20Metadata{}
		if multiResults[base].Success {
			meta.Name = decodeString(multiResults[base].ReturnData)
		}
		if multiResults[base+1].Success {
			meta.Symbol = decodeString(multiResults[base+1].ReturnData)
		}
		if multiResults[base+2].Success {
			meta.Decimals = decodeUint8(multiResults[base+2].ReturnData)
		}

		result[addr] = meta
	}

	// Cache results
	m.cacheMu.Lock()
	for _, addr := range uncached {
		if meta, ok := result[addr]; ok {
			m.cache[addr] = meta
		}
	}
	// Evict if over capacity
	if len(m.cache) > m.maxCache {
		evictCount := len(m.cache) / 10
		for k := range m.cache {
			if evictCount <= 0 {
				break
			}
			delete(m.cache, k)
			evictCount--
		}
	}
	m.cacheMu.Unlock()

	return result, nil
}

// ethCall makes an eth_call JSON-RPC request.
func (m *Multicall3Client) ethCall(ctx context.Context, to common.Address, callData []byte, blockNumber string) ([]byte, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}

	txObj := map[string]string{
		"to":   to.Hex(),
		"data": "0x" + hex.EncodeToString(callData),
	}

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_call",
		"params":  []any{txObj, blockNumber},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal eth_call request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create eth_call request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("eth_call RPC: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // defer close

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read eth_call response: %w", err)
	}

	var result struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  string `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal eth_call response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("eth_call RPC error %d: %s", result.Error.Code, result.Error.Message)
	}

	// Decode hex result
	hexStr := strings.TrimPrefix(result.Result, "0x")
	if hexStr == "" {
		return nil, nil
	}
	return hex.DecodeString(hexStr)
}

// encodeAggregate3 ABI-encodes the Aggregate3 call.
// selector(4) + offset(32) + length(32) + [for each call: target(32) + allowFailure(32) + calldata_offset(32) + calldata_length(32) + calldata_bytes(padded)]
func encodeAggregate3(calls []Multicall3Call) []byte {
	// Aggregate3 selector
	selector := []byte{0x82, 0xad, 0x56, 0xcb}

	// Dynamic array encoding:
	// offset to array (32) + array length (32) + elements
	// Each element is a tuple (address, bool, bytes)
	// Static part per element: target(32) + allowFailure(32) + offset_to_calldata(32)
	// Then dynamic calldata for each element

	numCalls := len(calls)
	staticSize := 32 + 32 + numCalls*3*32 // offset + length + static parts
	dynamicOffset := uint64(staticSize)

	// Build the encoded data
	var buf []byte
	buf = append(buf, selector...)

	// Offset to the array (always 0x20 for single parameter)
	buf = appendUint64(buf, 32)

	// Array length
	buf = appendUint64(buf, uint64(numCalls))

	// Calculate calldata offsets
	var callDataLens []int
	var callDataBytes [][]byte
	for _, call := range calls {
		callDataLens = append(callDataLens, len(call.CallData))
		callDataBytes = append(callDataBytes, call.CallData)
	}

	// For each call, write: target (32) + allowFailure (32) + calldata_offset (32)
	currentDynamicOffset := dynamicOffset
	var callDataOffsets []uint64
	for i := 0; i < numCalls; i++ {
		// target address (left-padded to 32 bytes)
		var targetBytes [32]byte
		copy(targetBytes[12:], calls[i].Target[:])
		buf = append(buf, targetBytes[:]...)

		// allowFailure (uint256: 0 or 1)
		if calls[i].AllowFailure {
			buf = appendUint64(buf, 1)
		} else {
			buf = appendUint64(buf, 0)
		}

		// calldata offset (relative to start of this tuple's dynamic data)
		callDataOffsets = append(callDataOffsets, currentDynamicOffset)
		paddedLen := ((callDataLens[i] + 31) / 32) * 32
		currentDynamicOffset += uint64(32 + 32 + paddedLen) // length + padded calldata
	}

	// Write calldata offsets
	for i := 0; i < numCalls; i++ {
		buf = appendUint64(buf, callDataOffsets[i])
	}

	// Write dynamic calldata
	for i := 0; i < numCalls; i++ {
		buf = appendUint64(buf, uint64(callDataLens[i]))
		buf = append(buf, callDataBytes[i]...)
		// Pad to 32-byte boundary
		if padLen := ((callDataLens[i] + 31) / 32) * 32; padLen > callDataLens[i] {
			buf = append(buf, make([]byte, padLen-callDataLens[i])...)
		}
	}

	return buf
}

// decodeAggregate3Result decodes the return data from Aggregate3.
// Returns an array of (bool success, bytes returnData) tuples.
// ABI layout per tuple: success(32) + returnDataOffset(32) where offset is
// relative to the start of that tuple.
func decodeAggregate3Result(data []byte) ([]Multicall3Result, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("result too short: %d bytes", len(data))
	}

	// Read offset to array
	arrayOffset := new(big.Int).SetBytes(data[0:32]).Uint64()
	if arrayOffset+32 > uint64(len(data)) {
		return nil, fmt.Errorf("array offset %d out of bounds", arrayOffset)
	}

	// Read array length
	arrayLen := new(big.Int).SetBytes(data[arrayOffset : arrayOffset+32]).Uint64()
	if arrayLen > 1000 {
		return nil, fmt.Errorf("unreasonable array length: %d", arrayLen)
	}

	results := make([]Multicall3Result, 0, arrayLen)
	// Each tuple: success(32) + returnDataOffset(32) = 64 bytes static part
	tupleStart := arrayOffset + 32

	for i := uint64(0); i < arrayLen; i++ {
		base := tupleStart + i*64
		if base+64 > uint64(len(data)) {
			break
		}

		success := data[base+31] == 1

		// returnDataOffset is relative to the start of this tuple
		returnDataOffset := new(big.Int).SetBytes(data[base+32 : base+64]).Uint64()
		rdAbsStart := base + returnDataOffset

		if rdAbsStart+32 > uint64(len(data)) {
			results = append(results, Multicall3Result{Success: success})
			continue
		}

		rdLen := new(big.Int).SetBytes(data[rdAbsStart : rdAbsStart+32]).Uint64()
		rdEnd := rdAbsStart + 32 + rdLen
		if rdEnd > uint64(len(data)) {
			rdEnd = uint64(len(data))
		}

		var returnData []byte
		if rdLen > 0 {
			returnData = make([]byte, rdLen)
			copy(returnData, data[rdAbsStart+32:rdEnd])
		}

		results = append(results, Multicall3Result{
			Success:    success,
			ReturnData: returnData,
		})
	}

	return results, nil
}

// appendUint64 appends a uint64 as a 32-byte big-endian value.
func appendUint64(buf []byte, v uint64) []byte {
	var b [32]byte
	new(big.Int).SetUint64(v).FillBytes(b[:])
	return append(buf, b[:]...)
}

// decodeString decodes an ABI-encoded string from return data.
func decodeString(data []byte) string {
	if len(data) < 64 {
		return ""
	}
	// Skip offset (first 32 bytes), read length
	strLen := new(big.Int).SetBytes(data[32:64]).Uint64()
	if strLen == 0 || 64+strLen > uint64(len(data)) {
		return ""
	}
	return string(data[64 : 64+strLen])
}

// decodeUint8 decodes a uint8 from the last byte of ABI-encoded return data.
func decodeUint8(data []byte) uint8 {
	if len(data) < 32 {
		return 0
	}
	return data[31]
}
