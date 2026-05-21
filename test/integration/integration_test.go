package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// TestPhase1Integration tests that all Phase 1 components work together
func TestPhase1Integration(t *testing.T) {
	// Create all core components
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	logBuffer := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, logBuffer)
	healthChecker := NewDefaultHealthChecker(registry, configManager, eventBus, metricsCollector, logger)

	// Verify all components are created
	if registry == nil {
		t.Fatal("registry is nil")
	}
	if configManager == nil {
		t.Fatal("configManager is nil")
	}
	if eventBus == nil {
		t.Fatal("eventBus is nil")
	}
	if metricsCollector == nil {
		t.Fatal("metricsCollector is nil")
	}
	if logger == nil {
		t.Fatal("logger is nil")
	}
	if healthChecker == nil {
		t.Fatal("healthChecker is nil")
	}

	t.Log("✓ All Phase 1 components created successfully")
}

// TestPhase1PluginLifecycle tests plugin lifecycle management
func TestPhase1PluginLifecycle(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	logger := NewDefaultLogger(LogLevelInfo)

	// Create a test plugin
	plugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		healthy: true,
	}

	// Register plugin
	if err := registry.Register(plugin); err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}
	logger.Info("plugin registered", "name", plugin.Name())

	// Get plugin
	retrieved, err := registry.Get(plugin.Name())
	if err != nil {
		t.Fatalf("failed to get plugin: %v", err)
	}
	if retrieved.Name() != plugin.Name() {
		t.Errorf("expected plugin name %s, got %s", plugin.Name(), retrieved.Name())
	}
	logger.Info("plugin retrieved", "name", retrieved.Name())

	// List plugins
	plugins := registry.List()
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}
	logger.Info("plugins listed", "count", len(plugins))

	// Unregister plugin
	if err := registry.Unregister(plugin.Name()); err != nil {
		t.Fatalf("failed to unregister plugin: %v", err)
	}
	logger.Info("plugin unregistered", "name", plugin.Name())

	// Verify plugin is gone
	plugins = registry.List()
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins after unregister, got %d", len(plugins))
	}

	t.Log("✓ Plugin lifecycle management works correctly")
}

// TestPhase1ConfigurationFlow tests configuration loading and validation
func TestPhase1ConfigurationFlow(t *testing.T) {
	configManager := NewDefaultConfigManager()
	logger := NewDefaultLogger(LogLevelInfo)

	// Load configuration
	config, err := configManager.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}
	logger.Info("configuration loaded", "deployment_mode", config.DeploymentMode)

	// Validate configuration
	if err := configManager.Validate(config); err != nil {
		t.Fatalf("failed to validate configuration: %v", err)
	}
	logger.Info("configuration validated")

	// Get configuration value
	value, err := configManager.Get("deployment_mode")
	if err != nil {
		t.Fatalf("failed to get configuration value: %v", err)
	}
	logger.Info("configuration value retrieved", "value", value)

	// Set configuration value
	if err := configManager.Set("log_level", "DEBUG"); err != nil {
		t.Fatalf("failed to set configuration value: %v", err)
	}
	logger.Info("configuration value set", "key", "log_level", "value", "DEBUG")

	t.Log("✓ Configuration management works correctly")
}

// TestPhase1EventBusFlow tests event publishing and subscription
func TestPhase1EventBusFlow(t *testing.T) {
	eventBus := NewDefaultEventBus()
	logger := NewDefaultLogger(LogLevelInfo)

	// Subscribe to events
	receivedEvents := make([]any, 0)
	handler := func(_ context.Context, event any) error {
		receivedEvents = append(receivedEvents, event)
		return nil
	}

	subID, err := eventBus.Subscribe(context.Background(), "test-topic", handler)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	logger.Info("subscribed to topic", "topic", "test-topic")

	// Publish event
	testEvent := map[string]any{"message": "test"}
	if err := eventBus.Publish(context.Background(), "test-topic", testEvent); err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}
	logger.Info("event published", "topic", "test-topic")

	// Give handler time to process
	time.Sleep(100 * time.Millisecond)

	// Verify event was received
	if len(receivedEvents) != 1 {
		t.Errorf("expected 1 event, got %d", len(receivedEvents))
	}

	// Unsubscribe
	if err := eventBus.Unsubscribe(subID); err != nil {
		t.Fatalf("failed to unsubscribe: %v", err)
	}
	logger.Info("unsubscribed from topic", "topic", "test-topic")

	t.Log("✓ Event bus works correctly")
}

// TestPhase1MetricsFlow tests metrics collection
func TestPhase1MetricsFlow(t *testing.T) {
	metricsCollector := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	tags := map[string]string{"service": "api"}

	// Record counter
	metricsCollector.RecordCounter("requests", 10, tags)
	logger.Info("counter recorded", "name", "requests", "value", 10)

	// Record gauge
	metricsCollector.RecordGauge("memory", 512.5, tags)
	logger.Info("gauge recorded", "name", "memory", "value", 512.5)

	// Record histogram
	for i := 1; i <= 5; i++ {
		metricsCollector.RecordHistogram("latency", float64(i*10), tags)
	}
	logger.Info("histogram recorded", "name", "latency", "count", 5)

	// Get metrics
	metrics := metricsCollector.GetMetrics()
	if metrics["counters"] == nil {
		t.Error("expected counters in metrics")
	}
	if metrics["gauges"] == nil {
		t.Error("expected gauges in metrics")
	}
	if metrics["histograms"] == nil {
		t.Error("expected histograms in metrics")
	}
	logger.Info("metrics retrieved")

	// Get specific metrics
	allMetrics := metricsCollector.GetMetrics()
	counterValue := allMetrics["counters"].(map[string]int64)["requests"]
	gaugeValue := allMetrics["gauges"].(map[string]float64)["memory"]
	histogramValues := allMetrics["histograms"].(map[string][]float64)["latency"]
	histStats := map[string]any{
		"count": len(histogramValues),
	}

	if counterValue != 10 {
		t.Errorf("expected counter 10, got %d", counterValue)
	}
	if gaugeValue != 512.5 {
		t.Errorf("expected gauge 512.5, got %f", gaugeValue)
	}
	if histStats["count"].(int) != 5 {
		t.Errorf("expected histogram count 5, got %d", histStats["count"].(int))
	}

	logger.Info("metrics verified", "counter", counterValue, "gauge", gaugeValue, "histogram_count", histStats["count"].(int))

	t.Log("✓ Metrics collection works correctly")
}

// TestPhase1LoggingFlow tests structured logging with correlation IDs
func TestPhase1LoggingFlow(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, logBuffer)

	// Log with correlation ID
	correlatedLogger := logger.WithCorrelationID("request-123")
	correlatedLogger.Info("processing request", "user_id", "user-456")

	// Verify log output
	output := logBuffer.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}

	if entry.CorrelationID != "request-123" {
		t.Errorf("expected correlation ID request-123, got %s", entry.CorrelationID)
	}
	if entry.Message != "processing request" {
		t.Errorf("expected message 'processing request', got %s", entry.Message)
	}
	if entry.Fields["user_id"] != "user-456" {
		t.Errorf("expected user_id user-456, got %v", entry.Fields["user_id"])
	}

	t.Log("✓ Structured logging works correctly")
}

// TestPhase1HealthCheckFlow tests health checking
func TestPhase1HealthCheckFlow(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	healthChecker := NewDefaultHealthChecker(registry, configManager, eventBus, metricsCollector, logger)

	// Perform health check
	status, err := healthChecker.Check(context.Background())
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("expected healthy status, got %s", status.Status)
	}

	logger.Info("health check passed", "status", status.Status)

	// Get health summary
	healthStatus, _ := healthChecker.Check(context.Background())
	summary := map[string]any{
		"status": healthStatus.Status,
	}
	if summary["status"] != "healthy" {
		t.Errorf("expected healthy status in summary, got %v", summary["status"])
	}

	logger.Info("health summary retrieved", "status", summary["status"])

	t.Log("✓ Health checking works correctly")
}

// TestPhase1EndToEndFlow tests complete end-to-end flow
func TestPhase1EndToEndFlow(t *testing.T) {
	// Create all components
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	logBuffer := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, logBuffer)
	healthChecker := NewDefaultHealthChecker(registry, configManager, eventBus, metricsCollector, logger)

	// Create a test plugin
	plugin := &MockPlugin{
		name:    "data-puller",
		version: "1.0.0",
		healthy: true,
	}

	// Register plugin
	_ = registry.Register(plugin)
	logger.WithCorrelationID("flow-1").Info("plugin registered", "name", plugin.Name())

	// Record metrics for plugin startup
	metricsCollector.RecordCounter("plugin_startups", 1, map[string]string{"plugin": plugin.Name()})
	logger.WithCorrelationID("flow-1").Info("metrics recorded", "metric", "plugin_startups")

	// Subscribe to plugin events
	eventCount := 0
	_, _ = eventBus.Subscribe(context.Background(), "plugin-events", func(_ context.Context, event any) error {
		eventCount++
		return nil
	})
	logger.WithCorrelationID("flow-1").Info("subscribed to plugin events")

	// Publish plugin event
	_ = eventBus.Publish(context.Background(), "plugin-events", map[string]any{
		"plugin": plugin.Name(),
		"event":  "started",
	})
	logger.WithCorrelationID("flow-1").Info("plugin event published")

	// Record more metrics
	metricsCollector.RecordHistogram("startup_time", 150.5, map[string]string{"plugin": plugin.Name()})
	logger.WithCorrelationID("flow-1").Info("startup time recorded", "duration_ms", 150.5)

	// Perform health check
	status, _ := healthChecker.Check(context.Background())
	logger.WithCorrelationID("flow-1").Info("health check completed", "status", status.Status)

	// Verify all components worked together
	if registry.List()[0].Name() != plugin.Name() {
		t.Error("plugin not registered correctly")
	}

	metrics := metricsCollector.GetMetrics()
	if metrics["counters"] == nil || metrics["histograms"] == nil {
		t.Error("metrics not recorded correctly")
	}

	if status.Status != "healthy" {
		t.Error("system not healthy")
	}

	// Verify logging
	logOutput := logBuffer.String()
	if !bytes.Contains([]byte(logOutput), []byte("flow-1")) {
		t.Error("correlation ID not in logs")
	}

	t.Log("✓ End-to-end flow works correctly")
}

// TestPhase1ConcurrentOperations tests concurrent operations across components
func TestPhase1ConcurrentOperations(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	var wg sync.WaitGroup
	numGoroutines := 20

	// Concurrent plugin registration
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			plugin := &MockPlugin{
				name:    "plugin-" + string(rune(id)),
				version: "1.0.0",
				healthy: true,
			}
			_ = registry.Register(plugin)
		}(i)
	}

	// Concurrent event publishing
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = eventBus.Publish(context.Background(), "test-topic", map[string]any{"id": id})
		}(i)
	}

	// Concurrent metrics recording
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			metricsCollector.RecordCounter("requests", 1, map[string]string{"id": string(rune(id))})
		}(i)
	}

	// Concurrent logging
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.WithCorrelationID("concurrent-" + string(rune(id))).Info("concurrent operation")
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		// All goroutines completed successfully
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for goroutines to complete")
	}

	// Verify results
	if len(registry.List()) != numGoroutines {
		t.Errorf("expected %d plugins, got %d", numGoroutines, len(registry.List()))
	}

	counterValue := metricsCollector.GetMetrics()["counters"].(map[string]int64)["requests"]
	if counterValue != int64(numGoroutines) {
		t.Errorf("expected counter %d, got %d", numGoroutines, counterValue)
	}

	logger.Info("concurrent operations completed successfully")
	t.Log("✓ Concurrent operations work correctly")
}

// TestPhase1ComponentInteraction tests interaction between components
func TestPhase1ComponentInteraction(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	logBuffer := &bytes.Buffer{}
	logger := NewDefaultLoggerWithOutput(LogLevelInfo, logBuffer)

	// Simulate a complete workflow
	correlatedLogger := logger.WithCorrelationID("workflow-1")

	// 1. Load configuration
	config, _ := configManager.Load()
	correlatedLogger.Info("configuration loaded", "mode", config.DeploymentMode)

	// 2. Register plugins
	for i := 0; i < 3; i++ {
		plugin := &MockPlugin{
			name:    "plugin-" + string(rune(i)),
			version: "1.0.0",
			healthy: true,
		}
		_ = registry.Register(plugin)
		metricsCollector.RecordCounter("plugins_registered", 1, nil)
	}
	correlatedLogger.Info("plugins registered", "count", len(registry.List()))

	// 3. Publish events
	for _, plugin := range registry.List() {
		_ = eventBus.Publish(context.Background(), "plugin-lifecycle", map[string]any{
			"plugin": plugin.Name(),
			"event":  "initialized",
		})
		metricsCollector.RecordCounter("plugin_events", 1, map[string]string{"plugin": plugin.Name()})
	}
	correlatedLogger.Info("plugin events published")

	// 4. Record metrics
	metricsCollector.RecordGauge("system_health", 100.0, nil)
	correlatedLogger.Info("system health recorded", "value", 100.0)

	// 5. Verify all components
	if len(registry.List()) != 3 {
		t.Error("plugins not registered correctly")
	}

	metrics := metricsCollector.GetMetrics()
	if metrics["counters"] == nil {
		t.Error("metrics not recorded")
	}

	logOutput := logBuffer.String()
	if !bytes.Contains([]byte(logOutput), []byte("workflow-1")) {
		t.Error("correlation ID not in logs")
	}

	correlatedLogger.Info("workflow completed successfully")
	t.Log("✓ Component interaction works correctly")
}

// TestPhase1ErrorHandling tests error handling across components
func TestPhase1ErrorHandling(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	logger := NewDefaultLogger(LogLevelInfo)

	// Try to get non-existent plugin
	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("expected error for non-existent plugin")
	}
	logger.Error("plugin not found", "error", err.Error())

	// Try to unregister non-existent plugin
	err = registry.Unregister("non-existent")
	if err == nil {
		t.Error("expected error for unregistering non-existent plugin")
	}
	logger.Error("unregister failed", "error", err.Error())

	t.Log("✓ Error handling works correctly")
}

// TestPhase1StateConsistency tests state consistency across operations
func TestPhase1StateConsistency(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	metricsCollector := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)

	// Register plugins and record metrics
	for i := 0; i < 5; i++ {
		plugin := &MockPlugin{
			name:    "plugin-" + string(rune(i)),
			version: "1.0.0",
			healthy: true,
		}
		_ = registry.Register(plugin)
		metricsCollector.RecordCounter("plugins", 1, nil)
	}

	// Verify state consistency
	pluginCount := len(registry.List())
	metricCount := metricsCollector.GetMetrics()["counters"].(map[string]int64)["plugins"]

	if pluginCount != 5 {
		t.Errorf("expected 5 plugins, got %d", pluginCount)
	}
	if metricCount != 5 {
		t.Errorf("expected metric count 5, got %d", metricCount)
	}

	logger.Info("state consistency verified", "plugins", pluginCount, "metrics", metricCount)
	t.Log("✓ State consistency maintained")
}
