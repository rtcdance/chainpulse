package indexing

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
)

type mockFinalityChecker struct{}

func (m *mockFinalityChecker) GetFinalizedBlockNumber(_ context.Context, _ string) (uint64, error) {
	return 100, nil
}

func (m *mockFinalityChecker) IsBlockFinalized(_ context.Context, _ string, blockNumber uint64) (bool, error) {
	return blockNumber <= 100, nil
}

func (m *mockFinalityChecker) IsBlockFinalizedWithStatus(_ context.Context, _ string, _ uint64) (*finality.FinalityResult, error) {
	return &finality.FinalityResult{IsFinalized: true}, nil
}

func TestEth2FinalityStrategy(t *testing.T) {
	t.Parallel()
	mock := &mockFinalityChecker{}
	s := NewEth2FinalityStrategy("ethereum", mock)

	if s.ChainID() != "ethereum" {
		t.Errorf("ChainID = %q, want ethereum", s.ChainID())
	}
	if s.Description() == "" {
		t.Error("Description should not be empty")
	}
	if s.RecommendedConfirmations() != 32 {
		t.Errorf("RecommendedConfirmations = %d, want 32", s.RecommendedConfirmations())
	}
	if !s.IsBlockSafe(context.Background(), 68, 100) {
		t.Error("block 68 should be safe at height 100")
	}
	if s.IsBlockSafe(context.Background(), 90, 100) {
		t.Error("block 90 should not be safe at height 100 (< 32 confirmations)")
	}

	finalized, err := s.IsBlockFinalized(context.Background(), 50)
	if err != nil {
		t.Fatalf("IsBlockFinalized error: %v", err)
	}
	if !finalized {
		t.Error("block 50 should be finalized")
	}
}

func TestProbabilisticFinalityStrategy(t *testing.T) {
	t.Parallel()
	mock := &mockFinalityChecker{}
	s := NewProbabilisticFinalityStrategy("bsc", mock, 15, 30)

	if s.ChainID() != "bsc" {
		t.Errorf("ChainID = %q, want bsc", s.ChainID())
	}
	if s.Description() == "" {
		t.Error("Description should not be empty")
	}
	if s.RecommendedConfirmations() != 15 {
		t.Errorf("RecommendedConfirmations = %d, want 15", s.RecommendedConfirmations())
	}
	if !s.IsBlockSafe(context.Background(), 85, 100) {
		t.Error("block 85 should be safe at height 100 (>= 15 confirmations)")
	}
	if s.IsBlockSafe(context.Background(), 90, 100) {
		t.Error("block 90 should not be safe at height 100 (< 15 confirmations)")
	}

	finalized, err := s.IsBlockFinalized(context.Background(), 50)
	if err != nil {
		t.Fatalf("IsBlockFinalized error: %v", err)
	}
	if !finalized {
		t.Error("block 50 should be finalized")
	}
}

func TestL2RollupFinalityStrategy(t *testing.T) {
	t.Parallel()
	mock := &mockFinalityChecker{}

	t.Run("optimistic", func(t *testing.T) {
		t.Parallel()
		s := NewL2RollupFinalityStrategy("optimism", mock, chainid.RollupOptimistic)
		if s.ChainID() != "optimism" {
			t.Errorf("ChainID = %q", s.ChainID())
		}
		if s.RecommendedConfirmations() != 12 {
			t.Errorf("RecommendedConfirmations = %d", s.RecommendedConfirmations())
		}
		desc := s.Description()
		if desc == "" {
			t.Error("Description should not be empty")
		}
		if !s.IsBlockSafe(context.Background(), 99, 100) {
			t.Error("L2 block 99 should be safe at height 100")
		}
	})

	t.Run("zk", func(t *testing.T) {
		t.Parallel()
		s := NewL2RollupFinalityStrategy("zksync", mock, chainid.RollupZK)
		desc := s.Description()
		if desc == "" {
			t.Error("Description should not be empty")
		}
	})

	t.Run("default_type", func(t *testing.T) {
		t.Parallel()
		s := NewL2RollupFinalityStrategy("unknown_l2", mock, chainid.RollupNone)
		desc := s.Description()
		if desc == "" {
			t.Error("Description should not be empty for default type")
		}
	})
}

func TestFinalityStrategyRegistry(t *testing.T) {
	t.Parallel()
	mock := &mockFinalityChecker{}
	r := NewFinalityStrategyRegistry()

	eth2 := NewEth2FinalityStrategy("ethereum", mock)
	r.Register(eth2)

	s, err := r.GetStrategy("ethereum")
	if err != nil {
		t.Fatalf("GetStrategy error: %v", err)
	}
	if s.ChainID() != "ethereum" {
		t.Errorf("ChainID = %q", s.ChainID())
	}

	_, err = r.GetStrategy("unknown")
	if err == nil {
		t.Error("GetStrategy for unknown chain should return error")
	}

	if !r.IsBlockSafeForChain(context.Background(), "ethereum", 68, 100) {
		t.Error("block 68 should be safe for ethereum at height 100")
	}
	if r.IsBlockSafeForChain(context.Background(), "unknown", 50, 100) {
		t.Error("unknown chain should return false for IsBlockSafeForChain")
	}

	finalized, err := r.IsBlockFinalizedForChain(context.Background(), "ethereum", 50)
	if err != nil {
		t.Fatalf("IsBlockFinalizedForChain error: %v", err)
	}
	if !finalized {
		t.Error("block 50 should be finalized")
	}

	_, err = r.IsBlockFinalizedForChain(context.Background(), "unknown", 50)
	if err == nil {
		t.Error("unknown chain should return error for IsBlockFinalizedForChain")
	}

	descs := r.ListDescriptions()
	if len(descs) != 1 {
		t.Errorf("ListDescriptions len = %d, want 1", len(descs))
	}
}

func TestGetDefaultFinalityConfig(t *testing.T) {
	t.Parallel()

	cfg := GetDefaultFinalityConfig(1)
	if cfg == nil {
		t.Fatal("expected config for chain 1 (Ethereum)")
	}
	if cfg.Strategy != "eth2" {
		t.Errorf("Strategy = %q, want eth2", cfg.Strategy)
	}

	cfg = GetDefaultFinalityConfig(56)
	if cfg == nil {
		t.Fatal("expected config for chain 56 (BSC)")
	}
	if cfg.Strategy != "probabilistic" {
		t.Errorf("Strategy = %q, want probabilistic", cfg.Strategy)
	}

	cfg = GetDefaultFinalityConfig(10)
	if cfg == nil {
		t.Fatal("expected config for chain 10 (OP Mainnet)")
	}

	cfg = GetDefaultFinalityConfig(999999)
	if cfg != nil {
		t.Error("expected nil config for unknown chain")
	}
}
