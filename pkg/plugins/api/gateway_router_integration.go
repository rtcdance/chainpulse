package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// GatewayRouterIntegration integrates the RequestRouter with the API Gateway
type GatewayRouterIntegration struct {
	router              *RequestRouter
	logger              core.Logger
	metrics             core.MetricsCollector
	eventQueryHandler   *EventQueryHandler
	subscriptionHandler *EventSubscriptionHandler
	healthCheckHandler  *HealthCheckHandler
	mu                  sync.RWMutex
	initialized         bool
}

// NewGatewayRouterIntegration creates a new gateway router integration
func NewGatewayRouterIntegration(
	logger core.Logger,
	metrics core.MetricsCollector,
	eventQueryHandler *EventQueryHandler,
	subscriptionHandler *EventSubscriptionHandler,
	healthCheckHandler *HealthCheckHandler,
) *GatewayRouterIntegration {
	return &GatewayRouterIntegration{
		router:              NewRequestRouter(logger, metrics),
		logger:              logger,
		metrics:             metrics,
		eventQueryHandler:   eventQueryHandler,
		subscriptionHandler: subscriptionHandler,
		healthCheckHandler:  healthCheckHandler,
		initialized:         false,
	}
}

// Initialize initializes the gateway router integration
func (gri *GatewayRouterIntegration) Initialize(ctx context.Context) error {
	gri.mu.Lock()
	defer gri.mu.Unlock()

	if gri.initialized {
		return nil
	}

	// Initialize router
	if err := gri.router.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

	// Register routes
	if err := gri.registerRoutes(); err != nil {
		return fmt.Errorf("failed to register routes: %w", err)
	}

	gri.initialized = true
	gri.logger.Info("Gateway router integration initialized")
	return nil
}

// registerRoutes registers all API routes
func (gri *GatewayRouterIntegration) registerRoutes() error {
	// Event Query Routes
	eventQueryRoute := NewRoute("event-query", "/events", "GET")
	if err := gri.router.RegisterRoute(eventQueryRoute); err != nil {
		return fmt.Errorf("failed to register event query route: %w", err)
	}

	eventByIDRoute := NewRoute("event-by-id", "/events/:id", "GET")
	if err := gri.router.RegisterRoute(eventByIDRoute); err != nil {
		return fmt.Errorf("failed to register event by ID route: %w", err)
	}

	eventByChainRoute := NewRoute("event-by-chain", "/events/chain/:chainId", "GET")
	if err := gri.router.RegisterRoute(eventByChainRoute); err != nil {
		return fmt.Errorf("failed to register event by chain route: %w", err)
	}

	eventByContractRoute := NewRoute("event-by-contract", "/events/contract/:address", "GET")
	if err := gri.router.RegisterRoute(eventByContractRoute); err != nil {
		return fmt.Errorf("failed to register event by contract route: %w", err)
	}

	eventByNameRoute := NewRoute("event-by-name", "/events/name/:eventName", "GET")
	if err := gri.router.RegisterRoute(eventByNameRoute); err != nil {
		return fmt.Errorf("failed to register event by name route: %w", err)
	}

	// Event Subscription Routes
	subscribeRoute := NewRoute("subscribe", "/events/subscribe", "GET")
	if err := gri.router.RegisterRoute(subscribeRoute); err != nil {
		return fmt.Errorf("failed to register subscribe route: %w", err)
	}

	subscribeChainRoute := NewRoute("subscribe-chain", "/events/subscribe/chain/:chainId", "GET")
	if err := gri.router.RegisterRoute(subscribeChainRoute); err != nil {
		return fmt.Errorf("failed to register subscribe chain route: %w", err)
	}

	subscribeContractRoute := NewRoute("subscribe-contract", "/events/subscribe/contract/:address", "GET")
	if err := gri.router.RegisterRoute(subscribeContractRoute); err != nil {
		return fmt.Errorf("failed to register subscribe contract route: %w", err)
	}

	subscribeNameRoute := NewRoute("subscribe-name", "/events/subscribe/name/:eventName", "GET")
	if err := gri.router.RegisterRoute(subscribeNameRoute); err != nil {
		return fmt.Errorf("failed to register subscribe name route: %w", err)
	}

	// Health Check Routes
	healthRoute := NewRoute("health", "/health", "GET")
	if err := gri.router.RegisterRoute(healthRoute); err != nil {
		return fmt.Errorf("failed to register health route: %w", err)
	}

	readyRoute := NewRoute("ready", "/health/ready", "GET")
	if err := gri.router.RegisterRoute(readyRoute); err != nil {
		return fmt.Errorf("failed to register ready route: %w", err)
	}

	liveRoute := NewRoute("live", "/health/live", "GET")
	if err := gri.router.RegisterRoute(liveRoute); err != nil {
		return fmt.Errorf("failed to register live route: %w", err)
	}

	componentsRoute := NewRoute("components", "/health/components", "GET")
	if err := gri.router.RegisterRoute(componentsRoute); err != nil {
		return fmt.Errorf("failed to register components route: %w", err)
	}

	gri.logger.Info("All routes registered successfully")
	return nil
}

// HandleRequest handles an incoming HTTP request
func (gri *GatewayRouterIntegration) HandleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Milliseconds()
		gri.metrics.RecordGauge("gateway_request_time_ms", float64(duration), nil)
	}()

	gri.mu.RLock()
	if !gri.initialized {
		gri.mu.RUnlock()
		http.Error(w, "Gateway not initialized", http.StatusInternalServerError)
		return
	}
	gri.mu.RUnlock()

	// Match route
	route, params, err := gri.router.MatchRoute(r.URL.Path)
	if err != nil {
		gri.logger.Warn("No route matched", "path", r.URL.Path)
		gri.metrics.RecordCounter("gateway_route_not_found", 1, nil)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Route to appropriate handler based on route ID
	switch route.ID {
	// Event Query Handlers
	case "event-query":
		gri.eventQueryHandler.HandleGetAllEvents(w, r)
	case "event-by-id":
		gri.eventQueryHandler.HandleGetEventByID(w, r, params["id"])
	case "event-by-chain":
		gri.eventQueryHandler.HandleGetEventsByChain(w, r, params["chainId"])
	case "event-by-contract":
		gri.eventQueryHandler.HandleGetEventsByContract(w, r, params["address"])
	case "event-by-name":
		gri.eventQueryHandler.HandleGetEventsByName(w, r, params["eventName"])

	// Event Subscription Handlers
	case "subscribe":
		gri.subscriptionHandler.HandleSubscribeAll(w, r)
	case "subscribe-chain":
		gri.subscriptionHandler.HandleSubscribeChain(w, r, params["chainId"])
	case "subscribe-contract":
		gri.subscriptionHandler.HandleSubscribeContract(w, r, params["address"])
	case "subscribe-name":
		gri.subscriptionHandler.HandleSubscribeName(w, r, params["eventName"])

	// Health Check Handlers
	case "health":
		gri.healthCheckHandler.HandleHealth(w, r)
	case "ready":
		gri.healthCheckHandler.HandleReady(w, r)
	case "live":
		gri.healthCheckHandler.HandleLive(w, r)
	case "components":
		gri.healthCheckHandler.HandleComponents(w, r)

	default:
		gri.logger.Warn("Unknown route", "routeId", route.ID)
		http.Error(w, "Not Found", http.StatusNotFound)
	}

	gri.metrics.RecordCounter("gateway_request_success", 1, nil)
}

// GetRouter returns the request router
func (gri *GatewayRouterIntegration) GetRouter() *RequestRouter {
	gri.mu.RLock()
	defer gri.mu.RUnlock()

	return gri.router
}

// GetMetrics returns gateway metrics
func (gri *GatewayRouterIntegration) GetMetrics() map[string]interface{} {
	gri.mu.RLock()
	defer gri.mu.RUnlock()

	return gri.router.GetMetrics()
}

// Close closes the gateway router integration
func (gri *GatewayRouterIntegration) Close(ctx context.Context) error {
	gri.mu.Lock()
	defer gri.mu.Unlock()

	if !gri.initialized {
		return nil
	}

	gri.initialized = false
	gri.logger.Info("Gateway router integration closed")
	return nil
}

// RequestRouterMiddleware is middleware for request routing
type RequestRouterMiddleware struct {
	router *RequestRouter
	logger core.Logger
}

// NewRequestRouterMiddleware creates a new request router middleware
func NewRequestRouterMiddleware(router *RequestRouter, logger core.Logger) *RequestRouterMiddleware {
	return &RequestRouterMiddleware{
		router: router,
		logger: logger,
	}
}

// Middleware wraps an HTTP handler with request routing
func (m *RequestRouterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to match route
		route, params, err := m.router.MatchRoute(r.URL.Path)
		if err != nil {
			// No route matched, pass to next handler
			next.ServeHTTP(w, r)
			return
		}

		// Store route and params in request context
		ctx := context.WithValue(r.Context(), routeContextKey, route)
		ctx = context.WithValue(ctx, paramsContextKey, params)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// GetRouteFromContext retrieves the route from request context
func GetRouteFromContext(r *http.Request) (*Route, bool) {
	route, ok := r.Context().Value(routeContextKey).(*Route)
	return route, ok
}

// GetParamsFromContext retrieves the route parameters from request context
func GetParamsFromContext(r *http.Request) (map[string]string, bool) {
	params, ok := r.Context().Value(paramsContextKey).(map[string]string)
	return params, ok
}

// ResponseWriter wraps http.ResponseWriter to capture response details
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response body
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}

// GetStatusCode returns the captured status code
func (rw *ResponseWriter) GetStatusCode() int {
	if rw.statusCode == 0 {
		return http.StatusOK
	}
	return rw.statusCode
}

// GetBody returns the captured response body
func (rw *ResponseWriter) GetBody() []byte {
	return rw.body
}

// NewResponseWriter creates a new response writer wrapper
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     0,
		body:           make([]byte, 0),
	}
}

// RequestLogger logs request details
type RequestLogger struct {
	logger core.Logger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger core.Logger) *RequestLogger {
	return &RequestLogger{
		logger: logger,
	}
}

// LogRequest logs request details
func (rl *RequestLogger) LogRequest(r *http.Request, statusCode int, duration time.Duration) {
	rl.logger.Info("Request processed",
		"method", r.Method,
		"path", r.URL.Path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogError logs request error
func (rl *RequestLogger) LogError(r *http.Request, err error) {
	rl.logger.Error("Request error",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err.Error(),
	)
}
