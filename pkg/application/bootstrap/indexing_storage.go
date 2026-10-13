package bootstrap

import (
	"context"
	"fmt"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"

	plugincache "github.com/rtcdance/chainpulse/pkg/plugins/cache"
	plugindatabase "github.com/rtcdance/chainpulse/pkg/plugins/database"
)

type monolithicIndexingStorageDeps struct {
	newDatabase func(logger core.Logger, config core.Config) core.DatabasePlugin
	newCache    func(config core.Config) core.CachePlugin
}

func defaultMonolithicIndexingStorageDeps() monolithicIndexingStorageDeps {
	return monolithicIndexingStorageDeps{
		newDatabase: newMonolithicIndexingDatabaseForMode,
		newCache:    newMonolithicIndexingCacheForConfig,
	}
}

type snapshotCompatibleDatabase struct {
	core.DatabasePlugin
	mu     sync.RWMutex
	blocks map[uint64]*blockchain.Block
}

func newSnapshotCompatibleDatabase(db core.DatabasePlugin) *snapshotCompatibleDatabase {
	return &snapshotCompatibleDatabase{
		DatabasePlugin: db,
		blocks:         make(map[uint64]*blockchain.Block),
	}
}

func (d *snapshotCompatibleDatabase) StoreBlockSnapshot(_ context.Context, block *blockchain.Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	copyBlock := *block
	d.blocks[block.Number] = &copyBlock
	return nil
}

func (d *snapshotCompatibleDatabase) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	if blocks, err := d.DatabasePlugin.GetAllBlocks(ctx); err == nil && len(blocks) > 0 {
		return blocks, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	results := make([]*blockchain.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		results = append(results, block)
	}
	return results, nil
}

func (d *snapshotCompatibleDatabase) GetBlock(ctx context.Context, blockNumber uint64) (*blockchain.Block, error) {
	if block, err := d.DatabasePlugin.GetBlock(ctx, blockNumber); err == nil && block != nil {
		return block, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.blocks[blockNumber], nil
}

// newMonolithicIndexingDatabaseForMode selects the indexing database.
//
// `DATABASE_TYPE=postgres` (the configured default) means the caller wants the
// port stack — the PostgreSQL adapter with chain scope and the atomic commit path
// — rather than the in-memory store. Until 2026-10-13 this function never
// returned it, which is why T02/T03/T05 were verified at the port but the port
// was never constructed in the deployed path.
//
// The in-memory store remains for a config that does not ask for postgres, so the
// zero-dependency playground keeps working.
func newMonolithicIndexingDatabaseForMode(logger core.Logger, config core.Config) core.DatabasePlugin {
	if config.DatabaseType == "postgres" && config.DatabaseURL != "" {
		adapter := plugindatabase.NewPostgreSQLPluginAdapter(logger, core.NewDefaultMetricsCollector())
		// I01: declare the chains this adapter serves. Declaring more than one
		// makes the unscoped port methods refuse to run, so a multi-chain
		// deployment cannot silently mix chains.
		if len(config.ActiveChains) > 1 {
			adapter.WithChainScope(config.ActiveChains...)
		}
		return newSnapshotCompatibleDatabase(adapter)
	}
	if config.DeploymentMode == "microservice" {
		return newSnapshotCompatibleDatabase(plugindatabase.NewMockDB())
	}
	return NewMonolithicMemoryDatabase(logger)
}

// newMonolithicIndexingCacheForConfig selects the indexing cache.
//
// `CACHE_TYPE=redis` means the Redis adapter; anything else keeps the in-memory
// cache. The adapter reports whether it actually reached Redis via Connected(),
// so a caller that cares can refuse to run against the fallback.
func newMonolithicIndexingCacheForConfig(config core.Config) core.CachePlugin {
	if config.CacheType == "redis" {
		return plugincache.NewRedisCachePluginAdapter(
			core.NewProductionLogger(),
			core.NewDefaultMetricsCollector(),
		)
	}
	return NewMonolithicMemoryCache()
}

// BuildMonolithicIndexingStorage creates started in-memory storage adapters for
// the monolithic indexing path.
func BuildMonolithicIndexingStorage(
	ctx context.Context,
	logger core.Logger,
	config core.Config,
) (core.DatabasePlugin, core.CachePlugin, error) {
	return buildMonolithicIndexingStorageWithDeps(ctx, logger, config, defaultMonolithicIndexingStorageDeps())
}

func buildMonolithicIndexingStorageWithDeps(
	ctx context.Context,
	logger core.Logger,
	config core.Config,
	deps monolithicIndexingStorageDeps,
) (core.DatabasePlugin, core.CachePlugin, error) {
	if logger == nil {
		return nil, nil, fmt.Errorf("logger is required")
	}

	database := deps.newDatabase(logger, config)
	cache := deps.newCache(config)

	if p, ok := database.(core.ConfigurablePlugin); ok {
		if err := p.Initialize(ctx, config); err != nil {
			return nil, nil, fmt.Errorf("initialize indexing database: %w", err)
		}
	}
	if p, ok := cache.(core.ConfigurablePlugin); ok {
		if err := p.Initialize(ctx, config); err != nil {
			return nil, nil, fmt.Errorf("initialize indexing cache: %w", err)
		}
	}
	if p, ok := database.(core.LifecyclePlugin); ok {
		if err := p.Start(ctx); err != nil {
			return nil, nil, fmt.Errorf("start indexing database: %w", err)
		}
	}
	if p, ok := cache.(core.LifecyclePlugin); ok {
		if err := p.Start(ctx); err != nil {
			if p, ok := database.(core.LifecyclePlugin); ok {
				_ = p.Stop(ctx)
			}
			return nil, nil, fmt.Errorf("start indexing cache: %w", err)
		}
	}

	return database, cache, nil
}
