package reliability

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewGracefulShutdownManager tests manager creation
func TestNewGracefulShutdownManager(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	assert.NotNil(t, manager)
	assert.Equal(t, "test-manager", manager.id)
	assert.NotNil(t, manager.services)
	assert.Equal(t, 30*time.Second, manager.connectionDrainTime)
	assert.Equal(t, 60*time.Second, manager.requestCompletionWait)
	assert.False(t, manager.shutdownInProgress)
}

// TestRegisterService tests registering a service
func TestRegisterService(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	assert.Equal(t, 1, len(manager.services))
	info, exists := manager.services["service-1"]
	assert.True(t, exists)
	assert.Equal(t, "service-1", info.ServiceID)
	assert.Equal(t, "running", info.Status)
}

// TestRegisterMultipleServices tests registering multiple services
func TestRegisterMultipleServices(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	for i := 1; i <= 5; i++ {
		serviceID := "service-" + string(rune(48+i))
		manager.RegisterService(serviceID)
	}

	assert.Equal(t, 5, len(manager.services))
}

// TestInitiateShutdown tests initiating shutdown
func TestInitiateShutdown(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")

	err := manager.InitiateShutdown(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), manager.metrics.ShutdownsInitiated)
	assert.Equal(t, int64(1), manager.metrics.ShutdownsCompleted)
}

// TestInitiateShutdownAlreadyInProgress tests initiating shutdown when already in progress
func TestInitiateShutdownAlreadyInProgress(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")

	// First shutdown
	_ = manager.InitiateShutdown(ctx)

	// Try second shutdown
	manager.shutdownInProgress = true
	err := manager.InitiateShutdown(ctx)

	assert.Error(t, err)
}

// TestUpdateConnectionCount tests updating connection count
func TestUpdateConnectionCount(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	err := manager.UpdateConnectionCount("service-1", 5)

	assert.NoError(t, err)
	assert.Equal(t, 5, manager.services["service-1"].ActiveConnections)
}

// TestUpdateConnectionCountNotFound tests updating non-existent service
func TestUpdateConnectionCountNotFound(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	err := manager.UpdateConnectionCount("nonexistent", 5)

	assert.Error(t, err)
}

// TestUpdateConnectionCountZero tests updating connection count to zero
func TestUpdateConnectionCountZero(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")
	_ = manager.UpdateConnectionCount("service-1", 5)

	err := manager.UpdateConnectionCount("service-1", 0)

	assert.NoError(t, err)
	assert.Equal(t, 0, manager.services["service-1"].ActiveConnections)
	assert.NotZero(t, manager.services["service-1"].LastConnectionClosed)
}

// TestUpdatePendingRequests tests updating pending requests
func TestUpdatePendingRequests(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	err := manager.UpdatePendingRequests("service-1", 10)

	assert.NoError(t, err)
	assert.Equal(t, 10, manager.services["service-1"].PendingRequests)
}

// TestUpdatePendingRequestsNotFound tests updating non-existent service
func TestUpdatePendingRequestsNotFound(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	err := manager.UpdatePendingRequests("nonexistent", 10)

	assert.Error(t, err)
}

// TestUpdatePendingRequestsZero tests updating pending requests to zero
func TestUpdatePendingRequestsZero(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")
	_ = manager.UpdatePendingRequests("service-1", 10)

	err := manager.UpdatePendingRequests("service-1", 0)

	assert.NoError(t, err)
	assert.Equal(t, 0, manager.services["service-1"].PendingRequests)
	assert.NotZero(t, manager.services["service-1"].LastRequestCompleted)
}

// TestGetShutdownStatus tests getting shutdown status
func TestGetShutdownStatus(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")
	_ = manager.UpdateConnectionCount("service-1", 5)
	_ = manager.UpdatePendingRequests("service-1", 10)

	status := manager.GetShutdownStatus()

	assert.NotNil(t, status)
	assert.Contains(t, status, "service-1")
	serviceStatus := status["service-1"].(map[string]interface{})
	assert.Equal(t, "running", serviceStatus["status"])
	assert.Equal(t, 5, serviceStatus["active_connections"])
	assert.Equal(t, 10, serviceStatus["pending_requests"])
}

// TestGetMetricsGracefulShutdown tests getting metrics
func TestGetMetricsGracefulShutdown(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")
	_ = manager.InitiateShutdown(ctx)

	metrics := manager.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "shutdowns_initiated")
	assert.Contains(t, metrics, "shutdowns_completed")
	assert.Contains(t, metrics, "connections_drained")
	assert.Contains(t, metrics, "requests_completed")
	assert.Contains(t, metrics, "average_shutdown_time")
	assert.Contains(t, metrics, "total_shutdown_time")
}

// TestIsShutdownInProgress tests checking if shutdown is in progress
func TestIsShutdownInProgress(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	assert.False(t, manager.IsShutdownInProgress())

	manager.shutdownInProgress = true

	assert.True(t, manager.IsShutdownInProgress())
}

// TestDeregister tests deregistering a service
func TestDeregister(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")
	assert.Equal(t, 1, len(manager.services))

	err := manager.Deregister("service-1")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(manager.services))
}

// TestDeregisterNotFound tests deregistering non-existent service
func TestDeregisterNotFound(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	err := manager.Deregister("nonexistent")

	assert.Error(t, err)
}

// TestShutdownInfoFields tests shutdown info fields
func TestShutdownInfoFields(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")
	_ = manager.UpdateConnectionCount("service-1", 5)
	_ = manager.UpdatePendingRequests("service-1", 10)

	info := manager.services["service-1"]

	assert.Equal(t, "service-1", info.ServiceID)
	assert.Equal(t, "running", info.Status)
	assert.Equal(t, 5, info.ActiveConnections)
	assert.Equal(t, 10, info.PendingRequests)
}

// TestMultipleServicesShutdown tests shutting down multiple services
func TestMultipleServicesShutdown(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 1; i <= 3; i++ {
		serviceID := "service-" + string(rune(48+i))
		manager.RegisterService(serviceID)
		_ = manager.UpdateConnectionCount(serviceID, i)
		_ = manager.UpdatePendingRequests(serviceID, 0) // Set to 0 so shutdown completes
	}

	err := manager.InitiateShutdown(ctx)

	assert.NoError(t, err)

	// All services should be stopped
	for i := 1; i <= 3; i++ {
		serviceID := "service-" + string(rune(48+i))
		assert.Equal(t, "stopped", manager.services[serviceID].Status)
	}
}

// TestShutdownMetricsTracking tests shutdown metrics tracking
func TestShutdownMetricsTracking(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")

	for i := 0; i < 3; i++ {
		_ = manager.InitiateShutdown(ctx)
	}

	assert.Equal(t, int64(3), manager.metrics.ShutdownsInitiated)
	assert.Equal(t, int64(3), manager.metrics.ShutdownsCompleted)
	assert.Greater(t, manager.metrics.AverageShutdownTime, time.Duration(0))
}

// TestConnectionDrainTime tests connection drain time
func TestConnectionDrainTime(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	assert.Equal(t, 30*time.Second, manager.connectionDrainTime)

	manager.connectionDrainTime = 10 * time.Second

	assert.Equal(t, 10*time.Second, manager.connectionDrainTime)
}

// TestRequestCompletionWait tests request completion wait time
func TestRequestCompletionWait(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	assert.Equal(t, 60*time.Second, manager.requestCompletionWait)

	manager.requestCompletionWait = 30 * time.Second

	assert.Equal(t, 30*time.Second, manager.requestCompletionWait)
}

// TestConcurrentServiceRegistration tests concurrent service registration
func TestConcurrentServiceRegistration(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			serviceID := "service-" + string(rune(48+(id%10)))
			manager.RegisterService(serviceID)
			atomic.AddInt32(&counter, 1)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&counter))
}

// TestConcurrentConnectionUpdates tests concurrent connection count updates
func TestConcurrentConnectionUpdates(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.UpdateConnectionCount("service-1", i)
			atomic.AddInt32(&counter, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&counter))
}

// TestConcurrentRequestUpdates tests concurrent pending request updates
func TestConcurrentRequestUpdates(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.UpdatePendingRequests("service-1", i)
			atomic.AddInt32(&counter, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(50), atomic.LoadInt32(&counter))
}

// TestShutdownContextCancellation tests shutdown with context cancellation
func TestShutdownContextCancellation(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	ctx, cancel := context.WithCancel(context.Background())

	manager.RegisterService("service-1")

	// Cancel context immediately
	cancel()

	err := manager.InitiateShutdown(ctx)

	// Should handle cancellation gracefully
	assert.NoError(t, err)
}

// TestShutdownWithActiveConnections tests shutdown with active connections
func TestShutdownWithActiveConnections(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")
	_ = manager.UpdateConnectionCount("service-1", 5)

	err := manager.InitiateShutdown(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "stopped", manager.services["service-1"].Status)
}

// TestShutdownWithPendingRequests tests shutdown with pending requests
func TestShutdownWithPendingRequests(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")
	_ = manager.UpdatePendingRequests("service-1", 10)

	err := manager.InitiateShutdown(ctx)

	assert.NoError(t, err)
	assert.Equal(t, "stopped", manager.services["service-1"].Status)
}

// TestShutdownStatusTransitions tests shutdown status transitions
func TestShutdownStatusTransitions(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")

	// Initial status
	assert.Equal(t, "running", manager.services["service-1"].Status)

	// After shutdown
	_ = manager.InitiateShutdown(ctx)

	assert.Equal(t, "stopped", manager.services["service-1"].Status)
}

// TestDrainStartAndCompleteTime tests drain start and complete times
func TestDrainStartAndCompleteTime(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")

	before := time.Now()
	_ = manager.InitiateShutdown(ctx)
	after := time.Now()

	info := manager.services["service-1"]

	assert.True(t, info.DrainStartTime.After(before) || info.DrainStartTime.Equal(before))
	assert.True(t, info.DrainCompleteTime.Before(after) || info.DrainCompleteTime.Equal(after))
	assert.True(t, info.DrainCompleteTime.After(info.DrainStartTime) || info.DrainCompleteTime.Equal(info.DrainStartTime))
}

// TestMetricsAverageShutdownTime tests average shutdown time calculation
func TestMetricsAverageShutdownTime(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	manager.RegisterService("service-1")

	for i := 0; i < 3; i++ {
		_ = manager.InitiateShutdown(ctx)
	}

	assert.Greater(t, manager.metrics.AverageShutdownTime, time.Duration(0))
	assert.Equal(t, manager.metrics.TotalShutdownTime/3, manager.metrics.AverageShutdownTime)
}

// TestEmptyServicesList tests shutdown with empty services list
func TestEmptyServicesList(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := manager.InitiateShutdown(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), manager.metrics.ShutdownsCompleted)
}

// TestGetShutdownStatusEmpty tests getting status with no services
func TestGetShutdownStatusEmpty(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	status := manager.GetShutdownStatus()

	assert.NotNil(t, status)
	assert.Equal(t, 0, len(status))
}

// TestMultipleDeregister tests deregistering multiple services
func TestMultipleDeregister(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	for i := 1; i <= 3; i++ {
		serviceID := "service-" + string(rune(48+i))
		manager.RegisterService(serviceID)
	}

	assert.Equal(t, 3, len(manager.services))

	for i := 1; i <= 3; i++ {
		serviceID := "service-" + string(rune(48+i))
		err := manager.Deregister(serviceID)
		assert.NoError(t, err)
	}

	assert.Equal(t, 0, len(manager.services))
}

// TestShutdownMetricsInitialization tests metrics initialization
func TestShutdownMetricsInitialization(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	assert.Equal(t, int64(0), manager.metrics.ShutdownsInitiated)
	assert.Equal(t, int64(0), manager.metrics.ShutdownsCompleted)
	assert.Equal(t, int64(0), manager.metrics.ShutdownsFailed)
	assert.Equal(t, int64(0), manager.metrics.ConnectionsDrained)
	assert.Equal(t, int64(0), manager.metrics.RequestsCompleted)
	assert.Equal(t, int64(0), manager.metrics.ForcedTerminations)
}

// TestConnectionCountTracking tests connection count tracking
func TestConnectionCountTracking(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	counts := []int{0, 5, 10, 3, 0}

	for _, count := range counts {
		_ = manager.UpdateConnectionCount("service-1", count)
		assert.Equal(t, count, manager.services["service-1"].ActiveConnections)
	}
}

// TestPendingRequestsTracking tests pending requests tracking
func TestPendingRequestsTracking(t *testing.T) {
	manager := NewGracefulShutdownManager("test-manager")

	manager.RegisterService("service-1")

	counts := []int{0, 10, 20, 5, 0}

	for _, count := range counts {
		_ = manager.UpdatePendingRequests("service-1", count)
		assert.Equal(t, count, manager.services["service-1"].PendingRequests)
	}
}
