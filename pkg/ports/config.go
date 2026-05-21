package ports

import "context"

// ConfigManager manages configuration
type ConfigManager interface {
	Load() (Config, error)
	Validate(config Config) error
	Get(key string) (any, error)
	Set(key string, value any) error
}

// HotReloadManager coordinates hot-reload of multiple plugins.
type HotReloadManager interface {
	RegisterPlugin(name string, plugin HotReloadablePlugin) error
	ReloadPlugin(ctx context.Context, name string, cfg Config) error
	ReloadAll(ctx context.Context, cfg Config) error
	GetPlugin(name string) (HotReloadablePlugin, error)
	ListPlugins() []string
}
