package core

import (
	"fmt"
	"sync"
	"time"
)

// PluginMetadata contains metadata about a plugin
type PluginMetadata struct {
	Name        string
	Version     string
	Description string
	Author      string
	LoadedAt    time.Time
	Status      PluginStatus
}

// PluginStatus represents the status of a plugin
type PluginStatus string

const (
	PluginStatusLoaded   PluginStatus = "loaded"
	PluginStatusRunning  PluginStatus = "running"
	PluginStatusStopped  PluginStatus = "stopped"
	PluginStatusError    PluginStatus = "error"
	PluginStatusUnloaded PluginStatus = "unloaded"
)

// Plugin defines the interface for loadable plugins
type Plugin interface {
	// GetName returns the plugin name
	GetName() string

	// GetVersion returns the plugin version
	GetVersion() string

	// GetDescription returns the plugin description
	GetDescription() string

	// Initialize initializes the plugin
	Initialize() error

	// Start starts the plugin
	Start() error

	// Stop stops the plugin
	Stop() error

	// GetStatus returns the plugin status
	GetStatus() PluginStatus

	// GetMetrics returns plugin metrics
	GetMetrics() map[string]interface{}
}

// PluginRegistry manages plugin lifecycle
type PluginRegistry struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
	metrics *RegistryMetrics
}

// RegistryMetrics tracks registry metrics
type RegistryMetrics struct {
	totalLoaded   int64
	totalUnloaded int64
	totalErrors   int64
	mu            sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
		metrics: &RegistryMetrics{},
	}
}

// Register registers a plugin
func (r *PluginRegistry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plugin == nil {
		return fmt.Errorf("plugin cannot be nil")
	}

	name := plugin.GetName()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Initialize plugin
	if err := plugin.Initialize(); err != nil {
		r.recordError()
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	r.plugins[name] = plugin
	r.recordLoaded()

	return nil
}

// Unregister unregisters a plugin
func (r *PluginRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop plugin if running
	if err := plugin.Stop(); err != nil {
		r.recordError()
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	delete(r.plugins, name)
	r.recordUnloaded()

	return nil
}

// Get retrieves a plugin by name
func (r *PluginRegistry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// Start starts a plugin
func (r *PluginRegistry) Start(name string) error {
	r.mu.RLock()
	plugin, exists := r.plugins[name]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := plugin.Start(); err != nil {
		r.recordError()
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	return nil
}

// Stop stops a plugin
func (r *PluginRegistry) Stop(name string) error {
	r.mu.RLock()
	plugin, exists := r.plugins[name]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := plugin.Stop(); err != nil {
		r.recordError()
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	return nil
}

// StartAll starts all registered plugins
func (r *PluginRegistry) StartAll() error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()

	for _, plugin := range plugins {
		if err := plugin.Start(); err != nil {
			r.recordError()
			return fmt.Errorf("failed to start plugin %s: %w", plugin.GetName(), err)
		}
	}

	return nil
}

// StopAll stops all registered plugins
func (r *PluginRegistry) StopAll() error {
	r.mu.RLock()
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	r.mu.RUnlock()

	for _, plugin := range plugins {
		if err := plugin.Stop(); err != nil {
			r.recordError()
			return fmt.Errorf("failed to stop plugin %s: %w", plugin.GetName(), err)
		}
	}

	return nil
}

// List returns all registered plugins
func (r *PluginRegistry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// GetMetadata returns metadata for all plugins
func (r *PluginRegistry) GetMetadata() []PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make([]PluginMetadata, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		metadata = append(metadata, PluginMetadata{
			Name:        plugin.GetName(),
			Version:     plugin.GetVersion(),
			Description: plugin.GetDescription(),
			Status:      plugin.GetStatus(),
			LoadedAt:    time.Now(),
		})
	}

	return metadata
}

// GetRegistryMetrics returns registry metrics
func (r *PluginRegistry) GetRegistryMetrics() map[string]interface{} {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	r.mu.RLock()
	activePlugins := int64(len(r.plugins))
	r.mu.RUnlock()

	return map[string]interface{}{
		"total_loaded":   r.metrics.totalLoaded,
		"total_unloaded": r.metrics.totalUnloaded,
		"active_plugins": activePlugins,
		"total_errors":   r.metrics.totalErrors,
	}
}

// Helper methods

func (r *PluginRegistry) recordLoaded() {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.totalLoaded++
}

func (r *PluginRegistry) recordUnloaded() {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.totalUnloaded++
}

func (r *PluginRegistry) recordError() {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.totalErrors++
}
