package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestAPIGatewayPluginDomainBridgeToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsDomainBridgeEnabled() {
		t.Fatal("expected domain bridge to be disabled by default")
	}

	plugin.SetDomainQueryService(nil)
	if plugin.IsDomainBridgeEnabled() {
		t.Fatal("expected domain bridge to remain disabled when service is nil")
	}
}

func TestAPIGatewayPluginDomainBridgeEnabledByUpstreamQueryEndpoints(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	plugin.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081"})
	if !plugin.IsDomainBridgeEnabled() {
		t.Fatal("expected upstream query endpoints to satisfy domain bridge signal")
	}
}

func TestAPIGatewayPluginEventQueryHandlerToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsEventQueryHandlerEnabled() {
		t.Fatal("expected event query handler to be disabled by default")
	}

	plugin.SetEventQueryHandler(nil)
	if plugin.IsEventQueryHandlerEnabled() {
		t.Fatal("expected event query handler to remain disabled when handler is nil")
	}
}

func TestAPIGatewayPluginRuntimeRoutesEnabledWhenHandlersWired(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	plugin.SetEventQueryHandler(NewEventQueryHandler(nil, logger, metrics))
	plugin.SetEventSubscriptionHandler(NewEventSubscriptionHandler(nil, logger, metrics))
	plugin.SetHealthCheckHandler(NewHealthCheckHandler(nil, logger, metrics))

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	if !plugin.IsRuntimeRoutesEnabled() {
		t.Fatal("expected runtime routes to be enabled when handlers are wired")
	}
}

func TestAPIGatewayPluginEventSubscriptionHandlerToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsEventSubscriptionHandlerEnabled() {
		t.Fatal("expected event subscription handler to be disabled by default")
	}

	plugin.SetEventSubscriptionHandler(nil)
	if plugin.IsEventSubscriptionHandlerEnabled() {
		t.Fatal("expected event subscription handler to remain disabled when handler is nil")
	}
}

func TestAPIGatewayPluginInitializeInjectsRateLimiterIntoSubscriptionHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	limiter := NewRateLimiter(logger, metrics, &RateLimitConfig{
		DefaultRequestsPerSecond: 10,
		DefaultBurstSize:         10,
	})

	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetRateLimitMiddleware(NewRateLimitMiddleware(limiter, logger))

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	if subscriptionHandler.rateLimiter != limiter {
		t.Fatal("expected gateway initialization to inject the shared rate limiter into subscription handler")
	}
}

func TestAPIGatewayPluginHealthCheckHandlerToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsHealthCheckHandlerEnabled() {
		t.Fatal("expected health check handler to be disabled by default")
	}

	plugin.SetHealthCheckHandler(nil)
	if plugin.IsHealthCheckHandlerEnabled() {
		t.Fatal("expected health check handler to remain disabled when handler is nil")
	}
}

func TestAPIGatewayPlugin_NameAndVersion(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)
	if plugin.Name() != "api-gateway" {
		t.Fatalf("expected name api-gateway, got %s", plugin.Name())
	}
	if plugin.Version() != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", plugin.Version())
	}
}

func TestAPIGatewayPlugin_SetAuthMiddleware(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsAuthMiddlewareEnabled() {
		t.Fatal("expected auth middleware disabled by default")
	}
	mw := &AuthMiddleware{}
	plugin.SetAuthMiddleware(mw)
	if !plugin.IsAuthMiddlewareEnabled() {
		t.Fatal("expected auth middleware enabled after set")
	}
}

func TestAPIGatewayPlugin_SetRateLimitMiddleware(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsRateLimitMiddlewareEnabled() {
		t.Fatal("expected rate limit middleware disabled by default")
	}
	mw := &RateLimitMiddleware{}
	plugin.SetRateLimitMiddleware(mw)
	if !plugin.IsRateLimitMiddlewareEnabled() {
		t.Fatal("expected rate limit middleware enabled after set")
	}
}

func TestAPIGatewayPlugin_SetCORSMiddleware(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	mw := &CORSMiddleware{}
	plugin.SetCORSMiddleware(mw)
	if plugin.corsMiddleware != mw {
		t.Fatal("expected CORS middleware to be set")
	}
}

func TestAPIGatewayPlugin_SetGraphQLHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &GraphQLHandler{}
	plugin.SetGraphQLHandler(h)
	if plugin.graphqlHandler != h {
		t.Fatal("expected GraphQL handler to be set")
	}
}

func TestAPIGatewayPlugin_SetDLQHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &DLQHandler{}
	plugin.SetDLQHandler(h)
	if plugin.dlqHandler != h {
		t.Fatal("expected DLQ handler to be set")
	}
}

func TestAPIGatewayPlugin_SetExportHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &ExportHandler{}
	plugin.SetExportHandler(h)
	if plugin.exportHandler != h {
		t.Fatal("expected export handler to be set")
	}
}

func TestAPIGatewayPlugin_SetStatsHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &StatsHandler{}
	plugin.SetStatsHandler(h)
	if plugin.statsHandler != h {
		t.Fatal("expected stats handler to be set")
	}
}

func TestAPIGatewayPlugin_SetAdminKeyHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &AdminKeyHandler{}
	plugin.SetAdminKeyHandler(h)
	if plugin.adminKeyHandler != h {
		t.Fatal("expected admin key handler to be set")
	}
}

func TestAPIGatewayPlugin_SetAdminAPIKeyHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &AdminAPIKeyHandler{}
	plugin.SetAdminAPIKeyHandler(h)
	if plugin.adminAPIKeyHandler != h {
		t.Fatal("expected admin API key handler to be set")
	}
}

func TestAPIGatewayPlugin_SetSIWEHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := &SIWEHandler{}
	plugin.SetSIWEHandler(h)
	if plugin.siweHandler != h {
		t.Fatal("expected SIWE handler to be set")
	}
}

func TestAPIGatewayPlugin_SetRuntimeSummaryProvider(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	provider := func(r *http.Request) any { return map[string]string{"test": "ok"} }
	plugin.SetRuntimeSummaryProvider(provider)
	if plugin.runtimeSummaryProvider == nil {
		t.Fatal("expected runtime summary provider to be set")
	}
}

func TestAPIGatewayPlugin_SetRuntimeMetricsProvider(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsMetricsRouteEnabled() {
		t.Fatal("expected metrics route disabled by default")
	}
	provider := func(r *http.Request) any { return map[string]string{} }
	plugin.SetRuntimeMetricsProvider(provider)
	if !plugin.IsMetricsRouteEnabled() {
		t.Fatal("expected metrics route enabled after set")
	}
}

func TestAPIGatewayPlugin_SetRuntimeControlProvider(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	provider := func(w http.ResponseWriter, r *http.Request) {}
	plugin.SetRuntimeControlProvider(provider)
	if plugin.runtimeControlProvider == nil {
		t.Fatal("expected runtime control provider to be set")
	}
}

func TestAPIGatewayPlugin_SetRuntimeReplayProvider(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	provider := func(w http.ResponseWriter, r *http.Request) {}
	plugin.SetRuntimeReplayProvider(provider)
	if plugin.runtimeReplayProvider == nil {
		t.Fatal("expected runtime replay provider to be set")
	}
}

func TestAPIGatewayPlugin_SetUpstreamQueryHTTPClient(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	client := &http.Client{}
	plugin.SetUpstreamQueryHTTPClient(client)
	if plugin.upstreamQueryHTTPClient != client {
		t.Fatal("expected upstream query HTTP client to be set")
	}
}

func TestAPIGatewayPlugin_SetUpstreamQueryHealthHTTPClient(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	client := &http.Client{}
	plugin.SetUpstreamQueryHealthHTTPClient(client)
	if plugin.upstreamQueryHealthHTTPClient != client {
		t.Fatal("expected upstream query health HTTP client to be set")
	}
}

func TestAPIGatewayPlugin_SetUpstreamQueryHealthHeaders(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	headers := map[string]string{"X-Custom": "value"}
	plugin.SetUpstreamQueryHealthHeaders(headers)
	if plugin.upstreamQueryHealthHeaders["X-Custom"] != "value" {
		t.Fatal("expected upstream query health headers to be set")
	}

	plugin.SetUpstreamQueryHealthHeaders(nil)
	if plugin.upstreamQueryHealthHeaders != nil {
		t.Fatal("expected upstream query health headers to be nil")
	}
}

func TestAPIGatewayPlugin_GetUpstreamQueryEndpoints(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	endpoints := []string{"http://api-1:8081", "http://api-2:8081"}
	plugin.SetUpstreamQueryEndpoints(endpoints)
	got := plugin.GetUpstreamQueryEndpoints()
	if len(got) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(got))
	}
}

func TestAPIGatewayPlugin_HTTPHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	h := plugin.HTTPHandler()
	if h != nil {
		t.Fatal("expected nil HTTP handler before initialization")
	}
}

func TestAPIGatewayPlugin_GetRouterIntegration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	ri := plugin.GetRouterIntegration()
	if ri != nil {
		t.Fatal("expected nil router integration before initialization")
	}
}

func TestAPIGatewayPlugin_RefreshUpstreamQueryBridgeHealth(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	plugin.RefreshUpstreamQueryBridgeHealth()
}

func TestAPIGatewayPlugin_SetREDRecorder(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.redRecorder != nil {
		t.Fatal("expected red recorder to be nil by default")
	}
	plugin.SetREDRecorder(nil)
}

func TestAPIGatewayPlugin_GetUpstreamQueryBridgeStatus_NoIntegration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	configured, attached, available := plugin.GetUpstreamQueryBridgeStatus()
	if configured != 0 {
		t.Fatalf("expected 0 configured, got %d", configured)
	}
	if attached != 0 {
		t.Fatalf("expected 0 attached, got %d", attached)
	}
	if available != 0 {
		t.Fatalf("expected 0 available, got %d", available)
	}

	plugin.SetUpstreamQueryEndpoints([]string{"http://api:8081"})
	configured, attached, available = plugin.GetUpstreamQueryBridgeStatus()
	if configured != 1 {
		t.Fatalf("expected 1 configured, got %d", configured)
	}
	if attached != 0 {
		t.Fatalf("expected 0 attached, got %d", attached)
	}
}

func TestAPIGatewayPlugin_GetRuntimeRouteInventory_NoIntegration(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	inventory := plugin.GetRuntimeRouteInventory()
	if (inventory != GatewayRuntimeRouteInventory{}) {
		t.Fatal("expected empty route inventory")
	}
}

func TestAPIGatewayPlugin_Health_NotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	err := plugin.Health()
	if err == nil {
		t.Fatal("expected health error when not running")
	}
}

func TestAPIGatewayPlugin_Start_NotInitialized(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	err := plugin.Start()
	if err == nil {
		t.Fatal("expected start error when not initialized")
	}
}

func TestAPIGatewayPlugin_ShutdownWithContext_NotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	err := plugin.ShutdownWithContext(context.Background())
	if err == nil {
		t.Fatal("expected shutdown error when not running")
	}
}

func TestAPIGatewayPlugin_RegisterHandler_NotInitialized(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	err := plugin.RegisterHandler("/test", nil)
	if err == nil {
		t.Fatal("expected register handler error when not initialized")
	}
}

func TestAPIGatewayPlugin_Stop_NotRunning(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	err := plugin.Stop()
	if err == nil {
		t.Fatal("expected stop error when not running")
	}
}
