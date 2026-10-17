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

	"github.com/ethereum/go-ethereum/common/hexutil"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/chainid"
)

// FinalityConfidence says HOW a finality answer was obtained, so a caller never
// has to infer it from a bool (finding F08).
type FinalityConfidence string

const (
	// ConfidenceUnknown: no usable answer.
	ConfidenceUnknown FinalityConfidence = "unknown"
	// ConfidenceDegraded: the height came from a fallback such as "latest", so
	// it carries NO finality guarantee.
	ConfidenceDegraded FinalityConfidence = "degraded"
	// ConfidenceSafe: the height came from the "safe" tag. Safe is NOT
	// finalized — it reflects the sequencer's/consensus view without the
	// finality guarantee.
	ConfidenceSafe FinalityConfidence = "safe"
	// ConfidenceFinalized: the height came from the "finalized" tag.
	ConfidenceFinalized FinalityConfidence = "finalized"
)

// FinalitySource names where the answer came from, for diagnostics and for
// callers that want to distinguish a safe-tag answer from a latest fallback.
type FinalitySource string

const (
	SourceUnknown        FinalitySource = "unknown"
	SourceFinalizedTag   FinalitySource = "finalized_tag"
	SourceSafeTag        FinalitySource = "safe_tag"
	SourceLatestFallback FinalitySource = "latest_fallback"
)

// FinalityResult contains the result of a finality check.
//
// IsFinalized and Degraded are the original fields. Confidence, Source and
// ObservedAt were added because a bool plus a "degraded" flag was not enough to
// stop a caller promoting an event: an L2 "safe" answer reported Degraded=false
// while carrying no finality guarantee.
type FinalityResult struct {
	IsFinalized bool // true if the block is at or below the finalized block
	Degraded    bool // true if the height carries no finality guarantee

	// Confidence is the explicit verdict: only ConfidenceFinalized means the
	// block may be treated as finalized.
	Confidence FinalityConfidence
	// Source is where the height came from.
	Source FinalitySource
	// ObservedAt is when the underlying height was observed (cache write time),
	// so a caller can judge staleness.
	ObservedAt time.Time
	// FinalizedBlockNumber is the height this verdict was computed against.
	FinalizedBlockNumber uint64
}

// FinalityChecker determines whether a block is finalized on a given chain.
// PoS Ethereum supports eth_getBlockByNumber("finalized"), while L2 rollups
// require tracking L1 batch finality.
type FinalityChecker interface {
	// GetFinalizedBlockNumber returns the latest finalized block number for the chain.
	// Returns an error if the chain does not support the finalized tag.
	//
	// The height alone is not enough to act on: it may come from an L2 "safe"
	// tag or a "latest" fallback, neither of which carries a finality
	// guarantee. Callers that promote data should use
	// GetFinalizedBlockNumberWithStatus.
	GetFinalizedBlockNumber(ctx context.Context, chainID string) (uint64, error)

	// GetFinalizedBlockNumberWithStatus returns the finalized height together
	// with how it was obtained. Only a result whose Confidence is
	// ConfidenceFinalized and which is not Degraded may be used to promote
	// data to confirmed/finalized; every other confidence describes a height
	// that can still be reorged.
	GetFinalizedBlockNumberWithStatus(ctx context.Context, chainID string) (*FinalityResult, error)

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
	logger           portcontract.Logger
}

type cachedFinality struct {
	blockNumber uint64
	cachedAt    time.Time
	degraded    bool // true when the value carries no finality guarantee
	source      FinalitySource
	confidence  FinalityConfidence
}

// NewRPCFinalityChecker creates a FinalityChecker that queries RPC nodes.
func NewRPCFinalityChecker(logger portcontract.Logger) *rpcFinalityChecker {
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
	delete(f.cache, chainID)
}

// GetFinalizedBlockNumber returns the latest finalized block number.
// For L1 chains: uses eth_getBlockByNumber("finalized").
// For L2 chains: uses eth_getBlockByNumber("safe") as an approximation,
// since L2 nodes may not support the "finalized" tag.
func (f *rpcFinalityChecker) GetFinalizedBlockNumber(ctx context.Context, chainID string) (uint64, error) {
	observation, err := f.observeFinality(ctx, chainID)
	if err != nil {
		return 0, err
	}
	return observation.blockNumber, nil
}

func (f *rpcFinalityChecker) observeFinality(ctx context.Context, chainID string) (cachedFinality, error) {
	// Check cache first
	if cached, ok := f.getCached(chainID); ok {
		return cached, nil
	}

	client, nodeURL := f.getClient(chainID)
	if client == nil {
		return cachedFinality{}, fmt.Errorf("no RPC endpoint registered for chain %s", chainID)
	}

	// Choose the block tag based on chain type and rollup mechanism
	tag := "finalized"
	numericID := chainid.ResolveChainID(chainID)
	if chainid.IsL2Chain(numericID) {
		// L2 nodes often don't support "finalized" tag; use "safe" as approximation
		tag = "safe"

		// Log rollup-specific finality characteristics
		rollupType := chainid.GetRollupType(numericID)
		switch rollupType {
		case chainid.RollupOptimistic:
			f.logger.Warn("using safe tag for optimistic rollup — true finality requires L1 batch confirmation",
				"chain_id", chainID, "rollup_type", rollupType.String())
		case chainid.RollupZK:
			f.logger.Warn("using safe tag for ZK rollup — true finality requires L1 proof verification",
				"chain_id", chainID, "rollup_type", rollupType.String())
		}
	}

	// The tag determines the verdict. "safe" is NOT finalized — it reflects the
	// sequencer's/consensus view without the finality guarantee, so it must
	// never be reported as finalized (finding F08).
	source := SourceFinalizedTag
	confidence := ConfidenceFinalized
	degraded := false
	if tag == "safe" {
		source = SourceSafeTag
		confidence = ConfidenceSafe
		degraded = true
	}

	blockNumber, err := f.getBlockByTag(ctx, client, nodeURL, tag)
	if err != nil {
		// Fallback: try "latest" if "finalized"/"safe" is not supported.
		// This is a serious degradation — the node cannot verify finality,
		// so events may appear finalized when they are not.
		if tag != "latest" {
			f.logger.Error("finality tag not supported, falling back to latest — finality guarantees lost",
				"chain_id", chainID, "tag", tag, "error", err.Error())
			blockNumber, err = f.getBlockByTag(ctx, client, nodeURL, "latest")
			if err != nil {
				return cachedFinality{}, fmt.Errorf("failed to get block number for chain %s: %w", chainID, err)
			}
			source = SourceLatestFallback
			confidence = ConfidenceDegraded
			degraded = true
		} else {
			return cachedFinality{}, fmt.Errorf("failed to get finalized block for chain %s: %w", chainID, err)
		}
	}

	// For L2 chains, apply a finality discount based on the rollup's
	// L1 challenge/proof window. The "safe" tag only reflects the L2
	// sequencer's view — true finality requires L1 confirmation.
	safeBlock := blockNumber
	if chainid.IsL2Chain(numericID) {
		info := chainid.GetL2ChainInfo(numericID)
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
	if f.clients[chainID] != client {
		f.cacheMu.Unlock()
		return cachedFinality{}, fmt.Errorf("finality provider changed during observation for chain %s", chainID)
	}
	observation := cachedFinality{
		blockNumber: blockNumber,
		cachedAt:    time.Now(),
		degraded:    degraded,
		source:      source,
		confidence:  confidence,
	}
	f.cache[chainID] = observation
	f.cacheMu.Unlock()

	return observation, nil
}

// IsBlockFinalized reports whether a block may be treated as finalized.
//
// It returns true ONLY when the height came from the "finalized" tag and the
// block is at or below it. A degraded answer — an L2 "safe" tag or a "latest"
// fallback — returns false even when the height suggests otherwise, because
// otherwise a caller using this bool would promote an event that has no
// finality guarantee (finding F08).
func (f *rpcFinalityChecker) IsBlockFinalized(ctx context.Context, chainID string, blockNumber uint64) (bool, error) {
	result, err := f.IsBlockFinalizedWithStatus(ctx, chainID, blockNumber)
	if err != nil {
		return false, fmt.Errorf("check block %d finality for chain %s: %w", blockNumber, chainID, err)
	}

	if result.Degraded || result.Confidence != ConfidenceFinalized {
		return false, nil
	}
	return result.IsFinalized, nil
}

// GetFinalizedBlockNumberWithStatus returns the finalized height together with
// the confidence and source it was obtained from.
//
// IsFinalized is true only for a non-degraded finalized-tag observation.
// A safe or latest height never implies finality, even in the status interface.
func (f *rpcFinalityChecker) GetFinalizedBlockNumberWithStatus(ctx context.Context, chainID string) (*FinalityResult, error) {
	meta, err := f.observeFinality(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("get finalized block number for chain %s: %w", chainID, err)
	}

	return &FinalityResult{
		IsFinalized:          !meta.degraded && meta.confidence == ConfidenceFinalized,
		Degraded:             meta.degraded,
		Confidence:           meta.confidence,
		Source:               meta.source,
		ObservedAt:           meta.cachedAt,
		FinalizedBlockNumber: meta.blockNumber,
	}, nil
}

// IsBlockFinalizedWithStatus returns the verdict together with how it was
// reached. When Degraded is true, or Confidence is not ConfidenceFinalized, the
// caller must NOT treat the block as finalized.
func (f *rpcFinalityChecker) IsBlockFinalizedWithStatus(ctx context.Context, chainID string, blockNumber uint64) (*FinalityResult, error) {
	result, err := f.GetFinalizedBlockNumberWithStatus(ctx, chainID)
	if err != nil {
		return nil, err
	}
	result.IsFinalized = result.IsFinalized && blockNumber <= result.FinalizedBlockNumber
	return result, nil
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

func (f *rpcFinalityChecker) getCached(chainID string) (cachedFinality, bool) {
	f.cacheMu.RLock()
	defer f.cacheMu.RUnlock()

	cached, ok := f.cache[chainID]
	if !ok {
		return cachedFinality{}, false
	}
	ttl := f.cacheTTL
	if cached.degraded {
		ttl = f.degradedCacheTTL
	}
	if time.Since(cached.cachedAt) > ttl {
		return cachedFinality{}, false
	}
	return cached, true
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

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("finality RPC returned HTTP %d", resp.StatusCode)
	}
	const maxResponseBytes = 2 << 20
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return 0, fmt.Errorf("read finality response: %w", err)
	}
	if len(payload) > maxResponseBytes {
		return 0, fmt.Errorf("finality RPC response exceeds %d bytes", maxResponseBytes)
	}
	var rpcResp struct {
		Version string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  *struct {
			Number string `json:"number"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &rpcResp); err != nil {
		return 0, fmt.Errorf("decode finality response: %w", err)
	}
	if rpcResp.Version != "2.0" || string(rpcResp.ID) != "1" {
		return 0, fmt.Errorf("invalid finality JSON-RPC version or response ID")
	}
	if rpcResp.Error != nil {
		return 0, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}
	if rpcResp.Result == nil {
		return 0, fmt.Errorf("block tag %s returned null result", tag)
	}
	number, err := hexutil.DecodeUint64(rpcResp.Result.Number)
	if err != nil {
		return 0, fmt.Errorf("invalid finality block quantity: %w", err)
	}
	return number, nil
}
