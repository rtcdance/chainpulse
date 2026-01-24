package integration

import (
	"context"
	"testing"
	"time"
)

// TestPhase10DeploymentIntegration tests Phase 10 deployment integration
func TestPhase10DeploymentIntegration(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
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

	t.Log("✓ All Phase 10 components created successfully")
}

// TestPhase10ConfigurationManagement tests configuration management
func TestPhase10ConfigurationManagement(t *testing.T) {
	configManager := NewDefaultConfigManager()

	config, err := configManager.Load()
	if err != nil {
		t.Fatalf("failed to load configuration: %v", err)
	}

	if config.DeploymentMode == "" {
		t.Errorf("expected deployment mode to be set, got empty")
	}

	t.Logf("✓ Configuration management works correctly")
}

// TestPhase10HealthCheckIntegration tests health check integration
func TestPhase10HealthCheckIntegration(t *testing.T) {
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	logger := NewDefaultLogger(LogLevelInfo)
	healthChecker := NewDefaultHealthChecker(registry, configManager, eventBus, metricsCollector, logger)

	ctx := context.Background()

	// Get initial health status
	health, err := healthChecker.Check(ctx)
	if err != nil {
		t.Fatalf("failed to get health status: %v", err)
	}

	if health.Status == "" {
		t.Errorf("expected health status to be set, got empty")
	}

	t.Logf("✓ Health check integration works correctly")
}

// TestPhase10MetricsCollection tests metrics collection
func TestPhase10MetricsCollection(t *testing.T) {
	metricsCollector := NewDefaultMetricsCollector()

	// Record some metrics
	metricsCollector.RecordCounter("requests_total", 100, map[string]string{"endpoint": "/health"})
	metricsCollector.RecordGauge("active_connections", 42, map[string]string{"service": "chainpulse"})
	metricsCollector.RecordHistogram("request_duration_ms", 150, map[string]string{"endpoint": "/api"})

	// Get metrics
	exported := metricsCollector.GetMetrics()

	if exported == nil {
		t.Errorf("expected exported metrics to be set, got nil")
	}

	if len(exported) == 0 {
		t.Errorf("expected exported metrics to contain data, got empty")
	}

	t.Logf("✓ Metrics collection works correctly")
}

// TestPhase10HealthMonitoring tests health monitoring
func TestPhase10HealthMonitoring(t *testing.T) {
	logger := NewDefaultLogger(LogLevelInfo)
	registry := NewDefaultPluginRegistry()
	configManager := NewDefaultConfigManager()
	eventBus := NewDefaultEventBus()
	metricsCollector := NewDefaultMetricsCollector()
	healthChecker := NewDefaultHealthChecker(registry, configManager, eventBus, metricsCollector, logger)

	ctx := context.Background()

	// Check health multiple times
	for i := 0; i < 5; i++ {
		health, err := healthChecker.Check(ctx)
		if err != nil {
			t.Fatalf("failed to check health: %v", err)
		}

		if health.Status == "" {
			t.Errorf("expected health status to be set, got empty")
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("✓ Health monitoring works correctly")
}
