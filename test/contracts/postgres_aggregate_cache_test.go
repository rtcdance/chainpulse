//go:build integration

package contracts

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type aggregateCacheFixture struct {
	mu     sync.Mutex
	values map[string][]byte
	sets   int
	fail   bool
}

func (c *aggregateCacheFixture) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return nil, fmt.Errorf("injected cache outage")
	}
	return append([]byte(nil), c.values[key]...), nil
}

func (c *aggregateCacheFixture) Set(_ context.Context, key string, value []byte, _ int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return fmt.Errorf("injected cache outage")
	}
	c.sets++
	c.values[key] = append([]byte(nil), value...)
	return nil
}

func (c *aggregateCacheFixture) Close() error { return nil }

type coldMissBarrier struct {
	*aggregateCacheFixture
	lookups atomic.Int64
	ready   chan struct{}
}

func (c *coldMissBarrier) Get(ctx context.Context, key string) ([]byte, error) {
	if index := c.lookups.Add(1); index <= 4 {
		if index == 4 {
			close(c.ready)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.ready:
			return nil, nil
		}
	}
	return c.aggregateCacheFixture.Get(ctx, key)
}

func TestPostgresConcurrentColdAggregatesReadOriginOnce(t *testing.T) {
	t.Parallel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set PostgreSQL integration gate with isolated endpoint")
	}
	for _, operation := range []string{"count", "dashboard"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			chain := "t11-cold-concurrent-" + operation
			db := openContractDBForReturn(t, dsn)
			_, err := db.ExecContext(ctx, `DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
			require.NoError(t, err)
			metrics := core.NewDefaultMetricsCollector()
			store := database.NewPostgreSQLPluginAdapter(testhelpers.NewTestLogger(), metrics).WithChainScope(chain)
			require.NoError(t, store.Initialize(ctx, core.Config{DatabaseURL: dsn}))
			require.NoError(t, store.Start(ctx))
			t.Cleanup(func() { require.NoError(t, store.Stop(context.Background())) })
			require.NoError(t, store.StoreEvent(ctx, txTestEvent(chain, 1)))
			cache := &coldMissBarrier{aggregateCacheFixture: &aggregateCacheFixture{values: make(map[string][]byte)}, ready: make(chan struct{})}
			store.WithCanonicalAggregateCache(cache)
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					if operation == "count" {
						result, err := store.QueryDomainEvents(ctx, &domainquery.Request{Limit: 10})
						require.NoError(t, err)
						if err == nil {
							require.EqualValues(t, 1, result.Total)
						}
					} else {
						total, _, _, _, err := store.CanonicalEventStats(ctx)
						require.NoError(t, err)
						require.EqualValues(t, 1, total)
					}
				})
			}
			wg.Wait()
			require.NoError(t, ctx.Err())
			require.EqualValues(t, 1, metrics.GetCounter("canonical_aggregate_origin_read", map[string]string{"operation": operation}), "four forced misses must execute one aggregate source read")
		})
	}
}

func TestPostgresAggregateStatsCacheUsesMutationRevision(t *testing.T) {
	t.Parallel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set PostgreSQL integration gate with isolated endpoint")
	}
	ctx := context.Background()
	const chainA, chainB = "t11-stats-cache-a", "t11-stats-cache-b"
	db := openContractDBForReturn(t, dsn)
	_, err := db.ExecContext(ctx, `DELETE FROM blockchain_events WHERE chain_id IN ($1,$2)`, chainA, chainB)
	require.NoError(t, err)
	backend := &aggregateCacheFixture{values: make(map[string][]byte)}
	store := newAdapter(t, dsn).WithChainScope(chainA).WithCanonicalAggregateCache(backend)
	read := func(total, reorged int64, chains, names map[string]int64) {
		t.Helper()
		got, byChain, byName, orphaned, err := store.CanonicalEventStats(ctx)
		require.NoError(t, err)
		require.Equal(t, total, got)
		require.Equal(t, reorged, orphaned)
		require.Equal(t, chains, byChain)
		require.Equal(t, names, byName)
	}
	read(0, 0, map[string]int64{}, map[string]int64{})
	require.Equal(t, 1, backend.sets)
	read(0, 0, map[string]int64{}, map[string]int64{})
	require.Equal(t, 1, backend.sets, "zero aggregates are valid cache hits")
	eventA, eventB := txTestEvent(chainA, 1), txTestEvent(chainB, 1)
	eventA.EventName, eventB.EventName = "Transfer", "Swap"
	require.NoError(t, store.StoreEvent(ctx, eventA))
	require.NoError(t, store.StoreEvent(ctx, eventB))
	read(1, 0, map[string]int64{chainA: 1}, map[string]int64{"Transfer": 1})
	store.WithChainScope(chainB, chainA)
	read(2, 0, map[string]int64{chainA: 1, chainB: 1}, map[string]int64{"Transfer": 1, "Swap": 1})
	sets := backend.sets
	store.WithChainScope(chainA, chainB)
	read(2, 0, map[string]int64{chainA: 1, chainB: 1}, map[string]int64{"Transfer": 1, "Swap": 1})
	require.Equal(t, sets, backend.sets, "scope ordering cannot create a different meaning")
	_, err = db.ExecContext(ctx, `UPDATE blockchain_events SET status='reorged' WHERE chain_id=$1`, chainA)
	require.NoError(t, err)
	read(1, 1, map[string]int64{chainB: 1}, map[string]int64{"Swap": 1})
	backend.mu.Lock()
	for key := range backend.values {
		backend.values[key] = []byte(`{"Total":1,"Reorged":0,"ByChain":{"outside":1},"ByName":{"Swap":1}}`)
	}
	backend.mu.Unlock()
	read(1, 1, map[string]int64{chainB: 1}, map[string]int64{"Swap": 1})
	backend.mu.Lock()
	backend.fail = true
	backend.mu.Unlock()
	read(1, 1, map[string]int64{chainB: 1}, map[string]int64{"Swap": 1})
}

type meteredAggregateCache struct {
	database.CanonicalAggregateCache
	hits atomic.Int64
}

func (c *meteredAggregateCache) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.CanonicalAggregateCache.Get(ctx, key)
	if err == nil && len(value) > 0 {
		c.hits.Add(1)
	}
	return value, err
}

func TestPostgresAggregateCacheRealRedis(t *testing.T) {
	t.Parallel()
	dsn, endpoint := postgresDSN(), redisCacheURL()
	if dsn == "" || endpoint == "" {
		t.Skip("set PostgreSQL and Redis integration gates with isolated endpoints")
	}
	ctx := context.Background()
	const chain = "t11-real-query-cache"
	db := openContractDBForReturn(t, dsn)
	_, err := db.ExecContext(ctx, `DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	backend, err := plugincache.NewCanonicalAggregateRedisCache(endpoint)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, backend.Close()) })
	metered := &meteredAggregateCache{CanonicalAggregateCache: backend}
	store := newAdapter(t, dsn).WithChainScope(chain).WithCanonicalAggregateCache(metered)
	require.NoError(t, store.StoreEvent(ctx, txTestEvent(chain, 1)))
	for range 2 {
		result, err := store.QueryDomainEvents(ctx, &domainquery.Request{Limit: 50})
		require.NoError(t, err)
		require.EqualValues(t, 1, result.Total)
	}
	require.EqualValues(t, 1, metered.hits.Load(), "second count must actually come from Redis")
	_, err = db.ExecContext(ctx, `DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	result, err := store.QueryDomainEvents(ctx, &domainquery.Request{Limit: 50})
	require.NoError(t, err)
	require.Zero(t, result.Total)
	require.Empty(t, result.Events)
	require.EqualValues(t, 1, metered.hits.Load(), "deleted data cannot reuse old cached count")
}

func TestPostgresAggregateCacheUsesMutationRevision(t *testing.T) {
	t.Parallel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	const chain = "t11-query-cache"
	db := openContractDBForReturn(t, dsn)
	_, err := db.ExecContext(ctx, `DELETE FROM blockchain_events WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	backend := &aggregateCacheFixture{values: make(map[string][]byte)}
	store := newAdapter(t, dsn).WithChainScope(chain).WithCanonicalAggregateCache(backend)
	read := func(want int64) {
		t.Helper()
		result, err := store.QueryDomainEvents(ctx, &domainquery.Request{Filter: map[string]any{}, Limit: 50})
		require.NoError(t, err)
		require.Equal(t, want, result.Total)
		require.Len(t, result.Events, int(want), "pages always come from the same PG snapshot")
	}
	read(0)
	require.Equal(t, 1, backend.sets)
	read(0)
	require.Equal(t, 1, backend.sets, "second read should use cached exact count")
	event := txTestEvent(chain, 1)
	require.NoError(t, store.StoreEvent(ctx, event))
	read(1)
	require.Equal(t, 2, backend.sets, "first insertion invalidates the absent-chain revision")
	read(1)
	require.Equal(t, 2, backend.sets)
	// A direct legacy writer does not advance checkpoint or reorg generation.
	_, err = db.ExecContext(ctx, `UPDATE blockchain_events SET status='reorged' WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	read(0)
	require.Equal(t, 3, backend.sets)
	backend.mu.Lock()
	for key := range backend.values {
		backend.values[key] = []byte("invalid count payload")
	}
	backend.mu.Unlock()
	read(0)
	require.Equal(t, 4, backend.sets, "malformed cache falls back to PG")
	backend.mu.Lock()
	backend.fail = true
	backend.mu.Unlock()
	read(0)
	_, err = db.ExecContext(ctx, `UPDATE blockchain_events SET status='confirmed' WHERE chain_id=$1`, chain)
	require.NoError(t, err)
	read(1)
}
