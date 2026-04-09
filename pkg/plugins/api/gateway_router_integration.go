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

// registerRoutes registers all API routes.
func (gri *GatewayRouterIntegration) registerRoutes() error {
	if err := gri.registerSubscriptionRoutes(); err != nil {
		return err
	}
	if err := gri.registerEventQueryRoutes(); err != nil {
		return err
	}
	if err := gri.registerHealthRoutes(); err != nil {
		return err
	}
	if err := gri.registerModelsRoutes(); err != nil {
		return err
	}
	if err := gri.registerRuntimeRoutes(); err != nil {
		return err
	}
	if err := gri.registerGraphQLRoutes(); err != nil {
		return err
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

func (gri *GatewayRouterIntegration) registerRoute(routeID, path, method string, priority int) error {
	route := NewRoute(routeID, path, method)
	if priority > 0 {
		route.SetPriority(priority)
	}
	if err := gri.router.RegisterRoute(route); err != nil {
		return fmt.Errorf("failed to register route %s: %w", routeID, err)
	}
	return nil
}

func (gri *GatewayRouterIntegration) registerSubscriptionRoutes() error {
	if !gri.shouldRegisterSubscriptionRoutes() {
		return nil
	}

	if err := gri.registerRoute("websocket-subscribe", "/ws", "GET", 100); err != nil {
		return err
	}
	if err := gri.registerRoute("subscribe", "/events/subscribe", "GET", 100); err != nil {
		return err
	}
	if err := gri.registerRoute("subscribe-chain", "/events/subscribe/chain/:chainId", "GET", 100); err != nil {
		return err
	}
	if err := gri.registerRoute("subscribe-contract", "/events/subscribe/contract/:address", "GET", 100); err != nil {
		return err
	}
	if err := gri.registerRoute("subscribe-name", "/events/subscribe/name/:eventName", "GET", 100); err != nil {
		return err
	}

	return nil
}

func (gri *GatewayRouterIntegration) registerEventQueryRoutes() error {
	if !gri.shouldRegisterEventQueryRoutes() {
		return nil
	}

	if err := gri.registerRoute("event-query", "/events", "GET", 0); err != nil {
		return err
	}
	if err := gri.registerRoute("event-by-id", "/events/:id", "GET", 50); err != nil {
		return err
	}
	if err := gri.registerRoute("event-by-chain", "/events/chain/:chainId", "GET", 50); err != nil {
		return err
	}
	if err := gri.registerRoute("event-by-contract", "/events/contract/:address", "GET", 50); err != nil {
		return err
	}
	if err := gri.registerRoute("event-by-name", "/events/name/:eventName", "GET", 50); err != nil {
		return err
	}

	return nil
}

func (gri *GatewayRouterIntegration) registerHealthRoutes() error {
	if err := gri.registerRoute("health", "/health", "GET", 0); err != nil {
		return err
	}
	if err := gri.registerRoute("ready", "/health/ready", "GET", 0); err != nil {
		return err
	}
	if err := gri.registerRoute("live", "/health/live", "GET", 0); err != nil {
		return err
	}
	if err := gri.registerRoute("components", "/health/components", "GET", 0); err != nil {
		return err
	}
	if err := gri.registerRoute("rollout", "/health/rollout", "GET", 0); err != nil {
		return err
	}
	return nil
}

func (gri *GatewayRouterIntegration) registerModelsRoutes() error {
	return gri.registerRoute("models", "/models", "GET", 0)
}

func (gri *GatewayRouterIntegration) registerRuntimeRoutes() error {
	if gri.runtimeSummaryProvider != nil {
		if err := gri.registerRoute("runtime-summary", "/runtime/summary", "GET", 0); err != nil {
			return err
		}
	}
	if gri.runtimeMetricsProvider != nil {
		if err := gri.registerRoute("runtime-metrics", "/metrics", "GET", 0); err != nil {
			return err
		}
	}
	if gri.runtimeControlProvider != nil {
		if err := gri.registerRoute("runtime-control", "/runtime/control", "GET", 0); err != nil {
			return err
		}
	}
	if gri.runtimeReplayProvider != nil {
		if err := gri.registerRoute("runtime-replay", "/runtime/indexing/dlq/replay", "POST", 0); err != nil {
			return err
		}
	}
	return nil
}

func (gri *GatewayRouterIntegration) registerGraphQLRoutes() error {
	if gri.graphqlHandler == nil && len(gri.upstreamQueryEndpoints) == 0 {
		return nil
	}
	return gri.registerRoute("graphql", "/graphql", "GET,POST,OPTIONS", 0)
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

type gatewayRouteHandler func(
	*GatewayRouterIntegration,
	http.ResponseWriter,
	http.ResponseWriter,
	*http.Request,
	*Route,
	map[string]string,
)

var gatewayRouteHandlers = map[string]gatewayRouteHandler{
	"event-query":         gatewayHandleEventQuery,
	"event-by-id":         gatewayHandleEventByID,
	"event-by-chain":      gatewayHandleEventByChain,
	"event-by-contract":   gatewayHandleEventByContract,
	"event-by-name":       gatewayHandleEventByName,
	"websocket-subscribe": gatewayHandleSubscribeAll,
	"subscribe":           gatewayHandleSubscribeAll,
	"subscribe-chain":     gatewayHandleSubscribeChain,
	"subscribe-contract":  gatewayHandleSubscribeContract,
	"subscribe-name":      gatewayHandleSubscribeName,
	"health":              gatewayHandleHealth,
	"ready":               gatewayHandleReady,
	"live":                gatewayHandleLive,
	"components":          gatewayHandleComponents,
	"rollout":             gatewayHandleRollout,
	"runtime-summary":     gatewayHandleRuntimeSummary,
	"runtime-metrics":     gatewayHandleRuntimeMetrics,
	"runtime-control":     gatewayHandleRuntimeControl,
	"runtime-replay":      gatewayHandleRuntimeReplay,
	"models":              gatewayHandleModels,
	"graphql":             gatewayHandleGraphQL,
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
	wrappedWriter := w
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

	handler, ok := gatewayRouteHandlers[route.ID]
	if !ok {
		gri.logger.Warn("Unknown route", "routeId", route.ID)
		http.Error(w, "Not Found", http.StatusNotFound)
	} else {
		handler(gri, w, wrappedWriter, normalizedReq, route, params)
	}

	gri.metrics.RecordCounter("gateway_request_success", 1, nil)
}

func (gri *GatewayRouterIntegration) handleQueryRoute(w http.ResponseWriter, r *http.Request, route *Route, routeID string, handle func()) {
	if gri.tryForwardQueryRequest(w, r, route) {
		return
	}
	if gri.eventQueryHandler == nil {
		respondGatewayQueryBridgeError(w, http.StatusBadGateway, routeID)
		return
	}
	handle()
}

func gatewayHandleEventQuery(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, route *Route, _ map[string]string) {
	gri.handleQueryRoute(w, r, route, "event-query", func() { gri.eventQueryHandler.HandleGetAllEvents(w, r) })
}

func gatewayHandleEventByID(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, route *Route, params map[string]string) {
	id := ""
	if params != nil {
		id = params["id"]
	}
	gri.handleQueryRoute(w, r, route, "event-by-id", func() { gri.eventQueryHandler.HandleGetEventByID(w, r, id) })
}

func gatewayHandleEventByChain(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, route *Route, params map[string]string) {
	chainID := ""
	if params != nil {
		chainID = params["chainId"]
	}
	gri.handleQueryRoute(w, r, route, "event-by-chain", func() { gri.eventQueryHandler.HandleGetEventsByChain(w, r, chainID) })
}

func gatewayHandleEventByContract(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, route *Route, params map[string]string) {
	address := ""
	if params != nil {
		address = params["address"]
	}
	gri.handleQueryRoute(w, r, route, "event-by-contract", func() { gri.eventQueryHandler.HandleGetEventsByContract(w, r, address) })
}

func gatewayHandleEventByName(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, route *Route, params map[string]string) {
	eventName := ""
	if params != nil {
		eventName = params["eventName"]
	}
	gri.handleQueryRoute(w, r, route, "event-by-name", func() { gri.eventQueryHandler.HandleGetEventsByName(w, r, eventName) })
}

func gatewayHandleSubscribeAll(gri *GatewayRouterIntegration, w http.ResponseWriter, wrapped http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.logger.Info("Handling WebSocket subscribe", "path", r.URL.Path, "handler_nil", gri.subscriptionHandler == nil)
	if gri.subscriptionHandler == nil {
		gri.logger.Error("subscriptionHandler is nil!")
		http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
		return
	}
	gri.subscriptionHandler.HandleSubscribeAll(wrapped, r)
}

func gatewayHandleSubscribeChain(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, params map[string]string) {
	chainID := ""
	if params != nil {
		chainID = params["chainId"]
	}
	gri.logger.Info("Handling WebSocket subscribe-chain", "path", r.URL.Path, "chainId", chainID)
	if gri.subscriptionHandler == nil {
		http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
		return
	}
	gri.subscriptionHandler.HandleSubscribeChain(w, r, chainID)
}

func gatewayHandleSubscribeContract(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, params map[string]string) {
	address := ""
	if params != nil {
		address = params["address"]
	}
	if gri.subscriptionHandler == nil {
		http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
		return
	}
	gri.subscriptionHandler.HandleSubscribeContract(w, r, address)
}

func gatewayHandleSubscribeName(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, params map[string]string) {
	eventName := ""
	if params != nil {
		eventName = params["eventName"]
	}
	if gri.subscriptionHandler == nil {
		http.Error(w, "Subscription handler not configured", http.StatusInternalServerError)
		return
	}
	gri.subscriptionHandler.HandleSubscribeName(w, r, eventName)
}

func gatewayHandleHealth(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.healthCheckHandler.HandleHealth(w, r)
}

func gatewayHandleReady(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.healthCheckHandler.HandleReady(w, r)
}

func gatewayHandleLive(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.healthCheckHandler.HandleLive(w, r)
}

func gatewayHandleComponents(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.healthCheckHandler.HandleComponents(w, r)
}

func gatewayHandleRollout(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.healthCheckHandler.HandleRollout(w, r)
}

func gatewayHandleRuntimeSummary(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.handleRuntimeSummary(w, r)
}

func gatewayHandleRuntimeMetrics(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.handleRuntimeMetrics(w, r)
}

func gatewayHandleRuntimeControl(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.handleRuntimeControl(w, r)
}

func gatewayHandleRuntimeReplay(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.handleRuntimeReplay(w, r)
}

func gatewayHandleModels(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, _ *Route, _ map[string]string) {
	gri.modelsHandler.HandleModels(w, r)
}

func gatewayHandleGraphQL(gri *GatewayRouterIntegration, w http.ResponseWriter, _ http.ResponseWriter, r *http.Request, route *Route, _ map[string]string) {
	if gri.tryForwardQueryRequest(w, r, route) {
		return
	}
	if gri.graphqlHandler == nil {
		http.Error(w, "GraphQL handler not configured", http.StatusInternalServerError)
		return
	}
	gri.graphqlHandler.Handle(w, r)
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
