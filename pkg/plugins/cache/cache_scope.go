package cache

import (
	"context"
	"fmt"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// CacheScope is the version of the data a cached value describes.
//
// A cache key MUST include it. Otherwise a value cached before a reorg is
// indistinguishable from one cached after it — the key is unchanged, so the
// stale canonical result keeps being served. That is the T11 acceptance:
// "after invalidation, no old canonical result is returned".
type CacheScope struct {
	// ChainID scopes the key to one chain, so two chains at the same height do
	// not share entries (invariant I01).
	ChainID string
	// Generation advances on a reorg. Bumping it makes every value written under
	// the previous generation unreachable in one step.
	Generation int64
	// SchemaVersion changes when the SHAPE of the cached value changes, so a
	// deploy that alters a response cannot read values written by the old code.
	SchemaVersion string
}

// VersionToken is the part of a key that changes when the meaning of the cached
// data changes.
func (s CacheScope) VersionToken() string {
	schema := s.SchemaVersion
	if schema == "" {
		schema = "1"
	}
	chain := s.ChainID
	if chain == "" {
		chain = "global"
	}
	return fmt.Sprintf("v%s:c%s:g%d", schema, chain, s.Generation)
}

// ScopeProvider returns the scope in force right now. It is a function rather
// than a value because a generation bump must take effect on the next key,
// without rebuilding the generator.
type ScopeProvider func() CacheScope

// VersionedKeyGenerator prefixes keys with the current scope.
type VersionedKeyGenerator struct {
	inner *CacheKeyGenerator
	scope ScopeProvider
}

// NewVersionedKeyGenerator wraps the existing key generator.
func NewVersionedKeyGenerator(prefix string, scope ScopeProvider) (*VersionedKeyGenerator, error) {
	if scope == nil {
		return nil, fmt.Errorf("scope provider is required")
	}
	return &VersionedKeyGenerator{
		inner: NewCacheKeyGenerator(prefix),
		scope: scope,
	}, nil
}

// Key versions an already-built key.
func (g *VersionedKeyGenerator) Key(baseKey string) string {
	return g.scope().VersionToken() + ":" + baseKey
}

// Scope returns the scope currently in force (useful for logging).
func (g *VersionedKeyGenerator) Scope() CacheScope {
	return g.scope()
}

// The typed helpers below cover the query paths T11 touches; they delegate to
// the existing generator so there is one place that knows the key shape.

// EventsByAddress versions the "events by address" key.
func (g *VersionedKeyGenerator) EventsByAddress(address string, offset, limit int) string {
	return g.Key(g.inner.GenerateEventsByAddressKey(address, offset, limit))
}

// EventsByBlock versions the "events by block" key.
func (g *VersionedKeyGenerator) EventsByBlock(blockNumber int64, offset, limit int) string {
	return g.Key(g.inner.GenerateEventsByBlockKey(blockNumber, offset, limit))
}

// Query versions a query-result key.
func (g *VersionedKeyGenerator) Query(queryType string, params map[string]any) string {
	return g.Key(g.inner.GenerateQueryKey(queryType, params))
}

// EventCount versions an event-count key.
func (g *VersionedKeyGenerator) EventCount(filters map[string]any) string {
	return g.Key(g.inner.GenerateEventCountKey(filters))
}

// Event versions an event-by-id key.
func (g *VersionedKeyGenerator) Event(eventID string) string {
	return g.Key(g.inner.GenerateEventKey(eventID))
}

// VersionedCache reads and writes through versioned keys.
//
// It does NOT delete the previous generation's entries: they simply become
// unreachable and expire on their TTL. That is deliberate — a reorg must not
// have to enumerate and delete an unknown set of keys before it can serve
// correct data, and a sweep that fails would leave the stale entries reachable.
type VersionedCache struct {
	cache core.CachePlugin
	keys  *VersionedKeyGenerator
}

// NewVersionedCache creates a versioned view over a cache plugin.
func NewVersionedCache(c core.CachePlugin, keys *VersionedKeyGenerator) (*VersionedCache, error) {
	if c == nil {
		return nil, fmt.Errorf("cache plugin is required")
	}
	if keys == nil {
		return nil, fmt.Errorf("key generator is required")
	}
	return &VersionedCache{cache: c, keys: keys}, nil
}

// Get returns the value stored under the current scope's key for baseKey.
// found is false for a miss, including the case where a value exists only under
// a previous generation.
func (v *VersionedCache) Get(ctx context.Context, baseKey string) (value []byte, found bool, err error) {
	stored, err := v.cache.Get(ctx, v.keys.Key(baseKey))
	if err != nil {
		return nil, false, fmt.Errorf("versioned cache get: %w", err)
	}
	if stored == nil {
		return nil, false, nil
	}
	return stored, true, nil
}

// Set stores a value under the current scope's key for baseKey.
func (v *VersionedCache) Set(ctx context.Context, baseKey string, value []byte, ttl int) error {
	if err := v.cache.Set(ctx, v.keys.Key(baseKey), value, ttl); err != nil {
		return fmt.Errorf("versioned cache set: %w", err)
	}
	return nil
}
