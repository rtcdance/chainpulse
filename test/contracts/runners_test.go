//go:build integration

// Package contracts holds the adapter contract suites. The helper functions
// (DatabaseContractTest, CacheContractTest) define the behaviour every
// implementation must satisfy; the Test... functions below bind them to
// concrete factories so the suite is actually executed (F02: previously this
// package contained only helpers, so `go test` reported zero tests and exited
// 0 — a false green).
package contracts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/cache"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// testConfig returns a minimal valid config for plugin lifecycle calls.
func testConfig() core.Config {
	return core.Config{
		BlockchainNodeURL: "http://localhost:8545",
		DataPullerType:    "https-jsonrpc",
		APIPort:           8080,
	}
}

// startDB drives the plugin lifecycle (Initialize when supported, then Start)
// and registers cleanup. The contract helpers assume a running plugin.
func startDB(t *testing.T, db core.DatabasePlugin) core.DatabasePlugin {
	t.Helper()
	ctx := context.Background()
	if init, ok := db.(ports.ConfigurablePlugin); ok {
		require.NoError(t, init.Initialize(ctx, testConfig()))
	}
	if lc, ok := db.(ports.LifecyclePlugin); ok {
		require.NoError(t, lc.Start(ctx))
		t.Cleanup(func() { _ = lc.Stop(context.Background()) })
	}
	return db
}

// startCache is startDB for the cache plugin.
func startCache(t *testing.T, c core.CachePlugin) core.CachePlugin {
	t.Helper()
	ctx := context.Background()
	if init, ok := c.(ports.ConfigurablePlugin); ok {
		require.NoError(t, init.Initialize(ctx, testConfig()))
	}
	if lc, ok := c.(ports.LifecyclePlugin); ok {
		require.NoError(t, lc.Start(ctx))
		t.Cleanup(func() { _ = lc.Stop(context.Background()) })
	}
	return c
}

// TestMonolithicMemoryDatabaseContract runs the database contract against the
// in-memory storage used by the zero-dependency playground / monolithic path.
func TestMonolithicMemoryDatabaseContract(t *testing.T) {
	t.Parallel()
	DatabaseContractTest(t, func(t *testing.T) core.DatabasePlugin {
		t.Helper()
		return startDB(t, bootstrap.NewMonolithicMemoryDatabase(testhelpers.NewTestLogger()))
	})
}

// TestMonolithicMemoryCacheContract runs the cache contract against the
// playground cache.
func TestMonolithicMemoryCacheContract(t *testing.T) {
	t.Parallel()
	CacheContractTest(t, func(t *testing.T) core.CachePlugin {
		t.Helper()
		return startCache(t, bootstrap.NewMonolithicMemoryCache())
	})
}

// TestInMemoryCacheContract runs the cache contract against the plugin cache
// used by the microservice paths.
func TestInMemoryCacheContract(t *testing.T) {
	t.Parallel()
	CacheContractTest(t, func(t *testing.T) core.CachePlugin {
		t.Helper()
		return startCache(t, cache.NewDefaultInMemoryCachePlugin(
			"test-cache", "1.0.0",
			testhelpers.NewTestLogger(),
			testhelpers.NewTestMetricsCollector(),
		))
	})
}

// TestPostgresDatabaseContract runs the database contract against the Postgres
// adapter (PostgreSQLPluginAdapter), which satisfies core.DatabasePlugin.
//
// Gated on a reachable DSN because the short lane must not require a database:
//
//	CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 \
//	CHAINPULSE_TEST_DATABASE_URL='postgres://user:pass@host:5432/db?sslmode=disable'
func TestPostgresDatabaseContract(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL to run the Postgres contract")
	}
	// The contract uses fixed ids and natural keys, so start from a clean table
	// to stay deterministic across runs.
	truncateBlockchainEvents(t, dsn)
	DatabaseContractTest(t, func(t *testing.T) core.DatabasePlugin {
		t.Helper()
		a := database.NewPostgreSQLPluginAdapter(
			testhelpers.NewTestLogger(),
			testhelpers.NewTestMetricsCollector(),
		)
		require.NoError(t, a.Initialize(context.Background(), core.Config{DatabaseURL: dsn}))
		require.NoError(t, a.Start(context.Background()))
		t.Cleanup(func() { _ = a.Stop(context.Background()) })
		return a
	})
}

// TestPortConformanceGaps is a tripwire, not an acceptance test.
//
// FINDING (T02): three of the four production/in-memory adapters do NOT
// implement the port interfaces the rest of the system is written against:
//
//  1. database.DefaultInMemoryDatabasePlugin — missing BatchStoreEvents.
//  2. database.PostgreSQLDatabase — exposes WriteEvent(ctx, *BlockchainEvent)
//     and GetEventByHash(hash); pkg/ports requires StoreEvent(ctx, any) and
//     GetEvent(ctx, id). Signature mismatch, not just a missing method.
//  3. cache.RedisCachePlugin — exposes Get(key) (*CacheEntry, error) and
//     Set(*CacheEntry); pkg/ports requires Get(ctx, key) ([]byte, error) and
//     Set(ctx, key, value, ttl).
//
// Consequence: the contract suite can only bind the monolithic in-memory
// implementations today, and §7.3's requirement "every required production
// factory has executed > 0 cases" cannot be met for Postgres/Redis until an
// adapter layer (or signature alignment) lands. This test fails on purpose the
// moment an adapter starts conforming, so its contract runner must be added in
// the same change.
func TestPortConformanceGaps(t *testing.T) {
	t.Parallel()

	logger := testhelpers.NewTestLogger()
	metrics := testhelpers.NewTestMetricsCollector()

	if _, ok := any(database.NewPostgreSQLDatabase(logger, metrics)).(core.DatabasePlugin); ok {
		t.Errorf("PostgreSQLDatabase now satisfies core.DatabasePlugin — " +
			"add a TestPostgresDatabaseContract runner binding this factory")
	}
	if _, ok := any(cache.NewRedisCachePlugin(logger, metrics)).(core.CachePlugin); ok {
		t.Errorf("RedisCachePlugin now satisfies core.CachePlugin — " +
			"add a TestRedisCacheContract runner binding this factory")
	}
	if _, ok := any(database.NewDefaultInMemoryDatabasePlugin(logger, metrics)).(core.DatabasePlugin); ok {
		t.Errorf("DefaultInMemoryDatabasePlugin now satisfies core.DatabasePlugin — " +
			"add a TestInMemoryDatabaseContract runner binding this factory")
	}

	// Sanity: the implementations the runners DO bind must conform, otherwise
	// those contract runners would be meaningless.
	_, monoDBOK := any(bootstrap.NewMonolithicMemoryDatabase(logger)).(core.DatabasePlugin)
	assert.True(t, monoDBOK, "MonolithicMemoryDatabase must satisfy core.DatabasePlugin")

	_, monoCacheOK := any(bootstrap.NewMonolithicMemoryCache()).(core.CachePlugin)
	assert.True(t, monoCacheOK, "MonolithicMemoryCache must satisfy core.CachePlugin")

	_, pluginCacheOK := any(cache.NewDefaultInMemoryCachePlugin("c", "1.0.0", logger, metrics)).(core.CachePlugin)
	assert.True(t, pluginCacheOK, "DefaultInMemoryCachePlugin must satisfy core.CachePlugin")

	// The Postgres adapter layer DOES satisfy the port (T02 "adapter layer").
	_, pgAdapterOK := any(database.NewPostgreSQLPluginAdapter(logger, metrics)).(core.DatabasePlugin)
	assert.True(t, pgAdapterOK, "PostgreSQLPluginAdapter must satisfy core.DatabasePlugin")
}
