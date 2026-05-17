package core

import (
	"fmt"
	"testing"
)

// MockPlugin implements Plugin interface for testing
type MockPlugin struct {
	name        string
	version     string
	description string
	status      PluginStatus
	initialized bool
	started     bool
}

func NewMockPlugin(name string) *MockPlugin {
	return &MockPlugin{
		name:        name,
		version:     "1.0.0",
		description: "Mock plugin for testing",
		status:      PluginStatusUnloaded,
	}
}

func (m *MockPlugin) GetName() string {
	return m.name
}

func (m *MockPlugin) GetVersion() string {
	return m.version
}

func (m *MockPlugin) GetDescription() string {
	return m.description
}

func (m *MockPlugin) Initialize() error {
	m.initialized = true
	m.status = PluginStatusLoaded
	return nil
}

func (m *MockPlugin) Start() error {
	if !m.initialized {
		return fmt.Errorf("plugin not initialized")
	}
	m.started = true
	m.status = PluginStatusRunning
	return nil
}

func (m *MockPlugin) Stop() error {
	m.started = false
	m.status = PluginStatusStopped
	return nil
}

func (m *MockPlugin) GetStatus() PluginStatus {
	return m.status
}

func (m *MockPlugin) GetMetrics() map[string]any {
	return map[string]any{
		"initialized": m.initialized,
		"started":     m.started,
	}
}

func TestPluginRegistryRegister(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("test-plugin")

	err := registry.Register(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	if !plugin.initialized {
		t.Error("Plugin was not initialized")
	}

	retrieved, err := registry.Get("test-plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin: %v", err)
	}

	if retrieved.GetName() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", retrieved.GetName())
	}
}

func TestPluginRegistryRegisterDuplicate(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin1 := NewMockPlugin("test-plugin")
	plugin2 := NewMockPlugin("test-plugin")

	err := registry.Register(plugin1)
	if err != nil {
		t.Fatalf("Failed to register first plugin: %v", err)
	}

	err = registry.Register(plugin2)
	if err == nil {
		t.Error("Expected error when registering duplicate plugin")
	}
}

func TestPluginRegistryRegisterNil(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	err := registry.Register(nil)
	if err == nil {
		t.Error("Expected error when registering nil plugin")
	}
}

func TestPluginRegistryUnregister(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("test-plugin")

	_ = registry.Register(plugin)
	_ = plugin.Start()

	err := registry.Unregister("test-plugin")
	if err != nil {
		t.Fatalf("Failed to unregister plugin: %v", err)
	}

	_, err = registry.Get("test-plugin")
	if err == nil {
		t.Error("Expected error when getting unregistered plugin")
	}
}

func TestPluginRegistryStart(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("test-plugin")

	_ = registry.Register(plugin)

	err := registry.Start("test-plugin")
	if err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}

	if !plugin.started {
		t.Error("Plugin was not started")
	}

	if plugin.GetStatus() != PluginStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", plugin.GetStatus())
	}
}

func TestPluginRegistryStop(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("test-plugin")

	_ = registry.Register(plugin)
	_ = registry.Start("test-plugin")

	err := registry.Stop("test-plugin")
	if err != nil {
		t.Fatalf("Failed to stop plugin: %v", err)
	}

	if plugin.started {
		t.Error("Plugin was not stopped")
	}

	if plugin.GetStatus() != PluginStatusStopped {
		t.Errorf("Expected status 'stopped', got '%s'", plugin.GetStatus())
	}
}

func TestPluginRegistryStartAll(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin1 := NewMockPlugin("plugin1")
	plugin2 := NewMockPlugin("plugin2")
	plugin3 := NewMockPlugin("plugin3")

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)
	_ = registry.Register(plugin3)

	err := registry.StartAll()
	if err != nil {
		t.Fatalf("Failed to start all plugins: %v", err)
	}

	if !plugin1.started || !plugin2.started || !plugin3.started {
		t.Error("Not all plugins were started")
	}
}

func TestPluginRegistryStopAll(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin1 := NewMockPlugin("plugin1")
	plugin2 := NewMockPlugin("plugin2")

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)
	_ = registry.StartAll()

	err := registry.StopAll()
	if err != nil {
		t.Fatalf("Failed to stop all plugins: %v", err)
	}

	if plugin1.started || plugin2.started {
		t.Error("Not all plugins were stopped")
	}
}

func TestPluginRegistryList(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin1 := NewMockPlugin("plugin1")
	plugin2 := NewMockPlugin("plugin2")

	_ = registry.Register(plugin1)
	_ = registry.Register(plugin2)

	plugins := registry.List()
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}
}

func TestPluginRegistryGetMetadata(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("test-plugin")

	if err := registry.Register(plugin); err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	metadata := registry.GetMetadata()
	if len(metadata) != 1 {
		t.Errorf("Expected 1 metadata entry, got %d", len(metadata))
	}

	if metadata[0].Name != "test-plugin" {
		t.Errorf("Expected name 'test-plugin', got '%s'", metadata[0].Name)
	}

	if metadata[0].Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", metadata[0].Version)
	}
}

func TestPluginRegistryMetrics(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin1 := NewMockPlugin("plugin1")
	plugin2 := NewMockPlugin("plugin2")

	if err := registry.Register(plugin1); err != nil {
		t.Fatalf("Failed to register plugin1: %v", err)
	}
	if err := registry.Register(plugin2); err != nil {
		t.Fatalf("Failed to register plugin2: %v", err)
	}

	metrics := registry.GetRegistryMetrics()

	totalLoaded := metrics["total_loaded"].(int64)
	if totalLoaded != 2 {
		t.Errorf("Expected 2 total loaded, got %d", totalLoaded)
	}

	activePlugins := metrics["active_plugins"].(int64)
	if activePlugins != 2 {
		t.Errorf("Expected 2 active plugins, got %d", activePlugins)
	}
}

func TestPluginRegistryMetricsIncludesPostureFields(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("plugin-ready")

	if err := registry.Register(plugin); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := registry.Start("plugin-ready"); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	metrics := registry.GetRegistryMetrics()
	if metrics["coverage_posture"] != "registry-running-only" {
		t.Fatalf("expected registry-running-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "registry-ready" {
		t.Fatalf("expected registry-ready, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "plugin registry has active plugins running without registry-level error drift" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestPluginRegistryRuntimeMetricsUnobserved(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "registry-empty" {
		t.Fatalf("expected registry-empty, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "registry-unobserved" {
		t.Fatalf("expected registry-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestPluginRegistryRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("plugin-ready")

	if err := registry.Register(plugin); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := registry.Start("plugin-ready"); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "registry-running-only" {
		t.Fatalf("expected registry-running-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "registry-ready" {
		t.Fatalf("expected registry-ready, got %v", metrics["runtime_posture"])
	}
}

func TestPluginRegistryRuntimeMetricsDegraded(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("plugin-error")

	if err := registry.Register(plugin); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	plugin.status = PluginStatusError
	registry.recordError()

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "registry-error-only" {
		t.Fatalf("expected registry-error-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "registry-degraded" {
		t.Fatalf("expected registry-degraded, got %v", metrics["runtime_posture"])
	}
}

func TestPluginRegistryGetNonexistent(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent plugin")
	}
}

func TestPluginRegistryStartNonexistent(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	err := registry.Start("nonexistent")
	if err == nil {
		t.Error("Expected error when starting nonexistent plugin")
	}
}

func TestPluginRegistryStopNonexistent(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	err := registry.Stop("nonexistent")
	if err == nil {
		t.Error("Expected error when stopping nonexistent plugin")
	}
}

func TestPluginRegistryUnregisterNonexistent(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering nonexistent plugin")
	}
}

func TestPluginRegistryMultipleOperations(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()
	plugin := NewMockPlugin("test-plugin")

	// Register
	_ = registry.Register(plugin)

	// Start
	_ = registry.Start("test-plugin")
	if !plugin.started {
		t.Error("Plugin not started")
	}

	// Stop
	_ = registry.Stop("test-plugin")
	if plugin.started {
		t.Error("Plugin not stopped")
	}

	// Start again
	_ = registry.Start("test-plugin")
	if !plugin.started {
		t.Error("Plugin not restarted")
	}

	// Unregister
	_ = registry.Unregister("test-plugin")

	// Verify unregistered
	_, err := registry.Get("test-plugin")
	if err == nil {
		t.Error("Plugin should be unregistered")
	}
}

func TestPluginRegistryConcurrentOperations(t *testing.T) {
	t.Parallel()
	registry := NewPluginRegistry()

	// Register plugins concurrently
	errChan := make(chan error, 5)
	for i := 1; i <= 5; i++ {
		go func(id int) {
			plugin := NewMockPlugin(fmt.Sprintf("plugin%d", id))
			errChan <- registry.Register(plugin)
		}(i)
	}

	// Check for errors
	for i := 0; i < 5; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("failed to register plugin: %v", err)
		}
	}

	// Give goroutines time to complete
	plugins := registry.List()
	if len(plugins) < 1 {
		t.Error("Expected at least 1 plugin registered")
	}
}
