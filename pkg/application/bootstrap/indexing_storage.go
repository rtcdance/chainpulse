package bootstrap

import (
	"context"
	"fmt"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"

	plugindatabase "github.com/rtcdance/chainpulse/pkg/plugins/database"
)

type monolithicIndexingStorageDeps struct {
	newDatabase func(logger core.Logger, config core.Config) core.DatabasePlugin
	newCache    func() core.CachePlugin
}

func defaultMonolithicIndexingStorageDeps() monolithicIndexingStorageDeps {
	return monolithicIndexingStorageDeps{
		newDatabase: newMonolithicIndexingDatabaseForMode,
		newCache: func() core.CachePlugin {
			return NewMonolithicMemoryCache()
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

func (d *snapshotCompatibleDatabase) StoreBlockSnapshot(_ context.Context, block *core.Block) error {
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
	return NewMonolithicMemoryDatabase(logger)
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
	cache := deps.newCache()

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
