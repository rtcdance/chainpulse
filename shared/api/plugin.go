package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// APIPlugin interface defines the interface for API plugins
type APIPlugin interface {
	APIService
	Initialize(config map[string]interface{}) error
	GetName() string
	SetMetricsCollector(collector *MetricsCollector)
}

// PluginRegistry holds all available API plugins
type PluginRegistry struct {
	plugins map[string]APIPlugin
	mutex   sync.RWMutex
}

// GlobalPluginRegistry is the global registry instance
var GlobalPluginRegistry = &PluginRegistry{
	plugins: make(map[string]APIPlugin),
}

// RegisterPlugin registers a new API plugin
func (r *PluginRegistry) RegisterPlugin(name string, plugin APIPlugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin with name %s already exists", name)
	}

	r.plugins[name] = plugin
	return nil
}

// GetPlugin returns a registered API plugin by name
func (r *PluginRegistry) GetPlugin(name string) (APIPlugin, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin with name %s not found", name)
	}

	return plugin, nil
}

// GetAvailablePlugins returns a list of available plugin names
func (r *PluginRegistry) GetAvailablePlugins() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// MultiProtocolAPI implements APIService using multiple API plugins
type MultiProtocolAPI struct {
	plugins          map[string]APIPlugin
	defaultPlugin    string
	metricsCollector *MetricsCollector
}

// NewMultiProtocolAPI creates a new multi-protocol API instance
func NewMultiProtocolAPI(defaultPlugin string) *MultiProtocolAPI {
	return &MultiProtocolAPI{
		plugins:       make(map[string]APIPlugin),
		defaultPlugin: defaultPlugin,
	}
}

// Initialize initializes the multi-protocol API with plugin configurations
func (mp *MultiProtocolAPI) Initialize(pluginConfigs map[string]map[string]interface{}) error {
	for pluginName, config := range pluginConfigs {
		plugin, err := GlobalPluginRegistry.GetPlugin(pluginName)
		if err != nil {
			return fmt.Errorf("failed to get plugin %s: %w", pluginName, err)
		}

		if err := plugin.Initialize(config); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", pluginName, err)
		}

		// Set metrics collector if available
		if mp.metricsCollector != nil {
			plugin.SetMetricsCollector(mp.metricsCollector)
		}

		mp.plugins[pluginName] = plugin
	}

	return nil
}

// SetMetricsCollector sets the metrics collector for the multi-protocol API
func (mp *MultiProtocolAPI) SetMetricsCollector(collector *MetricsCollector) {
	mp.metricsCollector = collector
	// Set collector for all plugins
	for _, plugin := range mp.plugins {
		plugin.SetMetricsCollector(collector)
	}
}

// Start starts all plugin services
func (mp *MultiProtocolAPI) Start(ctx context.Context) error {
	for name, plugin := range mp.plugins {
		go func(pluginName string, p APIPlugin) {
			if err := p.Start(ctx); err != nil {
				// Log error but continue with other plugins
				fmt.Printf("Error starting API plugin %s: %v\n", pluginName, err)
			}
		}(name, plugin)
	}
	return nil
}

// Stop stops all plugin services
func (mp *MultiProtocolAPI) Stop(ctx context.Context) error {
	var lastErr error
	for name, plugin := range mp.plugins {
		if err := plugin.Stop(ctx); err != nil {
			lastErr = fmt.Errorf("error stopping plugin %s: %w", name, err)
		}
	}
	return lastErr
}

// GetPlugin returns a specific plugin by name
func (mp *MultiProtocolAPI) GetPlugin(name string) (APIPlugin, error) {
	plugin, exists := mp.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return plugin, nil
}