package deployment

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewMicroserviceInitializer tests initializer creation
func TestNewMicroserviceInitializer(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)

	assert.NotNil(t, initializer)
	assert.Equal(t, config, initializer.config)
	assert.Equal(t, 0, len(initializer.services))
}

// TestInitialize tests microservice initialization
func TestMicroserviceInitializerInitialize(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)

	err := initializer.Initialize(context.Background())

	assert.NoError(t, err)
	assert.Greater(t, initializer.metrics.ServicesRegistered, 0)
}

// TestInitializeWithTimeout tests initialization with timeout
func TestInitializeWithTimeout(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := initializer.Initialize(ctx)

	assert.NoError(t, err)
}

// TestInitializeWithCancelledContext tests initialization with cancelled context
func TestInitializeWithCancelledContext(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := initializer.Initialize(ctx)

	assert.Error(t, err)
}

// TestHealthCheck tests health check
func TestMicroserviceInitializerHealthCheck(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	err := initializer.HealthCheck(context.Background())

	assert.NoError(t, err)
	assert.Greater(t, initializer.metrics.HealthChecksPassed, int64(0))
}

// TestHealthCheckNotInitialized tests health check without initialization
func TestHealthCheckNotInitialized(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)

	err := initializer.HealthCheck(context.Background())

	assert.Error(t, err)
	assert.Greater(t, initializer.metrics.HealthChecksFailed, int64(0))
}

// TestGetMetrics tests metrics retrieval
func TestMicroserviceInitializerGetMetrics(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	metrics := initializer.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Greater(t, metrics["services_registered"].(int), 0)
}

// TestGetRegisteredServices tests getting registered services
func TestMicroserviceInitializerGetRegisteredServices(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	services := initializer.GetRegisteredServices()

	assert.Greater(t, len(services), 0)
}

// TestShutdown tests graceful shutdown
func TestMicroserviceInitializerShutdown(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	err := initializer.Shutdown(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 0, len(initializer.services))
}

// TestShutdownWithTimeout tests shutdown with timeout
func TestShutdownWithTimeout(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 1 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := initializer.Shutdown(ctx)

	assert.NoError(t, err)
}

// TestMultipleHealthChecks tests multiple health checks
func TestMultipleHealthChecks(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	for i := 0; i < 5; i++ {
		_ = initializer.HealthCheck(context.Background())
	}

	assert.Equal(t, int64(5), initializer.metrics.HealthChecksPassed)
}

// TestConcurrentHealthChecks tests concurrent health checks
func TestConcurrentHealthChecks(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := initializer.HealthCheck(context.Background()); err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(10), atomic.LoadInt32(&successCount))
}

// TestInitializationMetrics tests initialization metrics
func TestInitializationMetrics(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.Greater(t, initializer.metrics.InitializationTime, time.Duration(0))
	assert.Equal(t, 4, initializer.metrics.ServicesRegistered)
	assert.Equal(t, 0, initializer.metrics.ServicesFailed)
}

// TestServiceRegistry tests service registry initialization
func TestServiceRegistry(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.serviceRegistry)
}

// TestMessageQueue tests message queue initialization
func TestMessageQueue(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.messageQueue)
}

// TestDistributedCache tests distributed cache initialization
func TestDistributedCache(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.distributedCache)
}

// TestShutdownClearsServices tests that shutdown clears services
func TestShutdownClearsServices(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	initialServices := len(initializer.services)
	assert.Greater(t, initialServices, 0)

	_ = initializer.Shutdown(context.Background())

	assert.Equal(t, 0, len(initializer.services))
}

// TestShutdownClearsInfrastructure tests that shutdown clears infrastructure
func TestShutdownClearsInfrastructure(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.serviceRegistry)
	assert.NotNil(t, initializer.messageQueue)
	assert.NotNil(t, initializer.distributedCache)

	_ = initializer.Shutdown(context.Background())

	assert.Nil(t, initializer.serviceRegistry)
	assert.Nil(t, initializer.messageQueue)
	assert.Nil(t, initializer.distributedCache)
}

// TestMetricsLastHealthCheckTime tests last health check time tracking
func TestMetricsLastHealthCheckTime(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	before := time.Now()
	_ = initializer.HealthCheck(context.Background())
	after := time.Now()

	assert.True(t, initializer.metrics.LastHealthCheckTime.After(before) || initializer.metrics.LastHealthCheckTime.Equal(before))
	assert.True(t, initializer.metrics.LastHealthCheckTime.Before(after) || initializer.metrics.LastHealthCheckTime.Equal(after))
}

// TestGetMetricsContent tests metrics content
func TestGetMetricsContent(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	metrics := initializer.GetMetrics()

	assert.Contains(t, metrics, "initialization_time")
	assert.Contains(t, metrics, "services_registered")
	assert.Contains(t, metrics, "services_failed")
	assert.Contains(t, metrics, "health_checks_passed")
	assert.Contains(t, metrics, "health_checks_failed")
	assert.Contains(t, metrics, "inter_service_calls")
	assert.Contains(t, metrics, "last_health_check")
}

// TestInitializeServices tests that all expected services are registered
func TestInitializeServices(t *testing.T) {
	t.Parallel()
	config := &DeploymentConfig{
		Mode:                    MicroserviceMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMicroserviceInitializer(config)
	_ = initializer.Initialize(context.Background())

	services := initializer.GetRegisteredServices()

	expectedServices := []string{"api-gateway", "data-puller", "event-processor", "query-service"}
	for _, service := range expectedServices {
		assert.Contains(t, services, service)
	}
}
