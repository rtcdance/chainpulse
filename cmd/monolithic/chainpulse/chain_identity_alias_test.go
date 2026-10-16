package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonolithicChainsRejectDuplicateNetworkAliases(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"ethereum,1", "31337,31337", "polygon,137"} {
		_, err := parseChains(raw)
		require.Error(t, err, "one network must not be indexed twice under aliases: %s", raw)
	}
	chains, err := parseChains("ethereum,polygon")
	require.NoError(t, err)
	require.Equal(t, []string{"ethereum", "polygon"}, chains)
}
