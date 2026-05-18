package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/observability"
)

var sharedHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     50,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	},
}

// RequestRouter manages route registration and request forwarding
type RequestRouter struct {
	routes          map[string]*Route
	loadBalancers   map[string]*LoadBalancer
	logger          core.Logger
	metrics         core.MetricsCollector
	httpClient      *http.Client
	mu              sync.RWMutex
	initialized     bool
	defaultTimeout  time.Duration
	circuitBreakers map[string]*CircuitBreaker
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	state           string // "closed", "open", "half-open"
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	threshold       int
	timeout         time.Duration
	mu              sync.RWMutex
}

// ForwardedRequest represents a forwarded request
type ForwardedRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
	Params  map[string]string
}

// AggregatedResponse represents aggregated response
type AggregatedResponse struct {
	Status  int
	Headers map[string]string
	Body    any
	Error   string
}

// NewRequestRouter creates a new request router
func NewRequestRouter(logger core.Logger, metrics core.MetricsCollector) *RequestRouter {
	return &RequestRouter{
		routes:          make(map[string]*Route),
		loadBalancers:   make(map[string]*LoadBalancer),
		logger:          logger,
		metrics:         metrics,
		httpClient:      sharedHTTPClient,
		initialized:     false,
		defaultTimeout:  30 * time.Second,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
}

// SetHTTPClient overrides the forwarding HTTP client.
func (rr *RequestRouter) SetHTTPClient(client *http.Client) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if client == nil {
		rr.httpClient = sharedHTTPClient
		return
	}
	rr.httpClient = client
}

// Initialize initializes the request router
func (rr *RequestRouter) Initialize(ctx context.Context) error {
	if rr.initialized {
		return nil
	}

	if rr.logger == nil {
		return fmt.Errorf("logger is required")
	}

	if rr.metrics == nil {
		return fmt.Errorf("metrics collector is required")
	}

	rr.initialized = true
	rr.logger.Info("Request router initialized")
	return nil
}

// RegisterRoute registers a new route
func (rr *RequestRouter) RegisterRoute(route *Route) error {
	if route == nil {
		return fmt.Errorf("route cannot be nil")
	}

	if route.ID == "" {
		return fmt.Errorf("route ID is required")
	}

	if route.Pattern == "" {
		return fmt.Errorf("route pattern is required")
	}

	rr.mu.Lock()
	defer rr.mu.Unlock()

	if _, exists := rr.routes[route.ID]; exists {
		return fmt.Errorf("route %s already exists", route.ID)
	}

	rr.routes[route.ID] = route

	// Create load balancer for this route
	rr.loadBalancers[route.ID] = NewLoadBalancer("round-robin")

	// Create circuit breaker for this route
	rr.circuitBreakers[route.ID] = NewCircuitBreaker(5, 30*time.Second)

	rr.logger.Info("Route registered", "routeId", route.ID, "pattern", route.Pattern)
	rr.metrics.RecordCounter("router_route_registered", 1, nil)

	return nil
}

// AttachHandler attaches a request handler to an existing route and its load balancer.
func (rr *RequestRouter) AttachHandler(routeID string, handler *RequestHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	rr.mu.Lock()
	defer rr.mu.Unlock()

	route, exists := rr.routes[routeID]
	if !exists {
		return fmt.Errorf("route %s not found", routeID)
	}
	lb, lbExists := rr.loadBalancers[routeID]
	if !lbExists {
		return fmt.Errorf("load balancer not found for route %s", routeID)
	}

	if err := route.AddHandler(handler); err != nil {
		return err
	}
	if err := lb.AddHandler(handler); err != nil {
		return err
	}

	rr.metrics.RecordCounter("router_handler_attached", 1, map[string]string{"route_id": routeID})
	return nil
}

// UnregisterRoute unregisters a route
func (rr *RequestRouter) UnregisterRoute(routeID string) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if _, exists := rr.routes[routeID]; !exists {
		return fmt.Errorf("route %s not found", routeID)
	}

	delete(rr.routes, routeID)
	delete(rr.loadBalancers, routeID)
	delete(rr.circuitBreakers, routeID)

	rr.logger.Info("Route unregistered", "routeId", routeID)
	rr.metrics.RecordCounter("router_route_unregistered", 1, nil)

	return nil
}

// MatchRoute finds a route that matches the given path
func (rr *RequestRouter) MatchRoute(path string) (*Route, map[string]string, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	rr.logger.Debug("Matching route", "path", path, "registered_routes", len(rr.routes))

	routes := make([]*Route, 0, len(rr.routes))
	for _, route := range rr.routes {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Priority > routes[j].Priority
	})

	for _, route := range routes {
		rr.logger.Debug("Trying route", "route_pattern", route.Pattern, "route_method", route.Method, "priority", route.Priority)
		if params, matched := route.Match(path); matched {
			rr.logger.Debug("Route matched", "route_id", route.ID, "path", path, "priority", route.Priority)
			return route, params, nil
		}
	}

	return nil, nil, fmt.Errorf("no route matches path: %s", path)
}

// ForwardRequest forwards a request to a handler
func (rr *RequestRouter) ForwardRequest(ctx context.Context, route *Route, req *ForwardedRequest) (*AggregatedResponse, error) {
	if route == nil {
		return nil, fmt.Errorf("route cannot be nil")
	}

	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		rr.metrics.RecordGauge("router_forward_request_time_ms", float64(duration), nil)
	}()

	// Get load balancer for this route
	rr.mu.RLock()
	lb, exists := rr.loadBalancers[route.ID]
	cb, cbExists := rr.circuitBreakers[route.ID]
	rr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("load balancer not found for route %s", route.ID)
	}

	if !cbExists {
		return nil, fmt.Errorf("circuit breaker not found for route %s", route.ID)
	}

	// Check circuit breaker
	if cb.IsOpen() {
		rr.logger.Warn("Circuit breaker open for route", "routeId", route.ID)
		rr.metrics.RecordCounter("router_circuit_breaker_open", 1, nil)
		return nil, fmt.Errorf("circuit breaker open for route %s", route.ID)
	}

	// Select handler using load balancer
	handler, err := lb.SelectHandler()
	if err != nil {
		rr.logger.Error("Failed to select handler", "routeId", route.ID, "error", err.Error())
		rr.metrics.RecordCounter("router_handler_selection_failed", 1, nil)
		return nil, err
	}

	// Forward request to handler
	response, err := rr.forwardToHandler(ctx, handler, req)
	if err != nil {
		handler.RecordError(err.Error())
		cb.RecordFailure()
		rr.logger.Error("Failed to forward request", "handlerId", handler.ID, "error", err.Error())
		rr.metrics.RecordCounter("router_forward_failed", 1, nil)
		return nil, err
	}

	// Record success
	handler.RecordRequest(time.Since(start), true)
	cb.RecordSuccess()
	rr.metrics.RecordCounter("router_forward_success", 1, nil)

	return response, nil
}

// forwardToHandler forwards a request to a specific handler
func (rr *RequestRouter) forwardToHandler(ctx context.Context, handler *RequestHandler, req *ForwardedRequest) (*AggregatedResponse, error) {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, handler.Endpoint+req.Path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	observability.InjectTraceHeaders(ctx, httpReq.Header)

	// Set body if present
	if len(req.Body) > 0 {
		httpReq.Body = io.NopCloser(bytes.NewReader(req.Body))
		httpReq.ContentLength = int64(len(req.Body))
	}

	client := rr.httpClient
	if client == nil {
		client = &http.Client{Timeout: rr.defaultTimeout}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			rr.logger.Error("Failed to close response body", "error", err.Error())
		}
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Copy response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &AggregatedResponse{
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    body,
	}, nil
}

// GetRoute returns a route by ID
func (rr *RequestRouter) GetRoute(routeID string) (*Route, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	route, exists := rr.routes[routeID]
	if !exists {
		return nil, fmt.Errorf("route %s not found", routeID)
	}

	return route, nil
}

// GetRoutes returns all routes
func (rr *RequestRouter) GetRoutes() []*Route {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	routes := make([]*Route, 0, len(rr.routes))
	for _, route := range rr.routes {
		routes = append(routes, route)
	}

	return routes
}

// GetMetrics returns router metrics
func (rr *RequestRouter) GetMetrics() map[string]any {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	metrics := make(map[string]any)
	metrics["route_count"] = len(rr.routes)
	metrics["load_balancer_count"] = len(rr.loadBalancers)

	// Collect load balancer metrics
	lbMetrics := make(map[string]any)
	for routeID, lb := range rr.loadBalancers {
		lbMetrics[routeID] = lb.GetMetrics()
	}
	metrics["load_balancers"] = lbMetrics

	return metrics
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        "closed",
		failureCount: 0,
		successCount: 0,
		threshold:    threshold,
		timeout:      timeout,
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "half-open" {
		cb.successCount++
		if cb.successCount >= 3 {
			cb.state = "closed"
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.threshold {
		cb.state = "open"
	}
}

// IsOpen checks if the circuit breaker is open.
// Uses write-lock to prevent TOCTOU race between state-read and state-transition.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "open" {
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = "half-open"
			cb.failureCount = 0
			cb.successCount = 0
			return false
		}
		return true
	}

	return false
}

// GetState returns the circuit breaker state
func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.state
}
