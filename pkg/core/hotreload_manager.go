package core

import (
	"context"
	"fmt"
	"sync"
)

// DefaultHotReloadManager implements HotReloadManager.
type DefaultHotReloadManager struct {
	mu      sync.RWMutex
	plugins map[string]HotReloadablePlugin
	logger  Logger
}

// NewDefaultHotReloadManager creates a hot-reload manager.
func NewDefaultHotReloadManager(logger Logger) *DefaultHotReloadManager {
	return &DefaultHotReloadManager{
		plugins: make(map[string]HotReloadablePlugin),
		logger:  logger,
	}
}

// RegisterPlugin registers a plugin for hot-reload management.
func (m *DefaultHotReloadManager) RegisterPlugin(name string, plugin HotReloadablePlugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %q is already registered", name)
	}
	m.plugins[name] = plugin
	if m.logger != nil {
		m.logger.Info("hot-reload plugin registered", "name", name, "reloadable", plugin.IsReloadable())
	}
	return nil
}

// ReloadPlugin triggers hot-reload of a single plugin.
func (m *DefaultHotReloadManager) ReloadPlugin(ctx context.Context, name string, cfg Config) error {
	m.mu.RLock()
	plugin, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %q not found", name)
	}
	if !plugin.IsReloadable() {
		return fmt.Errorf("plugin %q does not support hot-reload", name)
	}

	if m.logger != nil {
		m.logger.Info("reloading plugin", "name", name)
	}

	success, err := plugin.Reload(ctx, cfg)
	if err != nil {
		return fmt.Errorf("plugin %q reload failed: %w", name, err)
	}
	if !success {
		return fmt.Errorf("plugin %q reload returned false (partial failure)", name)
	}

	if m.logger != nil {
		m.logger.Info("plugin reloaded successfully", "name", name)
	}
	return nil
}

// ReloadAll triggers hot-reload of all registered plugins.
func (m *DefaultHotReloadManager) ReloadAll(ctx context.Context, cfg Config) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	m.mu.RUnlock()

	var lastErr error
	for _, name := range names {
		if err := m.ReloadPlugin(ctx, name, cfg); err != nil {
			lastErr = err
			if m.logger != nil {
				m.logger.Error("failed to reload plugin", "name", name, "error", err.Error())
			}
		}
	}
	return lastErr
}

// GetPlugin returns a registered plugin by name.
func (m *DefaultHotReloadManager) GetPlugin(name string) (HotReloadablePlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %q not found", name)
	}
	return plugin, nil
}

// ListPlugins returns names of all registered plugins.
func (m *DefaultHotReloadManager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}
