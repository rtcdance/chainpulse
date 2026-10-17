package finality

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrategyAliasesShareOneFinalityPolicy(t *testing.T) {
	t.Parallel()
	r, err := NewRegistry(Strategy{Mode: StrategyLatest}, map[string]Strategy{"ethereum": {Mode: StrategyTag, Tag: "finalized"}})
	require.NoError(t, err)
	require.Equal(t, Strategy{Mode: StrategyTag, Tag: "finalized"}, r.StrategyFor("1"))
	_, err = NewRegistry(Strategy{Mode: StrategyLatest}, map[string]Strategy{"ethereum": {Mode: StrategyTag, Tag: "safe"}, "1": {Mode: StrategyTag, Tag: "finalized"}})
	require.Error(t, err, "conflicting aliases must not choose a policy by map iteration order")
	_, err = NewRegistry(Strategy{Mode: StrategyLatest}, map[string]Strategy{" ": {Mode: StrategyTag, Tag: "safe"}})
	require.Error(t, err, "blank policy keys must not be silently ignored")
}
