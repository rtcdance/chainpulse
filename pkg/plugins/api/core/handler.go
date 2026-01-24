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
type DefaultErrorMapper struct {
}

// NewDefaultErrorMapper creates a new default error mapper
func NewDefaultErrorMapper() *DefaultErrorMapper {
	return &DefaultErrorMapper{}
}

// MapError maps an error to a response
func (m *DefaultErrorMapper) MapError(err error) (int, map[string]string, []byte) {
	if err == nil {
		return 200, make(map[string]string), []byte("")
	}

	// Default error mapping
	status := 500
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	body := []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error()))

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
