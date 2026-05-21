package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// ProtocolHandler defines the interface for protocol handlers
type ProtocolHandler interface {
	Start(ctx context.Context) error
	Stop() error
	HandleRequest(ctx context.Context, req any) (any, error)
	Health() core.HealthStatus
}

// RESTHandler handles REST API requests
type RESTHandler struct {
	port    int
	running bool
	mutex   sync.RWMutex
}

// NewRESTHandler creates a new REST handler
func NewRESTHandler(port int) *RESTHandler {
	return &RESTHandler{
		port: port,
	}
}

// Start starts the REST handler
func (rh *RESTHandler) Start(ctx context.Context) error {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if rh.running {
		return fmt.Errorf("REST handler already running")
	}

	rh.running = true
	return nil
}

// Stop stops the REST handler
func (rh *RESTHandler) Stop() error {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if !rh.running {
		return fmt.Errorf("REST handler not running")
	}

	rh.running = false
	return nil
}

// HandleRequest handles a REST request
func (rh *RESTHandler) HandleRequest(ctx context.Context, req any) (any, error) {
	rh.mutex.RLock()
	if !rh.running {
		rh.mutex.RUnlock()
		return nil, fmt.Errorf("REST handler not running")
	}
	rh.mutex.RUnlock()

	return map[string]any{
		"protocol": "REST",
		"status":   "ok",
	}, nil
}

// Health returns the health status
func (rh *RESTHandler) Health() core.HealthStatus {
	rh.mutex.RLock()
	defer rh.mutex.RUnlock()

	if rh.running {
		return core.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	}

	return core.HealthStatus{
		Status:    "unhealthy",
		Timestamp: time.Now(),
	}
}

// gRPCHandler handles gRPC requests
type gRPCHandler struct {
	port    int
	running bool
	mutex   sync.RWMutex
}

// NewgRPCHandler creates a new gRPC handler
func NewgRPCHandler(port int) *gRPCHandler {
	return &gRPCHandler{
		port: port,
	}
}

// Start starts the gRPC handler
func (gh *gRPCHandler) Start(ctx context.Context) error {
	gh.mutex.Lock()
	defer gh.mutex.Unlock()

	if gh.running {
		return fmt.Errorf("gRPC handler already running")
	}

	gh.running = true
	return nil
}

// Stop stops the gRPC handler
func (gh *gRPCHandler) Stop() error {
	gh.mutex.Lock()
	defer gh.mutex.Unlock()

	if !gh.running {
		return fmt.Errorf("gRPC handler not running")
	}

	gh.running = false
	return nil
}

// HandleRequest handles a gRPC request
func (gh *gRPCHandler) HandleRequest(ctx context.Context, req any) (any, error) {
	gh.mutex.RLock()
	if !gh.running {
		gh.mutex.RUnlock()
		return nil, fmt.Errorf("gRPC handler not running")
	}
	gh.mutex.RUnlock()

	return map[string]any{
		"protocol": "gRPC",
		"status":   "ok",
	}, nil
}

// Health returns the health status
func (gh *gRPCHandler) Health() core.HealthStatus {
	gh.mutex.RLock()
	defer gh.mutex.RUnlock()

	if gh.running {
		return core.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	}

	return core.HealthStatus{
		Status:    "unhealthy",
		Timestamp: time.Now(),
	}
}

// WebSocketHandler handles WebSocket requests
type WebSocketHandler struct {
	port    int
	running bool
	mutex   sync.RWMutex
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(port int) *WebSocketHandler {
	return &WebSocketHandler{
		port: port,
	}
}

// Start starts the WebSocket handler
func (wh *WebSocketHandler) Start(ctx context.Context) error {
	wh.mutex.Lock()
	defer wh.mutex.Unlock()

	if wh.running {
		return fmt.Errorf("WebSocket handler already running")
	}

	wh.running = true
	return nil
}

// Stop stops the WebSocket handler
func (wh *WebSocketHandler) Stop() error {
	wh.mutex.Lock()
	defer wh.mutex.Unlock()

	if !wh.running {
		return fmt.Errorf("WebSocket handler not running")
	}

	wh.running = false
	return nil
}

// HandleRequest handles a WebSocket request
func (wh *WebSocketHandler) HandleRequest(ctx context.Context, req any) (any, error) {
	wh.mutex.RLock()
	if !wh.running {
		wh.mutex.RUnlock()
		return nil, fmt.Errorf("WebSocket handler not running")
	}
	wh.mutex.RUnlock()

	return map[string]any{
		"protocol": "WebSocket",
		"status":   "ok",
	}, nil
}

// Health returns the health status
func (wh *WebSocketHandler) Health() core.HealthStatus {
	wh.mutex.RLock()
	defer wh.mutex.RUnlock()

	if wh.running {
		return core.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	}

	return core.HealthStatus{
		Status:    "unhealthy",
		Timestamp: time.Now(),
	}
}

// GraphQLHandler handles GraphQL requests
type GraphQLHandler struct {
	port    int
	running bool
	mutex   sync.RWMutex
}

// NewGraphQLHandler creates a new GraphQL handler
func NewGraphQLHandler(port int) *GraphQLHandler {
	return &GraphQLHandler{
		port: port,
	}
}

// Start starts the GraphQL handler
func (gh *GraphQLHandler) Start(ctx context.Context) error {
	gh.mutex.Lock()
	defer gh.mutex.Unlock()

	if gh.running {
		return fmt.Errorf("GraphQL handler already running")
	}

	gh.running = true
	return nil
}

// Stop stops the GraphQL handler
func (gh *GraphQLHandler) Stop() error {
	gh.mutex.Lock()
	defer gh.mutex.Unlock()

	if !gh.running {
		return fmt.Errorf("GraphQL handler not running")
	}

	gh.running = false
	return nil
}

// HandleRequest handles a GraphQL request
func (gh *GraphQLHandler) HandleRequest(ctx context.Context, req any) (any, error) {
	gh.mutex.RLock()
	if !gh.running {
		gh.mutex.RUnlock()
		return nil, fmt.Errorf("GraphQL handler not running")
	}
	gh.mutex.RUnlock()

	return map[string]any{
		"protocol": "GraphQL",
		"status":   "ok",
	}, nil
}

// Health returns the health status
func (gh *GraphQLHandler) Health() core.HealthStatus {
	gh.mutex.RLock()
	defer gh.mutex.RUnlock()

	if gh.running {
		return core.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	}

	return core.HealthStatus{
		Status:    "unhealthy",
		Timestamp: time.Now(),
	}
}

// MultiProtocolAPI manages multiple protocol handlers
type MultiProtocolAPI struct {
	handlers map[string]ProtocolHandler
	mutex    sync.RWMutex
}

// NewMultiProtocolAPI creates a new multi-protocol API
func NewMultiProtocolAPI() *MultiProtocolAPI {
	return &MultiProtocolAPI{
		handlers: make(map[string]ProtocolHandler),
	}
}

// RegisterHandler registers a protocol handler
func (mpa *MultiProtocolAPI) RegisterHandler(protocol string, handler ProtocolHandler) error {
	mpa.mutex.Lock()
	defer mpa.mutex.Unlock()

	if _, exists := mpa.handlers[protocol]; exists {
		return fmt.Errorf("handler already registered for protocol: %s", protocol)
	}

	mpa.handlers[protocol] = handler
	return nil
}

// StartAll starts all protocol handlers
func (mpa *MultiProtocolAPI) StartAll(ctx context.Context) error {
	mpa.mutex.RLock()
	handlers := make(map[string]ProtocolHandler)
	for protocol, handler := range mpa.handlers {
		handlers[protocol] = handler
	}
	mpa.mutex.RUnlock()

	for protocol, handler := range handlers {
		if err := handler.Start(ctx); err != nil {
			return fmt.Errorf("failed to start %s handler: %w", protocol, err)
		}
	}

	return nil
}

// StopAll stops all protocol handlers
func (mpa *MultiProtocolAPI) StopAll() error {
	mpa.mutex.RLock()
	handlers := make(map[string]ProtocolHandler)
	for protocol, handler := range mpa.handlers {
		handlers[protocol] = handler
	}
	mpa.mutex.RUnlock()

	for protocol, handler := range handlers {
		if err := handler.Stop(); err != nil {
			return fmt.Errorf("failed to stop %s handler: %w", protocol, err)
		}
	}

	return nil
}

// GetHandler gets a protocol handler
func (mpa *MultiProtocolAPI) GetHandler(protocol string) (ProtocolHandler, error) {
	mpa.mutex.RLock()
	defer mpa.mutex.RUnlock()

	handler, exists := mpa.handlers[protocol]
	if !exists {
		return nil, fmt.Errorf("handler not found for protocol: %s", protocol)
	}

	return handler, nil
}

// HealthAll returns health status of all handlers
func (mpa *MultiProtocolAPI) HealthAll() map[string]core.HealthStatus {
	mpa.mutex.RLock()
	handlers := make(map[string]ProtocolHandler)
	for protocol, handler := range mpa.handlers {
		handlers[protocol] = handler
	}
	mpa.mutex.RUnlock()

	health := make(map[string]core.HealthStatus)
	for protocol, handler := range handlers {
		health[protocol] = handler.Health()
	}

	return health
}
