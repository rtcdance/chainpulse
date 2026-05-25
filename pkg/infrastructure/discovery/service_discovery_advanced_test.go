package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRoutingUpdateStructure tests RoutingUpdate structure
func TestRoutingUpdateStructure(t *testing.T) {
	t.Parallel()
	services := []*ServiceInfo{
		{ID: "service-1", Name: "api", Port: 8080},
		{ID: "service-2", Name: "api", Port: 8081},
	}

	update := RoutingUpdate{
		ServiceName: "api",
		Services:    services,
		Timestamp:   time.Now(),
	}

	assert.Equal(t, "api", update.ServiceName)
	assert.Equal(t, 2, len(update.Services))
	assert.NotZero(t, update.Timestamp)
}

// TestNewAdvancedServiceDiscoveryClient tests creating a new advanced service discovery client
func TestNewAdvancedServiceDiscoveryClient(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}

	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	assert.NotNil(t, client)
	assert.Equal(t, discoveryClient, client.discoveryClient)
	assert.Equal(t, registry, client.registry)
	assert.NotNil(t, client.listeners)
	assert.False(t, client.running)
}

// TestRegisterRoutingUpdateListener tests registering a routing update listener
func TestRegisterRoutingUpdateListener(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}
	client.RegisterRoutingUpdateListener("api", listener)

	client.listenerMutex.RLock()
	assert.Equal(t, 1, len(client.listeners["api"]))
	client.listenerMutex.RUnlock()
}

// TestRegisterMultipleListeners tests registering multiple listeners
func TestRegisterMultipleListeners(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener1 := func(update RoutingUpdate) {}
	listener2 := func(update RoutingUpdate) {}

	client.RegisterRoutingUpdateListener("api", listener1)
	client.RegisterRoutingUpdateListener("api", listener2)

	client.listenerMutex.RLock()
	assert.Equal(t, 2, len(client.listeners["api"]))
	client.listenerMutex.RUnlock()
}

// TestRegisterListenersForDifferentServices tests registering listeners for different services
func TestRegisterListenersForDifferentServices(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}

	client.RegisterRoutingUpdateListener("api", listener)
	client.RegisterRoutingUpdateListener("auth", listener)
	client.RegisterRoutingUpdateListener("database", listener)

	client.listenerMutex.RLock()
	assert.Equal(t, 1, len(client.listeners["api"]))
	assert.Equal(t, 1, len(client.listeners["auth"]))
	assert.Equal(t, 1, len(client.listeners["database"]))
	client.listenerMutex.RUnlock()
}

// TestStartAdvancedServiceDiscoveryClient tests starting the client
func TestStartAdvancedServiceDiscoveryClient(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	err := client.Start(ctx)

	assert.NoError(t, err)
	client.runningMutex.RLock()
	assert.True(t, client.running)
	client.runningMutex.RUnlock()

	client.Stop()
}

func TestNewServiceEndpointCache(t *testing.T) {
	t.Parallel()

	cache := NewServiceEndpointCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("NewServiceEndpointCache returned nil")
	}
	if cache.cache == nil {
		t.Error("cache map is nil")
	}
	if cache.cacheTTL != 5*time.Minute {
		t.Errorf("cacheTTL = %v, want 5m", cache.cacheTTL)
	}
}

func TestServiceEndpointCache_SetAndGet(t *testing.T) {
	t.Parallel()

	cache := NewServiceEndpointCache(5 * time.Minute)
	services := []*ServiceInfo{
		{ID: "svc-1", Name: "api", Port: 8080},
		{ID: "svc-2", Name: "api", Port: 8081},
	}

	cache.Set("api", services)

	got, ok := cache.Get("api")
	if !ok {
		t.Fatal("expected to find cached services")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 services, got %d", len(got))
	}
}

func TestServiceEndpointCache_Get_NotFound(t *testing.T) {
	t.Parallel()

	cache := NewServiceEndpointCache(5 * time.Minute)
	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestServiceEndpointCache_Get_Expired(t *testing.T) {
	t.Parallel()

	cache := NewServiceEndpointCache(1 * time.Millisecond)
	cache.Set("api", []*ServiceInfo{{ID: "svc-1"}})

	time.Sleep(5 * time.Millisecond)

	_, ok := cache.Get("api")
	if ok {
		t.Error("expected expired entry not to be returned")
	}
}

func TestServiceEndpointCache_Invalidate(t *testing.T) {
	t.Parallel()

	cache := NewServiceEndpointCache(5 * time.Minute)
	cache.Set("api", []*ServiceInfo{{ID: "svc-1"}})
	cache.Invalidate("api")

	_, ok := cache.Get("api")
	if ok {
		t.Error("expected entry to be invalidated")
	}
}

func TestServiceEndpointCache_InvalidateAll(t *testing.T) {
	t.Parallel()

	cache := NewServiceEndpointCache(5 * time.Minute)
	cache.Set("api", []*ServiceInfo{{ID: "svc-1"}})
	cache.Set("web", []*ServiceInfo{{ID: "svc-2"}})

	cache.InvalidateAll()

	_, ok := cache.Get("api")
	if ok {
		t.Error("expected all entries to be invalidated")
	}
}

func TestRoundRobinStrategy_SelectService_Empty(t *testing.T) {
	t.Parallel()

	strategy := &RoundRobinStrategy{}
	svc := strategy.SelectService(nil)
	if svc != nil {
		t.Error("expected nil for empty services")
	}

	svc = strategy.SelectService([]*ServiceInfo{})
	if svc != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestRoundRobinStrategy_SelectService(t *testing.T) {
	strategy := &RoundRobinStrategy{}
	services := []*ServiceInfo{
		{ID: "svc-1"},
		{ID: "svc-2"},
		{ID: "svc-3"},
	}

	first := strategy.SelectService(services)
	if first.ID != "svc-1" {
		t.Errorf("expected svc-1, got %s", first.ID)
	}

	second := strategy.SelectService(services)
	if second.ID != "svc-2" {
		t.Errorf("expected svc-2, got %s", second.ID)
	}

	third := strategy.SelectService(services)
	if third.ID != "svc-3" {
		t.Errorf("expected svc-3, got %s", third.ID)
	}

	fourth := strategy.SelectService(services)
	if fourth.ID != "svc-1" {
		t.Errorf("expected svc-1 (wrapped), got %s", fourth.ID)
	}
}

func TestNewServiceConnectionPool(t *testing.T) {
	t.Parallel()

	pool := NewServiceConnectionPool(10)
	if pool == nil {
		t.Fatal("NewServiceConnectionPool returned nil")
	}
	if pool.maxConnections != 10 {
		t.Errorf("maxConnections = %d, want 10", pool.maxConnections)
	}
}

func TestServiceConnectionPool_SetAndGet(t *testing.T) {
	t.Parallel()

	pool := NewServiceConnectionPool(10)
	err := pool.SetConnection("svc-1", "conn-1")
	if err != nil {
		t.Fatalf("SetConnection failed: %v", err)
	}

	conn, err := pool.GetConnection("svc-1")
	if err != nil {
		t.Fatalf("GetConnection failed: %v", err)
	}
	if conn != "conn-1" {
		t.Errorf("expected conn-1, got %v", conn)
	}
}

func TestServiceConnectionPool_Get_NotFound(t *testing.T) {
	t.Parallel()

	pool := NewServiceConnectionPool(10)
	_, err := pool.GetConnection("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent connection")
	}
}

func TestServiceConnectionPool_Set_PoolFull(t *testing.T) {
	t.Parallel()

	pool := NewServiceConnectionPool(2)
	pool.SetConnection("svc-1", "conn-1")
	pool.SetConnection("svc-2", "conn-2")

	err := pool.SetConnection("svc-3", "conn-3")
	if err == nil {
		t.Error("expected error for full pool")
	}
}

func TestServiceConnectionPool_RemoveConnection(t *testing.T) {
	t.Parallel()

	pool := NewServiceConnectionPool(10)
	pool.SetConnection("svc-1", "conn-1")
	pool.RemoveConnection("svc-1")

	_, err := pool.GetConnection("svc-1")
	if err == nil {
		t.Error("expected error after removal")
	}
}

func TestServiceConnectionPool_ClearConnections(t *testing.T) {
	t.Parallel()

	pool := NewServiceConnectionPool(10)
	pool.SetConnection("svc-1", "conn-1")
	pool.SetConnection("svc-2", "conn-2")
	pool.ClearConnections()

	_, err := pool.GetConnection("svc-1")
	if err == nil {
		t.Error("expected error after clear")
	}
}

// TestStartAlreadyRunning tests starting when already running
func TestStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	_ = client.Start(ctx)

	err := client.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	client.Stop()
}

// TestStopAdvancedServiceDiscoveryClient tests stopping the client
func TestStopAdvancedServiceDiscoveryClient(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	_ = client.Start(ctx)
	client.Stop()

	client.runningMutex.RLock()
	assert.False(t, client.running)
	client.runningMutex.RUnlock()
}

// TestRoutingUpdateTimestamp tests routing update timestamp
func TestRoutingUpdateTimestamp(t *testing.T) {
	t.Parallel()
	before := time.Now()
	update := RoutingUpdate{
		ServiceName: "api",
		Timestamp:   time.Now(),
	}
	after := time.Now()

	assert.True(t, update.Timestamp.After(before) || update.Timestamp.Equal(before))
	assert.True(t, update.Timestamp.Before(after) || update.Timestamp.Equal(after))
}

// TestAdvancedServiceDiscoveryClientListenerStorage tests listener storage
func TestAdvancedServiceDiscoveryClientListenerStorage(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}

	client.RegisterRoutingUpdateListener("service-1", listener)
	client.RegisterRoutingUpdateListener("service-2", listener)

	client.listenerMutex.RLock()
	assert.Equal(t, 2, len(client.listeners))
	client.listenerMutex.RUnlock()
}

// TestAdvancedServiceDiscoveryClientConcurrentListenerRegistration tests concurrent listener registration
func TestAdvancedServiceDiscoveryClientConcurrentListenerRegistration(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	listener := func(update RoutingUpdate) {}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			client.RegisterRoutingUpdateListener("api", listener)
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	client.listenerMutex.RLock()
	assert.Equal(t, 10, len(client.listeners["api"]))
	client.listenerMutex.RUnlock()
}

// TestRoutingUpdateEmptyServices tests routing update with empty services
func TestRoutingUpdateEmptyServices(t *testing.T) {
	t.Parallel()
	update := RoutingUpdate{
		ServiceName: "api",
		Services:    []*ServiceInfo{},
		Timestamp:   time.Now(),
	}

	assert.Equal(t, 0, len(update.Services))
}

// TestRoutingUpdateMultipleServices tests routing update with multiple services
func TestRoutingUpdateMultipleServices(t *testing.T) {
	t.Parallel()
	services := make([]*ServiceInfo, 10)
	for i := 0; i < 10; i++ {
		services[i] = &ServiceInfo{ID: "service-" + string(rune(i))}
	}

	update := RoutingUpdate{
		ServiceName: "api",
		Services:    services,
		Timestamp:   time.Now(),
	}

	assert.Equal(t, 10, len(update.Services))
}

// TestAdvancedServiceDiscoveryClientInitialState tests initial state
func TestAdvancedServiceDiscoveryClientInitialState(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	client.runningMutex.RLock()
	assert.False(t, client.running)
	client.runningMutex.RUnlock()

	client.listenerMutex.RLock()
	assert.Equal(t, 0, len(client.listeners))
	client.listenerMutex.RUnlock()
}

// TestAdvancedServiceDiscoveryClientReferences tests client references
func TestAdvancedServiceDiscoveryClientReferences(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	assert.Equal(t, discoveryClient, client.discoveryClient)
	assert.Equal(t, registry, client.registry)
}

// TestRoutingUpdateServiceName tests routing update service name
func TestRoutingUpdateServiceName(t *testing.T) {
	t.Parallel()
	serviceNames := []string{"api", "auth", "database", "cache"}

	for _, name := range serviceNames {
		update := RoutingUpdate{
			ServiceName: name,
		}
		assert.Equal(t, name, update.ServiceName)
	}
}

// TestAdvancedServiceDiscoveryClientStopWhenNotRunning tests stopping when not running
func TestAdvancedServiceDiscoveryClientStopWhenNotRunning(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)

	// Should not panic
	client.Stop()

	client.runningMutex.RLock()
	assert.False(t, client.running)
	client.runningMutex.RUnlock()
}

// TestRoundRobinStrategySelectService returns nil for empty list
func TestRoundRobinStrategySelectService_Empty(t *testing.T) {
	t.Parallel()
	rrs := &RoundRobinStrategy{}
	got := rrs.SelectService(nil)
	assert.Nil(t, got)
	got = rrs.SelectService([]*ServiceInfo{})
	assert.Nil(t, got)
}

// TestRoundRobinStrategySelectService returns service and increments counter
func TestRoundRobinStrategySelectService_RoundRobin(t *testing.T) {
	t.Parallel()
	rrs := &RoundRobinStrategy{}
	svc1 := &ServiceInfo{ID: "svc1", Name: "test"}
	svc2 := &ServiceInfo{ID: "svc2", Name: "test"}

	s1 := rrs.SelectService([]*ServiceInfo{svc1, svc2})
	assert.Equal(t, "svc1", s1.ID)

	s2 := rrs.SelectService([]*ServiceInfo{svc1, svc2})
	assert.Equal(t, "svc2", s2.ID)

	s3 := rrs.SelectService([]*ServiceInfo{svc1, svc2})
	assert.Equal(t, "svc1", s3.ID)
}

// TestNewServiceLoadBalancer creates a load balancer
func TestNewServiceLoadBalancer(t *testing.T) {
	t.Parallel()
	sdc := &ServiceDiscoveryClient{}
	cache := &ServiceEndpointCache{}
	strategy := &RoundRobinStrategy{}
	lb := NewServiceLoadBalancer(sdc, cache, strategy)
	assert.NotNil(t, lb)
	assert.Equal(t, sdc, lb.discoveryClient)
	assert.Equal(t, cache, lb.cache)
	assert.Equal(t, strategy, lb.strategy)
}

// TestAdvancedServiceDiscoveryClientMultipleStarts tests multiple starts and stops
func TestAdvancedServiceDiscoveryClientMultipleStarts(t *testing.T) {
	t.Parallel()
	discoveryClient := &ServiceDiscoveryClient{}
	registry := &ServiceRegistry{}
	client := NewAdvancedServiceDiscoveryClient(discoveryClient, registry)
	ctx := context.Background()

	_ = client.Start(ctx)
	client.Stop()

	err := client.Start(ctx)
	assert.NoError(t, err)

	client.Stop()
}
