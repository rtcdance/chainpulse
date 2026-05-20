// Package reorgprofile defines reorg severity classification and chain-specific
// reorg profiles for Ethereum-based blockchains.
package reorgprofile

// ReorgSeverity classifies how severe a chain reorganization is.
type ReorgSeverity string

const (
	ReorgShallow  ReorgSeverity = "shallow"
	ReorgDeep     ReorgSeverity = "deep"
	ReorgCritical ReorgSeverity = "critical"
)

// ChainReorgProfile defines the expected reorg behavior for a specific chain.
type ChainReorgProfile struct {
	ChainID             uint64
	Name                string
	MaxExpectedDepth    uint64
	ShallowThreshold    uint64
	DeepThreshold       uint64
	IsL2                bool
	L2SequencerFinality bool
}

var MainnetReorgProfile = ChainReorgProfile{
	ChainID: 1, Name: "Ethereum Mainnet", MaxExpectedDepth: 64, ShallowThreshold: 4, DeepThreshold: 64,
}
var BSReorgProfile = ChainReorgProfile{
	ChainID: 56, Name: "BSC", MaxExpectedDepth: 20, ShallowThreshold: 5, DeepThreshold: 15,
}
var PolygonReorgProfile = ChainReorgProfile{
	ChainID: 137, Name: "Polygon", MaxExpectedDepth: 128, ShallowThreshold: 10, DeepThreshold: 64,
}
var ArbitrumReorgProfile = ChainReorgProfile{
	ChainID: 42161, Name: "Arbitrum One", MaxExpectedDepth: 0, ShallowThreshold: 0, DeepThreshold: 0,
	IsL2: true, L2SequencerFinality: true,
}
var OptimismReorgProfile = ChainReorgProfile{
	ChainID: 10, Name: "Optimism", MaxExpectedDepth: 5, ShallowThreshold: 2, DeepThreshold: 5, IsL2: true,
}

var KnownChainReorgProfiles = map[uint64]ChainReorgProfile{
	1: MainnetReorgProfile, 56: BSReorgProfile, 137: PolygonReorgProfile,
	42161: ArbitrumReorgProfile, 10: OptimismReorgProfile,
}

func GetReorgProfile(chainID uint64) ChainReorgProfile {
	if profile, ok := KnownChainReorgProfiles[chainID]; ok {
		return profile
	}
	return MainnetReorgProfile
}

func ClassifyReorg(depth uint64, profile ChainReorgProfile) ReorgSeverity {
	if profile.L2SequencerFinality && depth > 0 {
		return ReorgCritical
	}
	switch {
	case depth <= profile.ShallowThreshold:
		return ReorgShallow
	case depth <= profile.DeepThreshold:
		return ReorgDeep
	default:
		return ReorgCritical
	}
}

func IsReorgAnomalous(depth uint64, profile ChainReorgProfile) bool {
	return ClassifyReorg(depth, profile) == ReorgCritical
}

func ReorgProfileForChain(chainID uint64, name string, isL2 bool, sequencerFinality bool) ChainReorgProfile {
	if sequencerFinality {
		return ChainReorgProfile{ChainID: chainID, Name: name, MaxExpectedDepth: 0, IsL2: true, L2SequencerFinality: true}
	}
	if isL2 {
		return ChainReorgProfile{ChainID: chainID, Name: name, MaxExpectedDepth: 5, ShallowThreshold: 2, DeepThreshold: 5, IsL2: true}
	}
	return ChainReorgProfile{ChainID: chainID, Name: name, MaxExpectedDepth: 64, ShallowThreshold: 4, DeepThreshold: 64}
}