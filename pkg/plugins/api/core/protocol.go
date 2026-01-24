package core

import (
	"context"
	"fmt"
	"sync"
)

// ProtocolHandler defines the interface for protocol-specific handlers
type ProtocolHandler interface {
	// GetProtocolName returns the name of the protocol
	GetProtocolName() string

	// Start starts the protocol handler
	Start() error

	// Stop stops the protocol handler
	Stop() error

	// IsRunning returns whether the handler is running
	IsRunning() bool

	// RegisterRoute registers a route handler
	RegisterRoute(path string, handler Handler) error

	// Use adds middleware to the handler
	Use(middleware ...Middleware) error
}

// RequestProcessor defines the interface for processing requests
type RequestProcessor interface {
	// ProcessRequest processes a request and returns a response
	ProcessRequest(ctx context.Context, req Request) (Response, error)

	// HandleError handles an error and returns an error response
	HandleError(ctx context.Context, err error) Response
}

// ProtocolRegistry manages protocol handlers
type ProtocolRegistry struct {
	handlers map[string]ProtocolHandler
	mu       sync.RWMutex
}

// NewProtocolRegistry creates a new protocol registry
func NewProtocolRegistry() *ProtocolRegistry {
	return &ProtocolRegistry{
		handlers: make(map[string]ProtocolHandler),
	}
}

// Register registers a protocol handler
func (r *ProtocolRegistry) Register(name string, handler ProtocolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("protocol handler already registered: %s", name)
	}

	r.handlers[name] = handler
	return nil
}

// Get retrieves a protocol handler by name
func (r *ProtocolRegistry) Get(name string) (ProtocolHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[name]
	if !exists {
		return nil, fmt.Errorf("protocol handler not found: %s", name)
	}

	return handler, nil
}

// List returns all registered protocol handlers
func (r *ProtocolRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}

	return names
}

// StartAll starts all registered protocol handlers
func (r *ProtocolRegistry) StartAll() error {
	r.mu.RLock()
	handlers := make([]ProtocolHandler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		handlers = append(handlers, handler)
	}
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.Start(); err != nil {
			return fmt.Errorf("failed to start protocol handler %s: %w", handler.GetProtocolName(), err)
		}
	}

	return nil
}

// StopAll stops all registered protocol handlers
func (r *ProtocolRegistry) StopAll() error {
	r.mu.RLock()
	handlers := make([]ProtocolHandler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		handlers = append(handlers, handler)
	}
	r.mu.RUnlock()

	var lastErr error
	for _, handler := range handlers {
		if err := handler.Stop(); err != nil {
			lastErr = fmt.Errorf("failed to stop protocol handler %s: %w", handler.GetProtocolName(), err)
		}
	}

	return lastErr
}

// DefaultRequestProcessor provides default request processing
type DefaultRequestProcessor struct {
	apiLayer *APILayer
	mu       sync.RWMutex
}

// NewDefaultRequestProcessor creates a new default request processor
func NewDefaultRequestProcessor(apiLayer *APILayer) *DefaultRequestProcessor {
	return &DefaultRequestProcessor{
		apiLayer: apiLayer,
	}
}

// ProcessRequest processes a request through the API layer
func (p *DefaultRequestProcessor) ProcessRequest(ctx context.Context, req Request) (Response, error) {
	p.mu.RLock()
	apiLayer := p.apiLayer
	p.mu.RUnlock()

	if apiLayer == nil {
		return nil, fmt.Errorf("API layer not configured")
	}

	resp := apiLayer.Handle(req)
	return resp, nil
}

// HandleError handles an error and returns an error response
func (p *DefaultRequestProcessor) HandleError(ctx context.Context, err error) Response {
	resp := NewBaseResponse(nil)
	resp.SetStatus(500)
	resp.SetHeader("Content-Type", "application/json")
	resp.SetBody([]byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
	return resp
}

// BaseProtocolHandler provides a base implementation for protocol handlers
type BaseProtocolHandler struct {
	name      string
	processor RequestProcessor
	router    *APIRouter
	mu        sync.RWMutex
	running   bool
}

// NewBaseProtocolHandler creates a new base protocol handler
func NewBaseProtocolHandler(name string, processor RequestProcessor) *BaseProtocolHandler {
	return &BaseProtocolHandler{
		name:      name,
		processor: processor,
		router:    NewAPIRouter(),
	}
}

// GetProtocolName returns the protocol name
func (h *BaseProtocolHandler) GetProtocolName() string {
	return h.name
}

// IsRunning returns whether the handler is running
func (h *BaseProtocolHandler) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// RegisterRoute registers a route handler
func (h *BaseProtocolHandler) RegisterRoute(path string, handler Handler) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return fmt.Errorf("cannot register route while handler is running")
	}

	h.router.Register(path, handler)
	return nil
}

// Use adds middleware to the handler
func (h *BaseProtocolHandler) Use(middleware ...Middleware) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return fmt.Errorf("cannot add middleware while handler is running")
	}

	h.router.Use(middleware...)
	return nil
}

// SetRunning sets the running state
func (h *BaseProtocolHandler) SetRunning(running bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.running = running
}

// GetRouter returns the router
func (h *BaseProtocolHandler) GetRouter() *APIRouter {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.router
}

// GetProcessor returns the request processor
func (h *BaseProtocolHandler) GetProcessor() RequestProcessor {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.processor
}
