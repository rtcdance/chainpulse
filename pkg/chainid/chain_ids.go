package chainid

import (
	"strconv"
	"strings"
)

var knownChainIDs = map[string]int{
	"ethereum":     1,
	"mainnet":      1,
	"polygon":      137,
	"matic":        137,
	"bsc":          56,
	"binance":      56,
	"arbitrum":     42161,
	"optimism":     10,
	"base":         8453,
	"avalanche":    43114,
	"solana":       101,
	"mainnet-beta": 101,
	"cosmos":       118,
	"cosmoshub":    118,
	"zksync":       324,
	"scroll":       534352,
	"linea":        59144,
	"mantle":       5000,
}

var knownChainNames = map[int]string{
	1:      "ethereum",
	137:    "polygon",
	56:     "bsc",
	42161:  "arbitrum",
	10:     "optimism",
	8453:   "base",
	43114:  "avalanche",
	101:    "solana",
	118:    "cosmos",
	324:    "zksync",
	534352: "scroll",
	59144:  "linea",
	5000:   "mantle",
}

// evmChainIDs lists chain IDs that use the EVM execution environment
var evmChainIDs = map[int]bool{
	1:      true,
	137:    true,
	56:     true,
	42161:  true,
	10:     true,
	8453:   true,
	43114:  true,
	324:    true,
	534352: true,
	59144:  true,
	5000:   true,
}

// cosmosChainIDs lists chain IDs that use the Cosmos SDK
var cosmosChainIDs = map[int]bool{
	118: true,
}

// solanaChainIDs lists chain IDs that use the Solana blockchain
var solanaChainIDs = map[int]bool{
	101: true,
}

// l2ChainIDs lists EVM chain IDs that are Layer 2 rollups.
// L2 finality depends on L1 batch confirmation, not just L2 block depth.
var l2ChainIDs = map[int]bool{
	42161:    true, // Arbitrum One
	421614:   true, // Arbitrum Sepolia
	10:       true, // Optimism
	11155420: true, // Optimism Sepolia
	8453:     true, // Base
	84532:    true, // Base Sepolia
	324:      true, // zkSync Era
	300:      true, // zkSync Era Sepolia
	534352:   true, // Scroll
	534351:   true, // Scroll Sepolia
	59144:    true, // Linea
	59141:    true, // Linea Sepolia
	5000:     true, // Mantle
}

// RollupType classifies L2 rollup consensus mechanism
type RollupType int

const (
	RollupNone       RollupType = iota // Not an L2
	RollupOptimistic                   // Optimistic rollup (Arbitrum, Optimism, Base, Mantle)
	RollupZK                           // ZK rollup (zkSync, Scroll, Linea)
)

// L2ChainInfo describes a Layer 2 rollup chain's finality characteristics
type L2ChainInfo struct {
	ChainID        int
	Name           string
	RollupType     RollupType
	L1ChainID      int // Parent L1 chain ID (1 for Ethereum mainnet L2s)
	FinalityBlocks int // Approximate L1 blocks until L2 state is confirmed
}

// l2ChainInfo maps L2 chain IDs to their rollup-specific info
var l2ChainInfo = map[int]*L2ChainInfo{
	// Optimistic rollups
	42161:    {ChainID: 42161, Name: "arbitrum", RollupType: RollupOptimistic, L1ChainID: 1, FinalityBlocks: 960}, // ~4h at 12s blocks
	421614:   {ChainID: 421614, Name: "arbitrum-sepolia", RollupType: RollupOptimistic, L1ChainID: 11155111, FinalityBlocks: 960},
	10:       {ChainID: 10, Name: "optimism", RollupType: RollupOptimistic, L1ChainID: 1, FinalityBlocks: 50400}, // ~7 day challenge window
	11155420: {ChainID: 11155420, Name: "optimism-sepolia", RollupType: RollupOptimistic, L1ChainID: 11155111, FinalityBlocks: 50400},
	8453:     {ChainID: 8453, Name: "base", RollupType: RollupOptimistic, L1ChainID: 1, FinalityBlocks: 50400}, // OP Stack, same as Optimism
	84532:    {ChainID: 84532, Name: "base-sepolia", RollupType: RollupOptimistic, L1ChainID: 11155111, FinalityBlocks: 50400},
	5000:     {ChainID: 5000, Name: "mantle", RollupType: RollupOptimistic, L1ChainID: 1, FinalityBlocks: 960},
	// ZK rollups
	324:    {ChainID: 324, Name: "zksync", RollupType: RollupZK, L1ChainID: 1, FinalityBlocks: 720}, // ~2.4h for proof submission
	300:    {ChainID: 300, Name: "zksync-sepolia", RollupType: RollupZK, L1ChainID: 11155111, FinalityBlocks: 720},
	534352: {ChainID: 534352, Name: "scroll", RollupType: RollupZK, L1ChainID: 1, FinalityBlocks: 1440}, // ~4.8h for proof verification
	534351: {ChainID: 534351, Name: "scroll-sepolia", RollupType: RollupZK, L1ChainID: 11155111, FinalityBlocks: 1440},
	59144:  {ChainID: 59144, Name: "linea", RollupType: RollupZK, L1ChainID: 1, FinalityBlocks: 1440}, // ~4.8h for proof verification
	59141:  {ChainID: 59141, Name: "linea-sepolia", RollupType: RollupZK, L1ChainID: 11155111, FinalityBlocks: 1440},
}

// IsL2Chain returns true if the chain ID corresponds to an L2 rollup.
func IsL2Chain(chainID int) bool {
	return l2ChainIDs[chainID]
}

// GetRollupType returns the rollup type for an L2 chain ID.
// Returns RollupNone for non-L2 chains.
func GetRollupType(chainID int) RollupType {
	if info, ok := l2ChainInfo[chainID]; ok {
		return info.RollupType
	}
	return RollupNone
}

// GetL2ChainInfo returns the L2 chain info for a given chain ID.
// Returns nil if the chain is not an L2.
func GetL2ChainInfo(chainID int) *L2ChainInfo {
	return l2ChainInfo[chainID]
}

// String returns a human-readable name for the RollupType.
func (rt RollupType) String() string {
	switch rt {
	case RollupOptimistic:
		return "optimistic"
	case RollupZK:
		return "zk"
	default:
		return "none"
	}
}

// GetChainType returns the blockchain type for a given chain ID string.
// Returns "EVM", "Solana", "Cosmos", or "unknown".
func GetChainType(chainID string) string {
	id := ResolveChainID(chainID)
	if id == 0 {
		return "unknown"
	}
	if solanaChainIDs[id] {
		return "Solana"
	}
	if cosmosChainIDs[id] {
		return "Cosmos"
	}
	if evmChainIDs[id] {
		return "EVM"
	}
	return "unknown"
}

// ResolveChainID converts a symbolic or numeric chain identifier into an integer.
func ResolveChainID(value string) int {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return 0
	}
	if parsed, err := strconv.Atoi(trimmed); err == nil {
		return parsed
	}
	if chainID, ok := knownChainIDs[trimmed]; ok {
		return chainID
	}
	return 0
}

// ResolveChainName converts a numeric chain ID into its canonical string name.
// Returns the numeric string representation for unknown chain IDs.
func ResolveChainName(id int) string {
	if name, ok := knownChainNames[id]; ok {
		return name
	}
	return strconv.Itoa(id)
}

// IsSolanaChain returns true if the chain ID corresponds to a Solana chain.
func IsSolanaChain(chainID string) bool {
	id := ResolveChainID(chainID)
	return solanaChainIDs[id]
}

func IsCosmosChain(chainID string) bool {
	id := ResolveChainID(chainID)
	return cosmosChainIDs[id]
}