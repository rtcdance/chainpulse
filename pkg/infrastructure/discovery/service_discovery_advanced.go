package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RoutingUpdate represents a routing update
type RoutingUpdate struct {
	ServiceName string
	Services    []*ServiceInfo
	Timestamp   time.Time
}

// RoutingUpdateListener listens for routing updates
type RoutingUpdateListener func(update RoutingUpdate)

// AdvancedServiceDiscoveryClient provides advanced service discovery with automatic routing updates
type AdvancedServiceDiscoveryClient struct {
	discoveryClient *ServiceDiscoveryClient
	registry        *ServiceRegistry
	listeners       map[string][]RoutingUpdateListener
	listenerMutex   sync.RWMutex
	running         bool
	runningMutex    sync.RWMutex
}

// NewAdvancedServiceDiscoveryClient creates a new advanced service discovery client
func NewAdvancedServiceDiscoveryClient(discoveryClient *ServiceDiscoveryClient, registry *ServiceRegistry) *AdvancedServiceDiscoveryClient {
	return &AdvancedServiceDiscoveryClient{
		discoveryClient: discoveryClient,
		registry:        registry,
		listeners:       make(map[string][]RoutingUpdateListener),
	}
}

// RegisterRoutingUpdateListener registers a listener for routing updates
func (asdc *AdvancedServiceDiscoveryClient) RegisterRoutingUpdateListener(serviceName string, listener RoutingUpdateListener) {
	asdc.listenerMutex.Lock()
	defer asdc.listenerMutex.Unlock()

	asdc.listeners[serviceName] = append(asdc.listeners[serviceName], listener)
}

// Start starts the automatic routing update system
func (asdc *AdvancedServiceDiscoveryClient) Start(ctx context.Context) error {
	asdc.runningMutex.Lock()
	if asdc.running {
		asdc.runningMutex.Unlock()
		return fmt.Errorf("advanced service discovery client already running")
	}
	asdc.running = true
	asdc.runningMutex.Unlock()

	go asdc.updateLoop(ctx)
	return nil
}

// Stop stops the automatic routing update system
func (asdc *AdvancedServiceDiscoveryClient) Stop() {
	asdc.runningMutex.Lock()
	defer asdc.runningMutex.Unlock()
	asdc.running = false
}

// updateLoop performs periodic routing updates
func (asdc *AdvancedServiceDiscoveryClient) updateLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			asdc.performRoutingUpdates(ctx)
		}
	}
}

// performRoutingUpdates performs routing updates for all services
func (asdc *AdvancedServiceDiscoveryClient) performRoutingUpdates(ctx context.Context) {
	asdc.listenerMutex.RLock()
	serviceNames := make([]string, 0, len(asdc.listeners))
	for serviceName := range asdc.listeners {
		serviceNames = append(serviceNames, serviceName)
	}
	asdc.listenerMutex.RUnlock()

	for _, serviceName := range serviceNames {
		services, err := asdc.discoveryClient.GetServices(ctx, serviceName)
		if err != nil {
			continue
		}

		update := RoutingUpdate{
			ServiceName: serviceName,
			Services:    services,
			Timestamp:   time.Now(),
		}

		asdc.notifyListeners(serviceName, update)
	}
}

// notifyListeners notifies all listeners of a routing update
func (asdc *AdvancedServiceDiscoveryClient) notifyListeners(serviceName string, update RoutingUpdate) {
	asdc.listenerMutex.RLock()
	listeners := asdc.listeners[serviceName]
	asdc.listenerMutex.RUnlock()

	for _, listener := range listeners {
		go listener(update)
	}
}

// ServiceEndpointCache caches service endpoints
type ServiceEndpointCache struct {
	cache     map[string]*CachedEndpoint
	cacheTTL  time.Duration
	mutex     sync.RWMutex
}

// CachedEndpoint represents a cached endpoint
type CachedEndpoint struct {
	Services  []*ServiceInfo
	ExpiresAt time.Time
}

// NewServiceEndpointCache creates a new service endpoint cache
func NewServiceEndpointCache(cacheTTL time.Duration) *ServiceEndpointCache {
	return &ServiceEndpointCache{
		cache:    make(map[string]*CachedEndpoint),
		cacheTTL: cacheTTL,
	}
}

// Get retrieves cached endpoints
func (sec *ServiceEndpointCache) Get(serviceName string) ([]*ServiceInfo, bool) {
	sec.mutex.RLock()
	defer sec.mutex.RUnlock()

	cached, exists := sec.cache[serviceName]
	if !exists {
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	return cached.Services, true
}

// Set sets cached endpoints
func (sec *ServiceEndpointCache) Set(serviceName string, services []*ServiceInfo) {
	sec.mutex.Lock()
	defer sec.mutex.Unlock()

	sec.cache[serviceName] = &CachedEndpoint{
		Services:  services,
		ExpiresAt: time.Now().Add(sec.cacheTTL),
	}
}

// Invalidate invalidates cached endpoints
func (sec *ServiceEndpointCache) Invalidate(serviceName string) {
	sec.mutex.Lock()
	defer sec.mutex.Unlock()

	delete(sec.cache, serviceName)
}

// InvalidateAll invalidates all cached endpoints
func (sec *ServiceEndpointCache) InvalidateAll() {
	sec.mutex.Lock()
	defer sec.mutex.Unlock()

	sec.cache = make(map[string]*CachedEndpoint)
}

// ServiceLoadBalancer provides advanced load balancing
type ServiceLoadBalancer struct {
	discoveryClient *ServiceDiscoveryClient
	cache           *ServiceEndpointCache
	strategy        LoadBalancingStrategy
}

// LoadBalancingStrategy defines the load balancing strategy
type LoadBalancingStrategy interface {
	SelectService(services []*ServiceInfo) *ServiceInfo
}

// RoundRobinStrategy implements round-robin load balancing
type RoundRobinStrategy struct {
	counter int
	mutex   sync.Mutex
}

// SelectService selects a service using round-robin
func (rrs *RoundRobinStrategy) SelectService(services []*ServiceInfo) *ServiceInfo {
	if len(services) == 0 {
		return nil
	}

	rrs.mutex.Lock()
	defer rrs.mutex.Unlock()

	service := services[rrs.counter%len(services)]
	rrs.counter++

	return service
}

// NewServiceLoadBalancer creates a new service load balancer
func NewServiceLoadBalancer(discoveryClient *ServiceDiscoveryClient, cache *ServiceEndpointCache, strategy LoadBalancingStrategy) *ServiceLoadBalancer {
	return &ServiceLoadBalancer{
		discoveryClient: discoveryClient,
		cache:           cache,
		strategy:        strategy,
	}
}

// SelectService selects a service using the configured strategy
func (slb *ServiceLoadBalancer) SelectService(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	// Try to get from cache first
	services, cached := slb.cache.Get(serviceName)
	if !cached {
		var err error
		services, err = slb.discoveryClient.GetServices(ctx, serviceName)
		if err != nil {
			return nil, err
		}

		if len(services) == 0 {
			return nil, fmt.Errorf("no services available: %s", serviceName)
		}

		slb.cache.Set(serviceName, services)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no services available: %s", serviceName)
	}

	return slb.strategy.SelectService(services), nil
}

// ServiceConnectionPool manages connections to services
type ServiceConnectionPool struct {
	connections map[string]interface{}
	maxConnections int
	mutex       sync.RWMutex
}

// NewServiceConnectionPool creates a new service connection pool
func NewServiceConnectionPool(maxConnections int) *ServiceConnectionPool {
	return &ServiceConnectionPool{
		connections:    make(map[string]interface{}),
		maxConnections: maxConnections,
	}
}

// GetConnection retrieves a connection to a service
func (scp *ServiceConnectionPool) GetConnection(serviceID string) (interface{}, error) {
	scp.mutex.RLock()
	defer scp.mutex.RUnlock()

	conn, exists := scp.connections[serviceID]
	if !exists {
		return nil, fmt.Errorf("no connection found for service: %s", serviceID)
	}

	return conn, nil
}

// SetConnection sets a connection for a service
func (scp *ServiceConnectionPool) SetConnection(serviceID string, conn interface{}) error {
	scp.mutex.Lock()
	defer scp.mutex.Unlock()

	if len(scp.connections) >= scp.maxConnections {
		return fmt.Errorf("connection pool is full")
	}

	scp.connections[serviceID] = conn

	return nil
}

// RemoveConnection removes a connection for a service
func (scp *ServiceConnectionPool) RemoveConnection(serviceID string) {
	scp.mutex.Lock()
	defer scp.mutex.Unlock()

	delete(scp.connections, serviceID)
}

// ClearConnections clears all connections
func (scp *ServiceConnectionPool) ClearConnections() {
	scp.mutex.Lock()
	defer scp.mutex.Unlock()

	scp.connections = make(map[string]interface{})
}
