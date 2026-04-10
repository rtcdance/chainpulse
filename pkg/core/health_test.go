package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockPlugin is a mock plugin for testing
type MockPlugin struct {
	name    string
	version string
	healthy bool
	err     error
}

func (m *MockPlugin) Name() string                   { return m.name }
func (m *MockPlugin) Version() string                { return m.version }
func (m *MockPlugin) Initialize(config Config) error { return nil }
func (m *MockPlugin) Start() error                   { return nil }
func (m *MockPlugin) Stop() error                    { return nil }
func (m *MockPlugin) Health() error {
	if !m.healthy {
		return m.err
	}
	return nil
}

// MockPluginRegistry is a mock registry for testing
type MockPluginRegistry struct {
	plugins map[string]Plugin
}

func NewMockPluginRegistry() *MockPluginRegistry {
	return &MockPluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

func (m *MockPluginRegistry) Register(plugin Plugin) error {
	m.plugins[plugin.Name()] = plugin
	return nil
}

func (m *MockPluginRegistry) Unregister(name string) error {
	delete(m.plugins, name)
	return nil
}

func (m *MockPluginRegistry) Get(name string) (Plugin, error) {
	if p, ok := m.plugins[name]; ok {
		return p, nil
	}
	return nil, errors.New("plugin not found")
}

func (m *MockPluginRegistry) List() []Plugin {
	plugins := make([]Plugin, 0)
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

func (m *MockPluginRegistry) Start() error { return nil }
func (m *MockPluginRegistry) Stop() error  { return nil }

// MockConfigManager is a mock config manager for testing
type MockConfigManager struct {
	config Config
	valid  bool
}

func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		config: Config{
			DeploymentMode: "monolithic",
			LogLevel:       "INFO",
		},
		valid: true,
	}
}

func (m *MockConfigManager) Load() (Config, error) {
	return m.config, nil
}

func (m *MockConfigManager) Validate(config Config) error {
	if !m.valid {
		return errors.New("invalid configuration")
	}
	return nil
}

func (m *MockConfigManager) Get(key string) (interface{}, error) {
	return nil, nil
}

func (m *MockConfigManager) Set(key string, value interface{}) error {
	return nil
}

// TestNewDefaultHealthChecker tests health checker creation
func TestNewDefaultHealthChecker(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	if checker == nil {
		t.Fatal("expected health checker, got nil")
	}
}

// TestHealthCheckBasic tests basic health check
func TestHealthCheckBasic(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
	if status.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", status.Status)
	}
}

// TestHealthCheckWithHealthyPlugin tests health check with healthy plugin
func TestHealthCheckWithHealthyPlugin(t *testing.T) {
	registry := NewMockPluginRegistry()
	plugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		healthy: true,
	}
	if err := registry.Register(plugin); err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
	if status.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", status.Status)
	}
}

// TestHealthCheckWithUnhealthyPlugin tests health check with unhealthy plugin
func TestHealthCheckWithUnhealthyPlugin(t *testing.T) {
	registry := NewMockPluginRegistry()
	plugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		healthy: false,
		err:     errors.New("plugin error"),
	}
	if err := registry.Register(plugin); err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
	if status.Status != "unhealthy" {
		t.Errorf("expected unhealthy status, got %s", status.Status)
	}
}

// TestHealthCheckWithInvalidConfig tests health check with invalid configuration
func TestHealthCheckWithInvalidConfig(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	config.valid = false

	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, err := checker.Check(context.Background())
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
	if status.Status != "unhealthy" {
		t.Errorf("expected unhealthy status, got %s", status.Status)
	}
}

// TestGetLastCheckStatus tests getting last check status
func TestGetLastCheckStatus(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	_, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	status := checker.GetLastCheckStatus()
	if status.Status == "" {
		t.Error("expected status to be set")
	}
}

// TestGetLastCheckTime tests getting last check time
func TestGetLastCheckTime(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	before := time.Now()
	_, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	after := time.Now()

	checkTime := checker.GetLastCheckTime()
	if checkTime.Before(before) || checkTime.After(after) {
		t.Error("check time not within expected range")
	}
}

// TestSetCheckInterval tests setting check interval
func TestSetCheckInterval(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	interval := 60 * time.Second
	checker.SetCheckInterval(interval)

	if checker.GetCheckInterval() != interval {
		t.Errorf("expected interval %v, got %v", interval, checker.GetCheckInterval())
	}
}

// TestIsHealthy tests IsHealthy method
func TestIsHealthy(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	_, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if !checker.IsHealthy() {
		t.Error("expected system to be healthy")
	}
}

// TestGetHealthSummary tests getting health summary
func TestGetHealthSummary(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, err := checker.Check(context.Background())
	if err != nil {
		t.Logf("health check returned error: %v", err)
	}
	_ = status

	summary := checker.GetHealthSummary()
	if summary["status"] == nil {
		t.Error("expected status in summary")
	}
	if summary["message"] == nil {
		t.Error("expected message in summary")
	}
	if summary["details"] == nil {
		t.Error("expected details in summary")
	}
}

// TestPerformHealthCheck tests performing health check with logging
func TestPerformHealthCheck(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	err := checker.PerformHealthCheck(context.Background())
	if err != nil {
		t.Errorf("perform health check failed: %v", err)
	}
}

// TestHealthCheckDetails tests health check details
func TestHealthCheckDetails(t *testing.T) {
	registry := NewMockPluginRegistry()
	plugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		healthy: true,
	}
	err := registry.Register(plugin)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, _ := checker.Check(context.Background())

	if status.Details["plugins"] == nil {
		t.Error("expected plugins in details")
	}
	if status.Details["configuration"] == nil {
		t.Error("expected configuration in details")
	}
	if status.Details["event_bus"] == nil {
		t.Error("expected event_bus in details")
	}
	if status.Details["metrics"] == nil {
		t.Error("expected metrics in details")
	}
}

// TestHealthCheckDuration tests health check duration tracking
func TestHealthCheckDuration(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, _ := checker.Check(context.Background())

	duration := status.Details["check_duration_ms"]
	if duration == nil {
		t.Error("expected check_duration_ms in details")
	}
}

// TestMultipleHealthChecks tests multiple consecutive health checks
func TestMultipleHealthChecks(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)

	for i := 0; i < 5; i++ {
		status, err := checker.Check(context.Background())
		if err != nil {
			t.Errorf("health check %d failed: %v", i, err)
		}
		if status.Status != "healthy" {
			t.Errorf("health check %d returned unhealthy status", i)
		}
	}
}

// TestHealthCheckWithMultiplePlugins tests health check with multiple plugins
func TestHealthCheckWithMultiplePlugins(t *testing.T) {
	registry := NewMockPluginRegistry()
	for i := 0; i < 3; i++ {
		plugin := &MockPlugin{
			name:    "plugin-" + string(rune(i)),
			version: "1.0.0",
			healthy: true,
		}
		err := registry.Register(plugin)
		if err != nil {
			t.Fatalf("failed to register plugin %d: %v", i, err)
		}
	}

	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)
	status, _ := checker.Check(context.Background())

	if status.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", status.Status)
	}
}

// TestHealthCheckWithNilRegistry tests health check with nil registry
func TestHealthCheckWithNilRegistry(t *testing.T) {
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(nil, config, bus, metrics, logger)
	status, _ := checker.Check(context.Background())

	// Should still return a status even with nil registry
	if status.Status == "" {
		t.Error("expected status to be set")
	}
}

// TestHealthCheckWithNilConfigManager tests health check with nil config manager
func TestHealthCheckWithNilConfigManager(t *testing.T) {
	registry := NewMockPluginRegistry()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, nil, bus, metrics, logger)
	status, _ := checker.Check(context.Background())

	// Should still return a status even with nil config manager
	if status.Status == "" {
		t.Error("expected status to be set")
	}
}

// TestHealthCheckWithNilEventBus tests health check with nil event bus
func TestHealthCheckWithNilEventBus(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, nil, metrics, logger)
	status, _ := checker.Check(context.Background())

	// Should still return a status even with nil event bus
	if status.Status == "" {
		t.Error("expected status to be set")
	}
}

// TestHealthCheckWithNilMetricsCollector tests health check with nil metrics collector
func TestHealthCheckWithNilMetricsCollector(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, nil, logger)
	status, _ := checker.Check(context.Background())

	// Should still return a status even with nil metrics collector
	if status.Status == "" {
		t.Error("expected status to be set")
	}
}

// TestHealthCheckConcurrency tests concurrent health checks
func TestHealthCheckConcurrency(t *testing.T) {
	registry := NewMockPluginRegistry()
	config := NewMockConfigManager()
	bus := NewEventBus(nil)
	metrics := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	checker := NewDefaultHealthChecker(registry, config, bus, metrics, logger)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = checker.Check(context.Background())
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	status := checker.GetLastCheckStatus()
	if status.Status == "" {
		t.Error("expected status to be set after concurrent checks")
	}
}
