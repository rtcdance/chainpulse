package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/rtcdance/chainpulse/pkg/chainid"
)

func TestResolveChainID(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{input: "1", want: 1},
		{input: "ethereum", want: 1},
		{input: "polygon", want: 137},
		{input: "base", want: 8453},
		{input: "zksync", want: 324},
		{input: "scroll", want: 534352},
		{input: "linea", want: 59144},
		{input: "mantle", want: 5000},
		{input: "unknown", want: 0},
	}

	for _, tt := range tests {
		if got := chainid.ResolveChainID(tt.input); got != tt.want {
			t.Fatalf("chainid.ResolveChainID(%q)=%d want %d", tt.input, got, tt.want)
		}
	}
}

func TestResolveChainName(t *testing.T) {
	tests := []struct {
		id   int
		want string
	}{
		{id: 1, want: "ethereum"},
		{id: 137, want: "polygon"},
		{id: 56, want: "bsc"},
		{id: 42161, want: "arbitrum"},
		{id: 10, want: "optimism"},
		{id: 8453, want: "base"},
		{id: 43114, want: "avalanche"},
		{id: 324, want: "zksync"},
		{id: 534352, want: "scroll"},
		{id: 0, want: "0"},
		{id: 99999, want: "99999"},
	}

	for _, tt := range tests {
		if got := chainid.ResolveChainName(tt.id); got != tt.want {
			t.Fatalf("chainid.ResolveChainName(%d)=%q want %q", tt.id, got, tt.want)
		}
	}
}

func TestGetRollupType(t *testing.T) {
	tests := []struct {
		chainID int
		want    chainid.RollupType
	}{
		{1, chainid.RollupNone},           // Ethereum L1
		{137, chainid.RollupNone},         // Polygon
		{42161, chainid.RollupOptimistic}, // Arbitrum
		{10, chainid.RollupOptimistic},    // Optimism
		{8453, chainid.RollupOptimistic},  // Base
		{5000, chainid.RollupOptimistic},  // Mantle
		{324, chainid.RollupZK},           // zkSync Era
		{534352, chainid.RollupZK},        // Scroll
		{59144, chainid.RollupZK},         // Linea
		{99999, chainid.RollupNone},       // Unknown
	}

	for _, tt := range tests {
		got := chainid.GetRollupType(tt.chainID)
		assert.Equal(t, tt.want, got, "GetRollupType(%d)", tt.chainID)
	}
}

func TestGetL2ChainInfo(t *testing.T) {
	t.Run("arbitrum info", func(t *testing.T) {
		info := chainid.GetL2ChainInfo(42161)
		assert.NotNil(t, info)
		assert.Equal(t, chainid.RollupOptimistic, info.RollupType)
		assert.Equal(t, 1, info.L1ChainID)
		assert.Equal(t, "arbitrum", info.Name)
		assert.Greater(t, info.FinalityBlocks, 0)
	})

	t.Run("zksync info", func(t *testing.T) {
		info := chainid.GetL2ChainInfo(324)
		assert.NotNil(t, info)
		assert.Equal(t, chainid.RollupZK, info.RollupType)
		assert.Equal(t, 1, info.L1ChainID)
		assert.Equal(t, "zksync", info.Name)
	})

	t.Run("non-L2 returns nil", func(t *testing.T) {
		info := chainid.GetL2ChainInfo(1)
		assert.Nil(t, info)
	})

	t.Run("unknown chain returns nil", func(t *testing.T) {
		info := chainid.GetL2ChainInfo(99999)
		assert.Nil(t, info)
	})
}

func TestRollupTypeString(t *testing.T) {
	assert.Equal(t, "none", chainid.RollupNone.String())
	assert.Equal(t, "optimistic", chainid.RollupOptimistic.String())
	assert.Equal(t, "zk", chainid.RollupZK.String())
}

func TestIsL2ChainIncludesNewChains(t *testing.T) {
	assert.True(t, chainid.IsL2Chain(324), "zkSync Era should be L2")
	assert.True(t, chainid.IsL2Chain(534352), "Scroll should be L2")
	assert.True(t, chainid.IsL2Chain(59144), "Linea should be L2")
	assert.True(t, chainid.IsL2Chain(5000), "Mantle should be L2")
	assert.False(t, chainid.IsL2Chain(1), "Ethereum L1 should not be L2")
}

func TestGetChainTypeIncludesNewEVMChains(t *testing.T) {
	assert.Equal(t, "EVM", chainid.GetChainType("324"))
	assert.Equal(t, "EVM", chainid.GetChainType("534352"))
	assert.Equal(t, "EVM", chainid.GetChainType("59144"))
}
