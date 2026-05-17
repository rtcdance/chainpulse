package deployment

import (
	"context"
	"fmt"
	"os"
	"sync"

	"chainpulse/pkg/core"
)

// PluginFactory is a function that creates a plugin instance.
type PluginFactory func(ctx context.Context) (any, error)

// AdapterFactory creates plugins based on deployment mode.
// Plugins register themselves via RegisterFactory, eliminating
// the need for the factory to import concrete plugin packages.
type AdapterFactory struct {
	mode     DeploymentMode
	registry map[string]PluginFactory // key: "mq:memory", "cache:inmemory", "database:mock", etc.
	mu       sync.RWMutex
}

// NewAdapterFactory creates a new adapter factory
func NewAdapterFactory(mode DeploymentMode) *AdapterFactory {
	return &AdapterFactory{
		mode:     mode,
		registry: make(map[string]PluginFactory),
	}
}

// RegisterFactory registers a plugin factory for a given type key.
// Key format: "category:subtype" e.g. "mq:memory", "cache:inmemory", "database:mock"
func (f *AdapterFactory) RegisterFactory(key string, factory PluginFactory) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registry[key] = factory
}

// CreateMQPlugin creates an MQ plugin based on deployment mode and configuration.
func (f *AdapterFactory) CreateMQPlugin(ctx context.Context) (core.MQPlugin, error) {
	mqType := os.Getenv("CHAINPULSE_MQ_TYPE")
	if mqType == "" {
		mqType = "memory"
	}
	plugin, err := f.createPlugin(ctx, "mq", mqType)
	if err != nil {
		return nil, err
	}
	mq, ok := plugin.(core.MQPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin for mq:%s does not implement core.MQPlugin", mqType)
	}
	return mq, nil
}

// CreateCachePlugin creates a cache plugin based on deployment mode and configuration.
func (f *AdapterFactory) CreateCachePlugin(ctx context.Context) (core.CachePlugin, error) {
	cacheType := os.Getenv("CHAINPULSE_CACHE_TYPE")
	if cacheType == "" {
		cacheType = "inmemory"
	}
	plugin, err := f.createPlugin(ctx, "cache", cacheType)
	if err != nil {
		return nil, err
	}
	cache, ok := plugin.(core.CachePlugin)
	if !ok {
		return nil, fmt.Errorf("plugin for cache:%s does not implement core.CachePlugin", cacheType)
	}
	return cache, nil
}

// CreateDatabasePlugin creates a database plugin based on deployment mode and configuration.
func (f *AdapterFactory) CreateDatabasePlugin(ctx context.Context) (core.DatabasePlugin, error) {
	dbType := os.Getenv("CHAINPULSE_DATABASE_TYPE")
	if dbType == "" {
		dbType = "postgres"
	}
	plugin, err := f.createPlugin(ctx, "database", dbType)
	if err != nil {
		return nil, err
	}
	db, ok := plugin.(core.DatabasePlugin)
	if !ok {
		return nil, fmt.Errorf("plugin for database:%s does not implement core.DatabasePlugin", dbType)
	}
	return db, nil
}

func (f *AdapterFactory) createPlugin(ctx context.Context, category, subtype string) (any, error) {
	key := category + ":" + subtype
	f.mu.RLock()
	factory, ok := f.registry[key]
	f.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no plugin factory registered for %s (key: %s)", category, key)
	}

	plugin, err := factory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s plugin: %w", category, err)
	}
	return plugin, nil
}

// GetDeploymentModeFromEnv returns the deployment mode from environment
func GetDeploymentModeFromEnv() DeploymentMode {
	mode := os.Getenv("CHAINPULSE_DEPLOYMENT_MODE")
	switch mode {
	case "microservice":
		return MicroserviceMode
	default:
		return MonolithicMode
	}
}
