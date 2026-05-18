package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	corelib "github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	apicore "github.com/rtcdance/chainpulse/pkg/plugins/api/core"
	httpapi "github.com/rtcdance/chainpulse/pkg/plugins/api/http"
)

// APIGatewayPlugin is the main API gateway that routes requests to appropriate handlers
type APIGatewayPlugin struct {
	logger                        corelib.Logger
	metrics                       corelib.MetricsCollector
	httpPlugin                    *httpapi.HTTPPlugin
	handlers                      map[string]Handler
	domainQueryService            domainquery.Service
	eventQueryHandler             *EventQueryHandler
	eventSubHandler               *EventSubscriptionHandler
	healthCheckHandler            *HealthCheckHandler
	graphqlHandler                *GraphQLHandler
	dlqHandler                    *DLQHandler
	upstreamQueryEndpoints        []string
	upstreamQueryHTTPClient       *http.Client
	upstreamQueryHealthHTTPClient *http.Client
	upstreamQueryHealthHeaders    map[string]string
	runtimeMetricsProvider        func(*http.Request) any
	runtimeSummaryProvider        func(*http.Request) any
	runtimeControlProvider        func(http.ResponseWriter, *http.Request)
	runtimeReplayProvider         func(http.ResponseWriter, *http.Request)
	authMiddleware                *AuthMiddleware
	rateLimitMiddleware           *RateLimitMiddleware
	corsMiddleware                *CORSMiddleware
	routerIntegration             *GatewayRouterIntegration
	domainBridgeEnabled           bool
	eventQueryEnabled             bool
	runtimeRoutesEnabled          bool
	mu                            sync.RWMutex
	initialized                   bool
	running                       bool
}

// Handler defines the interface for API handlers
type Handler interface {
	Initialize(config *corelib.Config) error
	Handle(w http.ResponseWriter, r *http.Request)
	Stop() error
}

// NewAPIGatewayPlugin creates a new API gateway plugin
func NewAPIGatewayPlugin(logger corelib.Logger, metrics corelib.MetricsCollector) *APIGatewayPlugin {
	return &APIGatewayPlugin{
		logger:   logger,
		metrics:  metrics,
		handlers: make(map[string]Handler),
	}
}

// SetDomainQueryService sets an optional domain query service bridge for phased runtime migration.
func (g *APIGatewayPlugin) SetDomainQueryService(service domainquery.Service) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.domainQueryService = service
	g.domainBridgeEnabled = service != nil
}

// IsDomainBridgeEnabled returns whether domain query bridge is configured.
func (g *APIGatewayPlugin) IsDomainBridgeEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.domainBridgeEnabled || len(g.upstreamQueryEndpoints) > 0
}

// SetEventQueryHandler sets an optional runtime event query handler for phased route migration.
func (g *APIGatewayPlugin) SetEventQueryHandler(handler *EventQueryHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.eventQueryHandler = handler
	g.eventQueryEnabled = handler != nil
}

// IsEventQueryHandlerEnabled returns whether event query handler is configured.
func (g *APIGatewayPlugin) IsEventQueryHandlerEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.eventQueryEnabled
}

// SetEventSubscriptionHandler sets an optional subscription handler for runtime route composition.
func (g *APIGatewayPlugin) SetEventSubscriptionHandler(handler *EventSubscriptionHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.eventSubHandler = handler
}

// IsEventSubscriptionHandlerEnabled returns whether runtime subscription handler is configured.
func (g *APIGatewayPlugin) IsEventSubscriptionHandlerEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.eventSubHandler != nil
}

// SetHealthCheckHandler sets an optional health check handler for runtime route composition.
func (g *APIGatewayPlugin) SetHealthCheckHandler(handler *HealthCheckHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.healthCheckHandler = handler
}

// SetUpstreamQueryEndpoints sets optional upstream api-service endpoints for query forwarding.
func (g *APIGatewayPlugin) SetUpstreamQueryEndpoints(endpoints []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upstreamQueryEndpoints = append([]string(nil), endpoints...)
}

// SetUpstreamQueryHTTPClient overrides the HTTP client used for upstream query forwarding.
func (g *APIGatewayPlugin) SetUpstreamQueryHTTPClient(client *http.Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upstreamQueryHTTPClient = client
}

// SetUpstreamQueryHealthHTTPClient overrides the HTTP client used for upstream query health checks.
func (g *APIGatewayPlugin) SetUpstreamQueryHealthHTTPClient(client *http.Client) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.upstreamQueryHealthHTTPClient = client
}

// SetUpstreamQueryHealthHeaders overrides static headers used for upstream query health checks.
func (g *APIGatewayPlugin) SetUpstreamQueryHealthHeaders(headers map[string]string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(headers) == 0 {
		g.upstreamQueryHealthHeaders = nil
		return
	}

	copied := make(map[string]string, len(headers))
	for key, value := range headers {
		copied[key] = value
	}
	g.upstreamQueryHealthHeaders = copied
}

// SetRuntimeSummaryProvider sets an optional read-only runtime summary provider.
func (g *APIGatewayPlugin) SetRuntimeSummaryProvider(provider func(*http.Request) any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.runtimeSummaryProvider = provider
}

// SetRuntimeMetricsProvider sets an optional read-only runtime metrics provider.
func (g *APIGatewayPlugin) SetRuntimeMetricsProvider(provider func(*http.Request) any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.runtimeMetricsProvider = provider
}

// IsMetricsRouteEnabled returns whether runtime metrics provider is configured.
func (g *APIGatewayPlugin) IsMetricsRouteEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.runtimeMetricsProvider != nil
}

// SetRuntimeControlProvider sets an optional runtime control handler.
func (g *APIGatewayPlugin) SetRuntimeControlProvider(provider func(http.ResponseWriter, *http.Request)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.runtimeControlProvider = provider
}

// SetRuntimeReplayProvider sets an optional runtime replay handler.
func (g *APIGatewayPlugin) SetRuntimeReplayProvider(provider func(http.ResponseWriter, *http.Request)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.runtimeReplayProvider = provider
}

// SetAuthMiddleware wires an optional gateway auth middleware.
func (g *APIGatewayPlugin) SetAuthMiddleware(middleware *AuthMiddleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.authMiddleware = middleware
}

// IsAuthMiddlewareEnabled returns whether gateway auth middleware is configured.
func (g *APIGatewayPlugin) IsAuthMiddlewareEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.authMiddleware != nil
}

// SetRateLimitMiddleware wires an optional gateway rate limit middleware.
func (g *APIGatewayPlugin) SetRateLimitMiddleware(middleware *RateLimitMiddleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rateLimitMiddleware = middleware
}

// IsRateLimitMiddlewareEnabled returns whether gateway rate limit middleware is configured.
func (g *APIGatewayPlugin) IsRateLimitMiddlewareEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.rateLimitMiddleware != nil
}

// SetCORSMiddleware wires an optional CORS middleware that handles preflight
// requests and sets cross-origin headers. CORS middleware is applied outermost
// so OPTIONS preflight requests bypass auth and rate limiting.
func (g *APIGatewayPlugin) SetCORSMiddleware(middleware *CORSMiddleware) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.corsMiddleware = middleware
}

// SetGraphQLHandler wires an optional GraphQL handler.
func (g *APIGatewayPlugin) SetGraphQLHandler(handler *GraphQLHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.graphqlHandler = handler
}

// SetDLQHandler wires an optional DLQ handler.
func (g *APIGatewayPlugin) SetDLQHandler(handler *DLQHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.dlqHandler = handler
}

// IsHealthCheckHandlerEnabled returns whether runtime health handler is configured.
func (g *APIGatewayPlugin) IsHealthCheckHandlerEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.healthCheckHandler != nil
}

// IsRuntimeRoutesEnabled returns whether runtime route composition is enabled.
func (g *APIGatewayPlugin) IsRuntimeRoutesEnabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.runtimeRoutesEnabled
}

// GetRouterIntegration returns the initialized gateway router integration, if any.
func (g *APIGatewayPlugin) GetRouterIntegration() *GatewayRouterIntegration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.routerIntegration
}

// HTTPHandler returns the HTTP request handler for the initialized gateway.
func (g *APIGatewayPlugin) HTTPHandler() http.HandlerFunc {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.httpPlugin == nil {
		return nil
	}

	return g.httpPlugin.Handler()
}

// GetUpstreamQueryEndpoints returns a copy of configured upstream query endpoints.
func (g *APIGatewayPlugin) GetUpstreamQueryEndpoints() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	endpoints := make([]string, len(g.upstreamQueryEndpoints))
	copy(endpoints, g.upstreamQueryEndpoints)
	return endpoints
}

// GetUpstreamQueryBridgeStatus reports the current upstream query bridge state.
func (g *APIGatewayPlugin) GetUpstreamQueryBridgeStatus() (configured, attached, available int) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	configured = len(g.upstreamQueryEndpoints)
	if g.routerIntegration == nil {
		return configured, 0, 0
	}

	return g.routerIntegration.GetUpstreamQueryBridgeStatus()
}

// GetRuntimeRouteInventory reports the currently registered runtime-route footprint.
func (g *APIGatewayPlugin) GetRuntimeRouteInventory() GatewayRuntimeRouteInventory {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.routerIntegration == nil {
		return GatewayRuntimeRouteInventory{}
	}

	return g.routerIntegration.GetRuntimeRouteInventory()
}

// RefreshUpstreamQueryBridgeHealth actively refreshes upstream query handler health.
func (g *APIGatewayPlugin) RefreshUpstreamQueryBridgeHealth() {
	g.mu.RLock()
	integration := g.routerIntegration
	g.mu.RUnlock()

	if integration != nil {
		integration.RefreshUpstreamQueryBridgeHealth()
	}
}

// Name returns the plugin name
func (g *APIGatewayPlugin) Name() string {
	return "api-gateway"
}

// Version returns the plugin version
func (g *APIGatewayPlugin) Version() string {
	return "1.0.0"
}

// Health checks the plugin health
func (g *APIGatewayPlugin) Health() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.running {
		return fmt.Errorf("API gateway not running")
	}

	return nil
}

// Initialize initializes the API gateway
func (g *APIGatewayPlugin) Initialize(config corelib.Config) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.initialized {
		return fmt.Errorf("API gateway already initialized")
	}

	// Create HTTP plugin
	apiLayer := apicore.NewAPILayer()

	port := config.APIPort
	if port == 0 {
		port = 8080
	}
	g.httpPlugin = httpapi.NewHTTPPlugin("api-gateway", port, apiLayer)

	if g.shouldInitializeRuntimeIntegration() {
		authMiddleware := g.authMiddleware
		rateLimitMiddleware := g.rateLimitMiddleware

		if g.eventSubHandler != nil && rateLimitMiddleware != nil {
			g.eventSubHandler.SetRateLimiter(rateLimitMiddleware.Limiter())
		}

		integration := NewGatewayRouterIntegration(
			g.logger,
			g.metrics,
			g.eventQueryHandler,
			g.eventSubHandler,
			g.healthCheckHandler,
			g.runtimeSummaryProvider,
			g.runtimeMetricsProvider,
			g.runtimeControlProvider,
			g.runtimeReplayProvider,
		)
		integration.SetUpstreamQueryEndpoints(g.upstreamQueryEndpoints)
		integration.SetUpstreamQueryHTTPClient(g.upstreamQueryHTTPClient)
		integration.SetUpstreamQueryHealthHTTPClient(g.upstreamQueryHealthHTTPClient)
		integration.SetUpstreamQueryHealthHeaders(g.upstreamQueryHealthHeaders)
		if g.graphqlHandler != nil {
			integration.SetGraphQLHandler(g.graphqlHandler)
		}
		if g.dlqHandler != nil {
			integration.SetDLQHandler(g.dlqHandler)
		}
		if err := integration.Initialize(context.Background()); err != nil {
			return fmt.Errorf("failed to initialize gateway router integration: %w", err)
		}

		g.routerIntegration = integration
		g.httpPlugin.SetNativeHandler(g.wrapGatewayHandler(integration.HandleRequest, authMiddleware, rateLimitMiddleware))
		g.runtimeRoutesEnabled = true
	}

	g.initialized = true

	g.logger.Info("API gateway initialized", corelib.LogKeyComponent, "api_gateway", "domain_bridge_enabled", g.domainBridgeEnabled, "event_query_handler_set", g.eventQueryEnabled, "runtime_routes_enabled", g.runtimeRoutesEnabled)

	return nil
}

func (g *APIGatewayPlugin) shouldInitializeRuntimeIntegration() bool {
	return g.eventQueryHandler != nil ||
		g.eventSubHandler != nil ||
		g.healthCheckHandler != nil ||
		g.runtimeSummaryProvider != nil ||
		g.runtimeMetricsProvider != nil ||
		g.runtimeControlProvider != nil ||
		len(g.upstreamQueryEndpoints) > 0
}

//nolint:wsl,nlreturn // Security middleware stacking is intentionally explicit here.
func (g *APIGatewayPlugin) wrapGatewayHandler(handler http.HandlerFunc, authMiddleware *AuthMiddleware, rateLimitMiddleware *RateLimitMiddleware) http.HandlerFunc {
	var corsMw *CORSMiddleware
	g.mu.RLock()
	corsMw = g.corsMiddleware
	g.mu.RUnlock()

	wrapped := http.Handler(http.HandlerFunc(handler))

	if rateLimitMiddleware != nil {
		wrapped = rateLimitMiddleware.Middleware(rateLimitMiddleware.limiter)(wrapped)
	}
	if authMiddleware != nil {
		wrapped = authMiddleware.Handler(wrapped)
	}
	if corsMw != nil {
		wrapped = corsMw.Handler(wrapped)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		wrapped.ServeHTTP(w, r)
	}
}

// Start starts the API gateway
func (g *APIGatewayPlugin) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.initialized {
		return fmt.Errorf("API gateway not initialized")
	}

	if g.running {
		return fmt.Errorf("API gateway already running")
	}

	if err := g.httpPlugin.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP plugin: %w", err)
	}

	g.running = true

	g.logger.Info("API gateway started", corelib.LogKeyComponent, "api_gateway", corelib.LogKeyPort, 8080, "domain_bridge_enabled", g.domainBridgeEnabled, "event_query_handler_set", g.eventQueryEnabled, "runtime_routes_enabled", g.runtimeRoutesEnabled)

	return nil
}

// Stop stops the API gateway gracefully
func (g *APIGatewayPlugin) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return g.ShutdownWithContext(ctx)
}

// ShutdownWithContext stops the gateway gracefully with a context deadline,
// allowing in-flight HTTP requests to complete.
func (g *APIGatewayPlugin) ShutdownWithContext(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.running {
		return fmt.Errorf("API gateway not running")
	}

	if err := g.httpPlugin.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("failed to stop HTTP plugin: %w", err)
	}

	g.running = false

	g.logger.Info("API gateway stopped", corelib.LogKeyComponent, "api_gateway")

	return nil
}

// RegisterHandler registers a handler for a path
func (g *APIGatewayPlugin) RegisterHandler(path string, handler Handler) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.initialized {
		return fmt.Errorf("API gateway not initialized")
	}

	g.handlers[path] = handler
	return nil
}
