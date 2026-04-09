package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConsulClient defines the interface for Consul operations
type ConsulClient interface {
	RegisterService(ctx context.Context, serviceID, serviceName, address string, port int, tags []string) error
	DeregisterService(ctx context.Context, serviceID string) error
}

// ServiceInfo represents a registered service
type ServiceInfo struct {
	ID             string
	Name           string
	Address        string
	Port           int
	Tags           []string
	HealthCheckURL string
	Metadata       map[string]string
	RegisteredAt   time.Time
	LastHeartbeat  time.Time
	Status         string // "healthy", "unhealthy", "unknown"
}

// ServiceRegistry manages service registration and discovery
type ServiceRegistry struct {
	consul   ConsulClient
	services map[string]*ServiceInfo
	mutex    sync.RWMutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(consul ConsulClient) *ServiceRegistry {
	return &ServiceRegistry{
		consul:   consul,
		services: make(map[string]*ServiceInfo),
	}
}

// RegisterService registers a service
func (sr *ServiceRegistry) RegisterService(ctx context.Context, service ServiceInfo) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service.RegisteredAt = time.Now()
	service.LastHeartbeat = time.Now()
	service.Status = "healthy"

	// Register with Consul
	if err := sr.consul.RegisterService(ctx, service.ID, service.Name, service.Address, service.Port, service.Tags); err != nil {
		return fmt.Errorf("failed to register service with Consul: %w", err)
	}

	// Store locally
	sr.services[service.ID] = &service

	return nil
}

// DeregisterService deregisters a service
func (sr *ServiceRegistry) DeregisterService(ctx context.Context, serviceID string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	// Check if service exists
	if _, exists := sr.services[serviceID]; !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	// Deregister from Consul
	if err := sr.consul.DeregisterService(ctx, serviceID); err != nil {
		return fmt.Errorf("failed to deregister service from Consul: %w", err)
	}

	// Remove locally
	delete(sr.services, serviceID)

	return nil
}

// GetService retrieves a service by ID
func (sr *ServiceRegistry) GetService(_ context.Context, serviceID string) (*ServiceInfo, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return service, nil
}

// DiscoverServices discovers services by name
func (sr *ServiceRegistry) DiscoverServices(ctx context.Context, serviceName string) ([]*ServiceInfo, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var services []*ServiceInfo
	for _, service := range sr.services {
		if service.Name == serviceName && service.Status == "healthy" {
			services = append(services, service)
		}
	}

	return services, nil
}

// UpdateServiceStatus updates the status of a service
func (sr *ServiceRegistry) UpdateServiceStatus(ctx context.Context, serviceID, status string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	service, exists := sr.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	service.Status = status
	service.LastHeartbeat = time.Now()

	return nil
}

// ListServices lists all registered services
func (sr *ServiceRegistry) ListServices(ctx context.Context) []*ServiceInfo {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var services []*ServiceInfo
	for _, service := range sr.services {
		services = append(services, service)
	}

	return services
}

// GetHealthyServices returns only healthy services
func (sr *ServiceRegistry) GetHealthyServices(ctx context.Context) []*ServiceInfo {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	var services []*ServiceInfo
	for _, service := range sr.services {
		if service.Status == "healthy" {
			services = append(services, service)
		}
	}

	return services
}

// HealthChecker performs health checks on services
type HealthChecker struct {
	registry *ServiceRegistry
	interval time.Duration
	timeout  time.Duration
	mutex    sync.RWMutex
	running  bool
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(registry *ServiceRegistry, interval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		interval: interval,
		timeout:  timeout,
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) error {
	hc.mutex.Lock()
	if hc.running {
		hc.mutex.Unlock()
		return fmt.Errorf("health checker already running")
	}
	hc.running = true
	hc.mutex.Unlock()

	go hc.checkLoop(ctx)
	return nil
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.running = false
}

// checkLoop performs periodic health checks
func (hc *HealthChecker) checkLoop(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkAllServices(ctx)
		}
	}
}

// checkAllServices checks the health of all services
func (hc *HealthChecker) checkAllServices(ctx context.Context) {
	services := hc.registry.ListServices(ctx)

	for _, service := range services {
		go hc.checkService(ctx, service)
	}
}

// checkService checks the health of a single service
func (hc *HealthChecker) checkService(ctx context.Context, service *ServiceInfo) {
	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	// Perform health check (placeholder)
	healthy := hc.performHealthCheck(checkCtx, service)

	status := "unhealthy"
	if healthy {
		status = "healthy"
	}

	_ = hc.registry.UpdateServiceStatus(ctx, service.ID, status)
}

// performHealthCheck performs the actual health check
func (hc *HealthChecker) performHealthCheck(ctx context.Context, service *ServiceInfo) bool {
	// This is a placeholder for actual health check logic
	// In production, this would make HTTP requests to the health check endpoint
	return true
}

// ServiceDiscoveryClient provides service discovery functionality
type ServiceDiscoveryClient struct {
	registry *ServiceRegistry
	cache    map[string][]*ServiceInfo
	cacheTTL time.Duration
	mutex    sync.RWMutex
}

// NewServiceDiscoveryClient creates a new service discovery client
func NewServiceDiscoveryClient(registry *ServiceRegistry, cacheTTL time.Duration) *ServiceDiscoveryClient {
	return &ServiceDiscoveryClient{
		registry: registry,
		cache:    make(map[string][]*ServiceInfo),
		cacheTTL: cacheTTL,
	}
}

// GetService retrieves a service by name with load balancing
func (sdc *ServiceDiscoveryClient) GetService(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	services, err := sdc.GetServices(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no healthy services found: %s", serviceName)
	}

	// Simple round-robin load balancing
	return services[0], nil
}

// GetServices retrieves all services by name
func (sdc *ServiceDiscoveryClient) GetServices(ctx context.Context, serviceName string) ([]*ServiceInfo, error) {
	sdc.mutex.RLock()
	cached, exists := sdc.cache[serviceName]
	sdc.mutex.RUnlock()

	if exists {
		return cached, nil
	}

	services, err := sdc.registry.DiscoverServices(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	sdc.mutex.Lock()
	sdc.cache[serviceName] = services
	sdc.mutex.Unlock()

	return services, nil
}

// InvalidateCache invalidates the service cache
func (sdc *ServiceDiscoveryClient) InvalidateCache(serviceName string) {
	sdc.mutex.Lock()
	defer sdc.mutex.Unlock()
	delete(sdc.cache, serviceName)
}

// InvalidateAllCache invalidates all caches
func (sdc *ServiceDiscoveryClient) InvalidateAllCache() {
	sdc.mutex.Lock()
	defer sdc.mutex.Unlock()
	sdc.cache = make(map[string][]*ServiceInfo)
}

// LoadBalancer provides load balancing functionality
type LoadBalancer struct {
	discoveryClient *ServiceDiscoveryClient
	strategy        string // "round-robin", "least-connections", "random"
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(discoveryClient *ServiceDiscoveryClient, strategy string) *LoadBalancer {
	return &LoadBalancer{
		discoveryClient: discoveryClient,
		strategy:        strategy,
	}
}

// SelectService selects a service using the configured strategy
func (lb *LoadBalancer) SelectService(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	services, err := lb.discoveryClient.GetServices(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no services available: %s", serviceName)
	}

	switch lb.strategy {
	case "round-robin":
		return services[0], nil
	case "least-connections":
		return services[0], nil
	case "random":
		return services[0], nil
	default:
		return services[0], nil
	}
}
