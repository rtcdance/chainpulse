package discovery

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockConsulClient for testing
type MockConsulClient struct {
	mu                    sync.RWMutex
	registeredServices    map[string]bool
	registerCallCount     int32
	deregisterCallCount   int32
	registerError         error
	deregisterError       error
}

func (mcc *MockConsulClient) RegisterService(ctx context.Context, serviceID, serviceName, address string, port int, tags []string) error {
	mcc.mu.Lock()
	defer mcc.mu.Unlock()
	atomic.AddInt32(&mcc.registerCallCount, 1)
	if mcc.registerError != nil {
		return mcc.registerError
	}
	mcc.registeredServices[serviceID] = true
	return nil
}

func (mcc *MockConsulClient) DeregisterService(ctx context.Context, serviceID string) error {
	mcc.mu.Lock()
	defer mcc.mu.Unlock()
	atomic.AddInt32(&mcc.deregisterCallCount, 1)
	if mcc.deregisterError != nil {
		return mcc.deregisterError
	}
	delete(mcc.registeredServices, serviceID)
	return nil
}

// TestNewServiceRegistry tests registry creation
func TestNewServiceRegistry(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.services)
	assert.Equal(t, consul, registry.consul)
}

// TestRegisterService tests registering a service
func TestRegisterService(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
		Tags:    []string{"api", "v1"},
	}

	err := registry.RegisterService(ctx, service)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&consul.registerCallCount))
}

// TestRegisterServiceConsulError tests registration with Consul error
func TestRegisterServiceConsulError(t *testing.T) {
	consul := &MockConsulClient{
		registeredServices: make(map[string]bool),
		registerError:      assert.AnError,
	}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	err := registry.RegisterService(ctx, service)

	assert.Error(t, err)
}

// TestDeregisterService tests deregistering a service
func TestDeregisterService(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)
	err := registry.DeregisterService(ctx, "service-1")

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&consul.deregisterCallCount))
}

// TestDeregisterServiceNotFound tests deregistering non-existent service
func TestDeregisterServiceNotFound(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	err := registry.DeregisterService(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestGetService tests retrieving a service
func TestGetService(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)

	retrieved, err := registry.GetService(ctx, "service-1")

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "service-1", retrieved.ID)
	assert.Equal(t, "api-service", retrieved.Name)
}

// TestGetServiceNotFound tests retrieving non-existent service
func TestGetServiceNotFound(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	_, err := registry.GetService(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestDiscoverServices tests discovering services by name
func TestDiscoverServices(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "service-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "service-2", Name: "api-service", Address: "localhost", Port: 8081},
		{ID: "service-3", Name: "db-service", Address: "localhost", Port: 5432},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	discovered, err := registry.DiscoverServices(ctx, "api-service")

	assert.NoError(t, err)
	assert.Equal(t, 2, len(discovered))
}

// TestDiscoverServicesNoMatch tests discovering with no matching services
func TestDiscoverServicesNoMatch(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	discovered, err := registry.DiscoverServices(ctx, "nonexistent-service")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(discovered))
}

// TestUpdateServiceStatus tests updating service status
func TestUpdateServiceStatus(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)

	err := registry.UpdateServiceStatus(ctx, "service-1", "unhealthy")

	assert.NoError(t, err)

	retrieved, _ := registry.GetService(ctx, "service-1")
	assert.Equal(t, "unhealthy", retrieved.Status)
}

// TestUpdateServiceStatusNotFound tests updating non-existent service
func TestUpdateServiceStatusNotFound(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	err := registry.UpdateServiceStatus(ctx, "nonexistent", "unhealthy")

	assert.Error(t, err)
}

// TestListServices tests listing all services
func TestListServices(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "service-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "service-2", Name: "db-service", Address: "localhost", Port: 5432},
		{ID: "service-3", Name: "cache-service", Address: "localhost", Port: 6379},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	listed := registry.ListServices(ctx)

	assert.Equal(t, 3, len(listed))
}

// TestListServicesEmpty tests listing when no services registered
func TestListServicesEmpty(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	listed := registry.ListServices(ctx)

	assert.Equal(t, 0, len(listed))
}

// TestGetHealthyServices tests getting only healthy services
func TestGetHealthyServices(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "service-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "service-2", Name: "api-service", Address: "localhost", Port: 8081},
		{ID: "service-3", Name: "api-service", Address: "localhost", Port: 8082},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	// Mark one as unhealthy
	_ = registry.UpdateServiceStatus(ctx, "service-2", "unhealthy")

	healthy := registry.GetHealthyServices(ctx)

	assert.Equal(t, 2, len(healthy))
}

// TestNewHealthChecker tests health checker creation
func TestNewHealthChecker(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	hc := NewHealthChecker(registry, 1*time.Second, 5*time.Second)

	assert.NotNil(t, hc)
	assert.Equal(t, registry, hc.registry)
	assert.Equal(t, 1*time.Second, hc.interval)
	assert.Equal(t, 5*time.Second, hc.timeout)
}

// TestHealthCheckerStart tests starting health checker
func TestHealthCheckerStart(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	hc := NewHealthChecker(registry, 1*time.Second, 5*time.Second)
	ctx := context.Background()

	err := hc.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, hc.running)

	hc.Stop()
}

// TestHealthCheckerStartAlreadyRunning tests starting already running checker
func TestHealthCheckerStartAlreadyRunning(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	hc := NewHealthChecker(registry, 1*time.Second, 5*time.Second)
	ctx := context.Background()

	_ = hc.Start(ctx)
	err := hc.Start(ctx)

	assert.Error(t, err)

	hc.Stop()
}

// TestHealthCheckerStop tests stopping health checker
func TestHealthCheckerStop(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	hc := NewHealthChecker(registry, 1*time.Second, 5*time.Second)
	ctx := context.Background()

	_ = hc.Start(ctx)
	hc.Stop()

	assert.False(t, hc.running)
}

// TestNewServiceDiscoveryClient tests discovery client creation
func TestNewServiceDiscoveryClient(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)

	assert.NotNil(t, sdc)
	assert.Equal(t, registry, sdc.registry)
	assert.Equal(t, 5*time.Minute, sdc.cacheTTL)
}

// TestServiceDiscoveryClientGetService tests getting a service
func TestServiceDiscoveryClientGetService(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)

	retrieved, err := sdc.GetService(ctx, "api-service")

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "api-service", retrieved.Name)
}

// TestServiceDiscoveryClientGetServiceNotFound tests getting non-existent service
func TestServiceDiscoveryClientGetServiceNotFound(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	ctx := context.Background()

	_, err := sdc.GetService(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestServiceDiscoveryClientGetServices tests getting multiple services
func TestServiceDiscoveryClientGetServices(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "service-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "service-2", Name: "api-service", Address: "localhost", Port: 8081},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	retrieved, err := sdc.GetServices(ctx, "api-service")

	assert.NoError(t, err)
	assert.Equal(t, 2, len(retrieved))
}

// TestServiceDiscoveryClientCaching tests service caching
func TestServiceDiscoveryClientCaching(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)

	// First call
	_, _ = sdc.GetServices(ctx, "api-service")

	// Check cache
	sdc.mutex.RLock()
	cached, exists := sdc.cache["api-service"]
	sdc.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, 1, len(cached))
}

// TestServiceDiscoveryClientInvalidateCache tests cache invalidation
func TestServiceDiscoveryClientInvalidateCache(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)
	_, _ = sdc.GetServices(ctx, "api-service")

	sdc.InvalidateCache("api-service")

	sdc.mutex.RLock()
	_, exists := sdc.cache["api-service"]
	sdc.mutex.RUnlock()

	assert.False(t, exists)
}

// TestServiceDiscoveryClientInvalidateAllCache tests invalidating all caches
func TestServiceDiscoveryClientInvalidateAllCache(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "service-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "service-2", Name: "db-service", Address: "localhost", Port: 5432},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	_, _ = sdc.GetServices(ctx, "api-service")
	_, _ = sdc.GetServices(ctx, "db-service")

	sdc.InvalidateAllCache()

	sdc.mutex.RLock()
	cacheSize := len(sdc.cache)
	sdc.mutex.RUnlock()

	assert.Equal(t, 0, cacheSize)
}

// TestNewLoadBalancer tests load balancer creation
func TestNewLoadBalancer(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	lb := NewLoadBalancer(sdc, "round-robin")

	assert.NotNil(t, lb)
	assert.Equal(t, sdc, lb.discoveryClient)
	assert.Equal(t, "round-robin", lb.strategy)
}

// TestLoadBalancerSelectService tests selecting a service
func TestLoadBalancerSelectService(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	lb := NewLoadBalancer(sdc, "round-robin")
	ctx := context.Background()

	service := ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	_ = registry.RegisterService(ctx, service)

	selected, err := lb.SelectService(ctx, "api-service")

	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, "api-service", selected.Name)
}

// TestLoadBalancerSelectServiceNotFound tests selecting non-existent service
func TestLoadBalancerSelectServiceNotFound(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	sdc := NewServiceDiscoveryClient(registry, 5*time.Minute)
	lb := NewLoadBalancer(sdc, "round-robin")
	ctx := context.Background()

	_, err := lb.SelectService(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestConcurrentRegisterDeregister tests concurrent registration
func TestConcurrentRegisterDeregister(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			service := ServiceInfo{
				ID:      "service-" + string(rune(48+(id%10))),
				Name:    "api-service",
				Address: "localhost",
				Port:    8080 + id,
			}
			if err := registry.RegisterService(ctx, service); err == nil {
				atomic.AddInt32(&counter, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Greater(t, atomic.LoadInt32(&counter), int32(0))
}

// TestServiceInfoFields tests service info fields
func TestServiceInfoFields(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	service := ServiceInfo{
		ID:             "service-1",
		Name:           "api-service",
		Address:        "localhost",
		Port:           8080,
		Tags:           []string{"api", "v1"},
		HealthCheckURL: "http://localhost:8080/health",
		Metadata: map[string]string{
			"version": "1.0.0",
		},
	}

	_ = registry.RegisterService(ctx, service)

	retrieved, _ := registry.GetService(ctx, "service-1")

	assert.Equal(t, "service-1", retrieved.ID)
	assert.Equal(t, "api-service", retrieved.Name)
	assert.Equal(t, "localhost", retrieved.Address)
	assert.Equal(t, 8080, retrieved.Port)
	assert.Equal(t, 2, len(retrieved.Tags))
	assert.NotZero(t, retrieved.RegisteredAt)
	assert.NotZero(t, retrieved.LastHeartbeat)
	assert.Equal(t, "healthy", retrieved.Status)
}

// TestDiscoverServicesUnhealthy tests that unhealthy services are not discovered
func TestDiscoverServicesUnhealthy(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "service-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "service-2", Name: "api-service", Address: "localhost", Port: 8081},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	// Mark one as unhealthy
	_ = registry.UpdateServiceStatus(ctx, "service-2", "unhealthy")

	discovered, _ := registry.DiscoverServices(ctx, "api-service")

	assert.Equal(t, 1, len(discovered))
	assert.Equal(t, "service-1", discovered[0].ID)
}

// TestMultipleServiceTypes tests registering multiple service types
func TestMultipleServiceTypes(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	ctx := context.Background()

	services := []ServiceInfo{
		{ID: "api-1", Name: "api-service", Address: "localhost", Port: 8080},
		{ID: "db-1", Name: "db-service", Address: "localhost", Port: 5432},
		{ID: "cache-1", Name: "cache-service", Address: "localhost", Port: 6379},
	}

	for _, service := range services {
		_ = registry.RegisterService(ctx, service)
	}

	apiServices, _ := registry.DiscoverServices(ctx, "api-service")
	dbServices, _ := registry.DiscoverServices(ctx, "db-service")
	cacheServices, _ := registry.DiscoverServices(ctx, "cache-service")

	assert.Equal(t, 1, len(apiServices))
	assert.Equal(t, 1, len(dbServices))
	assert.Equal(t, 1, len(cacheServices))
}

// TestHealthCheckerPerformHealthCheck tests health check
func TestHealthCheckerPerformHealthCheck(t *testing.T) {
	consul := &MockConsulClient{registeredServices: make(map[string]bool)}
	registry := NewServiceRegistry(consul)
	hc := NewHealthChecker(registry, 1*time.Second, 5*time.Second)
	ctx := context.Background()

	service := &ServiceInfo{
		ID:      "service-1",
		Name:    "api-service",
		Address: "localhost",
		Port:    8080,
	}

	healthy := hc.performHealthCheck(ctx, service)

	assert.True(t, healthy)
}
