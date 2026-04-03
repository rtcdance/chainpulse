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
	totalLoaded := r.metrics.totalLoaded
	totalUnloaded := r.metrics.totalUnloaded
	totalErrors := r.metrics.totalErrors
	r.metrics.mu.RUnlock()

	activePlugins, loadedCount, runningCount, stoppedCount, errorCount := r.snapshotRuntimeCounts()
	coveragePosture := classifyPluginRegistryCoveragePosture(int(activePlugins), loadedCount, runningCount, stoppedCount, errorCount)
	runtimePosture := classifyPluginRegistryRuntimePosture(int(activePlugins), runningCount, errorCount, totalErrors)

	return map[string]interface{}{
		"total_loaded":     totalLoaded,
		"total_unloaded":   totalUnloaded,
		"active_plugins":   activePlugins,
		"total_errors":     totalErrors,
		"coverage_posture": coveragePosture,
		"runtime_posture":  runtimePosture,
		"reliability_hint": buildPluginRegistryReliabilityHint(coveragePosture, runtimePosture),
	}
}

// GetRuntimeMetrics returns a compact runtime surface for plugin-registry
// coverage and lifecycle readiness on top of the raw registry metrics.
func (r *PluginRegistry) GetRuntimeMetrics() map[string]interface{} {
	metrics := r.GetRegistryMetrics()
	loadedCount := 0
	runningCount := 0
	stoppedCount := 0
	errorCount := 0
	activePlugins, loadedCount, runningCount, stoppedCount, errorCount := r.snapshotRuntimeCounts()

	return map[string]interface{}{
		"total_loaded":     metrics["total_loaded"],
		"total_unloaded":   metrics["total_unloaded"],
		"active_plugins":   activePlugins,
		"total_errors":     metrics["total_errors"],
		"loaded_count":     loadedCount,
		"running_count":    runningCount,
		"stopped_count":    stoppedCount,
		"error_count":      errorCount,
		"coverage_posture": metrics["coverage_posture"],
		"runtime_posture":  metrics["runtime_posture"],
		"reliability_hint": metrics["reliability_hint"],
	}
}

func (r *PluginRegistry) snapshotRuntimeCounts() (int64, int, int, int, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activePlugins := int64(len(r.plugins))
	loadedCount := 0
	runningCount := 0
	stoppedCount := 0
	errorCount := 0
	for _, plugin := range r.plugins {
		switch plugin.GetStatus() {
		case PluginStatusLoaded:
			loadedCount++
		case PluginStatusRunning:
			runningCount++
		case PluginStatusStopped:
			stoppedCount++
		case PluginStatusError:
			errorCount++
		}
	}

	return activePlugins, loadedCount, runningCount, stoppedCount, errorCount
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

func classifyPluginRegistryCoveragePosture(activePlugins int, loadedCount int, runningCount int, stoppedCount int, errorCount int) string {
	if activePlugins == 0 {
		return "registry-empty"
	}
	if runningCount == activePlugins {
		return "registry-running-only"
	}
	if errorCount == activePlugins {
		return "registry-error-only"
	}
	if loadedCount > 0 || stoppedCount > 0 || errorCount > 0 {
		return "registry-mixed"
	}
	return "registry-loaded"
}

func classifyPluginRegistryRuntimePosture(activePlugins int, runningCount int, errorCount int, totalErrors int64) string {
	if activePlugins == 0 {
		return "registry-unobserved"
	}
	if errorCount > 0 || totalErrors > 0 {
		return "registry-degraded"
	}
	if runningCount == 0 {
		return "registry-idle"
	}
	if runningCount < activePlugins {
		return "registry-partial"
	}
	return "registry-ready"
}

func buildPluginRegistryReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "registry-degraded":
		return "plugin registry is observing plugin lifecycle errors; inspect failing plugins before treating the runtime as healthy"
	case runtimePosture == "registry-idle":
		return "plugin registry has loaded plugins but none are running; verify start sequencing before relying on active behavior"
	case runtimePosture == "registry-partial":
		return "plugin registry has only partial runtime coverage; confirm that required plugins have reached running state"
	case coveragePosture == "registry-mixed":
		return "plugin registry has mixed plugin states; continue observing lifecycle convergence"
	case runtimePosture == "registry-ready":
		return "plugin registry has active plugins running without registry-level error drift"
	default:
		return "plugin registry has not loaded plugins yet"
	}
}
