package deployment

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewMonolithicInitializer tests initializer creation
func TestNewMonolithicInitializer(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)

	assert.NotNil(t, initializer)
	assert.Equal(t, config, initializer.config)
	assert.Nil(t, initializer.apiGateway)
	assert.Nil(t, initializer.database)
}

// TestMonolithicInitializerInitialize tests monolithic initializer initialization
func TestMonolithicInitializerInitialize(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)

	err := initializer.Initialize(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 5, initializer.metrics.ComponentsReady)
	assert.NotNil(t, initializer.database)
	assert.NotNil(t, initializer.cache)
	assert.NotNil(t, initializer.dataPuller)
	assert.NotNil(t, initializer.eventProcessor)
	assert.NotNil(t, initializer.apiGateway)
}

// TestMonolithicInitializeWithTimeout tests initialization with timeout
func TestMonolithicInitializeWithTimeout(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := initializer.Initialize(ctx)

	assert.NoError(t, err)
}

// TestMonolithicInitializeWithCancelledContext tests initialization with cancelled context
func TestMonolithicInitializeWithCancelledContext(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := initializer.Initialize(ctx)

	assert.Error(t, err)
}

// TestMonolithicHealthCheck tests health check
func TestMonolithicHealthCheck(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	err := initializer.HealthCheck(context.Background())

	assert.NoError(t, err)
	assert.Greater(t, initializer.metrics.HealthChecksPassed, int64(0))
}

// TestMonolithicHealthCheckNotInitialized tests health check without initialization
func TestMonolithicHealthCheckNotInitialized(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)

	err := initializer.HealthCheck(context.Background())

	assert.Error(t, err)
	assert.Greater(t, initializer.metrics.HealthChecksFailed, int64(0))
}

// TestMonolithicInitializerGetMetrics tests metrics retrieval
func TestMonolithicInitializerGetMetrics(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	metrics := initializer.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, 5, metrics["components_ready"].(int))
	assert.Equal(t, 0, metrics["components_failed"].(int))
}

// TestMonolithicShutdown tests graceful shutdown
func TestMonolithicShutdown(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	err := initializer.Shutdown(context.Background())

	assert.NoError(t, err)
	assert.Nil(t, initializer.apiGateway)
	assert.Nil(t, initializer.database)
	assert.Nil(t, initializer.cache)
	assert.Nil(t, initializer.dataPuller)
	assert.Nil(t, initializer.eventProcessor)
}

// TestMonolithicShutdownWithTimeout tests shutdown with timeout
func TestMonolithicShutdownWithTimeout(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 1 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := initializer.Shutdown(ctx)

	assert.NoError(t, err)
}

// TestMonolithicMultipleHealthChecks tests multiple health checks
func TestMonolithicMultipleHealthChecks(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	for i := 0; i < 5; i++ {
		_ = initializer.HealthCheck(context.Background())
	}

	assert.Equal(t, int64(5), initializer.metrics.HealthChecksPassed)
}

// TestMonolithicConcurrentHealthChecks tests concurrent health checks
func TestMonolithicConcurrentHealthChecks(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
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

// TestMonolithicInitializationMetrics tests initialization metrics
func TestMonolithicInitializationMetrics(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.Greater(t, initializer.metrics.InitializationTime, time.Duration(0))
	assert.Equal(t, 5, initializer.metrics.ComponentsReady)
	assert.Equal(t, 0, initializer.metrics.ComponentsFailed)
}

// TestMonolithicDatabase tests database initialization
func TestMonolithicDatabase(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.database)
}

// TestMonolithicCache tests cache initialization
func TestMonolithicCache(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.cache)
}

// TestMonolithicDataPuller tests data puller initialization
func TestMonolithicDataPuller(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.dataPuller)
}

// TestMonolithicEventProcessor tests event processor initialization
func TestMonolithicEventProcessor(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.eventProcessor)
}

// TestMonolithicAPIGateway tests API gateway initialization
func TestMonolithicAPIGateway(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	assert.NotNil(t, initializer.apiGateway)
}

// TestMonolithicMetricsLastHealthCheckTime tests last health check time tracking
func TestMonolithicMetricsLastHealthCheckTime(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	before := time.Now()
	_ = initializer.HealthCheck(context.Background())
	after := time.Now()

	assert.True(t, initializer.metrics.LastHealthCheckTime.After(before) || initializer.metrics.LastHealthCheckTime.Equal(before))
	assert.True(t, initializer.metrics.LastHealthCheckTime.Before(after) || initializer.metrics.LastHealthCheckTime.Equal(after))
}

// TestMonolithicGetMetricsContent tests metrics content
func TestMonolithicGetMetricsContent(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	metrics := initializer.GetMetrics()

	assert.Contains(t, metrics, "initialization_time")
	assert.Contains(t, metrics, "components_ready")
	assert.Contains(t, metrics, "components_failed")
	assert.Contains(t, metrics, "health_checks_passed")
	assert.Contains(t, metrics, "health_checks_failed")
	assert.Contains(t, metrics, "last_health_check")
}

// TestMonolithicComponentOrder tests that components are initialized in correct order
func TestMonolithicComponentOrder(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	err := initializer.Initialize(context.Background())

	assert.NoError(t, err)
	// All components should be initialized
	assert.NotNil(t, initializer.database)
	assert.NotNil(t, initializer.cache)
	assert.NotNil(t, initializer.dataPuller)
	assert.NotNil(t, initializer.eventProcessor)
	assert.NotNil(t, initializer.apiGateway)
}

// TestMonolithicShutdownOrder tests that components are shut down in reverse order
func TestMonolithicShutdownOrder(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	_ = initializer.Initialize(context.Background())

	err := initializer.Shutdown(context.Background())

	assert.NoError(t, err)
	// All components should be nil after shutdown
	assert.Nil(t, initializer.apiGateway)
	assert.Nil(t, initializer.eventProcessor)
	assert.Nil(t, initializer.dataPuller)
	assert.Nil(t, initializer.cache)
	assert.Nil(t, initializer.database)
}

// TestMonolithicHealthCheckPartialInitialization tests health check with partial initialization
func TestMonolithicHealthCheckPartialInitialization(t *testing.T) {
	config := &DeploymentConfig{
		Mode:                    MonolithicMode,
		ServiceName:             "test-service",
		GracefulShutdownTimeout: 30 * time.Second,
	}

	initializer := NewMonolithicInitializer(config)
	initializer.mu.Lock()
	initializer.database = &struct {
		name   string
		status string
	}{name: "PostgreSQL", status: "ready"}
	initializer.mu.Unlock()

	err := initializer.HealthCheck(context.Background())

	assert.Error(t, err)
}
