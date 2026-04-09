package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// GatewayRouterIntegration integrates the RequestRouter with the API Gateway
type GatewayRouterIntegration struct {
	router                        *RequestRouter
	logger                        core.Logger
	metrics                       core.MetricsCollector
	eventQueryHandler             *EventQueryHandler
	subscriptionHandler           *EventSubscriptionHandler
	healthCheckHandler            *HealthCheckHandler
	modelsHandler                 *ModelsHandler
	graphqlHandler                *GraphQLHandler
	upstreamQueryEndpoints        []string
	upstreamQueryHTTPClient       *http.Client
	upstreamQueryHealthHTTPClient *http.Client
	runtimeMetricsProvider        func(*http.Request) interface{}
	runtimeSummaryProvider        func(*http.Request) interface{}
	runtimeControlProvider        func(http.ResponseWriter, *http.Request)
	runtimeReplayProvider         func(http.ResponseWriter, *http.Request)
	mu                            sync.RWMutex
	initialized                   bool
}

type GatewayRuntimeRouteInventory struct {
	RegisteredRouteCount int
	RuntimeRouteCount    int
	RuntimeSurfaceCount  int
	HealthRoutesEnabled  bool
	SummaryRouteEnabled  bool
	MetricsRouteEnabled  bool
	ControlRouteEnabled  bool
	ReplayRouteEnabled   bool
}

const gatewayAPIV1Prefix = "/api/v1"

// NewGatewayRouterIntegration creates a new gateway router integration
func NewGatewayRouterIntegration(
	logger core.Logger,
	metrics core.MetricsCollector,
	eventQueryHandler *EventQueryHandler,
	subscriptionHandler *EventSubscriptionHandler,
	healthCheckHandler *HealthCheckHandler,
	runtimeProviders ...interface{},
) *GatewayRouterIntegration {
	var (
		summaryProvider func(*http.Request) interface{}
		metricsProvider func(*http.Request) interface{}
		controlProvider func(http.ResponseWriter, *http.Request)
		replayProvider  func(http.ResponseWriter, *http.Request)
	)

	if len(runtimeProviders) > 0 {
		if provider, ok := runtimeProviders[0].(func(*http.Request) interface{}); ok {
			summaryProvider = provider
		}
	}
	if len(runtimeProviders) > 1 {
		if provider, ok := runtimeProviders[1].(func(*http.Request) interface{}); ok {
			metricsProvider = provider
		}
	}
	if len(runtimeProviders) > 2 {
		if provider, ok := runtimeProviders[2].(func(http.ResponseWriter, *http.Request)); ok {
			controlProvider = provider
		}
	}

	if len(runtimeProviders) > 3 {
		if provider, ok := runtimeProviders[3].(func(http.ResponseWriter, *http.Request)); ok {
			replayProvider = provider
		}
	}
	return &GatewayRouterIntegration{
		router:                 NewRequestRouter(logger, metrics),
		logger:                 logger,
		metrics:                metrics,
		eventQueryHandler:      eventQueryHandler,
		subscriptionHandler:    subscriptionHandler,
		healthCheckHandler:     healthCheckHandler,
		modelsHandler:          NewModelsHandler(logger, metrics),
		runtimeMetricsProvider: metricsProvider,
		runtimeSummaryProvider: summaryProvider,
		runtimeControlProvider: controlProvider,
		runtimeReplayProvider:  replayProvider,
		initialized:            false,
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

// SetUpstreamQueryEndpoints wires read-only query upstreams for gateway forwarding.
func (gri *GatewayRouterIntegration) SetUpstreamQueryEndpoints(endpoints []string) {
	gri.mu.Lock()
	defer gri.mu.Unlock()

	gri.upstreamQueryEndpoints = append([]string(nil), endpoints...)
}

// SetUpstreamQueryHealthHTTPClient overrides the HTTP client used for upstream query health checks.
func (gri *GatewayRouterIntegration) SetUpstreamQueryHealthHTTPClient(client *http.Client) {
	gri.mu.Lock()
	defer gri.mu.Unlock()
	gri.upstreamQueryHealthHTTPClient = client
}

// SetUpstreamQueryHTTPClient overrides the HTTP client used for upstream query forwarding.
//
//nolint:wsl // Setter keeps upstream client wiring explicit and simple.
func (gri *GatewayRouterIntegration) SetUpstreamQueryHTTPClient(client *http.Client) {
	gri.mu.Lock()
	defer gri.mu.Unlock()

	gri.upstreamQueryHTTPClient = client
	if gri.router != nil {
		gri.router.SetHTTPClient(client)
	}
}

// SetGraphQLHandler wires the GraphQL handler for route registration.
func (gri *GatewayRouterIntegration) SetGraphQLHandler(handler *GraphQLHandler) {
	gri.mu.Lock()
	defer gri.mu.Unlock()

	gri.graphqlHandler = handler
}

// registerRoutes registers all API routes
func (gri *GatewayRouterIntegration) registerRoutes() error {
	// Register subscription routes with HIGH priority (more specific paths)
	if gri.shouldRegisterSubscriptionRoutes() {
		websocketRoute := NewRoute("websocket-subscribe", "/ws", "GET")
		websocketRoute.SetPriority(100)
		if err := gri.router.RegisterRoute(websocketRoute); err != nil {
			return fmt.Errorf("failed to register websocket route: %w", err)
		}

		subscribeRoute := NewRoute("subscribe", "/events/subscribe", "GET")
		subscribeRoute.SetPriority(100)
		if err := gri.router.RegisterRoute(subscribeRoute); err != nil {
			return fmt.Errorf("failed to register subscribe route: %w", err)
		}

		subscribeChainRoute := NewRoute("subscribe-chain", "/events/subscribe/chain/:chainId", "GET")
		subscribeChainRoute.SetPriority(100)
		if err := gri.router.RegisterRoute(subscribeChainRoute); err != nil {
			return fmt.Errorf("failed to register subscribe chain route: %w", err)
		}

		subscribeContractRoute := NewRoute("subscribe-contract", "/events/subscribe/contract/:address", "GET")
		subscribeContractRoute.SetPriority(100)
		if err := gri.router.RegisterRoute(subscribeContractRoute); err != nil {
			return fmt.Errorf("failed to register subscribe contract route: %w", err)
		}

		subscribeNameRoute := NewRoute("subscribe-name", "/events/subscribe/name/:eventName", "GET")
		subscribeNameRoute.SetPriority(100)
		if err := gri.router.RegisterRoute(subscribeNameRoute); err != nil {
			return fmt.Errorf("failed to register subscribe name route: %w", err)
		}
	}

	// Then register event query routes (lower priority)
	if gri.shouldRegisterEventQueryRoutes() {
		eventQueryRoute := NewRoute("event-query", "/events", "GET")
		if err := gri.router.RegisterRoute(eventQueryRoute); err != nil {
			return fmt.Errorf("failed to register event query route: %w", err)
		}

		eventByIDRoute := NewRoute("event-by-id", "/events/:id", "GET")
		eventByIDRoute.SetPriority(50)
		if err := gri.router.RegisterRoute(eventByIDRoute); err != nil {
			return fmt.Errorf("failed to register event by ID route: %w", err)
		}

		eventByChainRoute := NewRoute("event-by-chain", "/events/chain/:chainId", "GET")
		eventByChainRoute.SetPriority(50)
		if err := gri.router.RegisterRoute(eventByChainRoute); err != nil {
			return fmt.Errorf("failed to register event by chain route: %w", err)
		}

		eventByContractRoute := NewRoute("event-by-contract", "/events/contract/:address", "GET")
		eventByContractRoute.SetPriority(50)
		if err := gri.router.RegisterRoute(eventByContractRoute); err != nil {
			return fmt.Errorf("failed to register event by contract route: %w", err)
		}

		eventByNameRoute := NewRoute("event-by-name", "/events/name/:eventName", "GET")
		eventByNameRoute.SetPriority(50)
		if err := gri.router.RegisterRoute(eventByNameRoute); err != nil {
			return fmt.Errorf("failed to register event by name route: %w", err)
		}
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

	rolloutRoute := NewRoute("rollout", "/health/rollout", "GET")
	if err := gri.router.RegisterRoute(rolloutRoute); err != nil {
		return fmt.Errorf("failed to register rollout route: %w", err)
	}

	modelsRoute := NewRoute("models", "/models", "GET")
	if err := gri.router.RegisterRoute(modelsRoute); err != nil {
		return fmt.Errorf("failed to register models route: %w", err)
	}

	if gri.runtimeSummaryProvider != nil {
		runtimeSummaryRoute := NewRoute("runtime-summary", "/runtime/summary", "GET")
		if err := gri.router.RegisterRoute(runtimeSummaryRoute); err != nil {
			return fmt.Errorf("failed to register runtime summary route: %w", err)
		}
	}
	if gri.runtimeMetricsProvider != nil {
		runtimeMetricsRoute := NewRoute("runtime-metrics", "/metrics", "GET")
		if err := gri.router.RegisterRoute(runtimeMetricsRoute); err != nil {
			return fmt.Errorf("failed to register runtime metrics route: %w", err)
		}
	}
	if gri.runtimeControlProvider != nil {
		runtimeControlRoute := NewRoute("runtime-control", "/runtime/control", "GET")
		if err := gri.router.RegisterRoute(runtimeControlRoute); err != nil {
			return fmt.Errorf("failed to register runtime control route: %w", err)
		}
	}

	if gri.runtimeReplayProvider != nil {
		runtimeReplayRoute := NewRoute("runtime-replay", "/runtime/indexing/dlq/replay", "POST")
		if err := gri.router.RegisterRoute(runtimeReplayRoute); err != nil {
			return fmt.Errorf("failed to register runtime replay route: %w", err)
		}
	}

	if gri.graphqlHandler != nil || len(gri.upstreamQueryEndpoints) > 0 {
		graphqlRoute := NewRoute("graphql", "/graphql", "GET,POST,OPTIONS")
		if err := gri.router.RegisterRoute(graphqlRoute); err != nil {
			return fmt.Errorf("failed to register graphql route: %w", err)
		}
	}

	if err := gri.attachUpstreamQueryHandlers(); err != nil {
		return err
	}

	if gri.upstreamQueryHTTPClient != nil {
		gri.router.SetHTTPClient(gri.upstreamQueryHTTPClient)
	}

	gri.logger.Info("All routes registered successfully")
	return nil
}

func (gri *GatewayRouterIntegration) shouldRegisterEventQueryRoutes() bool {
	return gri.eventQueryHandler != nil || len(gri.upstreamQueryEndpoints) > 0
}

func (gri *GatewayRouterIntegration) shouldRegisterSubscriptionRoutes() bool {
	return gri.subscriptionHandler != nil
}

func (gri *GatewayRouterIntegration) attachUpstreamQueryHandlers() error {
	if len(gri.upstreamQueryEndpoints) == 0 {
		return nil
	}

	routeIDs := []string{
		"event-query",
		"event-by-id",
		"event-by-chain",
		"event-by-contract",
		"event-by-name",
		"graphql",
	}

	for idx, endpoint := range gri.upstreamQueryEndpoints {
		handler := NewHandler(
			fmt.Sprintf("api-service-upstream-%d", idx+1),
			fmt.Sprintf("api-service-upstream-%d", idx+1),
			endpoint,
		)
		if gri.upstreamQueryHealthHTTPClient != nil {
			handler.SetHealthHTTPClient(gri.upstreamQueryHealthHTTPClient)
		}
		for _, routeID := range routeIDs {
			if err := gri.router.AttachHandler(routeID, handler); err != nil {
				return fmt.Errorf("failed to attach upstream handler to route %s: %w", routeID, err)
			}
		}
	}

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

	// Check if this is a WebSocket upgrade request - wrap response if so
	isWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
	var wrappedWriter http.ResponseWriter = w
	if isWebSocket {
		if hj, ok := w.(http.Hijacker); ok {
			_ = hj // Already hijackable
		} else {
			wrappedWriter = &HijackableResponseWriter{ResponseWriter: w}
		}
	}

	normalizedReq := normalizeGatewayAPIV1Request(r)

	// Match route
	route, params, err := gri.router.MatchRoute(normalizedReq.URL.Path)
	if err != nil {
		gri.logger.Warn("No route matched", "path", normalizedReq.URL.Path)
		gri.metrics.RecordCounter("gateway_route_not_found", 1, nil)
		gri.logger.Warn("Route match failed", "path", normalizedReq.URL.Path, "methods_tried", normalizedReq.Method)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if route.Method != "" && route.Method != normalizedReq.Method {
		methods := strings.Split(route.Method, ",")
		matched := false
		for _, m := range methods {
			if strings.TrimSpace(m) == normalizedReq.Method {
				matched = true
				break
			}
		}
		if !matched {
			gri.logger.Warn("Method mismatch", "route_method", route.Method, "request_method", normalizedReq.Method, "path", normalizedReq.URL.Path)
			w.Header().Set("Allow", route.Method)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			gri.metrics.RecordCounter("gateway_method_not_allowed", 1, map[string]string{
				"route_id": route.ID,
				"method":   normalizedReq.Method,
			})
			return
		}
	}

	// Route to appropriate handler based on route ID
	switch route.ID {
	// Event Query Handlers
	case "event-query":
		if gri.tryForwardQueryRequest(w, normalizedReq, route) {
			return
		}
		if gri.eventQueryHandler == nil {
			respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
			return
		}
		gri.eventQueryHandler.HandleGetAllEvents(w, normalizedReq)
	case "event-by-id":
		if gri.tryForwardQueryRequest(w, normalizedReq, route) {
			return
		}
		if gri.eventQueryHandler == nil {
			respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
			return
		}
		gri.eventQueryHandler.HandleGetEventByID(w, normalizedReq, params["id"])
	case "event-by-chain":
		if gri.tryForwardQueryRequest(w, normalizedReq, route) {
			return
		}
		if gri.eventQueryHandler == nil {
			respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
			return
		}
		gri.eventQueryHandler.HandleGetEventsByChain(w, normalizedReq, params["chainId"])
	case "event-by-contract":
		if gri.tryForwardQueryRequest(w, normalizedReq, route) {
			return
		}
		if gri.eventQueryHandler == nil {
			respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
			return
		}
		gri.eventQueryHandler.HandleGetEventsByContract(w, normalizedReq, params["address"])
	case "event-by-name":
		if gri.tryForwardQueryRequest(w, normalizedReq, route) {
			return
		}
		if gri.eventQueryHandler == nil {
			respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
			return
		}
		gri.eventQueryHandler.HandleGetEventsByName(w, normalizedReq, params["eventName"])

	// Event Subscription Handlers
	case "websocket-subscribe", "subscribe":
		gri.logger.Info("Handling WebSocket subscribe", "path", normalizedReq.URL.Path, "handler_nil", gri.subscriptionHandler == nil)
		if gri.subscriptionHandler == nil {
			gri.logger.Error("subscriptionHandler is nil!")
			http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
			return
		}
		gri.subscriptionHandler.HandleSubscribeAll(wrappedWriter, normalizedReq)
	case "subscribe-chain":
		gri.logger.Info("Handling WebSocket subscribe-chain", "path", normalizedReq.URL.Path, "chainId", params["chainId"])
		if gri.subscriptionHandler == nil {
			http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
			return
		}
		gri.subscriptionHandler.HandleSubscribeChain(w, normalizedReq, params["chainId"])
	case "subscribe-contract":
		if gri.subscriptionHandler == nil {
			http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
			return
		}
		gri.subscriptionHandler.HandleSubscribeContract(w, normalizedReq, params["address"])
	case "subscribe-name":
		if gri.subscriptionHandler == nil {
			http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
			return
		}
		gri.subscriptionHandler.HandleSubscribeName(w, normalizedReq, params["eventName"])

	// Health Check Handlers
	case "health":
		gri.healthCheckHandler.HandleHealth(w, normalizedReq)
	case "ready":
		gri.healthCheckHandler.HandleReady(w, normalizedReq)
	case "live":
		gri.healthCheckHandler.HandleLive(w, normalizedReq)
	case "components":
		gri.healthCheckHandler.HandleComponents(w, normalizedReq)
	case "rollout":
		gri.healthCheckHandler.HandleRollout(w, normalizedReq)
	case "runtime-summary":
		gri.handleRuntimeSummary(w, normalizedReq)
	case "runtime-metrics":
		gri.handleRuntimeMetrics(w, normalizedReq)
	case "runtime-control":
		gri.handleRuntimeControl(w, normalizedReq)
	case "runtime-replay":
		gri.handleRuntimeReplay(w, normalizedReq)

	// Models
	case "models":
		gri.modelsHandler.HandleModels(w, normalizedReq)

	// GraphQL
	case "graphql":
		if gri.tryForwardQueryRequest(w, normalizedReq, route) {
			return
		}
		if gri.graphqlHandler == nil {
			http.Error(w, "GraphQL handler not configured", http.StatusInternalServerError)
			return
		}
		gri.graphqlHandler.Handle(w, normalizedReq)

	default:
		gri.logger.Warn("Unknown route", "routeId", route.ID)
		http.Error(w, "Not Found", http.StatusNotFound)
	}

	gri.metrics.RecordCounter("gateway_request_success", 1, nil)
}

func normalizeGatewayAPIV1Request(r *http.Request) *http.Request {
	if r == nil || r.URL == nil {
		return r
	}

	path := r.URL.Path
	if path == gatewayAPIV1Prefix {
		path = "/"
	} else if strings.HasPrefix(path, gatewayAPIV1Prefix+"/") {
		path = strings.TrimPrefix(path, gatewayAPIV1Prefix)
	}

	if path == r.URL.Path {
		return r
	}

	cloned := r.Clone(r.Context())
	clonedURL := *r.URL
	clonedURL.Path = path
	clonedURL.RawPath = ""
	cloned.URL = &clonedURL
	return cloned
}

func (gri *GatewayRouterIntegration) handleRuntimeControl(w http.ResponseWriter, r *http.Request) {
	if gri.runtimeControlProvider == nil {
		http.Error(w, "runtime control unavailable", http.StatusNotFound)
		return
	}

	gri.runtimeControlProvider(w, r)
}

func (gri *GatewayRouterIntegration) handleRuntimeReplay(w http.ResponseWriter, r *http.Request) {
	if gri.runtimeReplayProvider == nil {
		http.Error(w, "runtime replay unavailable", http.StatusNotFound)

		return
	}

	gri.runtimeReplayProvider(w, r)
}

func (gri *GatewayRouterIntegration) tryForwardQueryRequest(w http.ResponseWriter, r *http.Request, route *Route) bool {
	if route == nil || len(route.GetHandlers()) == 0 {
		return false
	}

	var bodyBytes []byte
	if r != nil && r.Body != nil {
		consumedBody, err := io.ReadAll(r.Body)
		if err != nil {
			gri.logger.Error("Failed to read gateway request body", "routeId", route.ID, "error", err.Error())
			gri.metrics.RecordCounter("gateway_query_bridge_unavailable", 1, map[string]string{"route_id": route.ID})
			respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
			return true
		}
		bodyBytes = consumedBody
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	response, err := gri.router.ForwardRequest(r.Context(), route, &ForwardedRequest{
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: flattenRequestHeaders(r.Header),
		Body:    bodyBytes,
	})
	if err != nil {
		gri.logger.Error("Failed to forward gateway query request", "routeId", route.ID, "error", err.Error())
		gri.metrics.RecordCounter("gateway_query_bridge_unavailable", 1, map[string]string{"route_id": route.ID})
		respondGatewayQueryBridgeError(w, http.StatusBadGateway, route.ID)
		return true
	}

	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}
	if response.Status > 0 {
		w.WriteHeader(response.Status)
	}
	if body, ok := response.Body.([]byte); ok && len(body) > 0 {
		_, _ = w.Write(body)
	}
	return true
}

func respondGatewayQueryBridgeError(w http.ResponseWriter, statusCode int, routeID string) {
	payload := map[string]interface{}{
		"error":      "query_upstream_unavailable",
		"message":    "api-gateway could not reach an upstream api-service for this query request",
		"statusCode": statusCode,
		"timestamp":  time.Now().Unix(),
		"meta": map[string]interface{}{
			"routeId":          routeID,
			"bridgePosture":    "query-bridge-unavailable",
			"reliabilityHint":  "gateway query bridge is currently unavailable; verify api-service health and upstream bridge status",
			"responseBoundary": "gateway-query-bridge",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func flattenRequestHeaders(header http.Header) map[string]string {
	flattened := make(map[string]string, len(header))
	for key, values := range header {
		if len(values) > 0 {
			flattened[key] = values[0]
		}
	}
	return flattened
}

func (gri *GatewayRouterIntegration) handleRuntimeMetrics(w http.ResponseWriter, r *http.Request) {
	if gri.runtimeMetricsProvider == nil {
		http.Error(w, "runtime metrics unavailable", http.StatusNotFound)
		return
	}

	payload := gri.runtimeMetricsProvider(r)
	if payload == nil {
		http.Error(w, "runtime metrics unavailable", http.StatusServiceUnavailable)
		return
	}

	if textPayload, ok := payload.(string); ok {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(textPayload))

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (gri *GatewayRouterIntegration) handleRuntimeSummary(w http.ResponseWriter, r *http.Request) {
	if gri.runtimeSummaryProvider == nil {
		http.Error(w, "runtime summary unavailable", http.StatusNotFound)
		return
	}

	payload := gri.runtimeSummaryProvider(r)
	if payload == nil {
		http.Error(w, "runtime summary unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

// GetRouter returns the request router
func (gri *GatewayRouterIntegration) GetRouter() *RequestRouter {
	gri.mu.RLock()
	defer gri.mu.RUnlock()

	return gri.router
}

// GetUpstreamQueryBridgeStatus returns configured/attached/available counts for query upstream handlers.
func (gri *GatewayRouterIntegration) GetUpstreamQueryBridgeStatus() (configured, attached, available int) {
	gri.mu.RLock()
	defer gri.mu.RUnlock()

	configured = len(gri.upstreamQueryEndpoints)
	if gri.router == nil {
		return configured, 0, 0
	}

	route, err := gri.router.GetRoute("event-query")
	if err != nil || route == nil {
		return configured, 0, 0
	}

	handlers := route.GetHandlers()
	attached = len(handlers)
	for _, handler := range handlers {
		if handler != nil && handler.IsAvailable() {
			available++
		}
	}
	return configured, attached, available
}

// GetRuntimeRouteInventory returns the currently registered runtime-route footprint.
func (gri *GatewayRouterIntegration) GetRuntimeRouteInventory() GatewayRuntimeRouteInventory {
	gri.mu.RLock()
	router := gri.router
	gri.mu.RUnlock()

	if router == nil {
		return GatewayRuntimeRouteInventory{}
	}

	routes := router.GetRoutes()
	inventory := GatewayRuntimeRouteInventory{
		RegisteredRouteCount: len(routes),
	}

	runtimeRouteIDs := map[string]struct{}{
		"health":          {},
		"ready":           {},
		"live":            {},
		"components":      {},
		"rollout":         {},
		"runtime-summary": {},
		"runtime-metrics": {},
		"runtime-control": {},
		"runtime-replay":  {},
	}

	for _, route := range routes {
		if route == nil {
			continue
		}
		if _, ok := runtimeRouteIDs[route.ID]; ok {
			inventory.RuntimeRouteCount++
		}
		switch route.ID {
		case "health", "ready", "live", "components", "rollout":
			inventory.HealthRoutesEnabled = true
		case "runtime-summary":
			inventory.SummaryRouteEnabled = true
		case "runtime-metrics":
			inventory.MetricsRouteEnabled = true
		case "runtime-control":
			inventory.ControlRouteEnabled = true
		case "runtime-replay":
			inventory.ReplayRouteEnabled = true
		}
	}

	if inventory.HealthRoutesEnabled {
		inventory.RuntimeSurfaceCount++
	}
	if inventory.SummaryRouteEnabled {
		inventory.RuntimeSurfaceCount++
	}
	if inventory.MetricsRouteEnabled {
		inventory.RuntimeSurfaceCount++
	}
	if inventory.ControlRouteEnabled {
		inventory.RuntimeSurfaceCount++
	}

	if inventory.ReplayRouteEnabled {
		inventory.RuntimeSurfaceCount++
	}

	return inventory
}

// RefreshUpstreamQueryBridgeHealth actively refreshes upstream query handler health.
func (gri *GatewayRouterIntegration) RefreshUpstreamQueryBridgeHealth() {
	gri.mu.RLock()
	router := gri.router
	gri.mu.RUnlock()

	if router == nil {
		return
	}

	route, err := router.GetRoute("event-query")
	if err != nil || route == nil {
		return
	}

	for _, handler := range route.GetHandlers() {
		if handler != nil {
			handler.CheckHealth()
		}
	}
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

// Hijack implements http.Hijacker for WebSocket upgrade support
func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}

type HijackableResponseWriter struct {
	http.ResponseWriter
	conn *net.Conn
}

func (hw *HijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := hw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	if hw.conn != nil {
		return *hw.conn, bufio.NewReadWriter(bufio.NewReader(*hw.conn), bufio.NewWriter(*hw.conn)), nil
	}
	r, w := net.Pipe()
	hw.conn = &w
	return r, bufio.NewReadWriter(bufio.NewReader(r), bufio.NewWriter(w)), nil
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
