package deployment

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/stretchr/testify/assert"
)

// MonoMockPluginRegistry for testing
type MonoMockPluginRegistry struct{}

func (m *MonoMockPluginRegistry) Register(plugin core.Plugin) error {
	return nil
}

func (m *MonoMockPluginRegistry) Unregister(name string) error {
	return nil
}

func (m *MonoMockPluginRegistry) Get(name string) (core.Plugin, error) {
	return nil, fmt.Errorf("not found")
}

func (m *MonoMockPluginRegistry) List() []core.Plugin {
	return []core.Plugin{}
}

func (m *MonoMockPluginRegistry) Start() error {
	return nil
}

func (m *MonoMockPluginRegistry) Stop() error {
	return nil
}

// MonoMockConfigManager for testing
type MonoMockConfigManager struct{}

func (m *MonoMockConfigManager) Load() (core.Config, error) {
	return core.Config{ServiceName: "test-service"}, nil
}

func (m *MonoMockConfigManager) Validate(config core.Config) error {
	return nil
}

func (m *MonoMockConfigManager) Get(key string) (any, error) {
	return nil, nil
}

func (m *MonoMockConfigManager) Set(key string, value any) error {
	return nil
}

// MonoMockEventBus for testing
type MonoMockEventBus struct{}

func (m *MonoMockEventBus) Publish(ctx context.Context, topic string, event any) error {
	return nil
}

func (m *MonoMockEventBus) Subscribe(ctx context.Context, topic string, handler func(any)) (uint64, error) {
	return 0, nil
}

func (m *MonoMockEventBus) SubscribeNamed(ctx context.Context, topic, name string, handler func(any)) (uint64, error) {
	return 0, nil
}

func (m *MonoMockEventBus) Unsubscribe(subscriptionID uint64) error {
	return nil
}

// MonoMockLogger for testing
type MonoMockLogger struct{}

func (m *MonoMockLogger) Debug(msg string, fields ...any) {}
func (m *MonoMockLogger) Info(msg string, fields ...any)  {}
func (m *MonoMockLogger) Warn(msg string, fields ...any)  {}
func (m *MonoMockLogger) Error(msg string, fields ...any) {}
func (m *MonoMockLogger) Fatal(msg string, fields ...any) {}
func (m *MonoMockLogger) WithCorrelationID(id string) core.Logger {
	return m
}

// MonoMockMetricsCollector for testing
type MonoMockMetricsCollector struct{}

func (m *MonoMockMetricsCollector) RecordCounter(name string, value int64, tags map[string]string) {}

func (m *MonoMockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {}

func (m *MonoMockMetricsCollector) RecordHistogram(name string, value float64, tags map[string]string) {
}

func (m *MonoMockMetricsCollector) GetMetrics() map[string]any {
	return make(map[string]any)
}

// MonoMockHealthChecker for testing
type MonoMockHealthChecker struct{}

func (m *MonoMockHealthChecker) Check(ctx context.Context) (core.HealthStatus, error) {
	return core.HealthStatus{Status: "healthy"}, nil
}

// TestMonolithicNewDeployment tests deployment creation
func TestMonolithicNewDeployment(t *testing.T) {
	t.Parallel()
	config := core.Config{ServiceName: "test-service"}
	registry := &MonoMockPluginRegistry{}
	configManager := &MonoMockConfigManager{}
	eventBus := &MonoMockEventBus{}
	logger := &MonoMockLogger{}
	metricsCollector := &MonoMockMetricsCollector{}
	healthChecker := &MonoMockHealthChecker{}

	md := NewMonolithicDeployment(config, registry, configManager, eventBus, logger, metricsCollector, healthChecker)

	assert.NotNil(t, md)
	assert.False(t, md.isRunning)
	assert.Equal(t, 0, md.GetServiceCount())
}

// TestMonolithicRegisterService tests service registration
func TestMonolithicRegisterService(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	err := md.RegisterService("service1", func() error { return nil }, func() error { return nil }, func() error { return nil })

	assert.NoError(t, err)
	assert.Equal(t, 1, md.GetServiceCount())
}

// TestMonolithicGetServiceCount tests service count retrieval
func TestMonolithicGetServiceCount(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	_ = md.RegisterService("service1", func() error { return nil }, func() error { return nil }, func() error { return nil })
	_ = md.RegisterService("service2", func() error { return nil }, func() error { return nil }, func() error { return nil })

	count := md.GetServiceCount()

	assert.Equal(t, 2, count)
}

// TestMonolithicGetServices tests service names retrieval
func TestMonolithicGetServices(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	_ = md.RegisterService("service1", func() error { return nil }, func() error { return nil }, func() error { return nil })
	_ = md.RegisterService("service2", func() error { return nil }, func() error { return nil }, func() error { return nil })

	services := md.GetServices()

	assert.Equal(t, 2, len(services))
	assert.Contains(t, services, "service1")
	assert.Contains(t, services, "service2")
}

// TestMonolithicInitialize tests deployment initialization
func TestMonolithicInitialize(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	_ = md.RegisterService("service1", func() error { return nil }, func() error { return nil }, func() error { return nil })

	err := md.Initialize(context.Background())

	assert.NoError(t, err)
}

// TestMonolithicGetHealth tests health status
func TestMonolithicGetHealth(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	health, err := md.GetHealth(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "unhealthy", health.Status)
}

// TestMonolithicGetHealthRunning tests health status when running
func TestMonolithicGetHealthRunning(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	md.mu.Lock()
	md.isRunning = true
	md.mu.Unlock()

	health, err := md.GetHealth(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "healthy", health.Status)
}

// TestMonolithicGetMetrics tests metrics retrieval
func TestMonolithicGetMetrics(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	metrics := md.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, false, metrics["is_running"])
	assert.Equal(t, "monolithic", metrics["deployment_mode"])
}

// TestMonolithicConcurrentOperations tests concurrent operations
func TestMonolithicConcurrentOperations(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = md.GetServiceCount()
			_ = md.IsRunning()
			_ = md.GetServices()
			atomic.AddInt32(&successCount, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&successCount))
}

// TestMonolithicInitializeMultipleServices tests initializing multiple services
func TestMonolithicInitializeMultipleServices(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	var initCount int32

	for i := 0; i < 3; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		_ = md.RegisterService(
			serviceName,
			func() error {
				atomic.AddInt32(&initCount, 1)
				return nil
			},
			func() error { return nil },
			func() error { return nil },
		)
	}

	_ = md.Initialize(context.Background())

	assert.Equal(t, int32(3), atomic.LoadInt32(&initCount))
}

// TestMonolithicInitializeServiceFailure tests initialization failure
func TestMonolithicInitializeServiceFailure(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	_ = md.RegisterService(
		"service1",
		func() error { return fmt.Errorf("initialization failed") },
		func() error { return nil },
		func() error { return nil },
	)

	err := md.Initialize(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialization failed")
}

// TestMonolithicShutdownNotRunning tests shutdown when not running
func TestMonolithicShutdownNotRunning(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	err := md.Shutdown()

	assert.Error(t, err)
}

// TestMonolithicStopNotRunning tests stop when not running
func TestMonolithicStopNotRunning(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	err := md.Stop(context.Background())

	assert.NoError(t, err)
}

// TestMonolithicMultipleServices tests registering multiple services
func TestMonolithicMultipleServices(t *testing.T) {
	t.Parallel()
	md := NewMonolithicDeployment(
		core.Config{ServiceName: "test"},
		&MonoMockPluginRegistry{},
		&MonoMockConfigManager{},
		&MonoMockEventBus{},
		&MonoMockLogger{},
		&MonoMockMetricsCollector{},
		&MonoMockHealthChecker{},
	)

	for i := 0; i < 5; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		_ = md.RegisterService(
			serviceName,
			func() error { return nil },
			func() error { return nil },
			func() error { return nil },
		)
	}

	assert.Equal(t, 5, md.GetServiceCount())
}
