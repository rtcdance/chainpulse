package mq

import (
	"context"
	"fmt"
	"sync"
)

// MQPlugin interface defines the interface for message queue plugins
type MQPlugin interface {
	MessageQueue
	Initialize(config map[string]interface{}) error
	GetName() string
	SetMetricsCollector(collector *MetricsCollector)
}

// PluginRegistry holds all available MQ plugins
type PluginRegistry struct {
	plugins map[string]MQPlugin
	mutex   sync.RWMutex
}

// GlobalPluginRegistry is the global registry instance
var GlobalPluginRegistry = &PluginRegistry{
	plugins: make(map[string]MQPlugin),
}

// RegisterPlugin registers a new MQ plugin
func (r *PluginRegistry) RegisterPlugin(name string, plugin MQPlugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin with name %s already exists", name)
	}

	r.plugins[name] = plugin
	return nil
}

// GetPlugin returns a registered MQ plugin by name
func (r *PluginRegistry) GetPlugin(name string) (MQPlugin, error) {
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

// MultiProtocolMQ implements MessageQueue using multiple MQ plugins
type MultiProtocolMQ struct {
	plugins          map[string]MQPlugin
	defaultPlugin    string
	metricsCollector *MetricsCollector
}

// NewMultiProtocolMQ creates a new multi-protocol message queue instance
func NewMultiProtocolMQ(defaultPlugin string) *MultiProtocolMQ {
	return &MultiProtocolMQ{
		plugins:       make(map[string]MQPlugin),
		defaultPlugin: defaultPlugin,
	}
}

// Initialize initializes the multi-protocol MQ with plugin configurations
func (mp *MultiProtocolMQ) Initialize(pluginConfigs map[string]map[string]interface{}) error {
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

// SetMetricsCollector sets the metrics collector for the multi-protocol MQ
func (mp *MultiProtocolMQ) SetMetricsCollector(collector *MetricsCollector) {
	mp.metricsCollector = collector
	// Set collector for all plugins
	for _, plugin := range mp.plugins {
		plugin.SetMetricsCollector(collector)
	}
}

// Publish sends a message using the default or specified plugin
func (mp *MultiProtocolMQ) Publish(topic string, message interface{}) error {
	plugin, exists := mp.plugins[mp.defaultPlugin]
	if !exists {
		return fmt.Errorf("default plugin %s not found", mp.defaultPlugin)
	}

	return plugin.Publish(topic, message)
}

// PublishToPlugin sends a message using a specific plugin
func (mp *MultiProtocolMQ) PublishToPlugin(pluginName, topic string, message interface{}) error {
	plugin, exists := mp.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	return plugin.Publish(topic, message)
}

// Consume reads messages from the default or specified plugin
func (mp *MultiProtocolMQ) Consume(ctx context.Context, topic string, handler MessageHandler) error {
	plugin, exists := mp.plugins[mp.defaultPlugin]
	if !exists {
		return fmt.Errorf("default plugin %s not found", mp.defaultPlugin)
	}

	return plugin.Consume(ctx, topic, handler)
}

// ConsumeFromPlugin reads messages from a specific plugin
func (mp *MultiProtocolMQ) ConsumeFromPlugin(ctx context.Context, pluginName, topic string, handler MessageHandler) error {
	plugin, exists := mp.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	return plugin.Consume(ctx, topic, handler)
}

// Close closes all plugin connections
func (mp *MultiProtocolMQ) Close() error {
	var lastErr error
	for name, plugin := range mp.plugins {
		if err := plugin.Close(); err != nil {
			lastErr = fmt.Errorf("error closing plugin %s: %w", name, err)
		}
	}
	return lastErr
}
