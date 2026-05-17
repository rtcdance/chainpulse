package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/ethereum/go-ethereum/common"

	core "chainpulse/pkg/core"
)

// txTypeCacheEntry stores both the transaction type and receipt status.
type txTypeCacheEntry struct {
	txType   uint8
	txStatus uint8
}

// RPCTxTypeResolver resolves EIP-2718 transaction types via eth_getTransactionReceipt.
// It maintains a bounded cache to avoid repeated RPC calls for the same transaction.
type RPCTxTypeResolver struct {
	client   *http.Client
	rpcURL   string
	cache    map[common.Hash]txTypeCacheEntry
	cacheMu  sync.RWMutex
	maxCache int
}

// NewRPCTxTypeResolver creates a new resolver with the given RPC URL and HTTP client.
// If client is nil, it uses http.DefaultClient.
func NewRPCTxTypeResolver(rpcURL string, client *http.Client) *RPCTxTypeResolver {
	if client == nil {
		client = http.DefaultClient
	}
	return &RPCTxTypeResolver{
		client:   client,
		rpcURL:   rpcURL,
		cache:    make(map[common.Hash]txTypeCacheEntry, 1024),
		maxCache: 10000,
	}
}

// ResolveTxType resolves the EIP-2718 transaction type and receipt status for the given transaction hash.
// It first checks the local cache, then falls back to an eth_getTransactionReceipt RPC call.
// On error or unavailable, it returns TxLegacy (0) as a safe default with status=1 (success).
func (r *RPCTxTypeResolver) ResolveTxType(ctx context.Context, txHash string) (uint8, uint8, error) {
	hash := common.HexToHash(txHash)

	// Check cache first
	if entry, ok := r.getFromCache(hash); ok {
		return entry.txType, entry.txStatus, nil
	}

	// Fall back to RPC
	txType, txStatus, err := r.resolveViaRPC(ctx, txHash)
	if err != nil {
		// Non-fatal: return legacy as default, assume success
		return core.TxLegacy, core.TxStatusSuccess, fmt.Errorf("failed to resolve tx type via RPC for %s: %w", txHash, err)
	}

	// Cache the result
	r.addToCache(hash, txTypeCacheEntry{txType: txType, txStatus: txStatus})

	return txType, txStatus, nil
}

// getFromCache retrieves a cached transaction type and status.
func (r *RPCTxTypeResolver) getFromCache(hash common.Hash) (txTypeCacheEntry, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	entry, ok := r.cache[hash]
	return entry, ok
}

// addToCache adds a transaction type and status to the cache with bounded eviction.
func (r *RPCTxTypeResolver) addToCache(hash common.Hash, entry txTypeCacheEntry) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// Evict if at capacity — remove ~10% of entries to amortize cost
	if len(r.cache) >= r.maxCache {
		toRemove := r.maxCache / 10
		count := 0
		for k := range r.cache {
			delete(r.cache, k)
			count++
			if count >= toRemove {
				break
			}
		}
	}

	r.cache[hash] = entry
}

// resolveViaRPC fetches the transaction type and receipt status via eth_getTransactionReceipt.
func (r *RPCTxTypeResolver) resolveViaRPC(ctx context.Context, txHash string) (uint8, uint8, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getTransactionReceipt",
		"params":  []string{txHash},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, 0, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.rpcURL, bytes.NewReader(body))
	if err != nil {
		return 0, 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return 0, 0, fmt.Errorf("RPC request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // defer close

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("RPC returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  *struct {
			Type   string `json:"type"`
			Status string `json:"status"` // EIP-658: "0x0" = failed, "0x1" = success
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, 0, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != nil {
		return 0, 0, fmt.Errorf("RPC error %d: %s", result.Error.Code, result.Error.Message)
	}

	if result.Result == nil {
		return 0, 0, fmt.Errorf("transaction receipt not found for %s", txHash)
	}

	// Parse the type hex string (e.g., "0x0", "0x2", "0x3")
	txType, err := parseHexUint8(result.Result.Type)
	if err != nil {
		return 0, 0, fmt.Errorf("parse type field %q: %w", result.Result.Type, err)
	}

	// Parse the status hex string (e.g., "0x0" = failed, "0x1" = success per EIP-658)
	// Pre-EIP-658 receipts don't have status — treat missing as success.
	txStatus := core.TxStatusSuccess
	if result.Result.Status != "" {
		if s, err := parseHexUint8(result.Result.Status); err == nil {
			if s == 0 {
				txStatus = core.TxStatusFailed
			}
		}
	}

	return txType, txStatus, nil
}

// parseHexUint8 parses a hex string like "0x0", "0x1", "0x2", "0x3" into a uint8.
func parseHexUint8(hexStr string) (uint8, error) {
	if len(hexStr) < 2 || hexStr[:2] != "0x" {
		// Try decimal
		var v uint8
		if _, err := fmt.Sscanf(hexStr, "%d", &v); err == nil {
			return v, nil
		}
		return 0, fmt.Errorf("invalid hex string: %s", hexStr)
	}

	var v uint8
	if _, err := fmt.Sscanf(hexStr, "0x%x", &v); err != nil {
		return 0, fmt.Errorf("parse hex %s: %w", hexStr, err)
	}
	return v, nil
}

// CacheSize returns the current number of cached transaction types (for monitoring).
func (r *RPCTxTypeResolver) CacheSize() int {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return len(r.cache)
}

// ClearCache removes all cached entries.
func (r *RPCTxTypeResolver) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache = make(map[common.Hash]txTypeCacheEntry, 1024)
}

// Ensure RPCTxTypeResolver implements TxTypeResolver at compile time.
var _ core.TxTypeResolver = (*RPCTxTypeResolver)(nil)
