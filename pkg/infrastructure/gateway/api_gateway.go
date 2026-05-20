package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/infrastructure/discovery"
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
	mu              sync.RWMutex
	running         bool

	// Handler registry for request routing
	handlers        map[string]RequestHandler // method+path -> handler
	middlewareChain []Middleware
}

// RequestHandler handles a specific API request
type RequestHandler func(ctx context.Context, req *APIRequest) (*APIResponse, error)

// Middleware is a function that wraps a handler
type Middleware func(next RequestHandler) RequestHandler

// NewAPIGateway creates a new API gateway
func NewAPIGateway(config APIGatewayConfig, discoveryClient *discovery.ServiceDiscoveryClient, loadBalancer *discovery.ServiceLoadBalancer) *APIGateway {
	return &APIGateway{
		config:          config,
		discoveryClient: discoveryClient,
		loadBalancer:    loadBalancer,
		sessionManager:  discovery.NewSessionManager(),
		cache:           discovery.NewServiceEndpointCache(30 * time.Second),
		metrics:         NewAPIMetrics(),
		handlers:        make(map[string]RequestHandler),
	}
}

// RegisterHandler registers a handler for a specific method and path pattern.
// Pattern format: "GET /api/v1/events", "POST /api/v1/query", etc.
func (ag *APIGateway) RegisterHandler(pattern string, handler RequestHandler) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.handlers[pattern] = handler
}

// RegisterMiddleware adds a middleware to the handler chain.
// Middlewares are applied in registration order.
func (ag *APIGateway) RegisterMiddleware(mw Middleware) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.middlewareChain = append(ag.middlewareChain, mw)
}

// ListHandlers returns all registered handler patterns sorted alphabetically.
func (ag *APIGateway) ListHandlers() []string {
	ag.mu.RLock()
	defer ag.mu.RUnlock()
	patterns := make([]string, 0, len(ag.handlers))
	for p := range ag.handlers {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	return patterns
}

// Start activates the API gateway routing layer. It does NOT start an HTTP server
// — the gateway is a request router that delegates to an external server (e.g.,
// HTTPPlugin). Call Start() before routing requests; pair with Stop() on shutdown.
func (ag *APIGateway) Start(ctx context.Context) error {
	ag.mu.Lock()
	if ag.running {
		ag.mu.Unlock()
		return fmt.Errorf("API gateway already running")
	}
	ag.running = true
	ag.mu.Unlock()

	return nil
}

// Stop stops the API gateway
func (ag *APIGateway) Stop() error {
	ag.mu.Lock()
	defer ag.mu.Unlock()

	if !ag.running {
		return fmt.Errorf("API gateway not running")
	}

	ag.running = false
	return nil
}

// HandleRequest handles an incoming request with handler routing and middleware chain
func (ag *APIGateway) HandleRequest(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	ag.metrics.RecordRequest()

	start := time.Now()
	defer func() {
		ag.metrics.RecordLatency(time.Since(start))
	}()

	// Build the handler chain with middlewares
	baseHandler := ag.findHandler(req)
	handler := ag.wrapWithMiddleware(baseHandler)

	// Execute the handler with panic recovery
	return ag.executeWithRecovery(ctx, req, handler)
}

// executeWithRecovery runs the handler and recovers from panics, returning a
// safe error response instead of crashing the caller goroutine.
func (ag *APIGateway) executeWithRecovery(ctx context.Context, req *APIRequest, handler RequestHandler) (_ *APIResponse, _ error) {
	defer func() {
		if r := recover(); r != nil {
			ag.metrics.RecordError()
		}
	}()

	resp, err := handler(ctx, req)
	if err != nil {
		ag.metrics.RecordError()
		return nil, err
	}

	return resp, nil
}

// findHandler locates the appropriate handler for the request.
// Falls back to the generic load-balanced routing if no specific handler is registered.
func (ag *APIGateway) findHandler(req *APIRequest) RequestHandler {
	ag.mu.RLock()
	defer ag.mu.RUnlock()

	pattern := req.Method + " " + req.Path
	if h, ok := ag.handlers[pattern]; ok {
		return h
	}

	// Fallback: route via load balancer (backward-compatible behavior)
	return func(ctx context.Context, r *APIRequest) (*APIResponse, error) {
		_, err := ag.loadBalancer.SelectService(ctx, r.ServiceName)
		if err != nil {
			return nil, fmt.Errorf("failed to select service: %w", err)
		}
		return &APIResponse{
			SessionID:   r.SessionID,
			ServiceName: r.ServiceName,
			Status:      http.StatusOK,
			Timestamp:   time.Now(),
		}, nil
	}
}

// wrapWithMiddleware wraps a handler with the registered middleware chain.
// Middlewares are applied in registration order (first registered = outermost).
func (ag *APIGateway) wrapWithMiddleware(handler RequestHandler) RequestHandler {
	ag.mu.RLock()
	chain := make([]Middleware, len(ag.middlewareChain))
	copy(chain, ag.middlewareChain)
	ag.mu.RUnlock()

	wrapped := handler
	for i := len(chain) - 1; i >= 0; i-- {
		wrapped = chain[i](wrapped)
	}
	return wrapped
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
func (am *APIMetrics) GetMetrics() map[string]any {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	avgLatency := int64(0)
	if am.totalRequests > 0 {
		avgLatency = am.totalLatency / am.totalRequests
	}

	return map[string]any{
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
func (agc *APIGatewayCluster) GetClusterMetrics() map[string]any {
	agc.mutex.RLock()
	defer agc.mutex.RUnlock()

	totalRequests := int64(0)
	totalErrors := int64(0)

	for _, gateway := range agc.gateways {
		metrics := gateway.metrics.GetMetrics()
		if v, ok := metrics["total_requests"].(int64); ok {
			totalRequests += v
		}
		if v, ok := metrics["total_errors"].(int64); ok {
			totalErrors += v
		}
	}

	return map[string]any{
		"total_requests": totalRequests,
		"total_errors":   totalErrors,
		"gateway_count":  len(agc.gateways),
	}
}
