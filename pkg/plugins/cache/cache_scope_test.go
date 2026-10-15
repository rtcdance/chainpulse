package cache

import (
	"context"
	"sync"
	"testing"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// mutableScope lets a test bump the generation the way a reorg does.
type mutableScope struct {
	mu    sync.Mutex
	scope CacheScope
}

func (m *mutableScope) get() CacheScope {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scope
}

func (m *mutableScope) set(scope CacheScope) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scope = scope
}

func newTestCache(t *testing.T) portcontract.CachePlugin {
	t.Helper()

	c := NewDefaultInMemoryCachePlugin("test", "1.0.0",
		core.NewDefaultLogger(core.LogLevelError), core.NewDefaultMetricsCollector())
	require.NoError(t, c.Initialize(context.Background(), core.Config{}))
	require.NoError(t, c.Start(context.Background()))
	t.Cleanup(func() { _ = c.Stop(context.Background()) })
	return c
}

func TestVersionTokenReflectsScope(t *testing.T) {
	t.Parallel()

	base := CacheScope{ChainID: "ethereum", Generation: 1, SchemaVersion: "2"}
	assert.Equal(t, "v2:cethereum:g1", base.VersionToken())

	// A generation bump changes the token: that is the whole mechanism.
	bumped := base
	bumped.Generation = 2
	assert.NotEqual(t, base.VersionToken(), bumped.VersionToken())

	// So does a different chain, so two chains never share entries (I01).
	other := base
	other.ChainID = "polygon"
	assert.NotEqual(t, base.VersionToken(), other.VersionToken())

	// And so does a schema change, so a deploy cannot read old-shaped values.
	reshaped := base
	reshaped.SchemaVersion = "3"
	assert.NotEqual(t, base.VersionToken(), reshaped.VersionToken())

	// Empty fields fall back to explicit defaults rather than an empty segment.
	assert.Equal(t, "v1:cglobal:g0", CacheScope{}.VersionToken())
}

func TestVersionedKeysDifferAcrossGenerations(t *testing.T) {
	t.Parallel()

	scope := &mutableScope{scope: CacheScope{ChainID: "ethereum", Generation: 1}}
	keys, err := NewVersionedKeyGenerator("cp", scope.get)
	require.NoError(t, err)

	before := keys.EventsByAddress("0xabc", 0, 50)
	scope.set(CacheScope{ChainID: "ethereum", Generation: 2})
	after := keys.EventsByAddress("0xabc", 0, 50)

	assert.NotEqual(t, before, after, "the same query must key differently after a generation bump")
}

// TestVersionedCacheDoesNotServeStaleCanonicalResultAfterReorg is the T11
// acceptance test: a value cached before a reorg must not be served after it,
// without anyone having to enumerate and delete the old keys.
func TestVersionedCacheDoesNotServeStaleCanonicalResultAfterReorg(t *testing.T) {
	t.Parallel()

	scope := &mutableScope{scope: CacheScope{ChainID: "ethereum", Generation: 1, SchemaVersion: "1"}}
	keys, err := NewVersionedKeyGenerator("cp", scope.get)
	require.NoError(t, err)

	raw := newTestCache(t)
	vc, err := NewVersionedCache(raw, keys)
	require.NoError(t, err)

	ctx := context.Background()
	const baseKey = "events:address:0xabc:offset:0:limit:50"

	// Capture the pre-reorg versioned key so we can later show the entry was not
	// deleted, only made unreachable.
	oldKey := keys.Key(baseKey)

	// Cache the pre-reorg ("old canonical") result.
	require.NoError(t, vc.Set(ctx, baseKey, []byte("OLD-CANONICAL"), 60))

	value, found, err := vc.Get(ctx, baseKey)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []byte("OLD-CANONICAL"), value)

	// A reorg advances the generation.
	scope.set(CacheScope{ChainID: "ethereum", Generation: 2, SchemaVersion: "1"})

	value, found, err = vc.Get(ctx, baseKey)
	require.NoError(t, err)
	assert.False(t, found, "the pre-reorg canonical result must NOT be served after the reorg")
	assert.Nil(t, value)

	// The new generation caches and serves independently.
	require.NoError(t, vc.Set(ctx, baseKey, []byte("NEW-CANONICAL"), 60))
	value, found, err = vc.Get(ctx, baseKey)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []byte("NEW-CANONICAL"), value)

	// The old entry was never deleted — it is simply unreachable, and expires on
	// its TTL. That is why a failed sweep cannot leave stale data reachable.
	stale, err := raw.Get(ctx, oldKey)
	require.NoError(t, err)
	assert.Equal(t, []byte("OLD-CANONICAL"), stale,
		"the old entry is versioned away, not deleted")
	assert.NotEqual(t, oldKey, keys.Key(baseKey), "the new key must differ from the old one")
}

func TestVersionedCacheSeparatesChains(t *testing.T) {
	t.Parallel()

	scope := &mutableScope{scope: CacheScope{ChainID: "ethereum", Generation: 1}}
	keys, err := NewVersionedKeyGenerator("cp", scope.get)
	require.NoError(t, err)
	vc, err := NewVersionedCache(newTestCache(t), keys)
	require.NoError(t, err)

	ctx := context.Background()
	const baseKey = "events:block:500"

	require.NoError(t, vc.Set(ctx, baseKey, []byte("ethereum-row"), 60))

	// Same logical query on another chain must not see it (I01).
	scope.set(CacheScope{ChainID: "polygon", Generation: 1})

	_, found, err := vc.Get(ctx, baseKey)
	require.NoError(t, err)
	assert.False(t, found, "a cached value must not leak across chains")
}

func TestVersionedKeyGeneratorRequiresScope(t *testing.T) {
	t.Parallel()

	_, err := NewVersionedKeyGenerator("cp", nil)
	require.Error(t, err)
}

func TestVersionedCacheRequiresDependencies(t *testing.T) {
	t.Parallel()

	keys, err := NewVersionedKeyGenerator("cp", func() CacheScope { return CacheScope{} })
	require.NoError(t, err)

	_, err = NewVersionedCache(nil, keys)
	require.Error(t, err)

	_, err = NewVersionedCache(newTestCache(t), nil)
	require.Error(t, err)
}
