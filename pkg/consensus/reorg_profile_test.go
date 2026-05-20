package consensus

import "testing"

func TestClassifyReorg(t *testing.T) {
	tests := []struct {
		name    string
		depth   uint64
		profile ChainReorgProfile
		want    ReorgSeverity
	}{
		// Mainnet
		{"mainnet 1 block", 1, MainnetReorgProfile, ReorgShallow},
		{"mainnet 4 blocks", 4, MainnetReorgProfile, ReorgShallow},
		{"mainnet 10 blocks", 10, MainnetReorgProfile, ReorgDeep},
		{"mainnet 64 blocks", 64, MainnetReorgProfile, ReorgDeep},
		{"mainnet 65 blocks", 65, MainnetReorgProfile, ReorgCritical},

		// BSC
		{"bsc 3 blocks", 3, BSReorgProfile, ReorgShallow},
		{"bsc 10 blocks", 10, BSReorgProfile, ReorgDeep},
		{"bsc 16 blocks", 16, BSReorgProfile, ReorgCritical},

		// Polygon
		{"polygon 5 blocks", 5, PolygonReorgProfile, ReorgShallow},
		{"polygon 50 blocks", 50, PolygonReorgProfile, ReorgDeep},
		{"polygon 100 blocks", 100, PolygonReorgProfile, ReorgCritical},

		// Arbitrum (sequencer finality)
		{"arbitrum 0 blocks", 0, ArbitrumReorgProfile, ReorgShallow},
		{"arbitrum 1 block", 1, ArbitrumReorgProfile, ReorgCritical},

		// Optimism
		{"optimism 1 block", 1, OptimismReorgProfile, ReorgShallow},
		{"optimism 3 blocks", 3, OptimismReorgProfile, ReorgDeep},
		{"optimism 6 blocks", 6, OptimismReorgProfile, ReorgCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyReorg(tt.depth, tt.profile)
			if got != tt.want {
				t.Errorf("ClassifyReorg(%d, %s) = %s, want %s",
					tt.depth, tt.profile.Name, got, tt.want)
			}
		})
	}
}

func TestIsReorgAnomalous(t *testing.T) {
	if IsReorgAnomalous(1, MainnetReorgProfile) {
		t.Error("1-block mainnet reorg should not be anomalous")
	}
	if !IsReorgAnomalous(100, MainnetReorgProfile) {
		t.Error("100-block mainnet reorg should be anomalous")
	}
	if !IsReorgAnomalous(1, ArbitrumReorgProfile) {
		t.Error("any Arbitrum reorg should be anomalous (sequencer finality)")
	}
}

func TestGetReorgProfile(t *testing.T) {
	// Known chain
	p := GetReorgProfile(1)
	if p.Name != "Ethereum Mainnet" {
		t.Errorf("expected Mainnet profile for chain 1, got %s", p.Name)
	}

	// Unknown chain defaults to Mainnet
	p2 := GetReorgProfile(99999)
	if p2.Name != "Ethereum Mainnet" {
		t.Errorf("expected Mainnet default for unknown chain, got %s", p2.Name)
	}
}

func TestReorgProfileForChain(t *testing.T) {
	// L2 with sequencer finality
	p := ReorgProfileForChain(8453, "Base", true, true)
	if !p.L2SequencerFinality {
		t.Error("expected sequencer finality")
	}
	if ClassifyReorg(1, p) != ReorgCritical {
		t.Error("any reorg on sequencer-finality chain should be critical")
	}

	// L2 without sequencer finality
	p2 := ReorgProfileForChain(999, "UnknownL2", true, false)
	if !p2.IsL2 {
		t.Error("expected L2 profile")
	}

	// L1
	p3 := ReorgProfileForChain(999, "UnknownL1", false, false)
	if p3.IsL2 {
		t.Error("expected L1 profile")
	}
}

func TestZeroDepthReorg(t *testing.T) {
	// Zero depth should always be shallow
	if ClassifyReorg(0, MainnetReorgProfile) != ReorgShallow {
		t.Error("0-depth reorg should be shallow")
	}
}
