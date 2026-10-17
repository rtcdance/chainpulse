//go:build integration

package contracts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func TestMonolithicFactoryStartsRequestedRealRedis(t *testing.T) {
	dsn, endpoint := postgresDSN(), redisCacheURL()
	if dsn == "" || endpoint == "" {
		t.Skip("set PostgreSQL and Redis integration gates with isolated endpoints")
	}
	ctx := context.Background()
	db, cache, err := bootstrap.BuildMonolithicIndexingStorage(ctx, testhelpers.NewTestLogger(), core.Config{
		DatabaseType: "postgres", DatabaseURL: dsn, CacheType: "redis", CacheConnectionURL: endpoint, ActiveChains: []string{"factory-delivery"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.(ports.LifecyclePlugin).Stop(ctx)) })
	connected, ok := cache.(interface{ Connected() bool })
	require.True(t, ok)
	require.True(t, connected.Connected())
	value := []byte{0, 0xff, 0x80, 1}
	require.NoError(t, cache.Set(ctx, "factory-delivery:binary", value, 60))
	got, err := cache.Get(ctx, "factory-delivery:binary")
	require.NoError(t, err)
	require.Equal(t, value, got)
	require.NoError(t, cache.Delete(ctx, "factory-delivery:binary"))
	require.NoError(t, cache.(ports.LifecyclePlugin).Stop(ctx))
	require.False(t, connected.Connected(), "shutdown must release the actual Redis pool")
}
