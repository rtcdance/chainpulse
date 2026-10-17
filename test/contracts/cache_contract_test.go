//go:build integration

package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// CacheContractTest defines the contract that all cache implementations must satisfy
func CacheContractTest(t *testing.T, factory func(t *testing.T) core.CachePlugin) {
	t.Run("set_and_get", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		err := cache.Set(ctx, "key1", []byte("value1"), 60)
		require.NoError(t, err)

		value, err := cache.Get(ctx, "key1")
		require.NoError(t, err)
		assert.Equal(t, []byte("value1"), value)
	})

	t.Run("get_nonexistent_key", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		value, err := cache.Get(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("delete_key", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		err := cache.Set(ctx, "delete-key", []byte("delete-value"), 60)
		require.NoError(t, err)

		err = cache.Delete(ctx, "delete-key")
		require.NoError(t, err)

		value, err := cache.Get(ctx, "delete-key")
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("ttl_expiration", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		err := cache.Set(ctx, "ttl-key", []byte("ttl-value"), 1)
		require.NoError(t, err)

		value, err := cache.Get(ctx, "ttl-key")
		require.NoError(t, err)
		assert.Equal(t, []byte("ttl-value"), value)

		time.Sleep(2 * time.Second)

		value, err = cache.Get(ctx, "ttl-key")
		require.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("overwrite_key", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		err := cache.Set(ctx, "overwrite", []byte("value1"), 60)
		require.NoError(t, err)

		err = cache.Set(ctx, "overwrite", []byte("value2"), 60)
		require.NoError(t, err)

		value, err := cache.Get(ctx, "overwrite")
		require.NoError(t, err)
		assert.Equal(t, []byte("value2"), value)
	})

	t.Run("stats_tracking", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		require.NoError(t, cache.Set(ctx, "stats-key", []byte("stats-value"), 60))
		_, err := cache.Get(ctx, "stats-key")
		require.NoError(t, err)
		_, err = cache.Get(ctx, "nonexistent")
		require.NoError(t, err)

		stats := cache.GetStats()
		assert.Greater(t, stats.HitCount, int64(0))
		assert.Greater(t, stats.MissCount, int64(0))
	})

	// A cache stores bytes, not text. Every other case here uses ASCII, for
	// which a naive `string(value)` / `[]byte(string)` implementation looks
	// correct — but encoding/json replaces every non-UTF-8 byte of a Go string
	// with U+FFFD, so a value that reaches a JSON-encoded backend as a string
	// comes back silently corrupted. This payload loses 6 of its 10 bytes that
	// way, so the case has teeth against exactly that implementation.
	t.Run("binary_safe_value", func(t *testing.T) {
		cache := factory(t)
		ctx := context.Background()

		value := []byte{0x00, 0x91, 0xAB, 0xCD, 0xEF, 0x10, 0x00, 0xFF, 0xFE, 0x80}
		require.NoError(t, cache.Set(ctx, "binary-key", value, 60))

		got, err := cache.Get(ctx, "binary-key")
		require.NoError(t, err)
		assert.Equal(t, value, got, "binary value did not survive the round trip")
	})
}
