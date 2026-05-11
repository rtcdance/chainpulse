package health

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"chainpulse/pkg/infrastructure/discovery"
	"github.com/stretchr/testify/assert"
)

// MockServiceRegistry is a mock implementation of ServiceRegistry
type MockServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]*discovery.ServiceInfo
	statuses map[string]string
}

// NewMockServiceRegistry creates a new mock service registry
func NewMockServiceRegistry() *MockServiceRegistry {
	return &MockServiceRegistry{
		services: make(map[string]*discovery.ServiceInfo),
		statuses: make(map[string]string),
	}
}

// RegisterService registers a service
func (m *MockServiceRegistry) RegisterService(ctx context.Context, service *discovery.ServiceInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[service.ID] = service
	m.statuses[service.ID] = "healthy"
	return nil
}

// DeregisterService deregisters a service
func (m *MockServiceRegistry) DeregisterService(ctx context.Context, serviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.services, serviceID)
	delete(m.statuses, serviceID)
	return nil
}

// UpdateServiceStatus updates service status
func (m *MockServiceRegistry) UpdateServiceStatus(ctx context.Context, serviceID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses[serviceID] = status
	return nil
}

// GetService retrieves a service
func (m *MockServiceRegistry) GetService(ctx context.Context, serviceID string) (*discovery.ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	service, exists := m.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found")
	}
	return service, nil
}

// DiscoverService discovers services
func (m *MockServiceRegistry) DiscoverService(ctx context.Context, serviceName string) ([]*discovery.ServiceInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var services []*discovery.ServiceInfo
	for _, service := range m.services {
		if service.Name == serviceName {
			services = append(services, service)
		}
	}
	return services, nil
}

// GetStatus retrieves service status
func (m *MockServiceRegistry) GetStatus(serviceID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statuses[serviceID]
}

// TestNewHealthCheckSystem tests HealthCheckSystem creation
func TestNewHealthCheckSystem(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	assert.NotNil(t, hcs)
	assert.Equal(t, registry, hcs.registry)
	assert.NotNil(t, hcs.endpoints)
	assert.NotNil(t, hcs.results)
	assert.False(t, hcs.running)
}

// TestRegisterHealthCheckEndpoint tests registering a health check endpoint
func TestRegisterHealthCheckEndpoint(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	endpoint := HealthCheckEndpoint{
		ServiceID: "service-1",
		URL:       "http://localhost:8080/health",
		Interval:  10 * time.Second,
		Timeout:   5 * time.Second,
	}

	ctx := context.Background()
	err := hcs.RegisterHealthCheckEndpoint(ctx, endpoint)

	assert.NoError(t, err)

	hcs.mutex.RLock()
	registered, exists := hcs.endpoints["service-1"]
	hcs.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, endpoint.URL, registered.URL)
}

// TestStartHealthCheckSystem tests starting the health check system
func TestStartHealthCheckSystem(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	ctx := context.Background()
	err := hcs.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, hcs.running)

	hcs.Stop()
}

// TestStartHealthCheckSystemAlreadyRunning tests starting already running system
func TestStartHealthCheckSystemAlreadyRunning(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	ctx := context.Background()
	err := hcs.Start(ctx)
	assert.NoError(t, err)

	err = hcs.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	hcs.Stop()
}

// TestStopHealthCheckSystem tests stopping the health check system
func TestStopHealthCheckSystem(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	ctx := context.Background()
	err := hcs.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, hcs.running)

	hcs.Stop()
	assert.False(t, hcs.running)
}

// TestGetHealthCheckResult tests retrieving health check result
func TestGetHealthCheckResult(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	result := &HealthCheckResult{
		ServiceID: "service-1",
		Healthy:   true,
		Message:   "Service is healthy",
		Timestamp: time.Now(),
	}

	hcs.mutex.Lock()
	hcs.results["service-1"] = result
	hcs.mutex.Unlock()

	retrieved, err := hcs.GetHealthCheckResult("service-1")

	assert.NoError(t, err)
	assert.Equal(t, result.ServiceID, retrieved.ServiceID)
	assert.Equal(t, result.Healthy, retrieved.Healthy)
}

// TestGetHealthCheckResultNotFound tests retrieving non-existent result
func TestGetHealthCheckResultNotFound(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	_, err := hcs.GetHealthCheckResult("nonexistent-service")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no health check result found")
}

// TestGetAllHealthCheckResults tests retrieving all results
func TestGetAllHealthCheckResults(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	// Add multiple results
	for i := 0; i < 3; i++ {
		result := &HealthCheckResult{
			ServiceID: fmt.Sprintf("service-%d", i),
			Healthy:   true,
			Timestamp: time.Now(),
		}
		hcs.mutex.Lock()
		hcs.results[fmt.Sprintf("service-%d", i)] = result
		hcs.mutex.Unlock()
	}

	results := hcs.GetAllHealthCheckResults()

	assert.Equal(t, 3, len(results))
}

// TestNewFailureDetector tests FailureDetector creation
func TestNewFailureDetector(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)

	assert.NotNil(t, fd)
	assert.Equal(t, hcs, fd.healthCheckSystem)
	assert.Equal(t, 3, fd.failureThreshold)
	assert.NotNil(t, fd.failureCount)
}

// TestDetectFailuresNoFailures tests failure detection with no failures
func TestDetectFailuresNoFailures(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)

	// Add healthy results
	for i := 0; i < 3; i++ {
		result := &HealthCheckResult{
			ServiceID: fmt.Sprintf("service-%d", i),
			Healthy:   true,
			Timestamp: time.Now(),
		}
		hcs.mutex.Lock()
		hcs.results[fmt.Sprintf("service-%d", i)] = result
		hcs.mutex.Unlock()
	}

	ctx := context.Background()
	failedServices := fd.DetectFailures(ctx)

	assert.Equal(t, 0, len(failedServices))
}

// TestDetectFailuresWithFailures tests failure detection with failures
func TestDetectFailuresWithFailures(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 2)

	// Add unhealthy results
	for i := 0; i < 3; i++ {
		result := &HealthCheckResult{
			ServiceID: fmt.Sprintf("service-%d", i),
			Healthy:   false,
			Timestamp: time.Now(),
		}
		hcs.mutex.Lock()
		hcs.results[fmt.Sprintf("service-%d", i)] = result
		hcs.mutex.Unlock()
	}

	ctx := context.Background()

	// First detection - count = 1
	failedServices := fd.DetectFailures(ctx)
	assert.Equal(t, 0, len(failedServices))

	// Second detection - count = 2, should trigger
	failedServices = fd.DetectFailures(ctx)
	assert.Equal(t, 3, len(failedServices))
}

// TestResetFailureCount tests resetting failure count
func TestResetFailureCount(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)

	// Manually set failure count
	fd.mutex.Lock()
	fd.failureCount["service-1"] = 5
	fd.mutex.Unlock()

	fd.ResetFailureCount("service-1")

	fd.mutex.RLock()
	count := fd.failureCount["service-1"]
	fd.mutex.RUnlock()

	assert.Equal(t, 0, count)
}

// TestNewAutomaticDeregistration tests AutomaticDeregistration creation
func TestNewAutomaticDeregistration(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)
	ad := NewAutomaticDeregistration(registry, fd, 30*time.Second)

	assert.NotNil(t, ad)
	assert.Equal(t, registry, ad.registry)
	assert.Equal(t, fd, ad.failureDetector)
	assert.Equal(t, 30*time.Second, ad.deregistrationTTL)
	assert.False(t, ad.running)
}

// TestStartAutomaticDeregistration tests starting automatic deregistration
func TestStartAutomaticDeregistration(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)
	ad := NewAutomaticDeregistration(registry, fd, 30*time.Second)

	ctx := context.Background()
	err := ad.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, ad.running)

	ad.Stop()
}

// TestStartAutomaticDeregistrationAlreadyRunning tests starting already running system
func TestStartAutomaticDeregistrationAlreadyRunning(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)
	ad := NewAutomaticDeregistration(registry, fd, 30*time.Second)

	ctx := context.Background()
	err := ad.Start(ctx)
	assert.NoError(t, err)

	err = ad.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	ad.Stop()
}

// TestStopAutomaticDeregistration tests stopping automatic deregistration
func TestStopAutomaticDeregistration(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	fd := NewFailureDetector(hcs, 3)
	ad := NewAutomaticDeregistration(registry, fd, 30*time.Second)

	ctx := context.Background()
	err := ad.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, ad.running)

	ad.Stop()
	assert.False(t, ad.running)
}

// TestHealthCheckResultFields tests HealthCheckResult fields
func TestHealthCheckResultFields(t *testing.T) {
	result := &HealthCheckResult{
		ServiceID:    "service-1",
		ServiceName:  "my-service",
		Healthy:      true,
		Message:      "Service is healthy",
		Timestamp:    time.Now(),
		ResponseTime: 100 * time.Millisecond,
	}

	assert.Equal(t, "service-1", result.ServiceID)
	assert.Equal(t, "my-service", result.ServiceName)
	assert.True(t, result.Healthy)
	assert.Equal(t, "Service is healthy", result.Message)
	assert.False(t, result.Timestamp.IsZero())
	assert.Equal(t, 100*time.Millisecond, result.ResponseTime)
}

// TestHealthCheckEndpointFields tests HealthCheckEndpoint fields
func TestHealthCheckEndpointFields(t *testing.T) {
	endpoint := HealthCheckEndpoint{
		ServiceID: "service-1",
		URL:       "http://localhost:8080/health",
		Interval:  10 * time.Second,
		Timeout:   5 * time.Second,
	}

	assert.Equal(t, "service-1", endpoint.ServiceID)
	assert.Equal(t, "http://localhost:8080/health", endpoint.URL)
	assert.Equal(t, 10*time.Second, endpoint.Interval)
	assert.Equal(t, 5*time.Second, endpoint.Timeout)
}

// TestConcurrentHealthCheckRegistration tests concurrent endpoint registration
func TestConcurrentHealthCheckRegistration(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			endpoint := HealthCheckEndpoint{
				ServiceID: fmt.Sprintf("service-%d", id),
				URL:       fmt.Sprintf("http://localhost:808%d/health", id),
				Interval:  10 * time.Second,
				Timeout:   5 * time.Second,
			}
			ctx := context.Background()
			err := hcs.RegisterHealthCheckEndpoint(ctx, endpoint)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	hcs.mutex.RLock()
	count := len(hcs.endpoints)
	hcs.mutex.RUnlock()

	assert.Equal(t, numGoroutines, count)
}

// TestConcurrentResultRetrieval tests concurrent result retrieval
func TestConcurrentResultRetrieval(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	// Add results
	for i := 0; i < 5; i++ {
		result := &HealthCheckResult{
			ServiceID: fmt.Sprintf("service-%d", i),
			Healthy:   true,
			Timestamp: time.Now(),
		}
		hcs.mutex.Lock()
		hcs.results[fmt.Sprintf("service-%d", i)] = result
		hcs.mutex.Unlock()
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			serviceID := fmt.Sprintf("service-%d", id%5)
			_, err := hcs.GetHealthCheckResult(serviceID)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

// TestFailureDetectionThreshold tests failure detection threshold
func TestFailureDetectionThreshold(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)
	threshold := 3
	fd := NewFailureDetector(hcs, threshold)

	// Add unhealthy result
	result := &HealthCheckResult{
		ServiceID: "service-1",
		Healthy:   false,
		Timestamp: time.Now(),
	}
	hcs.mutex.Lock()
	hcs.results["service-1"] = result
	hcs.mutex.Unlock()

	ctx := context.Background()

	// Detect failures multiple times
	for i := 0; i < threshold-1; i++ {
		failedServices := fd.DetectFailures(ctx)
		assert.Equal(t, 0, len(failedServices))
	}

	// Should trigger on threshold
	failedServices := fd.DetectFailures(ctx)
	assert.Equal(t, 1, len(failedServices))
	assert.Contains(t, failedServices, "service-1")
}

// TestHealthCheckResultTimestamp tests result timestamp
func TestHealthCheckResultTimestamp(t *testing.T) {
	before := time.Now()
	result := &HealthCheckResult{
		ServiceID: "service-1",
		Healthy:   true,
		Timestamp: time.Now(),
	}
	after := time.Now()

	assert.True(t, result.Timestamp.After(before) || result.Timestamp.Equal(before))
	assert.True(t, result.Timestamp.Before(after) || result.Timestamp.Equal(after))
}

// TestHealthCheckResponseTime tests response time measurement
func TestHealthCheckResponseTime(t *testing.T) {
	result := &HealthCheckResult{
		ServiceID:    "service-1",
		Healthy:      true,
		ResponseTime: 150 * time.Millisecond,
	}

	assert.Equal(t, 150*time.Millisecond, result.ResponseTime)
	assert.Greater(t, result.ResponseTime, time.Duration(0))
}

func TestHealthCheckSystemStopWaitsForGoroutines(t *testing.T) {
	registry := NewMockServiceRegistry()
	hcs := NewHealthCheckSystem(registry)

	// Register a health check endpoint
	ctx := context.Background()
	endpoint := HealthCheckEndpoint{
		ServiceID: "test-service",
		URL:       "http://localhost:9999",
		Timeout:   1 * time.Second,
	}
	_ = hcs.RegisterHealthCheckEndpoint(ctx, endpoint)

	// Start the health check system
	err := hcs.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, hcs.running)

	// Stop should complete without hanging (proves wg.Wait() works)
	done := make(chan struct{})
	go func() {
		hcs.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success — Stop() waited for goroutines and returned
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() hung — goroutine leak not fixed")
	}

	assert.False(t, hcs.running)
}

func TestAutomaticDeregistrationStopWaitsForGoroutine(t *testing.T) {
	registry := NewMockServiceRegistry()
	fd := NewFailureDetector(NewHealthCheckSystem(registry), 3)
	ad := NewAutomaticDeregistration(registry, fd, 30*time.Second)

	ctx := context.Background()
	err := ad.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, ad.running)

	done := make(chan struct{})
	go func() {
		ad.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("AutomaticDeregistration.Stop() hung")
	}

	assert.False(t, ad.running)
}
