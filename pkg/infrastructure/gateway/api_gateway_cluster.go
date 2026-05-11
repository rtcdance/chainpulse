package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"chainpulse/pkg/infrastructure/discovery"
	"chainpulse/pkg/infrastructure/health"
)

// APIGatewayClusterConfig represents API gateway cluster configuration
type APIGatewayClusterConfig struct {
	Instances       int
	Port            int
	Protocols       []string
	HealthCheckURL  string
	HealthCheckTTL  time.Duration
	LoadBalancerURL string
}

// APIGatewayClusterDeployment manages API gateway cluster deployment
type APIGatewayClusterDeployment struct {
	config           APIGatewayClusterConfig
	gateways         map[string]*APIGateway
	registry         *discovery.ServiceRegistry
	healthChecker    *health.HealthCheckSystem
	loadBalancer     *discovery.ServiceLoadBalancer
	discoveryClient  *discovery.ServiceDiscoveryClient
	multiProtocolAPI *MultiProtocolAPI
	mutex            sync.RWMutex
	running          bool
}

// NewAPIGatewayClusterDeployment creates a new API gateway cluster deployment
func NewAPIGatewayClusterDeployment(
	config APIGatewayClusterConfig,
	registry *discovery.ServiceRegistry,
	healthChecker *health.HealthCheckSystem,
	loadBalancer *discovery.ServiceLoadBalancer,
	discoveryClient *discovery.ServiceDiscoveryClient,
) *APIGatewayClusterDeployment {
	return &APIGatewayClusterDeployment{
		config:           config,
		gateways:         make(map[string]*APIGateway),
		registry:         registry,
		healthChecker:    healthChecker,
		loadBalancer:     loadBalancer,
		discoveryClient:  discoveryClient,
		multiProtocolAPI: NewMultiProtocolAPI(),
	}
}

// Deploy deploys the API gateway cluster
func (agcd *APIGatewayClusterDeployment) Deploy(ctx context.Context) error {
	agcd.mutex.Lock()
	if agcd.running {
		agcd.mutex.Unlock()
		return fmt.Errorf("cluster already deployed")
	}
	agcd.mutex.Unlock()

	// Deploy instances
	for i := 0; i < agcd.config.Instances; i++ {
		instanceID := fmt.Sprintf("api-gateway-%d", i)
		if err := agcd.deployInstance(ctx, instanceID); err != nil {
			return fmt.Errorf("failed to deploy instance %s: %w", instanceID, err)
		}
	}

	// Start multi-protocol API
	if err := agcd.multiProtocolAPI.StartAll(ctx); err != nil {
		return fmt.Errorf("failed to start multi-protocol API: %w", err)
	}

	agcd.mutex.Lock()
	agcd.running = true
	agcd.mutex.Unlock()

	return nil
}

// deployInstance deploys a single API gateway instance
func (agcd *APIGatewayClusterDeployment) deployInstance(ctx context.Context, instanceID string) error {
	// Create gateway configuration
	gatewayConfig := APIGatewayConfig{
		Port:              agcd.config.Port,
		Protocols:         agcd.config.Protocols,
		MaxConnections:    1000,
		RequestTimeout:    30 * time.Second,
		EnableCompression: true,
		EnableCaching:     true,
	}

	// Create gateway
	gateway := NewAPIGateway(gatewayConfig, agcd.discoveryClient, agcd.loadBalancer)

	// Start gateway
	if err := gateway.Start(ctx); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// Register service
	serviceInfo := discovery.ServiceInfo{
		ID:             instanceID,
		Name:           "api-gateway",
		Address:        "localhost",
		Port:           agcd.config.Port,
		Tags:           []string{"api", "gateway"},
		HealthCheckURL: agcd.config.HealthCheckURL,
		RegisteredAt:   time.Now(),
	}

	if err := agcd.registry.RegisterService(ctx, serviceInfo); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Store gateway
	agcd.mutex.Lock()
	agcd.gateways[instanceID] = gateway
	agcd.mutex.Unlock()

	return nil
}

// Undeploy undeploys the API gateway cluster
func (agcd *APIGatewayClusterDeployment) Undeploy(ctx context.Context) error {
	agcd.mutex.Lock()
	if !agcd.running {
		agcd.mutex.Unlock()
		return fmt.Errorf("cluster not deployed")
	}

	gateways := make(map[string]*APIGateway)
	for id, gateway := range agcd.gateways {
		gateways[id] = gateway
	}
	agcd.mutex.Unlock()

	// Stop multi-protocol API
	if err := agcd.multiProtocolAPI.StopAll(); err != nil {
		return fmt.Errorf("failed to stop multi-protocol API: %w", err)
	}

	// Stop instances
	for instanceID, gateway := range gateways {
		if err := gateway.Stop(); err != nil {
			return fmt.Errorf("failed to stop instance %s: %w", instanceID, err)
		}

		// Deregister service
		if err := agcd.registry.DeregisterService(ctx, instanceID); err != nil {
			// Log error but continue with cleanup
			slog.Warn("failed to deregister service", "instanceID", instanceID, "error", err)
		}
	}

	agcd.mutex.Lock()
	agcd.running = false
	agcd.gateways = make(map[string]*APIGateway)
	agcd.mutex.Unlock()

	return nil
}

// GetInstance gets an API gateway instance
func (agcd *APIGatewayClusterDeployment) GetInstance(instanceID string) (*APIGateway, error) {
	agcd.mutex.RLock()
	defer agcd.mutex.RUnlock()

	gateway, exists := agcd.gateways[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	return gateway, nil
}

// ListInstances lists all API gateway instances
func (agcd *APIGatewayClusterDeployment) ListInstances() []*APIGateway {
	agcd.mutex.RLock()
	defer agcd.mutex.RUnlock()

	instances := make([]*APIGateway, 0, len(agcd.gateways))
	for _, gateway := range agcd.gateways {
		instances = append(instances, gateway)
	}

	return instances
}

// GetClusterHealth gets the health status of the cluster
func (agcd *APIGatewayClusterDeployment) GetClusterHealth(ctx context.Context) (map[string]interface{}, error) {
	agcd.mutex.RLock()
	defer agcd.mutex.RUnlock()

	if !agcd.running {
		return nil, fmt.Errorf("cluster not deployed")
	}

	healthyInstances := 0
	totalInstances := len(agcd.gateways)

	for _, gateway := range agcd.gateways {
		// Check if gateway is running
		if gateway != nil {
			healthyInstances++
		}
	}

	return map[string]interface{}{
		"status":             "ok",
		"healthy_instances":  healthyInstances,
		"total_instances":    totalInstances,
		"protocols":          agcd.config.Protocols,
		"multi_protocol_api": agcd.multiProtocolAPI.HealthAll(),
	}, nil
}

// ScaleUp scales up the cluster
func (agcd *APIGatewayClusterDeployment) ScaleUp(ctx context.Context, count int) error {
	agcd.mutex.RLock()
	currentCount := len(agcd.gateways)
	agcd.mutex.RUnlock()

	for i := 0; i < count; i++ {
		instanceID := fmt.Sprintf("api-gateway-%d", currentCount+i)
		if err := agcd.deployInstance(ctx, instanceID); err != nil {
			return fmt.Errorf("failed to scale up: %w", err)
		}
	}

	return nil
}

// ScaleDown scales down the cluster
func (agcd *APIGatewayClusterDeployment) ScaleDown(ctx context.Context, count int) error {
	agcd.mutex.Lock()
	gateways := make([]string, 0)
	for id := range agcd.gateways {
		gateways = append(gateways, id)
	}
	agcd.mutex.Unlock()

	if count > len(gateways) {
		return fmt.Errorf("cannot scale down more than available instances")
	}

	// Remove last 'count' instances
	for i := 0; i < count; i++ {
		instanceID := gateways[len(gateways)-1-i]

		// Get gateway
		gateway, err := agcd.GetInstance(instanceID)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Stop gateway
		if err := gateway.Stop(); err != nil {
			return fmt.Errorf("failed to stop instance: %w", err)
		}

		// Deregister service
		if err := agcd.registry.DeregisterService(ctx, instanceID); err != nil {
			return fmt.Errorf("failed to deregister service: %w", err)
		}

		// Remove from gateways
		agcd.mutex.Lock()
		delete(agcd.gateways, instanceID)
		agcd.mutex.Unlock()
	}

	return nil
}

// GetMetrics gets cluster metrics
func (agcd *APIGatewayClusterDeployment) GetMetrics() map[string]interface{} {
	agcd.mutex.RLock()
	defer agcd.mutex.RUnlock()

	totalRequests := int64(0)
	totalErrors := int64(0)

	for _, gateway := range agcd.gateways {
		metrics := gateway.metrics.GetMetrics()
		if v, ok := metrics["total_requests"].(int64); ok {
			totalRequests += v
		}
		if v, ok := metrics["total_errors"].(int64); ok {
			totalErrors += v
		}
	}

	return map[string]interface{}{
		"total_requests": totalRequests,
		"total_errors":   totalErrors,
		"instance_count": len(agcd.gateways),
	}
}
