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

// GetRuntimeMetrics returns a compact runtime surface for protocol-registry
// readiness on top of handler registration and running-state coverage.
func (r *ProtocolRegistry) GetRuntimeMetrics() map[string]interface{} {
	r.mu.RLock()
	handlerCount := len(r.handlers)
	runningCount := 0
	for _, handler := range r.handlers {
		if handler != nil && handler.IsRunning() {
			runningCount++
		}
	}
	r.mu.RUnlock()

	coveragePosture := classifyProtocolRegistryCoveragePosture(handlerCount, runningCount)
	runtimePosture := classifyProtocolRegistryRuntimePosture(handlerCount, runningCount)

	return map[string]interface{}{
		"handler_count":    handlerCount,
		"running_count":    runningCount,
		"coverage_posture": coveragePosture,
		"runtime_posture":  runtimePosture,
		"reliability_hint": buildProtocolRegistryReliabilityHint(coveragePosture, runtimePosture),
	}
}

func classifyProtocolRegistryCoveragePosture(handlerCount int, runningCount int) string {
	if handlerCount == 0 {
		return "protocol-registry-empty"
	}
	if runningCount == 0 {
		return "protocol-registry-registered"
	}
	if runningCount < handlerCount {
		return "protocol-registry-partial"
	}
	return "protocol-registry-active"
}

func classifyProtocolRegistryRuntimePosture(handlerCount int, runningCount int) string {
	if handlerCount == 0 {
		return "protocol-registry-unobserved"
	}
	if runningCount == 0 {
		return "protocol-registry-watch"
	}
	if runningCount < handlerCount {
		return "protocol-registry-degraded"
	}
	return "protocol-registry-ready"
}

func buildProtocolRegistryReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "protocol-registry-unobserved":
		return "protocol registry has no registered handlers yet"
	case runtimePosture == "protocol-registry-watch":
		return "protocol registry has registered handlers but none are running; verify start sequencing before treating it as active"
	case runtimePosture == "protocol-registry-degraded":
		return "protocol registry has only partial running coverage; verify whether all registered handlers are expected to be active"
	case coveragePosture == "protocol-registry-active":
		return "protocol registry has full running coverage across registered handlers and is ready for protocol dispatch"
	default:
		return "protocol registry has registered handlers and requires runtime observation"
	}
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

// GetRuntimeMetrics returns a compact runtime surface for request-processor
// readiness on top of API-layer wiring and route/error coverage.
func (p *DefaultRequestProcessor) GetRuntimeMetrics() map[string]interface{} {
	p.mu.RLock()
	apiLayer := p.apiLayer
	p.mu.RUnlock()

	routeCount := 0
	middlewareCount := 0
	errorMapperConfigured := false
	apiLayerConfigured := apiLayer != nil
	layerCoveragePosture := "layer-empty"

	if apiLayerConfigured {
		layerMetrics := apiLayer.GetRuntimeMetrics()
		routeCount, _ = layerMetrics["route_count"].(int)
		middlewareCount, _ = layerMetrics["middleware_count"].(int)
		errorMapperConfigured, _ = layerMetrics["error_mapper_configured"].(bool)
		layerCoveragePosture, _ = layerMetrics["coverage_posture"].(string)
	}

	coveragePosture := classifyRequestProcessorCoveragePosture(apiLayerConfigured, layerCoveragePosture)
	runtimePosture := classifyRequestProcessorRuntimePosture(apiLayerConfigured, routeCount, errorMapperConfigured)

	return map[string]interface{}{
		"api_layer_configured":    apiLayerConfigured,
		"route_count":             routeCount,
		"middleware_count":        middlewareCount,
		"error_mapper_configured": errorMapperConfigured,
		"coverage_posture":        coveragePosture,
		"runtime_posture":         runtimePosture,
		"reliability_hint":        buildRequestProcessorReliabilityHint(coveragePosture, runtimePosture),
	}
}

// ProcessRequest processes a request through the API layer
func (p *DefaultRequestProcessor) ProcessRequest(_ context.Context, req Request) (Response, error) {
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
func (p *DefaultRequestProcessor) HandleError(_ context.Context, err error) Response {
	resp := NewBaseResponse(nil)
	resp.SetStatus(500)
	resp.SetHeader("Content-Type", "application/json")
	resp.SetBody([]byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
	return resp
}

func classifyRequestProcessorCoveragePosture(apiLayerConfigured bool, layerCoveragePosture string) string {
	if !apiLayerConfigured {
		return "request-processor-unconfigured"
	}
	switch layerCoveragePosture {
	case "layer-guarded":
		return "request-processor-guarded"
	case "layer-routes-only":
		return "request-processor-routes-only"
	default:
		return "request-processor-partial"
	}
}

func classifyRequestProcessorRuntimePosture(apiLayerConfigured bool, routeCount int, errorMapperConfigured bool) string {
	if !apiLayerConfigured {
		return "request-processor-unobserved"
	}
	if routeCount == 0 || !errorMapperConfigured {
		return "request-processor-watch"
	}
	return "request-processor-ready"
}

func buildRequestProcessorReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "request-processor-unobserved":
		return "request processor has no API layer configured yet"
	case runtimePosture == "request-processor-watch":
		return "request processor wiring is incomplete; verify API routes and error mapping before relying on runtime behavior"
	case coveragePosture == "request-processor-routes-only":
		return "request processor has routed API coverage without middleware guardrails; verify whether bare processing is intentional"
	case coveragePosture == "request-processor-guarded":
		return "request processor is backed by routed, guarded API-layer coverage and is ready for request handling"
	default:
		return "request processor has partial API-layer coverage; continue observing runtime wiring"
	}
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

// GetRuntimeMetrics returns a compact runtime surface for protocol-handler
// readiness on top of router wiring, processor presence, and running state.
func (h *BaseProtocolHandler) GetRuntimeMetrics() map[string]interface{} {
	h.mu.RLock()
	running := h.running
	processorConfigured := h.processor != nil
	router := h.router
	name := h.name
	h.mu.RUnlock()

	routeCount := 0
	middlewareCount := 0
	if router != nil {
		routeCount = router.RouteCount()
		middlewareCount = router.MiddlewareCount()
	}

	coveragePosture := classifyBaseProtocolHandlerCoveragePosture(routeCount, middlewareCount, processorConfigured)
	runtimePosture := classifyBaseProtocolHandlerRuntimePosture(running, routeCount, processorConfigured)

	return map[string]interface{}{
		"protocol_name":        name,
		"running":              running,
		"processor_configured": processorConfigured,
		"route_count":          routeCount,
		"middleware_count":     middlewareCount,
		"coverage_posture":     coveragePosture,
		"runtime_posture":      runtimePosture,
		"reliability_hint":     buildBaseProtocolHandlerReliabilityHint(coveragePosture, runtimePosture),
	}
}

func classifyBaseProtocolHandlerCoveragePosture(routeCount int, middlewareCount int, processorConfigured bool) string {
	if routeCount == 0 && !processorConfigured {
		return "protocol-handler-unconfigured"
	}
	if routeCount > 0 && middlewareCount == 0 {
		return "protocol-handler-routes-only"
	}
	if routeCount > 0 && middlewareCount > 0 && processorConfigured {
		return "protocol-handler-guarded"
	}
	return "protocol-handler-partial"
}

func classifyBaseProtocolHandlerRuntimePosture(running bool, routeCount int, processorConfigured bool) string {
	if routeCount == 0 && !processorConfigured {
		return "protocol-handler-unobserved"
	}
	if !processorConfigured {
		return "protocol-handler-degraded"
	}
	if !running {
		return "protocol-handler-idle"
	}
	return "protocol-handler-ready"
}

func buildBaseProtocolHandlerReliabilityHint(coveragePosture string, runtimePosture string) string {
	switch {
	case runtimePosture == "protocol-handler-degraded":
		return "protocol handler is missing a request processor; configure processing before relying on runtime behavior"
	case runtimePosture == "protocol-handler-idle":
		return "protocol handler has wiring but is not running yet; verify start sequencing before treating it as active"
	case coveragePosture == "protocol-handler-routes-only":
		return "protocol handler has routes without middleware guardrails; verify whether bare routing is intentional"
	case coveragePosture == "protocol-handler-guarded" && runtimePosture == "protocol-handler-ready":
		return "protocol handler has processor, routes, middleware, and running state aligned for request handling"
	default:
		return "protocol handler has not been fully configured yet"
	}
}
