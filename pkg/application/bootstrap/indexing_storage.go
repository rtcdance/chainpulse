package bootstrap

import (
	"context"
	"fmt"
	"sync"

	"chainpulse/pkg/core"

	indexingadapter "chainpulse/pkg/adapters/indexing"

	plugindatabase "chainpulse/pkg/plugins/database"
)

type monolithicIndexingStorageDeps struct {
	newDatabase func(logger core.Logger, config core.Config) core.DatabasePlugin
	newCache    func() core.CachePlugin
}

func defaultMonolithicIndexingStorageDeps() monolithicIndexingStorageDeps {
	return monolithicIndexingStorageDeps{
		newDatabase: func(logger core.Logger, config core.Config) core.DatabasePlugin {
			return newMonolithicIndexingDatabaseForMode(logger, config)
		},
		newCache: func() core.CachePlugin {
			return indexingadapter.NewMonolithicMemoryCache()
		},
	}
}

type snapshotCompatibleDatabase struct {
	core.DatabasePlugin
	mu     sync.RWMutex
	blocks map[uint64]*core.Block
}

func newSnapshotCompatibleDatabase(db core.DatabasePlugin) *snapshotCompatibleDatabase {
	return &snapshotCompatibleDatabase{
		DatabasePlugin: db,
		blocks:         make(map[uint64]*core.Block),
	}
}

func (d *snapshotCompatibleDatabase) StoreBlockSnapshot(ctx context.Context, block *core.Block) error {
	_ = ctx
	if block == nil {
		return fmt.Errorf("block is nil")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	copyBlock := *block
	d.blocks[block.Number] = &copyBlock
	return nil
}

func (d *snapshotCompatibleDatabase) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	if blocks, err := d.DatabasePlugin.GetAllBlocks(ctx); err == nil && len(blocks) > 0 {
		return blocks, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	results := make([]*core.Block, 0, len(d.blocks))
	for _, block := range d.blocks {
		results = append(results, block)
	}
	return results, nil
}

func (d *snapshotCompatibleDatabase) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	if block, err := d.DatabasePlugin.GetBlock(ctx, blockNumber); err == nil && block != nil {
		return block, nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.blocks[blockNumber], nil
}

func newMonolithicIndexingDatabaseForMode(logger core.Logger, config core.Config) core.DatabasePlugin {
	if config.DeploymentMode == "microservice" {
		return newSnapshotCompatibleDatabase(plugindatabase.NewMockDB())
	}
	return indexingadapter.NewMonolithicMemoryDatabase(logger)
}

// BuildMonolithicIndexingStorage creates started in-memory storage adapters for
// the monolithic indexing path.
func BuildMonolithicIndexingStorage(
	logger core.Logger,
	config core.Config,
) (core.DatabasePlugin, core.CachePlugin, error) {
	return buildMonolithicIndexingStorageWithDeps(logger, config, defaultMonolithicIndexingStorageDeps())
}

func buildMonolithicIndexingStorageWithDeps(
	logger core.Logger,
	config core.Config,
	deps monolithicIndexingStorageDeps,
) (core.DatabasePlugin, core.CachePlugin, error) {
	if logger == nil {
		return nil, nil, fmt.Errorf("logger is required")
	}

	database := deps.newDatabase(logger, config)
	cache := deps.newCache()

	if err := database.Initialize(config); err != nil {
		return nil, nil, fmt.Errorf("initialize indexing database: %w", err)
	}
	if err := cache.Initialize(config); err != nil {
		return nil, nil, fmt.Errorf("initialize indexing cache: %w", err)
	}
	if err := database.Start(); err != nil {
		return nil, nil, fmt.Errorf("start indexing database: %w", err)
	}
	if err := cache.Start(); err != nil {
		_ = database.Stop()
		return nil, nil, fmt.Errorf("start indexing cache: %w", err)
	}

	return database, cache, nil
}
