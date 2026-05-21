package chainid

import (
	"testing"
)

func TestChainIDResolveChainID(t *testing.T) {
	id := ResolveChainID("ethereum")
	if id != 1 {
		t.Errorf("expected 1, got %d", id)
	}
	id2 := ResolveChainID("polygon")
	if id2 != 137 {
		t.Errorf("expected 137, got %d", id2)
	}
	id3 := ResolveChainID("unknown")
	if id3 != 0 {
		t.Errorf("expected 0 for unknown, got %d", id3)
	}
}

func TestChainIDResolveChainName(t *testing.T) {
	name := ResolveChainName(1)
	if name != "Ethereum" {
		t.Errorf("expected Ethereum, got %q", name)
	}
	name2 := ResolveChainName(9999)
	if name2 != "" {
		t.Errorf("expected empty for unknown, got %q", name2)
	}
}

func TestChainTypeDetection(t *testing.T) {
	ctype := GetChainType("ethereum")
	if ctype == "" {
		t.Error("expected non-empty chain type")
	}
	ctype2 := GetChainType("solana")
	if ctype2 == "" {
		t.Error("expected non-empty chain type for solana")
	}
}

func TestSolanaAndCosmosDetection(t *testing.T) {
	if !IsSolanaChain("solana") {
		t.Error("expected solana to be solana chain")
	}
	if IsSolanaChain("ethereum") {
		t.Error("expected ethereum to not be solana chain")
	}
	if IsCosmosChain("ethereum") {
		t.Error("expected ethereum to not be cosmos chain")
	}
}

func TestRollupDetection(t *testing.T) {
	rt := GetRollupType(10)
	if rt != RollupOptimistic {
		t.Errorf("expected optimistic rollup for optimism, got %d", rt)
	}
	rt2 := GetRollupType(1)
	if rt2 != RollupNone {
		t.Errorf("expected none for ethereum, got %d", rt2)
	}
}

func TestRollupTypeString(t *testing.T) {
	if RollupNone.String() != "none" {
		t.Errorf("expected none, got %q", RollupNone.String())
	}
}
