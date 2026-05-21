package consensus

// ReorgSeverity classifies how severe a chain reorganization is relative to
// the expected behavior of the chain.
type ReorgSeverity string

const (
	// ReorgShallow indicates a minor reorg within normal operating parameters.
	// On PoS Ethereum, 1-4 block reorgs can happen due to attestation timing.
	ReorgShallow ReorgSeverity = "shallow"

	// ReorgDeep indicates a reorg beyond normal expectations but not yet
	// critical. Requires investigation but may be explainable.
	ReorgDeep ReorgSeverity = "deep"

	// ReorgCritical indicates an anomalous reorg that should never happen
	// under normal consensus rules. May indicate an attack or software bug.
	ReorgCritical ReorgSeverity = "critical"
)

// ChainReorgProfile defines the expected reorg behavior for a specific chain.
// Different chains have fundamentally different reorg characteristics:
// - PoS Ethereum: reorgs should not exceed 1-2 epochs (finality gadget)
// - BSC: shorter block times mean more frequent but typically shallow reorgs
// - L2s: some have no reorgs (Arbitrum), others have limited reorgs (Optimism)
type ChainReorgProfile struct {
	ChainID             uint64 `json:"chain_id"`
	Name                string `json:"name"`
	MaxExpectedDepth    uint64 `json:"max_expected_depth"` // deepest reorg considered normal
	ShallowThreshold    uint64 `json:"shallow_threshold"`  // <= this: shallow
	DeepThreshold       uint64 `json:"deep_threshold"`     // <= this: deep, >: critical
	IsL2                bool   `json:"is_l2"`
	L2SequencerFinality bool   `json:"l2_sequencer_finality"` // true if L2 has no reorgs under normal operation
}

// Predefined reorg profiles for major chains.

var MainnetReorgProfile = ChainReorgProfile{
	ChainID:          1,
	Name:             "Ethereum Mainnet",
	MaxExpectedDepth: 64, // 2 epochs (finality takes ~12.8min)
	ShallowThreshold: 4,
	DeepThreshold:    64,
	IsL2:             false,
}

var BSReorgProfile = ChainReorgProfile{
	ChainID:          56,
	Name:             "BSC",
	MaxExpectedDepth: 20,
	ShallowThreshold: 5,
	DeepThreshold:    15,
	IsL2:             false,
}

var PolygonReorgProfile = ChainReorgProfile{
	ChainID:          137,
	Name:             "Polygon",
	MaxExpectedDepth: 128,
	ShallowThreshold: 10,
	DeepThreshold:    64,
	IsL2:             false,
}

var ArbitrumReorgProfile = ChainReorgProfile{
	ChainID:             42161,
	Name:                "Arbitrum One",
	MaxExpectedDepth:    0, // sequencer finality: no reorgs under normal operation
	ShallowThreshold:    0,
	DeepThreshold:       0,
	IsL2:                true,
	L2SequencerFinality: true,
}

var OptimismReorgProfile = ChainReorgProfile{
	ChainID:          10,
	Name:             "Optimism",
	MaxExpectedDepth: 5, // sequencer window
	ShallowThreshold: 2,
	DeepThreshold:    5,
	IsL2:             true,
}

// KnownChainReorgProfiles maps chain IDs to their reorg profiles.
var KnownChainReorgProfiles = map[uint64]ChainReorgProfile{
	1:     MainnetReorgProfile,
	56:    BSReorgProfile,
	137:   PolygonReorgProfile,
	42161: ArbitrumReorgProfile,
	10:    OptimismReorgProfile,
}

// GetReorgProfile returns the reorg profile for a given chain ID.
// Returns the Mainnet profile as default if the chain is not known.
func GetReorgProfile(chainID uint64) ChainReorgProfile {
	if profile, ok := KnownChainReorgProfiles[chainID]; ok {
		return profile
	}
	return MainnetReorgProfile
}

// ClassifyReorg determines the severity of a reorg given its depth and the
// chain's reorg profile.
func ClassifyReorg(depth uint64, profile ChainReorgProfile) ReorgSeverity {
	// L2s with sequencer finality: any reorg is critical
	if profile.L2SequencerFinality && depth > 0 {
		return ReorgCritical
	}

	if depth <= profile.ShallowThreshold {
		return ReorgShallow
	}
	if depth <= profile.DeepThreshold {
		return ReorgDeep
	}
	return ReorgCritical
}

// IsReorgAnomalous returns true if the reorg depth exceeds the chain's
// maximum expected depth, indicating a potential attack or software bug.
func IsReorgAnomalous(depth uint64, profile ChainReorgProfile) bool {
	return ClassifyReorg(depth, profile) == ReorgCritical
}

// ReorgProfileForChain returns a custom reorg profile for chains not in
// the predefined list, using sensible defaults based on L2 status.
func ReorgProfileForChain(chainID uint64, name string, isL2 bool, sequencerFinality bool) ChainReorgProfile {
	if sequencerFinality {
		return ChainReorgProfile{
			ChainID:             chainID,
			Name:                name,
			MaxExpectedDepth:    0,
			ShallowThreshold:    0,
			DeepThreshold:       0,
			IsL2:                true,
			L2SequencerFinality: true,
		}
	}

	if isL2 {
		return ChainReorgProfile{
			ChainID:          chainID,
			Name:             name,
			MaxExpectedDepth: 5,
			ShallowThreshold: 2,
			DeepThreshold:    5,
			IsL2:             true,
		}
	}

	// Default L1 profile (conservative)
	return ChainReorgProfile{
		ChainID:          chainID,
		Name:             name,
		MaxExpectedDepth: 64,
		ShallowThreshold: 4,
		DeepThreshold:    64,
		IsL2:             false,
	}
}
