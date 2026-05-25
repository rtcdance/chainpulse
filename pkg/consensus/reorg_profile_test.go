package consensus

import (
	"testing"
)

func TestGetReorgProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		chainID uint64
		want    string
	}{
		{"mainnet", 1, "Ethereum Mainnet"},
		{"bsc", 56, "BSC"},
		{"polygon", 137, "Polygon"},
		{"arbitrum", 42161, "Arbitrum One"},
		{"optimism", 10, "Optimism"},
		{"unknown_defaults_to_mainnet", 99999, "Ethereum Mainnet"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := GetReorgProfile(tc.chainID)
			if got.Name != tc.want {
				t.Errorf("GetReorgProfile(%d).Name = %q, want %q", tc.chainID, got.Name, tc.want)
			}
		})
	}
}

func TestClassifyReorg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		depth    uint64
		profile  ChainReorgProfile
		severity ReorgSeverity
	}{
		{"shallow_mainnet", 3, MainnetReorgProfile, ReorgShallow},
		{"deep_mainnet", 50, MainnetReorgProfile, ReorgDeep},
		{"critical_mainnet", 100, MainnetReorgProfile, ReorgCritical},
		{"shallow_boundary", 4, MainnetReorgProfile, ReorgShallow},
		{"deep_boundary", 64, MainnetReorgProfile, ReorgDeep},
		{"critical_boundary", 65, MainnetReorgProfile, ReorgCritical},
		{"l2_sequencer_no_reorg", 0, ArbitrumReorgProfile, ReorgShallow},
		{"l2_sequencer_critical", 1, ArbitrumReorgProfile, ReorgCritical},
		{"optimism_shallow", 1, OptimismReorgProfile, ReorgShallow},
		{"optimism_deep", 4, OptimismReorgProfile, ReorgDeep},
		{"optimism_critical", 10, OptimismReorgProfile, ReorgCritical},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyReorg(tc.depth, tc.profile)
			if got != tc.severity {
				t.Errorf("ClassifyReorg(%d) = %q, want %q", tc.depth, got, tc.severity)
			}
		})
	}
}

func TestIsReorgAnomalous(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		depth     uint64
		profile   ChainReorgProfile
		anomalous bool
	}{
		{"normal", 3, MainnetReorgProfile, false},
		{"deep_but_not_anomalous", 64, MainnetReorgProfile, false},
		{"anomalous", 65, MainnetReorgProfile, true},
		{"l2_anomalous", 1, ArbitrumReorgProfile, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsReorgAnomalous(tc.depth, tc.profile)
			if got != tc.anomalous {
				t.Errorf("IsReorgAnomalous(%d) = %v, want %v", tc.depth, got, tc.anomalous)
			}
		})
	}
}

func TestReorgProfileForChain(t *testing.T) {
	t.Parallel()

	t.Run("sequencer_finality_l2", func(t *testing.T) {
		t.Parallel()
		p := ReorgProfileForChain(999, "TestL2", true, true)
		if p.L2SequencerFinality != true {
			t.Error("expected sequencer finality")
		}
		if p.MaxExpectedDepth != 0 {
			t.Error("expected 0 max depth")
		}
	})

	t.Run("regular_l2", func(t *testing.T) {
		t.Parallel()
		p := ReorgProfileForChain(998, "TestL2Reg", true, false)
		if p.IsL2 != true {
			t.Error("expected IsL2")
		}
		if p.MaxExpectedDepth != 5 {
			t.Error("expected 5 max depth")
		}
	})

	t.Run("default_l1", func(t *testing.T) {
		t.Parallel()
		p := ReorgProfileForChain(997, "TestL1", false, false)
		if p.IsL2 != false {
			t.Error("expected not L2")
		}
		if p.MaxExpectedDepth != 64 {
			t.Errorf("expected 64 max depth, got %d", p.MaxExpectedDepth)
		}
	})

	t.Run("custom_name", func(t *testing.T) {
		t.Parallel()
		p := ReorgProfileForChain(12345, "CustomChain", false, false)
		if p.Name != "CustomChain" {
			t.Errorf("Name = %q", p.Name)
		}
		if p.ChainID != 12345 {
			t.Errorf("ChainID = %d", p.ChainID)
		}
	})
}
