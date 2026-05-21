package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// sortTopological orders plugins by dependency using Kahn's algorithm.
// Plugins implementing DependentPlugin declare dependencies by name.
// Returns error if a dependency is missing or a cycle is detected.
func sortTopological(plugins []Plugin) ([]Plugin, error) {
	inDegree := make(map[string]int, len(plugins))
	byName := make(map[string]Plugin, len(plugins))

	for _, p := range plugins {
		byName[p.Name()] = p
	}
	for _, p := range plugins {
		n := p.Name()
		if _, ok := inDegree[n]; !ok {
			inDegree[n] = 0
		}
		if dep, ok := p.(DependentPlugin); ok {
			for _, d := range dep.Dependencies() {
				if _, exists := byName[d]; !exists {
					return nil, fmt.Errorf("plugin %q depends on %q which is not registered", n, d)
				}
				inDegree[n]++
			}
		}
	}

	var queue []string
	for n, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, n)
		}
	}

	sorted := make([]Plugin, 0, len(plugins))
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, byName[name])

		for _, other := range plugins {
			dep, ok := other.(DependentPlugin)
			if !ok {
				continue
			}
			for _, d := range dep.Dependencies() {
				if d == name {
					inDegree[other.Name()]--
					if inDegree[other.Name()] == 0 {
						queue = append(queue, other.Name())
					}
				}
			}
		}
	}

	if len(sorted) != len(plugins) {
		return nil, fmt.Errorf("dependency cycle detected among %d plugins", len(plugins)-len(sorted))
	}

	return sorted, nil
}

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
		version := ""
		if vp, ok := any(plugin).(interface{ Version() string }); ok {
			version = vp.Version()
		}
		r.logger.Info("plugin registered", "name", name, "version", version)
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

// Start starts all registered plugins with the given context.
// Plugins are started in topological order based on declared dependencies.
func (r *DefaultPluginRegistry) Start(ctx context.Context) error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()

	sorted, err := sortTopological(plugins)
	if err != nil {
		return fmt.Errorf("plugin start order: %w", err)
	}

	for _, plugin := range sorted {
		lp, ok := plugin.(LifecyclePlugin)
		if !ok {
			if r.logger != nil {
				r.logger.Info("plugin has no lifecycle", "name", plugin.Name())
			}
			continue
		}
		if err := lp.Start(ctx); err != nil {
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

// Stop stops all registered plugins in reverse topological order with the given context.
// Each plugin's Stop call is bounded by a 30-second timeout to prevent a hung plugin
// from blocking the entire shutdown sequence.
func (r *DefaultPluginRegistry) Stop(ctx context.Context) error {
	const stopTimeout = 30 * time.Second

	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()

	sorted, err := sortTopological(plugins)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to compute stop order", "error", err)
		}
		sorted = plugins
	}

	for i := len(sorted) - 1; i >= 0; i-- {
		plugin := sorted[i]
		lp, ok := plugin.(LifecyclePlugin)
		if !ok {
			continue
		}

		stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
		done := make(chan error, 1)
		go func() {
			done <- lp.Stop(stopCtx)
		}()
		select {
		case stopErr := <-done:
			cancel()
			if stopErr != nil {
				if r.logger != nil {
					r.logger.Error("failed to stop plugin", "name", plugin.Name(), "error", stopErr)
				}
			} else {
				if r.logger != nil {
					r.logger.Info("plugin stopped", "name", plugin.Name())
				}
			}
		case <-stopCtx.Done():
			cancel()
			if r.logger != nil {
				r.logger.Warn("plugin stop timed out, continuing shutdown", "name", plugin.Name(), "timeout", stopTimeout)
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
