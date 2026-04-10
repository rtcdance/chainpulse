package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/infrastructure/discovery"
)

// APIGatewayConfig represents API gateway configuration
type APIGatewayConfig struct {
	Port              int
	Protocols         []string // "rest", "grpc", "websocket", "graphql"
	MaxConnections    int
	RequestTimeout    time.Duration
	EnableCompression bool
	EnableCaching     bool
}

// APIGateway represents an API gateway instance
type APIGateway struct {
	config          APIGatewayConfig
	discoveryClient *discovery.ServiceDiscoveryClient
	loadBalancer    *discovery.ServiceLoadBalancer
	sessionManager  *discovery.SessionManager
	cache           *discovery.ServiceEndpointCache
	metrics         *APIMetrics
	mutex           sync.RWMutex
	running         bool
}

// NewAPIGateway creates a new API gateway
func NewAPIGateway(config APIGatewayConfig, discoveryClient *discovery.ServiceDiscoveryClient, loadBalancer *discovery.ServiceLoadBalancer) *APIGateway {
	return &APIGateway{
		config:          config,
		discoveryClient: discoveryClient,
		loadBalancer:    loadBalancer,
		sessionManager:  discovery.NewSessionManager(),
		cache:           discovery.NewServiceEndpointCache(30 * time.Second),
		metrics:         NewAPIMetrics(),
	}
}

// Start starts the API gateway
func (ag *APIGateway) Start(ctx context.Context) error {
	ag.mutex.Lock()
	if ag.running {
		ag.mutex.Unlock()
		return fmt.Errorf("API gateway already running")
	}
	ag.running = true
	ag.mutex.Unlock()

	return nil
}

// Stop stops the API gateway
func (ag *APIGateway) Stop() error {
	ag.mutex.Lock()
	defer ag.mutex.Unlock()

	if !ag.running {
		return fmt.Errorf("API gateway not running")
	}

	ag.running = false
	return nil
}

// HandleRequest handles an incoming request
func (ag *APIGateway) HandleRequest(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	ag.metrics.RecordRequest()

	start := time.Now()
	defer func() {
		ag.metrics.RecordLatency(time.Since(start))
	}()

	// Get session
	session, err := ag.sessionManager.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Route request to appropriate service
	_, err = ag.loadBalancer.SelectService(ctx, req.ServiceName)
	if err != nil {
		ag.metrics.RecordError()
		return nil, fmt.Errorf("failed to select service: %w", err)
	}

	// Create response
	response := &APIResponse{
		SessionID:   session.ID,
		ServiceName: req.ServiceName,
		Status:      200,
		Timestamp:   time.Now(),
	}

	return response, nil
}

// APIRequest represents an API request
type APIRequest struct {
	SessionID   string
	ServiceName string
	Protocol    string
	Method      string
	Path        string
	Headers     map[string]string
	Body        []byte
}

// APIResponse represents an API response
type APIResponse struct {
	SessionID   string
	ServiceName string
	Status      int
	Headers     map[string]string
	Body        []byte
	Timestamp   time.Time
}

// APIMetrics tracks API gateway metrics
type APIMetrics struct {
	totalRequests int64
	totalErrors   int64
	totalLatency  int64
	mutex         sync.RWMutex
}

// NewAPIMetrics creates new API metrics
func NewAPIMetrics() *APIMetrics {
	return &APIMetrics{}
}

// RecordRequest records a request
func (am *APIMetrics) RecordRequest() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.totalRequests++
}

// RecordError records an error
func (am *APIMetrics) RecordError() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.totalErrors++
}

// RecordLatency records latency
func (am *APIMetrics) RecordLatency(latency time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.totalLatency += latency.Milliseconds()
}

// GetMetrics returns current metrics
func (am *APIMetrics) GetMetrics() map[string]interface{} {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	avgLatency := int64(0)
	if am.totalRequests > 0 {
		avgLatency = am.totalLatency / am.totalRequests
	}

	return map[string]interface{}{
		"total_requests": am.totalRequests,
		"total_errors":   am.totalErrors,
		"avg_latency_ms": avgLatency,
	}
}

// APIGatewayCluster manages multiple API gateway instances
type APIGatewayCluster struct {
	gateways map[string]*APIGateway
	mutex    sync.RWMutex
}

// NewAPIGatewayCluster creates a new API gateway cluster
func NewAPIGatewayCluster() *APIGatewayCluster {
	return &APIGatewayCluster{
		gateways: make(map[string]*APIGateway),
	}
}

// AddGateway adds a gateway to the cluster
func (agc *APIGatewayCluster) AddGateway(id string, gateway *APIGateway) error {
	agc.mutex.Lock()
	defer agc.mutex.Unlock()

	if _, exists := agc.gateways[id]; exists {
		return fmt.Errorf("gateway already exists: %s", id)
	}

	agc.gateways[id] = gateway
	return nil
}

// RemoveGateway removes a gateway from the cluster
func (agc *APIGatewayCluster) RemoveGateway(id string) error {
	agc.mutex.Lock()
	defer agc.mutex.Unlock()

	if _, exists := agc.gateways[id]; !exists {
		return fmt.Errorf("gateway not found: %s", id)
	}

	delete(agc.gateways, id)
	return nil
}

// GetGateway gets a gateway from the cluster
func (agc *APIGatewayCluster) GetGateway(id string) (*APIGateway, error) {
	agc.mutex.RLock()
	defer agc.mutex.RUnlock()

	gateway, exists := agc.gateways[id]
	if !exists {
		return nil, fmt.Errorf("gateway not found: %s", id)
	}

	return gateway, nil
}

// ListGateways lists all gateways in the cluster
func (agc *APIGatewayCluster) ListGateways() []*APIGateway {
	agc.mutex.RLock()
	defer agc.mutex.RUnlock()

	gateways := make([]*APIGateway, 0, len(agc.gateways))
	for _, gateway := range agc.gateways {
		gateways = append(gateways, gateway)
	}

	return gateways
}

// GetClusterMetrics gets metrics from all gateways
func (agc *APIGatewayCluster) GetClusterMetrics() map[string]interface{} {
	agc.mutex.RLock()
	defer agc.mutex.RUnlock()

	totalRequests := int64(0)
	totalErrors := int64(0)

	for _, gateway := range agc.gateways {
		metrics := gateway.metrics.GetMetrics()
		totalRequests += metrics["total_requests"].(int64)
		totalErrors += metrics["total_errors"].(int64)
	}

	return map[string]interface{}{
		"total_requests": totalRequests,
		"total_errors":   totalErrors,
		"gateway_count":  len(agc.gateways),
	}
}
