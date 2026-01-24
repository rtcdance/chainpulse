package core

import (
	"fmt"
	"sync"
)

// DefaultPluginRegistry is the default implementation of PluginRegistry
type DefaultPluginRegistry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
	logger  Logger
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(logger Logger) *DefaultPluginRegistry {
	return &DefaultPluginRegistry{
		plugins: make(map[string]Plugin),
		logger:  logger,
	}
}

// Register registers a plugin
func (r *DefaultPluginRegistry) Register(plugin Plugin) error {
	if plugin == nil {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"plugin cannot be nil",
			nil,
		)
	}

	name := plugin.Name()
	if name == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"plugin name cannot be empty",
			nil,
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeDuplicate,
			fmt.Sprintf("plugin %s already registered", name),
			nil,
		)
	}

	r.plugins[name] = plugin
	if r.logger != nil {
		r.logger.Info("plugin registered", "name", name, "version", plugin.Version())
	}

	return nil
}

// Unregister unregisters a plugin
func (r *DefaultPluginRegistry) Unregister(name string) error {
	if name == "" {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"plugin name cannot be empty",
			nil,
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return NewSystemError(
			ErrorTypePermanent,
			ErrorCodeNotFound,
			fmt.Sprintf("plugin %s not found", name),
			nil,
		)
	}

	delete(r.plugins, name)
	if r.logger != nil {
		r.logger.Info("plugin unregistered", "name", name)
	}

	return nil
}

// Get retrieves a plugin by name
func (r *DefaultPluginRegistry) Get(name string) (Plugin, error) {
	if name == "" {
		return nil, NewSystemError(
			ErrorTypePermanent,
			ErrorCodeValidation,
			"plugin name cannot be empty",
			nil,
		)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, NewSystemError(
			ErrorTypePermanent,
			ErrorCodeNotFound,
			fmt.Sprintf("plugin %s not found", name),
			nil,
		)
	}

	return plugin, nil
}

// List returns all registered plugins
func (r *DefaultPluginRegistry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// Start starts all registered plugins
func (r *DefaultPluginRegistry) Start() error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()

	for _, plugin := range plugins {
		if err := plugin.Start(); err != nil {
			if r.logger != nil {
				r.logger.Error("failed to start plugin", "name", plugin.Name(), "error", err)
			}
			return NewSystemError(
				ErrorTypeCritical,
				ErrorCodeInternalError,
				fmt.Sprintf("failed to start plugin %s", plugin.Name()),
				err,
			)
		}

		if r.logger != nil {
			r.logger.Info("plugin started", "name", plugin.Name())
		}
	}

	return nil
}

// Stop stops all registered plugins
func (r *DefaultPluginRegistry) Stop() error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()

	// Stop plugins in reverse order
	for i := len(plugins) - 1; i >= 0; i-- {
		plugin := plugins[i]
		if err := plugin.Stop(); err != nil {
			if r.logger != nil {
				r.logger.Error("failed to stop plugin", "name", plugin.Name(), "error", err)
			}
			// Continue stopping other plugins even if one fails
		} else {
			if r.logger != nil {
				r.logger.Info("plugin stopped", "name", plugin.Name())
			}
		}
	}

	return nil
}

// Count returns the number of registered plugins
func (r *DefaultPluginRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// Clear removes all plugins
func (r *DefaultPluginRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins = make(map[string]Plugin)
	if r.logger != nil {
		r.logger.Info("plugin registry cleared")
	}
}
