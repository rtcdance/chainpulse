package finality

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// FinalityResult contains the result of a finality check, including whether
// the result is based on a degraded (unreliable) data source.
type FinalityResult struct {
	IsFinalized bool // true if the block is at or below the finalized block
	Degraded    bool // true if the finalized block number came from a "latest" fallback
}

// FinalityChecker determines whether a block is finalized on a given chain.
// PoS Ethereum supports eth_getBlockByNumber("finalized"), while L2 rollups
// require tracking L1 batch finality.
type FinalityChecker interface {
	// GetFinalizedBlockNumber returns the latest finalized block number for the chain.
	// Returns an error if the chain does not support the finalized tag.
	GetFinalizedBlockNumber(ctx context.Context, chainID string) (uint64, error)

	// IsBlockFinalized returns true if the given block number is at or below
	// the finalized block for the chain.
	IsBlockFinalized(ctx context.Context, chainID string, blockNumber uint64) (bool, error)

	// IsBlockFinalizedWithStatus returns a FinalityResult that includes both the
	// finalization status and whether the result is from a degraded source.
	// When Degraded is true, the finalization guarantee is unreliable.
	IsBlockFinalizedWithStatus(ctx context.Context, chainID string, blockNumber uint64) (*FinalityResult, error)
}

// rpcFinalityChecker implements FinalityChecker using Ethereum JSON-RPC.
type rpcFinalityChecker struct {
	clients          map[string]*http.Client // chainID -> HTTP client
	nodeURLs         map[string]string       // chainID -> RPC URL
	cache            map[string]cachedFinality
	cacheMu          sync.RWMutex
	cacheTTL         time.Duration
	degradedCacheTTL time.Duration // shorter TTL for degraded (unreliable) values
	logger           core.Logger
}

type cachedFinality struct {
	blockNumber uint64
	cachedAt    time.Time
	degraded    bool // true if value came from "latest" fallback
}

// NewRPCFinalityChecker creates a FinalityChecker that queries RPC nodes.
func NewRPCFinalityChecker(logger core.Logger) *rpcFinalityChecker {
	return &rpcFinalityChecker{
		clients:          make(map[string]*http.Client),
		nodeURLs:         make(map[string]string),
		cache:            make(map[string]cachedFinality),
		cacheTTL:         12 * time.Second, // Slot time is 12s; finality advances every epoch (32 slots ≈ 6.4min). Cache refreshes per slot.
		degradedCacheTTL: 2 * time.Second,  // Degraded values are unreliable; refresh more aggressively.
		logger:           logger,
	}
}

// RegisterChain registers an RPC endpoint for a chain.
func (f *rpcFinalityChecker) RegisterChain(chainID, nodeURL string) {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()
	f.clients[chainID] = &http.Client{Timeout: 10 * time.Second}
	f.nodeURLs[chainID] = nodeURL
}

// GetFinalizedBlockNumber returns the latest finalized block number.
// For L1 chains: uses eth_getBlockByNumber("finalized").
// For L2 chains: uses eth_getBlockByNumber("safe") as an approximation,
// since L2 nodes may not support the "finalized" tag.
func (f *rpcFinalityChecker) GetFinalizedBlockNumber(ctx context.Context, chainID string) (uint64, error) {
	// Check cache first
	if cached, ok := f.getCached(chainID); ok {
		return cached, nil
	}

	client, nodeURL := f.getClient(chainID)
	if client == nil {
		return 0, fmt.Errorf("no RPC endpoint registered for chain %s", chainID)
	}

	// Choose the block tag based on chain type and rollup mechanism
	tag := "finalized"
	numericID := core.ResolveChainID(chainID)
	if core.IsL2Chain(numericID) {
		// L2 nodes often don't support "finalized" tag; use "safe" as approximation
		tag = "safe"

		// Log rollup-specific finality characteristics
		rollupType := core.GetRollupType(numericID)
		switch rollupType {
		case core.RollupOptimistic:
			f.logger.Warn("using safe tag for optimistic rollup — true finality requires L1 batch confirmation",
				"chain_id", chainID, "rollup_type", rollupType.String())
		case core.RollupZK:
			f.logger.Warn("using safe tag for ZK rollup — true finality requires L1 proof verification",
				"chain_id", chainID, "rollup_type", rollupType.String())
		}
	}

	blockNumber, err := f.getBlockByTag(ctx, client, nodeURL, tag)
	degraded := false
	if err != nil {
		// Fallback: try "latest" if "finalized"/"safe" is not supported.
		// This is a serious degradation — the node cannot verify finality,
		// so events may appear finalized when they are not.
		if tag != "latest" {
			f.logger.Error("finality tag not supported, falling back to latest — finality guarantees lost",
				"chain_id", chainID, "tag", tag, "error", err.Error())
			blockNumber, err = f.getBlockByTag(ctx, client, nodeURL, "latest")
			if err != nil {
				return 0, fmt.Errorf("failed to get block number for chain %s: %w", chainID, err)
			}
			degraded = true
		} else {
			return 0, fmt.Errorf("failed to get finalized block for chain %s: %w", chainID, err)
		}
	}

	// For L2 chains, apply a finality discount based on the rollup's
	// L1 challenge/proof window. The "safe" tag only reflects the L2
	// sequencer's view — true finality requires L1 confirmation.
	safeBlock := blockNumber
	if core.IsL2Chain(numericID) {
		info := core.GetL2ChainInfo(numericID)
		if info != nil && info.FinalityBlocks > 0 {
			margin := uint64(info.FinalityBlocks)
			if blockNumber > margin {
				blockNumber -= margin
			} else {
				blockNumber = 0 // chain too young for reliable finality
			}
			f.logger.Info("L2 finality discount applied",
				"chain_id", chainID, "safe_block", safeBlock,
				"finality_blocks", margin, "finalized_block", blockNumber)
		}
	}

	// Update cache
	f.cacheMu.Lock()
	f.cache[chainID] = cachedFinality{
		blockNumber: blockNumber,
		cachedAt:    time.Now(),
		degraded:    degraded,
	}
	f.cacheMu.Unlock()

	return blockNumber, nil
}

// IsBlockFinalized checks if a block number is at or below the finalized block.
func (f *rpcFinalityChecker) IsBlockFinalized(ctx context.Context, chainID string, blockNumber uint64) (bool, error) {
	result, err := f.IsBlockFinalizedWithStatus(ctx, chainID, blockNumber)
	if err != nil {
		return false, err
	}
	return result.IsFinalized, nil
}

// IsBlockFinalizedWithStatus returns a FinalityResult that includes both the
// finalization status and whether the result came from a degraded data source.
// When Degraded is true, the caller should treat the result as unreliable —
// the block may not actually be finalized.
func (f *rpcFinalityChecker) IsBlockFinalizedWithStatus(ctx context.Context, chainID string, blockNumber uint64) (*FinalityResult, error) {
	finalized, err := f.GetFinalizedBlockNumber(ctx, chainID)
	if err != nil {
		return nil, err
	}
	return &FinalityResult{
		IsFinalized: blockNumber <= finalized,
		Degraded:    f.IsDegraded(chainID),
	}, nil
}

// IsDegraded returns true if the cached finality value for the given chain
// came from a "latest" fallback rather than a true "finalized" or "safe" tag.
// When degraded, the finality guarantees are lost — events may appear finalized
// when they are not.
func (f *rpcFinalityChecker) IsDegraded(chainID string) bool {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()
	return f.cache[chainID].degraded
}

func (f *rpcFinalityChecker) getClient(chainID string) (*http.Client, string) {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()
	return f.clients[chainID], f.nodeURLs[chainID]
}

func (f *rpcFinalityChecker) getCached(chainID string) (uint64, bool) {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()

	cached, ok := f.cache[chainID]
	if !ok {
		return 0, false
	}
	ttl := f.cacheTTL
	if cached.degraded {
		ttl = f.degradedCacheTTL
	}
	if time.Since(cached.cachedAt) > ttl {
		return 0, false
	}
	return cached.blockNumber, true
}

// getBlockByTag calls eth_getBlockByNumber with a block tag ("finalized", "safe", "latest")
// and returns the block number.
func (f *rpcFinalityChecker) getBlockByTag(ctx context.Context, client *http.Client, nodeURL, tag string) (uint64, error) {
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params":  []any{tag, false},
		"id":      1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", nodeURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Body = io.NopCloser(bytes.NewReader(body))

	resp, err := client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // defer close

	var rpcResp struct {
		Result struct {
			Number string `json:"number"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return 0, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result.Number == "" {
		return 0, fmt.Errorf("block tag %s returned null result", tag)
	}

	// Parse hex block number
	numberStr := rpcResp.Result.Number
	if len(numberStr) > 2 && numberStr[:2] == "0x" {
		var number uint64
		for _, c := range numberStr[2:] {
			number = number * 16
			switch {
			case c >= '0' && c <= '9':
				number += uint64(c - '0')
			case c >= 'a' && c <= 'f':
				number += uint64(c-'a') + 10
			case c >= 'A' && c <= 'F':
				number += uint64(c-'A') + 10
			}
		}
		return number, nil
	}

	return 0, fmt.Errorf("invalid block number format: %s", numberStr)
}
