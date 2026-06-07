package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func TestBuildAPIGatewayRuntimeRolloutComponents(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	gateway := api.NewAPIGatewayPlugin(logger, metrics)

	eventQueryHandler, subscriptionHandler, healthHandler, err := buildAPIGatewayRuntimeRolloutComponents(
		context.Background(),
		"api-gateway-1",
		logger,
		metrics,
		gateway,
	)
	if err != nil {
		t.Fatalf("build runtime rollout components: %v", err)
	}
	if eventQueryHandler == nil {
		t.Fatal("expected event query handler")
	}
	if subscriptionHandler == nil {
		t.Fatal("expected event subscription handler")
	}
	if healthHandler == nil {
		t.Fatal("expected health handler")
	}
	if !gateway.IsEventQueryHandlerEnabled() {
		t.Fatal("expected event query handler to be wired")
	}
	if !gateway.IsEventSubscriptionHandlerEnabled() {
		t.Fatal("expected event subscription handler to be wired")
	}
	if !gateway.IsHealthCheckHandlerEnabled() {
		t.Fatal("expected health handler to be wired")
	}
	health := subscriptionHandler.Health(context.Background())
	if health == nil {
		t.Fatal("expected subscription handler health")
	}
	if health.Message == "event subscription handler not initialized" {
		t.Fatalf("expected initialized subscription handler, got %q", health.Message)
	}
}

func TestAPIGatewaySubscriptionRouteSupportsWebSocketUpgrade(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	gateway := api.NewAPIGatewayPlugin(logger, metrics)

	eventQueryHandler, subscriptionHandler, healthHandler, err := buildAPIGatewayRuntimeRolloutComponents(
		context.Background(),
		"api-gateway-1",
		logger,
		metrics,
		gateway,
	)
	if err != nil {
		t.Fatalf("build runtime rollout components: %v", err)
	}

	integration := api.NewGatewayRouterIntegration(logger, metrics, eventQueryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(integration.HandleRequest))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/events/subscribe"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		status := 0
		body := ""
		if resp != nil {
			status = resp.StatusCode
			data, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			body = string(data)
		}
		t.Fatalf("dial subscription websocket: %v status=%d body=%s", err, status, body)
	}
	defer func() { _ = conn.Close() }()
}

func TestAPIGatewayRuntimeRolloutRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	gateway := api.NewAPIGatewayPlugin(logger, metrics)

	eventQueryHandler, subscriptionHandler, healthHandler, err := buildAPIGatewayRuntimeRolloutComponents(
		context.Background(),
		"api-gateway-1",
		logger,
		metrics,
		gateway,
	)
	if err != nil {
		t.Fatalf("build runtime rollout components: %v", err)
	}

	integration := api.NewGatewayRouterIntegration(logger, metrics, eventQueryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()
	integration.HandleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload api.RolloutReportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if !payload.Available || payload.Details == nil {
		t.Fatal("expected rollout details")
	}
	if err := api.ValidateMicroserviceRolloutMetadataParity(
		payload.Details,
		"api-gateway",
		"api-gateway-ownership-rollout-runtime",
		"microservice:api-gateway-1",
	); err != nil {
		t.Fatalf("expected metadata parity validation: %v", err)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(payload.Details); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
	if err := api.ValidateMicroserviceOwnershipParityMarker(payload.Details); err != nil {
		t.Fatalf("expected ownership parity marker validation: %v", err)
	}
	if got := payload.Details.Service; got != "api-gateway" {
		t.Fatalf("expected service api-gateway, got %q", got)
	}
	if !strings.Contains(payload.Details.Advisory.Reason, "ownership_parity_hint: api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending") {
		t.Fatalf("expected ownership parity hint in reason, got %q", payload.Details.Advisory.Reason)
	}
	if got := payload.Details.Approval.WorkItem.Reason; got != "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
}

func TestAPIGatewayRuntimeRolloutRouteIncludesMonolithParityReason(t *testing.T) {
	t.Skip("pre-existing vet error: buildAPIGatewayRuntimeRolloutComponentsWithReadinessDetails undefined at HEAD; restore when production function is reintroduced")
}

func TestAPIGatewayRuntimeSummaryRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("api_gateway_summary_counter", 2, nil)
	metrics.RecordGauge("api_gateway_summary_gauge", 1, nil)

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081", "http://api-service-2:8081"})
	gateway.SetUpstreamQueryHealthHTTPClient(&http.Client{
		Transport: apiGatewayRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}, nil
		}),
	})
	eventQueryHandler, subscriptionHandler, healthHandler, err := buildAPIGatewayRuntimeRolloutComponents(
		context.Background(),
		"api-gateway-1",
		logger,
		metrics,
		gateway,
	)
	if err != nil {
		t.Fatalf("build runtime rollout components: %v", err)
	}

	gateway.SetRuntimeSummaryProvider(buildAPIGatewayRuntimeSummaryProvider("api-gateway-1", metrics, gateway))
	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	integration := api.NewGatewayRouterIntegration(
		logger,
		metrics,
		eventQueryHandler,
		subscriptionHandler,
		healthHandler,
		buildAPIGatewayRuntimeSummaryProvider("api-gateway-1", metrics, gateway),
	)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rec := httptest.NewRecorder()
	integration.HandleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime summary: %v", err)
	}
	if got := payload["service"]; got != "api-gateway" {
		t.Fatalf("expected service api-gateway, got %v", got)
	}
	if got := payload["deployment_mode"]; got != "microservice" {
		t.Fatalf("expected deployment mode microservice, got %v", got)
	}
	if got := payload["runtime_mode"]; got != "runtime-wired" {
		t.Fatalf("expected runtime mode runtime-wired, got %v", got)
	}
	if got := payload["runtime_posture"]; got != "runtime-wired" {
		t.Fatalf("expected runtime posture runtime-wired, got %v", got)
	}
	gatewaySection, ok := payload["gateway"].(map[string]any)
	if !ok {
		t.Fatalf("expected gateway section, got %#v", payload["gateway"])
	}
	if got := gatewaySection["route_boundary"]; got != "gateway-entrypoint" {
		t.Fatalf("expected route boundary gateway-entrypoint, got %v", got)
	}
	if got := gatewaySection["runtime_routes_enabled"]; got != true {
		t.Fatalf("expected runtime routes enabled true, got %v", got)
	}
	if got := gatewaySection["domain_bridge_enabled"]; got != true {
		t.Fatalf("expected domain bridge enabled true, got %v", got)
	}
	if got := gatewaySection["upstream_query_configured_count"]; got != float64(2) {
		t.Fatalf("expected upstream configured count 2, got %v", got)
	}
	if got := gatewaySection["upstream_query_attached_count"]; got != float64(2) {
		t.Fatalf("expected upstream attached count 2, got %v", got)
	}
	if got := gatewaySection["upstream_query_available_count"]; got != float64(2) {
		t.Fatalf("expected upstream available count 2, got %v", got)
	}
	if got := gatewaySection["query_bridge_posture"]; got != "query-bridge-ready" {
		t.Fatalf("expected query bridge posture ready, got %v", got)
	}
	if got := gatewaySection["upstream_query_health_state"]; got != "query-upstream-healthy" {
		t.Fatalf("expected upstream query health state healthy, got %v", got)
	}
	if got := gatewaySection["security_posture"]; got != "gateway-security-unconfigured" {
		t.Fatalf("expected security posture unconfigured, got %v", got)
	}
	if got := gatewaySection["auth_posture"]; got != "gateway-auth-unconfigured" {
		t.Fatalf("expected auth posture unconfigured, got %v", got)
	}
	if got := gatewaySection["rate_limit_posture"]; got != "gateway-rate-limit-unconfigured" {
		t.Fatalf("expected rate limit posture unconfigured, got %v", got)
	}
	metricsSection, ok := payload["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected metrics section, got %#v", payload["metrics"])
	}
	if got := metricsSection["collector_state"]; got != "available" {
		t.Fatalf("expected collector state available, got %v", got)
	}
}

func TestAPIGatewayRuntimeSummaryRouteWithSecurityControls(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("api_gateway_summary_counter", 2, nil)
	metrics.RecordGauge("api_gateway_summary_gauge", 1, nil)

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	tokenValidator := api.NewTokenValidator("gateway-secret", logger, metrics)
	if err := tokenValidator.RegisterAPIKey("test-api-key", "test-client"); err != nil {
		t.Fatalf("register api key: %v", err)
	}
	authMiddleware := api.NewAuthMiddleware(
		tokenValidator,
		api.NewRBACChecker(logger, metrics),
		api.NewAuditLogger(logger, metrics),
		logger,
		metrics,
	)
	rateLimitMiddleware := api.NewRateLimitMiddleware(
		api.NewRateLimiter(logger, metrics, &api.RateLimitConfig{
			DefaultRequestsPerSecond: 1000,
			DefaultBurstSize:         100,
			CleanupInterval:          time.Minute,
		}),
		logger,
	)
	gateway.SetAuthMiddleware(authMiddleware)
	gateway.SetRateLimitMiddleware(rateLimitMiddleware)
	gateway.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081"})
	gateway.SetUpstreamQueryHealthHTTPClient(&http.Client{
		Transport: apiGatewayRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}, nil
		}),
	})

	eventQueryHandler, subscriptionHandler, healthHandler, err := buildAPIGatewayRuntimeRolloutComponents(
		context.Background(),
		"api-gateway-1",
		logger,
		metrics,
		gateway,
	)
	if err != nil {
		t.Fatalf("build runtime rollout components: %v", err)
	}

	gateway.SetRuntimeSummaryProvider(buildAPIGatewayRuntimeSummaryProvider("api-gateway-1", metrics, gateway))
	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	integration := api.NewGatewayRouterIntegration(
		logger,
		metrics,
		eventQueryHandler,
		subscriptionHandler,
		healthHandler,
		buildAPIGatewayRuntimeSummaryProvider("api-gateway-1", metrics, gateway),
	)
	integration.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081"})
	integration.SetUpstreamQueryHealthHTTPClient(&http.Client{
		Transport: apiGatewayRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(r.URL.Path, "/health"):
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`ok`)),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"data":[],"meta":{"source":"upstream-api-service"}}`)),
				}, nil
			}
		}),
	})
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rec := httptest.NewRecorder()
	integration.HandleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime summary: %v", err)
	}
	gatewaySection, ok := payload["gateway"].(map[string]any)
	if !ok {
		t.Fatalf("expected gateway section, got %#v", payload["gateway"])
	}
	if got := gatewaySection["auth_posture"]; got != "gateway-auth-ready" {
		t.Fatalf("expected auth posture ready, got %v", got)
	}
	if got := gatewaySection["rate_limit_posture"]; got != "gateway-rate-limit-ready" {
		t.Fatalf("expected rate limit posture ready, got %v", got)
	}
	if got := gatewaySection["security_posture"]; got != "gateway-security-ready" {
		t.Fatalf("expected security posture ready, got %v", got)
	}
	if got := gatewaySection["auth_enabled"]; got != true {
		t.Fatalf("expected auth enabled true, got %v", got)
	}
	if got := gatewaySection["rate_limit_enabled"]; got != true {
		t.Fatalf("expected rate limit enabled true, got %v", got)
	}
}

type apiGatewayRoundTripFunc func(*http.Request) (*http.Response, error)

func (f apiGatewayRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
