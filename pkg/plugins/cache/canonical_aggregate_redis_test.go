package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCanonicalRedisCacheRejectsUnsafeConfiguration(t *testing.T) {
	t.Parallel()
	for _, url := range []string{"", "http://user:secret@localhost:6379", "redis://user:secret@localhost:invalid"} {
		backend, err := NewCanonicalAggregateRedisCache(url)
		require.Error(t, err)
		require.Nil(t, backend)
		require.NotContains(t, err.Error(), "secret")
	}
}

func TestCanonicalRedisCacheHonorsCancellationAndBounds(t *testing.T) {
	t.Parallel()
	backend, err := NewCanonicalAggregateRedisCache("redis://127.0.0.1:1")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, backend.Close()) })
	require.True(t, backend.client.Options().ContextTimeoutEnabled)
	require.LessOrEqual(t, backend.client.Options().ReadTimeout, 50*time.Millisecond)
	require.LessOrEqual(t, backend.client.Options().PoolSize, 8)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = backend.Get(ctx, "key")
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, backend.Set(ctx, "key", []byte("1"), 30), context.Canceled)
	require.Error(t, backend.Set(context.Background(), "key", []byte("1"), 0))
	require.Error(t, backend.Set(context.Background(), "key", make([]byte, 65537), 30))
}
