package bootstrap

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestSIWEProductionRefusesLocalNonceState(t *testing.T) {
	t.Parallel()
	_, _, err := BuildSIWENonceStore(context.Background(), core.Config{CacheType: "memory"}, "production")
	require.ErrorContains(t, err, "shared Redis")
}

func TestSIWEChainIsExplicit(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{"31337", "ethereum"} {
		id, err := ResolveSIWEChainID(raw)
		require.NoError(t, err)
		require.Positive(t, id.Sign())
	}
	for _, raw := range []string{"", "0", "-1", "unknown-chain"} {
		_, err := ResolveSIWEChainID(raw)
		require.Error(t, err)
	}
}

func TestSIWEInvalidRedisConfigDoesNotLeakCredentials(t *testing.T) {
	t.Parallel()
	_, _, err := BuildSIWENonceStore(context.Background(), core.Config{CacheType: "redis", CacheConnectionURL: "redis://secret-user:secret-password@host:not-port"}, "production")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-password")
	require.NotContains(t, err.Error(), "secret-user")
}
