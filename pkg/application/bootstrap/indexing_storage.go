package bootstrap

import (
	"fmt"

	indexingadapter "chainpulse/pkg/adapters/indexing"
	"chainpulse/pkg/core"
)

type monolithicIndexingStorageDeps struct {
	newDatabase func(logger core.Logger) core.DatabasePlugin
	newCache    func() core.CachePlugin
}

func defaultMonolithicIndexingStorageDeps() monolithicIndexingStorageDeps {
	return monolithicIndexingStorageDeps{
		newDatabase: func(logger core.Logger) core.DatabasePlugin {
			return indexingadapter.NewMonolithicMemoryDatabase(logger)
		},
		newCache: func() core.CachePlugin {
			return indexingadapter.NewMonolithicMemoryCache()
		},
	}
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

	database := deps.newDatabase(logger)
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
