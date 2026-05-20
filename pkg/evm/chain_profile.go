package evm

import (
	"fmt"
	"time"
)

// ChainProfile contains chain-specific configuration for multi-chain indexing.
// Each chain has unique characteristics: block time, finality depth, RPC rate limits, etc.
type ChainProfile struct {
	ChainID           string        // Numeric chain ID (e.g., "1", "56", "137")
	Name              string        // Chain name (e.g., "ethereum", "bsc", "polygon")
	BlockTime         time.Duration // Average block production interval
	ConfirmationDepth uint64        // Blocks after which event is considered final
	MaxReorgDepth     uint64        // Maximum expected reorg depth for rollback
	RPCRateLimit      int           // Maximum RPC requests per second
	MaxEthGetLogs     int           // Maximum results per eth_getLogs call
	GasToken          string        // Native gas token symbol (e.g., "ETH", "BNB", "MATIC")
}

// Validate checks that the ChainProfile has valid values.
func (p ChainProfile) Validate() error {
	if p.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("name is required for chain %s", p.ChainID)
	}
	if p.BlockTime <= 0 {
		return fmt.Errorf("block_time must be positive for chain %s", p.ChainID)
	}
	if p.ConfirmationDepth == 0 {
		return fmt.Errorf("confirmation_depth must be positive for chain %s", p.ChainID)
	}
	if p.MaxReorgDepth == 0 {
		return fmt.Errorf("max_reorg_depth must be positive for chain %s", p.ChainID)
	}
	if p.RPCRateLimit <= 0 {
		return fmt.Errorf("rpc_rate_limit must be positive for chain %s", p.ChainID)
	}
	if p.MaxEthGetLogs <= 0 {
		return fmt.Errorf("max_eth_get_logs must be positive for chain %s", p.ChainID)
	}
	return nil
}

// KnownChains provides well-known chain profiles for popular EVM chains.
// These serve as sensible defaults that can be overridden by user configuration.
var KnownChains = map[string]ChainProfile{
	"1": {
		ChainID:           "1",
		Name:              "ethereum",
		BlockTime:         12 * time.Second,
		ConfirmationDepth: 12,
		MaxReorgDepth:     10,
		RPCRateLimit:      50,
		MaxEthGetLogs:     10000,
		GasToken:          "ETH",
	},
	"56": {
		ChainID:           "56",
		Name:              "bsc",
		BlockTime:         3 * time.Second,
		ConfirmationDepth: 15,
		MaxReorgDepth:     20,
		RPCRateLimit:      100,
		MaxEthGetLogs:     5000,
		GasToken:          "BNB",
	},
	"137": {
		ChainID:           "137",
		Name:              "polygon",
		BlockTime:         2 * time.Second,
		ConfirmationDepth: 128,
		MaxReorgDepth:     50,
		RPCRateLimit:      30,
		MaxEthGetLogs:     10000,
		GasToken:          "MATIC",
	},
	"42161": {
		ChainID:           "42161",
		Name:              "arbitrum",
		BlockTime:         1 * time.Second,
		ConfirmationDepth: 10,
		MaxReorgDepth:     5,
		RPCRateLimit:      50,
		MaxEthGetLogs:     10000,
		GasToken:          "ETH",
	},
	"10": {
		ChainID:           "10",
		Name:              "optimism",
		BlockTime:         2 * time.Second,
		ConfirmationDepth: 10,
		MaxReorgDepth:     5,
		RPCRateLimit:      50,
		MaxEthGetLogs:     10000,
		GasToken:          "ETH",
	},
	"8453": {
		ChainID:           "8453",
		Name:              "base",
		BlockTime:         2 * time.Second,
		ConfirmationDepth: 10,
		MaxReorgDepth:     5,
		RPCRateLimit:      50,
		MaxEthGetLogs:     10000,
		GasToken:          "ETH",
	},
}

// GetChainProfile returns the ChainProfile for a given chain ID.
// Falls back to defaults if the chain is not in KnownChains.
func GetChainProfile(chainID string) (ChainProfile, error) {
	if profile, ok := KnownChains[chainID]; ok {
		return profile, nil
	}
	return ChainProfile{}, fmt.Errorf("unknown chain_id: %s", chainID)
}

// GetOrBuildChainProfile returns the ChainProfile for a given chain ID,
// building a minimal default profile if the chain is not in KnownChains.
func GetOrBuildChainProfile(chainID string) ChainProfile {
	if profile, ok := KnownChains[chainID]; ok {
		return profile
	}
	// Build a conservative default profile for unknown chains
	return ChainProfile{
		ChainID:           chainID,
		Name:              "unknown",
		BlockTime:         10 * time.Second,
		ConfirmationDepth: 24,
		MaxReorgDepth:     10,
		RPCRateLimit:      20,
		MaxEthGetLogs:     2000,
		GasToken:          "UNKNOWN",
	}
}

// RegisterChainProfile adds or updates a chain profile in KnownChains.
// This allows users to add custom chains at startup.
func RegisterChainProfile(profile ChainProfile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	KnownChains[profile.ChainID] = profile
	return nil
}
