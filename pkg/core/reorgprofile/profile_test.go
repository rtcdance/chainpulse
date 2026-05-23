package reorgprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReorgProfileKnownChain(t *testing.T) {
	t.Parallel()
	profile := GetReorgProfile(1)
	assert.Equal(t, "Ethereum Mainnet", profile.Name)
	assert.Equal(t, uint64(1), profile.ChainID)
}

func TestGetReorgProfileUnknownChain(t *testing.T) {
	t.Parallel()
	profile := GetReorgProfile(99999)
	assert.Equal(t, "Ethereum Mainnet", profile.Name)
	assert.Equal(t, uint64(1), profile.ChainID)
}

func TestGetReorgProfileBSC(t *testing.T) {
	t.Parallel()
	profile := GetReorgProfile(56)
	assert.Equal(t, "BSC", profile.Name)
	assert.False(t, profile.IsL2)
}

func TestGetReorgProfilePolygon(t *testing.T) {
	t.Parallel()
	profile := GetReorgProfile(137)
	assert.Equal(t, "Polygon", profile.Name)
	assert.Equal(t, uint64(128), profile.MaxExpectedDepth)
}

func TestGetReorgProfileArbitrum(t *testing.T) {
	t.Parallel()
	profile := GetReorgProfile(42161)
	assert.Equal(t, "Arbitrum One", profile.Name)
	assert.True(t, profile.IsL2)
	assert.True(t, profile.L2SequencerFinality)
}

func TestGetReorgProfileOptimism(t *testing.T) {
	t.Parallel()
	profile := GetReorgProfile(10)
	assert.Equal(t, "Optimism", profile.Name)
	assert.True(t, profile.IsL2)
	assert.False(t, profile.L2SequencerFinality)
}

func TestClassifyReorgShallow(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(3, MainnetReorgProfile)
	assert.Equal(t, ReorgShallow, result)
}

func TestClassifyReorgAtThreshold(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(4, MainnetReorgProfile)
	assert.Equal(t, ReorgShallow, result)
}

func TestClassifyReorgDeep(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(10, MainnetReorgProfile)
	assert.Equal(t, ReorgDeep, result)
}

func TestClassifyReorgAtDeepThreshold(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(64, MainnetReorgProfile)
	assert.Equal(t, ReorgDeep, result)
}

func TestClassifyReorgCritical(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(65, MainnetReorgProfile)
	assert.Equal(t, ReorgCritical, result)
}

func TestClassifyReorgL2SequencerFinality(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(1, ArbitrumReorgProfile)
	assert.Equal(t, ReorgCritical, result)
}

func TestClassifyReorgL2SequencerFinalityZeroDepth(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(0, ArbitrumReorgProfile)
	assert.Equal(t, ReorgShallow, result)
}

func TestClassifyReorgZeroDepth(t *testing.T) {
	t.Parallel()
	result := ClassifyReorg(0, MainnetReorgProfile)
	assert.Equal(t, ReorgShallow, result)
}

func TestIsReorgAnomalousTrue(t *testing.T) {
	t.Parallel()
	assert.True(t, IsReorgAnomalous(65, MainnetReorgProfile))
}

func TestIsReorgAnomalousFalse(t *testing.T) {
	t.Parallel()
	assert.False(t, IsReorgAnomalous(4, MainnetReorgProfile))
}

func TestIsReorgAnomalousL2Finality(t *testing.T) {
	t.Parallel()
	assert.True(t, IsReorgAnomalous(1, ArbitrumReorgProfile))
	assert.False(t, IsReorgAnomalous(0, ArbitrumReorgProfile))
}

func TestReorgProfileForChainSequencerFinality(t *testing.T) {
	t.Parallel()
	profile := ReorgProfileForChain(7777, "TestChain", true, true)
	assert.Equal(t, uint64(7777), profile.ChainID)
	assert.Equal(t, "TestChain", profile.Name)
	assert.True(t, profile.IsL2)
	assert.True(t, profile.L2SequencerFinality)
	assert.Equal(t, uint64(0), profile.MaxExpectedDepth)
}

func TestReorgProfileForChainL2NoSequencerFinality(t *testing.T) {
	t.Parallel()
	profile := ReorgProfileForChain(7778, "L2Chain", true, false)
	assert.True(t, profile.IsL2)
	assert.False(t, profile.L2SequencerFinality)
	assert.Equal(t, uint64(5), profile.MaxExpectedDepth)
	assert.Equal(t, uint64(2), profile.ShallowThreshold)
}

func TestReorgProfileForChainL1(t *testing.T) {
	t.Parallel()
	profile := ReorgProfileForChain(7779, "L1Chain", false, false)
	assert.False(t, profile.IsL2)
	assert.False(t, profile.L2SequencerFinality)
	assert.Equal(t, uint64(64), profile.MaxExpectedDepth)
	assert.Equal(t, uint64(4), profile.ShallowThreshold)
	assert.Equal(t, uint64(64), profile.DeepThreshold)
}

func TestKnownChainReorgProfilesHasAll(t *testing.T) {
	t.Parallel()
	assert.Contains(t, KnownChainReorgProfiles, uint64(1))
	assert.Contains(t, KnownChainReorgProfiles, uint64(56))
	assert.Contains(t, KnownChainReorgProfiles, uint64(137))
	assert.Contains(t, KnownChainReorgProfiles, uint64(42161))
	assert.Contains(t, KnownChainReorgProfiles, uint64(10))
}

func TestClassifyReorgBSC(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ReorgShallow, ClassifyReorg(5, BSReorgProfile))
	assert.Equal(t, ReorgDeep, ClassifyReorg(10, BSReorgProfile))
	assert.Equal(t, ReorgDeep, ClassifyReorg(15, BSReorgProfile))
	assert.Equal(t, ReorgCritical, ClassifyReorg(20, BSReorgProfile))
}
