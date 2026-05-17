package core

import (
	"fmt"
	"sync"
)

// Handler defines the protocol-agnostic handler interface
type Handler interface {
	// Handle processes a request and returns a response
	Handle(req Request) (Response, error)
}

// HandlerFunc is a function type that implements Handler
type HandlerFunc func(req Request) (Response, error)

// Handle implements the Handler interface
func (f HandlerFunc) Handle(req Request) (Response, error) {
	return f(req)
}

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler

// APIRouter routes requests to appropriate handlers
type APIRouter struct {
	handlers   map[string]Handler
	middleware []Middleware
	mu         sync.RWMutex
}

// NewAPIRouter creates a new API router
func NewAPIRouter() *APIRouter {
	return &APIRouter{
		handlers:   make(map[string]Handler),
		middleware: make([]Middleware, 0),
	}
}

// Register registers a handler for a specific route
func (r *APIRouter) Register(route string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[route] = handler
}

// RegisterFunc registers a handler function for a specific route
func (r *APIRouter) RegisterFunc(route string, handler HandlerFunc) {
	r.Register(route, handler)
}

// Use adds middleware to the router
func (r *APIRouter) Use(middleware ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// Route returns a handler for a specific route
func (r *APIRouter) Route(route string) Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.handlers[route]
	if !ok {
		return nil
	}

	// Apply middleware in reverse order
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	return handler
}

// RouteCount returns the number of registered routes.
func (r *APIRouter) RouteCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.handlers)
}

// MiddlewareCount returns the number of registered middleware entries.
func (r *APIRouter) MiddlewareCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.middleware)
}

// GetRuntimeMetrics returns a compact runtime surface for route coverage and
// router readiness on top of the registered routes and middleware stack.
func (r *APIRouter) GetRuntimeMetrics() map[string]any {
	routeCount := r.RouteCount()
	middlewareCount := r.MiddlewareCount()

	coveragePosture := classifyRouterCoveragePosture(routeCount, middlewareCount)
	runtimePosture := classifyRouterRuntimePosture(routeCount, middlewareCount)

	return map[string]any{
		"route_count":      routeCount,
		"middleware_count": middlewareCount,
		"coverage_posture": coveragePosture,
		"runtime_posture":  runtimePosture,
		"reliability_hint": buildRouterReliabilityHint(coveragePosture, runtimePosture),
	}
}

// Handle processes a request through the router
func (r *APIRouter) Handle(req Request) (Response, error) {
	route := req.Path()
	handler := r.Route(route)

	if handler == nil {
		resp := NewBaseResponse(nil)
		resp.SetStatus(404)
		resp.SetBody([]byte("Not Found"))
		return resp, fmt.Errorf("route not found: %s", route)
	}

	return handler.Handle(req)
}

// ErrorMapper maps errors to protocol-specific error responses
type ErrorMapper interface {
	// MapError maps an error to a response
	MapError(err error) (int, map[string]string, []byte)
}

// DefaultErrorMapper provides default error mapping
type DefaultErrorMapper struct{}

// NewDefaultErrorMapper creates a new default error mapper
func NewDefaultErrorMapper() *DefaultErrorMapper {
	return &DefaultErrorMapper{}
}

// MapError maps an error to a response using the unified APIError format.
func (m *DefaultErrorMapper) MapError(err error) (int, map[string]string, []byte) {
	if err == nil {
		return 200, make(map[string]string), []byte("")
	}

	// Use the api package's MapErrorToAPIError via a deferred import to
	// avoid circular dependency. The api package imports core, and core
	// must not import api. We resolve this by doing the mapping inline
	// with the same logic, or by accepting a mapper function at wire time.
	//
	// For now, produce a consistent error response without leaking internals.
	status := 500
	code := "INTERNAL_SERVER_ERROR"
	message := "an internal error occurred"

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := []byte(fmt.Sprintf(`{"error":"%s","message":"%s","statusCode":%d}`, code, message, status))

	return status, headers, body
}

// APILayer provides the unified API layer
type APILayer struct {
	router      *APIRouter
	errorMapper ErrorMapper
	mu          sync.RWMutex
}

// NewAPILayer creates a new API layer
func NewAPILayer() *APILayer {
	return &APILayer{
		router:      NewAPIRouter(),
		errorMapper: NewDefaultErrorMapper(),
	}
}

// RegisterHandler registers a handler for a specific route
func (a *APILayer) RegisterHandler(route string, handler Handler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.router.Register(route, handler)
}

// RegisterHandlerFunc registers a handler function for a specific route
func (a *APILayer) RegisterHandlerFunc(route string, handler HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.router.RegisterFunc(route, handler)
}

// Use adds middleware to the API layer
func (a *APILayer) Use(middleware ...Middleware) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.router.Use(middleware...)
}

// SetErrorMapper sets the error mapper
func (a *APILayer) SetErrorMapper(mapper ErrorMapper) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.errorMapper = mapper
}

// GetRuntimeMetrics returns a compact runtime surface for API-layer route
// readiness and error-mapper wiring on top of router/runtime state.
func (a *APILayer) GetRuntimeMetrics() map[string]any {
	a.mu.RLock()
	router := a.router
	errorMapper := a.errorMapper
	a.mu.RUnlock()

	routerMetrics := router.GetRuntimeMetrics()
	routeCount, _ := routerMetrics["route_count"].(int)
	middlewareCount, _ := routerMetrics["middleware_count"].(int)
	errorMapperConfigured := errorMapper != nil

	coveragePosture := classifyAPILayerCoveragePosture(routeCount, middlewareCount, errorMapperConfigured)
	runtimePosture := classifyAPILayerRuntimePosture(routeCount, middlewareCount, errorMapperConfigured)

	return map[string]any{
		"route_count":             routeCount,
		"middleware_count":        middlewareCount,
		"error_mapper_configured": errorMapperConfigured,
		"coverage_posture":        coveragePosture,
		"runtime_posture":         runtimePosture,
		"reliability_hint":        buildAPILayerReliabilityHint(coveragePosture, runtimePosture),
	}
}

// Handle processes a request through the API layer
func (a *APILayer) Handle(req Request) Response {
	a.mu.RLock()
	router := a.router
	errorMapper := a.errorMapper
	a.mu.RUnlock()

	resp := NewBaseResponse(nil)

	// Route the request
	result, err := router.Handle(req)
	if err != nil {
		status, headers, body := errorMapper.MapError(err)
		resp.SetStatus(status)
		for k, v := range headers {
			resp.SetHeader(k, v)
		}
		resp.SetBody(body)
		return resp
	}

	return result
}

func classifyRouterCoveragePosture(routeCount int, middlewareCount int) string {
	if routeCount == 0 {
		return "router-empty"
	}
	if middlewareCount == 0 {
		return "router-routes-only"
	}
	return "router-guarded"
}

func classifyRouterRuntimePosture(routeCount int, middlewareCount int) string {
	if routeCount == 0 {
		return "router-unobserved"
	}
	if middlewareCount == 0 {
		return "router-watch"
	}
	return "router-ready"
}

func buildRouterReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "router-unobserved":
		return "api router has no registered routes yet"
	case runtimePosture == "router-watch":
		return "api router has routes but no middleware guardrails; verify whether this path is intentionally bare"
	case coveragePosture == "router-guarded":
		return "api router has registered routes with middleware coverage and is ready for request handling"
	default:
		return "api router is configured with routes"
	}
}

func classifyAPILayerCoveragePosture(routeCount int, middlewareCount int, errorMapperConfigured bool) string {
	if routeCount == 0 {
		return "layer-empty"
	}
	if middlewareCount == 0 && !errorMapperConfigured {
		return "layer-routes-only"
	}
	if errorMapperConfigured && middlewareCount == 0 {
		return "layer-mapped"
	}
	return "layer-guarded"
}

func classifyAPILayerRuntimePosture(routeCount int, middlewareCount int, errorMapperConfigured bool) string {
	if routeCount == 0 {
		return "layer-unobserved"
	}
	if !errorMapperConfigured {
		return "layer-watch"
	}
	if middlewareCount == 0 {
		return "layer-ready"
	}
	return "layer-hardened"
}

func buildAPILayerReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "layer-unobserved":
		return "api layer has no registered routes yet"
	case runtimePosture == "layer-watch":
		return "api layer has routes but no error mapper configured; verify error translation before relying on runtime behavior"
	case runtimePosture == "layer-ready":
		return "api layer has routes and error mapping configured; add middleware only if route hardening is required"
	case coveragePosture == "layer-guarded":
		return "api layer has routes, error mapping, and middleware coverage and is ready for request handling"
	default:
		return "api layer is configured with routes"
	}
}
