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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// StorageReader reads contract storage slots via eth_getStorageAt RPC.
// It also provides helpers for common patterns like proxy implementation
// slots (EIP-1967) and ERC-20 balance lookups.
type StorageReader struct {
	client *http.Client
	rpcURL string
}

// NewStorageReader creates a StorageReader that targets the given RPC URL.
// If client is nil, http.DefaultClient is used.
func NewStorageReader(rpcURL string, client *http.Client) *StorageReader {
	if client == nil {
		client = http.DefaultClient
	}
	return &StorageReader{client: client, rpcURL: rpcURL}
}

// EIP-1967 proxy storage slots (keccak256 of well-known strings).
var (
	// EIP1967ImplementationSlot is the storage slot for proxy implementation address.
	// keccak256("eip1967.proxy.implementation") - 1
	EIP1967ImplementationSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	// EIP1967AdminSlot is the storage slot for proxy admin address.
	// keccak256("eip1967.proxy.admin") - 1
	EIP1967AdminSlot = common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103")
	// EIP1967BeaconSlot is the storage slot for proxy beacon address.
	// keccak256("eip1967.proxy.beacon") - 1
	EIP1967BeaconSlot = common.HexToHash("0xa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d50")
)

// ReadStorageSlot reads a single storage slot from a contract via eth_getStorageAt.
func (r *StorageReader) ReadStorageSlot(ctx context.Context, address common.Address, slot common.Hash, blockNumber string) (common.Hash, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getStorageAt",
		"params":  []string{address.Hex(), slot.Hex(), blockNumber},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return common.Hash{}, fmt.Errorf("marshal eth_getStorageAt request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.rpcURL, bytes.NewReader(body))
	if err != nil {
		return common.Hash{}, fmt.Errorf("create eth_getStorageAt request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return common.Hash{}, fmt.Errorf("eth_getStorageAt RPC call: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // defer close

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return common.Hash{}, fmt.Errorf("read eth_getStorageAt response: %w", err)
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
		return common.Hash{}, fmt.Errorf("unmarshal eth_getStorageAt response: %w", err)
	}
	if result.Error != nil {
		return common.Hash{}, fmt.Errorf("eth_getStorageAt RPC error %d: %s", result.Error.Code, result.Error.Message)
	}

	return parseHash(result.Result)
}

// ReadProxyImplementation reads the EIP-1967 implementation address from a proxy contract.
func (r *StorageReader) ReadProxyImplementation(ctx context.Context, proxyAddress common.Address, blockNumber string) (common.Address, error) {
	slotValue, err := r.ReadStorageSlot(ctx, proxyAddress, EIP1967ImplementationSlot, blockNumber)
	if err != nil {
		return common.Address{}, fmt.Errorf("read proxy implementation slot: %w", err)
	}
	// The slot contains the implementation address in the last 20 bytes
	return common.BytesToAddress(slotValue[12:]), nil
}

// ReadERC20BalanceOf reads the balanceOf mapping for an ERC-20 token.
// The mapping key is keccak256(abi.encode(account, slot)) where slot is typically 0
// for the first mapping in ERC-20 (balances).
func (r *StorageReader) ReadERC20BalanceOf(ctx context.Context, tokenAddr, account common.Address, balanceSlotIndex int, blockNumber string) (*big.Int, error) {
	// Compute mapping key: keccak256(padLeft32(account) ++ padLeft32(slotIndex))
	key := computeMappingKey(account, big.NewInt(int64(balanceSlotIndex)))

	slotValue, err := r.ReadStorageSlot(ctx, tokenAddr, key, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("read ERC-20 balance slot: %w", err)
	}

	return new(big.Int).SetBytes(slotValue[:]), nil
}

// computeMappingKey computes the storage key for a Solidity mapping:
// keccak256(padLeft32(key) ++ padLeft32(slot))
func computeMappingKey(key common.Address, slot *big.Int) common.Hash {
	var keyBytes [32]byte
	copy(keyBytes[12:], key[:]) // left-pad address to 32 bytes

	var slotBytes [32]byte
	copy(slotBytes[32-len(slot.Bytes()):], slot.Bytes()) // left-pad to 32 bytes

	combined := append(keyBytes[:], slotBytes[:]...)
	return common.BytesToHash(crypto.Keccak256(combined))
}

// parseHash parses a hex string like "0x0000...1234" into common.Hash.
func parseHash(hexStr string) (common.Hash, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	if len(hexStr) > 64 {
		hexStr = hexStr[len(hexStr)-64:] // take last 32 bytes
	} else if len(hexStr) < 64 {
		hexStr = strings.Repeat("0", 64-len(hexStr)) + hexStr
	}

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid hex in storage slot value: %w", err)
	}

	var hash common.Hash
	copy(hash[32-len(b):], b)
	return hash, nil
}
