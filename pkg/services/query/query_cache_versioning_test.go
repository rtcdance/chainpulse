package query

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
)

// versionedCacheMock stores entries keyed by the FULL cache key so a test can
// observe exactly which key a value was written under and prove that a
// generation bump makes the previous key unreachable (T11 acceptance: after
// invalidation, no old canonical result is returned).
type versionedCacheMock struct {
	mu      sync.RWMutex
	results map[string]cachedResultEntry
	singles map[string]*blockchain.BlockchainEvent
	setKeys []string
	getKeys []string
}

type cachedResultEntry struct {
	events []blockchain.BlockchainEvent
	total  int64
}

func newVersionedCacheMock() *versionedCacheMock {
	return &versionedCacheMock{
		results: make(map[string]cachedResultEntry),
		singles: make(map[string]*blockchain.BlockchainEvent),
	}
}

func (m *versionedCacheMock) Initialize(context.Context) error { return nil }
func (m *versionedCacheMock) Start(context.Context) error      { return nil }
func (m *versionedCacheMock) Stop(context.Context) error       { return nil }

func (m *versionedCacheMock) Get(_ context.Context, key string) ([]blockchain.BlockchainEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getKeys = append(m.getKeys, key)
	if e, ok := m.results[key]; ok {
		return e.events, nil
	}
	return nil, nil
}

func (m *versionedCacheMock) GetSingle(_ context.Context, key string) (*blockchain.BlockchainEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getKeys = append(m.getKeys, key)
	if e, ok := m.singles[key]; ok {
		return e, nil
	}
	return nil, nil
}

func (m *versionedCacheMock) Set(_ context.Context, key string, value []blockchain.BlockchainEvent, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setKeys = append(m.setKeys, key)
	m.results[key] = cachedResultEntry{events: value}
	return nil
}

func (m *versionedCacheMock) SetSingle(_ context.Context, key string, value *blockchain.BlockchainEvent, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setKeys = append(m.setKeys, key)
	if value != nil {
		m.singles[key] = value
	}
	return nil
}

func (m *versionedCacheMock) SetQueryResult(_ context.Context, key string, events []blockchain.BlockchainEvent, total int64, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setKeys = append(m.setKeys, key)
	m.results[key] = cachedResultEntry{events: events, total: total}
	return nil
}

func (m *versionedCacheMock) GetQueryResult(_ context.Context, key string) ([]blockchain.BlockchainEvent, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getKeys = append(m.getKeys, key)
	if e, ok := m.results[key]; ok {
		return e.events, e.total, nil
	}
	return nil, 0, nil
}

func (m *versionedCacheMock) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.results, key)
	delete(m.singles, key)
	return nil
}

func (m *versionedCacheMock) Health(context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy"}
}

func (m *versionedCacheMock) setKeysSnapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.setKeys))
	copy(out, m.setKeys)
	return out
}

func (m *versionedCacheMock) getKeysSnapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.getKeys))
	copy(out, m.getKeys)
	return out
}

func containsPrefix(keys []string, prefix string) bool {
	for _, k := range keys {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// filterChainLabel extracts a chain marker from a filter the way the adapter
// sees it; test helpers only understand the shapes the production handler
// builds (scalar string or $in array).
func filterChainLabel(filter map[string]any) string {
	if v, ok := filter["chainId"]; ok {
		switch t := v.(type) {
		case string:
			return t
		case map[string]any:
			if in, ok := t["$in"].([]any); ok && len(in) > 0 {
				return fmt.Sprintf("%v", in[0])
			}
		}
	}
	if v, ok := filter["network"].(string); ok {
		return v
	}
	return "none"
}

// newGenerationEchoMongo returns a mongo adapter whose result always reflects
// the CURRENT generation: the adapter is the source of truth, so a stale cache
// hit is detectable by the returned event ID.
func newGenerationEchoMongo(generation, calls *atomic.Int64) *configurableMockAdapter {
	return &configurableMockAdapter{
		healthy: true,
		queryFn: func(_ context.Context, req *QueryRequest) (*QueryResult, error) {
			calls.Add(1)
			return &QueryResult{
				Events: []blockchain.BlockchainEvent{
					{ID: fmt.Sprintf("ev-g%d:%s", generation.Load(), filterChainLabel(req.Filter))},
				},
				Total: 1,
			}, nil
		},
	}
}

func newVersionedTestService(t *testing.T, mongo MongoDBAdapter, cache CacheService) *DefaultQueryService {
	t.Helper()
	return NewQueryService(mongo, &configurableMockAdapter{healthy: true}, cache, &MockLogger{}, NewMockMetricsCollector())
}

func newTestVersioner(t *testing.T, generation *atomic.Int64) *plugincache.LiveScopeResolver {
	t.Helper()
	versioner, err := plugincache.NewLiveScopeResolver(func(context.Context, string) (int64, error) {
		return generation.Load(), nil
	}, "1")
	if err != nil {
		t.Fatalf("NewLiveScopeResolver: %v", err)
	}
	return versioner
}

// T11 core acceptance: after a reorg generation bump, the same request must
// NOT be served the result cached under the previous generation.
func TestQueryService_VersionedCache_ServesFreshDataAfterReorgGenerationBump(t *testing.T) {
	t.Parallel()

	var generation, calls atomic.Int64
	cache := newVersionedCacheMock()
	mongo := newGenerationEchoMongo(&generation, &calls)
	svc := newVersionedTestService(t, mongo, cache)
	svc.SetCacheScopeVersioner(newTestVersioner(t, &generation))
	initQueryService(t, svc)

	ctx := context.Background()
	req := &QueryRequest{
		CacheKey: "events:list",
		Filter:   map[string]any{"chainId": "1"},
		Limit:    10,
	}

	first, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	if first.CacheHit {
		t.Fatalf("first query unexpectedly served from cache")
	}
	if len(first.Events) != 1 || first.Events[0].ID != "ev-g0:1" {
		t.Fatalf("first query events = %+v, want ev-g0:1", first.Events)
	}

	generation.Store(1)

	second, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if second.CacheHit {
		t.Fatalf("stale hit: served cached result after generation bump")
	}
	if len(second.Events) != 1 || second.Events[0].ID != "ev-g1:1" {
		t.Fatalf("second query events = %+v, want ev-g1:1 (fresh canonical data)", second.Events)
	}

	setKeys := cache.setKeysSnapshot()
	if !containsPrefix(setKeys, "v1:c1:g0:events:list") {
		t.Errorf("set keys = %v, want entry under v1:c1:g0:events:list", setKeys)
	}
	if !containsPrefix(setKeys, "v1:c1:g1:events:list") {
		t.Errorf("set keys = %v, want entry under v1:c1:g1:events:list after bump", setKeys)
	}
}

// I01 chain isolation: two chains using the SAME base key must never share a
// cache entry.
func TestQueryService_VersionedCache_SeparatesChainsOnSameBaseKey(t *testing.T) {
	t.Parallel()

	var generation, calls atomic.Int64
	cache := newVersionedCacheMock()
	mongo := newGenerationEchoMongo(&generation, &calls)
	svc := newVersionedTestService(t, mongo, cache)
	svc.SetCacheScopeVersioner(newTestVersioner(t, &generation))
	initQueryService(t, svc)

	ctx := context.Background()
	reqEth := &QueryRequest{
		CacheKey: "shared-key",
		Filter:   map[string]any{"chainId": "1"},
		Limit:    10,
	}
	reqPolygon := &QueryRequest{
		CacheKey: "shared-key",
		Filter:   map[string]any{"chainId": "137"},
		Limit:    10,
	}

	eth, err := svc.Query(ctx, reqEth)
	if err != nil {
		t.Fatalf("ethereum query: %v", err)
	}
	if len(eth.Events) != 1 || eth.Events[0].ID != "ev-g0:1" {
		t.Fatalf("ethereum events = %+v, want ev-g0:1", eth.Events)
	}

	polygon, err := svc.Query(ctx, reqPolygon)
	if err != nil {
		t.Fatalf("polygon query: %v", err)
	}
	if polygon.CacheHit {
		t.Fatalf("chain isolation broken: polygon served ethereum's cached entry")
	}
	if len(polygon.Events) != 1 || polygon.Events[0].ID != "ev-g0:137" {
		t.Fatalf("polygon events = %+v, want ev-g0:137", polygon.Events)
	}

	setKeys := cache.setKeysSnapshot()
	if !containsPrefix(setKeys, "v1:c1:g0:shared-key") {
		t.Errorf("set keys = %v, want v1:c1:g0:shared-key", setKeys)
	}
	if !containsPrefix(setKeys, "v1:c137:g0:shared-key") {
		t.Errorf("set keys = %v, want v1:c137:g0:shared-key", setKeys)
	}
}

// The handler emits numeric chains as a $in array (numeric, numeric-string,
// canonical name). All aliases resolving to one chain must hit the SAME
// versioned entry; a $in spanning multiple chains must NOT.
func TestQueryService_VersionedCache_NormalizesChainInFilter(t *testing.T) {
	t.Parallel()

	var generation, calls atomic.Int64
	cache := newVersionedCacheMock()
	mongo := newGenerationEchoMongo(&generation, &calls)
	svc := newVersionedTestService(t, mongo, cache)
	svc.SetCacheScopeVersioner(newTestVersioner(t, &generation))
	initQueryService(t, svc)

	ctx := context.Background()

	scalar := &QueryRequest{
		CacheKey: "events:chain",
		Filter:   map[string]any{"chainId": "137"},
		Limit:    10,
	}
	if _, err := svc.Query(ctx, scalar); err != nil {
		t.Fatalf("scalar query: %v", err)
	}

	aliased := &QueryRequest{
		CacheKey: "events:chain",
		Filter:   map[string]any{"chainId": map[string]any{"$in": []any{float64(137), "137", "polygon"}}},
		Limit:    10,
	}
	hit, err := svc.Query(ctx, aliased)
	if err != nil {
		t.Fatalf("aliased query: %v", err)
	}
	if !hit.CacheHit {
		t.Fatalf("$in aliases of chain 137 did not normalize to the scalar entry (set keys: %v)", cache.setKeysSnapshot())
	}

	mixed := &QueryRequest{
		CacheKey: "events:chain",
		Filter:   map[string]any{"chainId": map[string]any{"$in": []any{float64(1), float64(137)}}},
		Limit:    10,
	}
	mixedResult, err := svc.Query(ctx, mixed)
	if err != nil {
		t.Fatalf("mixed query: %v", err)
	}
	if mixedResult.CacheHit {
		t.Fatalf("multi-chain $in shared a single-chain cache entry")
	}
}

// Cold and hot executions of the same request must return byte-identical
// event payloads (only meta fields differ).
func TestQueryService_VersionedCache_HotColdPayloadByteIdentical(t *testing.T) {
	t.Parallel()

	var generation, calls atomic.Int64
	cache := newVersionedCacheMock()
	mongo := newGenerationEchoMongo(&generation, &calls)
	svc := newVersionedTestService(t, mongo, cache)
	svc.SetCacheScopeVersioner(newTestVersioner(t, &generation))
	initQueryService(t, svc)

	ctx := context.Background()
	req := &QueryRequest{
		CacheKey: "events:identical",
		Filter:   map[string]any{"chainId": "1"},
		Limit:    10,
	}

	cold, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("cold query: %v", err)
	}
	hot, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("hot query: %v", err)
	}
	if !hot.CacheHit {
		t.Fatalf("second query was not a cache hit (keys: %v)", cache.getKeysSnapshot())
	}

	coldJSON, err := json.Marshal(cold.Events)
	if err != nil {
		t.Fatalf("marshal cold events: %v", err)
	}
	hotJSON, err := json.Marshal(hot.Events)
	if err != nil {
		t.Fatalf("marshal hot events: %v", err)
	}
	if string(coldJSON) != string(hotJSON) {
		t.Errorf("hot/cold payloads differ:\ncold: %s\nhot:  %s", coldJSON, hotJSON)
	}
	if cold.Total != hot.Total {
		t.Errorf("hot/cold totals differ: cold=%d hot=%d", cold.Total, hot.Total)
	}
}

// Fail-closed: when the scope cannot be resolved, the query must bypass the
// cache entirely (never read, never write) instead of risking a stale hit.
func TestQueryService_VersionedCache_VersionKeyErrorBypassesCache(t *testing.T) {
	t.Parallel()

	var generation, calls atomic.Int64
	cache := newVersionedCacheMock()
	mongo := newGenerationEchoMongo(&generation, &calls)
	svc := newVersionedTestService(t, mongo, cache)
	versioner, err := plugincache.NewLiveScopeResolver(func(context.Context, string) (int64, error) {
		return 0, fmt.Errorf("reorg_generations unreachable")
	}, "1")
	if err != nil {
		t.Fatalf("NewLiveScopeResolver: %v", err)
	}
	svc.SetCacheScopeVersioner(versioner)
	initQueryService(t, svc)

	ctx := context.Background()
	req := &QueryRequest{
		CacheKey: "events:bypass",
		Filter:   map[string]any{"chainId": "1"},
		Limit:    10,
	}

	first, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("first query: %v", err)
	}
	if first.CacheHit {
		t.Fatalf("cache hit despite versioner error")
	}
	second, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if second.CacheHit {
		t.Fatalf("second query hit cache despite versioner error")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("adapter calls = %d, want 2 (both queries must hit the adapter)", got)
	}
	if setKeys := cache.setKeysSnapshot(); len(setKeys) != 0 {
		t.Errorf("set keys = %v, want none (cache writes must be bypassed)", setKeys)
	}
}

// Regression guard: with no versioner wired (nil), the legacy raw key path is
// unchanged — existing deployments keep their current behavior.
func TestQueryService_CacheKeyLegacyPassthroughWithoutVersioner(t *testing.T) {
	t.Parallel()

	var generation, calls atomic.Int64
	cache := newVersionedCacheMock()
	mongo := newGenerationEchoMongo(&generation, &calls)
	svc := newVersionedTestService(t, mongo, cache)
	initQueryService(t, svc)

	ctx := context.Background()
	req := &QueryRequest{
		CacheKey: "raw-key",
		Filter:   map[string]any{"chainId": "1"},
		Limit:    10,
	}

	if _, err := svc.Query(ctx, req); err != nil {
		t.Fatalf("first query: %v", err)
	}
	setKeys := cache.setKeysSnapshot()
	if len(setKeys) != 1 || setKeys[0] != "raw-key" {
		t.Errorf("set keys = %v, want exactly [raw-key] when no versioner is wired", setKeys)
	}

	hot, err := svc.Query(ctx, req)
	if err != nil {
		t.Fatalf("second query: %v", err)
	}
	if !hot.CacheHit {
		t.Errorf("legacy path lost its cache hit")
	}
}

// QueryByHash participates in versioning under the global scope (a hash lookup
// is not chain-scoped): a generation bump must make the single-event entry
// unreachable.
func TestQueryService_VersionedCache_QueryByHashAfterGenerationBump(t *testing.T) {
	t.Parallel()

	var generation atomic.Int64
	cache := newVersionedCacheMock()
	mongo := &configurableMockAdapter{
		healthy: true,
		queryByHashFn: func(context.Context, string) (*blockchain.BlockchainEvent, error) {
			return &blockchain.BlockchainEvent{ID: fmt.Sprintf("hash-g%d", generation.Load())}, nil
		},
	}
	svc := newVersionedTestService(t, mongo, cache)
	svc.SetCacheScopeVersioner(newTestVersioner(t, &generation))
	initQueryService(t, svc)

	ctx := context.Background()

	first, err := svc.QueryByHash(ctx, "0xabc")
	if err != nil {
		t.Fatalf("first query by hash: %v", err)
	}
	if first.ID != "hash-g0" {
		t.Fatalf("first event ID = %q, want hash-g0", first.ID)
	}
	if setKeys := cache.setKeysSnapshot(); !containsPrefix(setKeys, "v1:cglobal:g0:event:0xabc") {
		t.Errorf("set keys = %v, want v1:cglobal:g0:event:0xabc", setKeys)
	}

	generation.Store(1)

	second, err := svc.QueryByHash(ctx, "0xabc")
	if err != nil {
		t.Fatalf("second query by hash: %v", err)
	}
	if second.ID != "hash-g1" {
		t.Fatalf("stale hit: second event ID = %q, want hash-g1 (fresh)", second.ID)
	}
	if setKeys := cache.setKeysSnapshot(); !containsPrefix(setKeys, "v1:cglobal:g1:event:0xabc") {
		t.Errorf("set keys = %v, want v1:cglobal:g1:event:0xabc after bump", setKeys)
	}
}
