package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
)

// FinalityStrategy defines how a specific chain determines block finality.
//
// Different blockchains use different finality mechanisms:
//   - Ethereum PoS: economic finality after 2 epochs (~12.8 min), with 32-slot safe window
//   - Polygon PoS: checkpoint-based finality via Heimdall layer to Ethereum
//   - BSC: probabilistic finality with 15-block safe window (Parliat) or fast finality (Parlia+)
//   - Solana: PoH with 40-slot optimistic confirmation
//   - L2 Rollups: finality requires L1 batch/proof confirmation
type FinalityStrategy interface {
	// ChainID returns the chain identifier for this strategy.
	ChainID() string

	// Description returns a human-readable explanation of this chain's finality model.
	Description() string

	// IsBlockFinalized checks whether a block at the given height is finalized.
	IsBlockFinalized(ctx context.Context, blockNumber uint64) (bool, error)

	// IsBlockSafe checks whether a block has enough confirmations to be
	// considered safe (beyond the typical reorg window), even if not fully finalized.
	IsBlockSafe(ctx context.Context, blockNumber uint64, currentHeight uint64) bool

	// RecommendedConfirmations returns the number of confirmations recommended
	// before treating indexed events as reliable.
	RecommendedConfirmations() uint64
}

// Eth2FinalityStrategy implements FinalityStrategy for Ethereum (PoS).
//
// Post-Merge Ethereum uses Gasper consensus:
//   - Slot: 12 seconds
//   - Epoch: 32 slots (~6.4 minutes)
//   - Safe: 1 epoch (32 blocks) — very low reorg probability
//   - Finalized: 2 epochs (64 blocks) — economic finality with 2/3 validator stake
//   - RPC tag "safe" ≈ justified checkpoint (1 epoch)
//   - RPC tag "finalized" ≈ finalized checkpoint (2 epochs)
type Eth2FinalityStrategy struct {
	chainID         string
	rpcFinality     finality.FinalityChecker
	safeBlocks      uint64
	finalizedBlocks uint64
}

// NewEth2FinalityStrategy creates a finality strategy for Ethereum PoS.
func NewEth2FinalityStrategy(chainID string, rpcFinality finality.FinalityChecker) *Eth2FinalityStrategy {
	return &Eth2FinalityStrategy{
		chainID:         chainID,
		rpcFinality:     rpcFinality,
		safeBlocks:      32, // 1 epoch
		finalizedBlocks: 64, // 2 epochs
	}
}

func (s *Eth2FinalityStrategy) ChainID() string { return s.chainID }

func (s *Eth2FinalityStrategy) Description() string {
	return "Ethereum PoS: economic finality after 2 epochs (64 blocks/~12.8 min), " +
		"safe after 1 epoch (32 blocks/~6.4 min)"
}

func (s *Eth2FinalityStrategy) IsBlockFinalized(ctx context.Context, blockNumber uint64) (bool, error) {
	return s.rpcFinality.IsBlockFinalized(ctx, s.chainID, blockNumber)
}

func (s *Eth2FinalityStrategy) IsBlockSafe(ctx context.Context, blockNumber uint64, currentHeight uint64) bool {
	return currentHeight >= blockNumber+s.safeBlocks
}

func (s *Eth2FinalityStrategy) RecommendedConfirmations() uint64 { return s.safeBlocks }

// ProbabilisticFinalityStrategy implements FinalityStrategy for chains with
// probabilistic finality (no explicit finality tags, reorg risk decreases
// exponentially with confirmations).
//
// Used by: BSC, Polygon Edge, Avalanche C-Chain, and other EVM chains
// that don't have Ethereum's PoS finality.
type ProbabilisticFinalityStrategy struct {
	chainID      string
	rpcFinality  finality.FinalityChecker
	confirmSafe  uint64
	confirmFinal uint64
}

// NewProbabilisticFinalityStrategy creates a probabilistic finality strategy.
// confirmSafe: blocks after which reorg is very unlikely.
// confirmFinal: blocks after which state is considered final.
func NewProbabilisticFinalityStrategy(chainID string, rpcFinality finality.FinalityChecker, confirmSafe, confirmFinal uint64) *ProbabilisticFinalityStrategy {
	return &ProbabilisticFinalityStrategy{
		chainID:      chainID,
		rpcFinality:  rpcFinality,
		confirmSafe:  confirmSafe,
		confirmFinal: confirmFinal,
	}
}

func (s *ProbabilisticFinalityStrategy) ChainID() string { return s.chainID }

func (s *ProbabilisticFinalityStrategy) Description() string {
	return fmt.Sprintf("Probabilistic finality: safe after %d confirmations, final after %d confirmations",
		s.confirmSafe, s.confirmFinal)
}

func (s *ProbabilisticFinalityStrategy) IsBlockFinalized(ctx context.Context, blockNumber uint64) (bool, error) {
	return s.rpcFinality.IsBlockFinalized(ctx, s.chainID, blockNumber)
}

func (s *ProbabilisticFinalityStrategy) IsBlockSafe(ctx context.Context, blockNumber uint64, currentHeight uint64) bool {
	return currentHeight >= blockNumber+s.confirmSafe
}

func (s *ProbabilisticFinalityStrategy) RecommendedConfirmations() uint64 { return s.confirmSafe }

// L2RollupFinalityStrategy implements FinalityStrategy for L2 rollups.
//
// L2 rollups have a different finality model:
//   - Optimistic rollups (Arbitrum, OP Mainnet): transactions are "safe" after
//     the sequencer includes them, but truly finalized only after the L1 challenge
//     window (7 days for Optimism v0, ~1 day for Arbitrum)
//   - ZK rollups (zkSync, StarkNet): transactions are final after the L1 proof
//     verification transaction is confirmed
//
// In practice:
//   - L2 safe: after sequencer confirmation (seconds)
//   - L2 finalized: after L1 batch confirmation (minutes to days)
type L2RollupFinalityStrategy struct {
	chainID      string
	rpcFinality  finality.FinalityChecker
	rollupType   chainid.RollupType
	l1SafeBlocks uint64
}

// NewL2RollupFinalityStrategy creates an L2 rollup finality strategy.
func NewL2RollupFinalityStrategy(chainID string, rpcFinality finality.FinalityChecker, rollupType chainid.RollupType) *L2RollupFinalityStrategy {
	return &L2RollupFinalityStrategy{
		chainID:      chainID,
		rpcFinality:  rpcFinality,
		rollupType:   rollupType,
		l1SafeBlocks: 12, // L1 safe after 12 confirmations
	}
}

func (s *L2RollupFinalityStrategy) ChainID() string { return s.chainID }

func (s *L2RollupFinalityStrategy) Description() string {
	switch s.rollupType {
	case chainid.RollupOptimistic:
		return "Optimistic rollup: sequencer-safe immediately, L1-finalized after challenge window (~7d)"
	case chainid.RollupZK:
		return "ZK rollup: sequencer-safe immediately, L1-finalized after proof verification (~hours)"
	default:
		return "L2 rollup: sequencer view is safe, L1 batch confirmation provides true finality"
	}
}

func (s *L2RollupFinalityStrategy) IsBlockFinalized(ctx context.Context, blockNumber uint64) (bool, error) {
	return s.rpcFinality.IsBlockFinalized(ctx, s.chainID, blockNumber)
}

func (s *L2RollupFinalityStrategy) IsBlockSafe(ctx context.Context, blockNumber uint64, currentHeight uint64) bool {
	return currentHeight > blockNumber // L2 sequencer output is immediately "safe" for most practical purposes
}

func (s *L2RollupFinalityStrategy) RecommendedConfirmations() uint64 { return 12 }

// FinalityStrategyRegistry manages finality strategies for multiple chains.
// It provides chain-aware finality checks to the indexing pipeline.
type FinalityStrategyRegistry struct {
	mu         sync.RWMutex
	strategies map[string]FinalityStrategy // chainID → strategy
}

// NewFinalityStrategyRegistry creates a new registry.
func NewFinalityStrategyRegistry() *FinalityStrategyRegistry {
	return &FinalityStrategyRegistry{
		strategies: make(map[string]FinalityStrategy),
	}
}

// Register adds a finality strategy for a chain.
func (r *FinalityStrategyRegistry) Register(strategy FinalityStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[strategy.ChainID()] = strategy
}

// GetStrategy returns the finality strategy for a chain.
// Returns an error if the chain is not registered.
func (r *FinalityStrategyRegistry) GetStrategy(chainID string) (FinalityStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategy, ok := r.strategies[chainID]
	if !ok {
		return nil, fmt.Errorf("no finality strategy registered for chain %s", chainID)
	}
	return strategy, nil
}

// IsBlockSafeForChain checks if a block is safe for the given chain.
func (r *FinalityStrategyRegistry) IsBlockSafeForChain(ctx context.Context, chainID string, blockNumber uint64, currentHeight uint64) bool {
	strategy, err := r.GetStrategy(chainID)
	if err != nil {
		return false
	}
	return strategy.IsBlockSafe(ctx, blockNumber, currentHeight)
}

// IsBlockFinalizedForChain checks if a block is finalized for the given chain.
func (r *FinalityStrategyRegistry) IsBlockFinalizedForChain(ctx context.Context, chainID string, blockNumber uint64) (bool, error) {
	strategy, err := r.GetStrategy(chainID)
	if err != nil {
		return false, fmt.Errorf("get finality strategy for chain %s: %w", chainID, err)
	}
	return strategy.IsBlockFinalized(ctx, blockNumber)
}

// ListDescriptions returns descriptions of all registered strategies.
func (r *FinalityStrategyRegistry) ListDescriptions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	descs := make([]string, 0, len(r.strategies))
	for _, s := range r.strategies {
		descs = append(descs, fmt.Sprintf("[%s] %s", s.ChainID(), s.Description()))
	}
	return descs
}

// Built-in chain finality configurations.
// These match the well-known chain IDs.
var defaultFinalityConfigs = map[uint64]struct {
	Strategy    string // "eth2", "probabilistic", "l2-optimistic", "l2-zk"
	SafeBlocks  uint64
	FinalBlocks uint64
}{
	1:     {Strategy: "eth2", SafeBlocks: 32, FinalBlocks: 64},            // Ethereum
	56:    {Strategy: "probabilistic", SafeBlocks: 15, FinalBlocks: 30},   // BSC
	137:   {Strategy: "probabilistic", SafeBlocks: 128, FinalBlocks: 256}, // Polygon
	10:    {Strategy: "l2-optimistic", SafeBlocks: 0, FinalBlocks: 0},     // OP Mainnet
	42161: {Strategy: "l2-optimistic", SafeBlocks: 0, FinalBlocks: 0},     // Arbitrum
	324:   {Strategy: "l2-zk", SafeBlocks: 0, FinalBlocks: 0},             // zkSync Era
	43114: {Strategy: "probabilistic", SafeBlocks: 6, FinalBlocks: 12},    // Avalanche C-Chain
	250:   {Strategy: "probabilistic", SafeBlocks: 30, FinalBlocks: 60},   // Fantom
}

// GetDefaultFinalityConfig returns the default finality config for a chain ID.
// Returns nil for unknown chains.
func GetDefaultFinalityConfig(chainID uint64) *struct {
	Strategy    string
	SafeBlocks  uint64
	FinalBlocks uint64
} {
	cfg, ok := defaultFinalityConfigs[chainID]
	if !ok {
		return nil
	}
	return &cfg
}

// Ensure type-level use of core package
var _ = chainid.RollupOptimistic

// Ensure finality strategies implement the interface
var (
	_ FinalityStrategy = (*Eth2FinalityStrategy)(nil)
	_ FinalityStrategy = (*ProbabilisticFinalityStrategy)(nil)
	_ FinalityStrategy = (*L2RollupFinalityStrategy)(nil)
)

// Ensure time and context are recognized
var _ = time.Now
