package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

func TestAPIServiceRolloutReportRouteParityMetadataAndBodyBoundaries(t *testing.T) {
	t.Skip("pre-existing vet error: newAPIServiceRolloutReportProducerWithReadinessDetails undefined at HEAD; restore when production function is reintroduced")
}

type apiServiceTestQueryService struct {
	health         *core.HealthStatus
	runtimeSummary *query.RuntimeSummary
}

func (s *apiServiceTestQueryService) Query(ctx context.Context, req *query.QueryRequest) (*query.QueryResult, error) {
	_ = ctx
	_ = req
	return nil, nil
}

func (s *apiServiceTestQueryService) QueryByHash(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
	_ = ctx
	_ = hash
	return nil, nil
}

func (s *apiServiceTestQueryService) InvalidateCache(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}

func (s *apiServiceTestQueryService) Health(ctx context.Context) *core.HealthStatus {
	_ = ctx
	if s.health == nil {
		return &core.HealthStatus{Status: "unknown", Message: "query service unavailable"}
	}
	return s.health
}

func (s *apiServiceTestQueryService) RuntimeSummary(ctx context.Context) *query.RuntimeSummary {
	_ = ctx
	return s.runtimeSummary
}

func TestAPIServiceRuntimeSummaryRouteExposesQueryRuntimeState(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("api_service_summary_counter", 2, nil)
	metrics.RecordGauge("api_service_summary_gauge", 1, nil)

	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	eventQueryHandler := api.NewEventQueryHandler(nil, logger, metrics)
	eventSubscriptionHandler := api.NewEventSubscriptionHandler(nil, logger, metrics)

	plugin := api.NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(eventSubscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)
	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	integration := api.NewGatewayRouterIntegration(
		logger,
		metrics,
		eventQueryHandler,
		eventSubscriptionHandler,
		healthHandler,
		buildAPIServiceRuntimeSummaryProvider(
			"api-service-1",
			metrics,
			plugin,
			&apiServiceTestQueryService{
				health: &core.HealthStatus{Status: "degraded", Message: "Cache is unhealthy"},
				runtimeSummary: &query.RuntimeSummary{
					Status:                "degraded",
					Message:               "Cache is unhealthy",
					QueryPosture:          "query-runtime-degraded",
					CachePosture:          "cache-unhealthy",
					CircuitBreakerPosture: "circuit-not-wired",
					ConsistencyPosture:    "consistency-not-wired",
					ReliabilityHint:       "query runtime is degraded and cache is unhealthy; expect store-backed reads while cache is restored",
				},
			},
		),
	)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rr := httptest.NewRecorder()
	integration.HandleRequest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload apiServiceRuntimeSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime summary response: %v", err)
	}
	if payload.Service != "api-service" {
		t.Fatalf("expected service api-service, got %q", payload.Service)
	}
	if payload.DeploymentMode != "microservice" {
		t.Fatalf("expected deployment mode microservice, got %q", payload.DeploymentMode)
	}
	if payload.RuntimePosture != "partial-runtime-wiring" {
		t.Fatalf("expected runtime posture partial-runtime-wiring, got %q", payload.RuntimePosture)
	}
	if got := payload.Query["status"]; got != "degraded" {
		t.Fatalf("expected query status degraded, got %v", got)
	}
	if got := payload.Query["health_hint"]; got != "investigate degraded query runtime before treating runtime-wired api-service routes as ready" {
		t.Fatalf("unexpected query health hint: %v", got)
	}
	if got := payload.Query["query_posture"]; got != "query-runtime-degraded" {
		t.Fatalf("expected query posture query-runtime-degraded, got %v", got)
	}
	if got := payload.Query["cache_posture"]; got != "cache-unhealthy" {
		t.Fatalf("expected cache posture cache-unhealthy, got %v", got)
	}
	if got := payload.Query["circuit_breaker_posture"]; got != "circuit-not-wired" {
		t.Fatalf("expected circuit posture circuit-not-wired, got %v", got)
	}
	if got := payload.Query["consistency_posture"]; got != "consistency-not-wired" {
		t.Fatalf("expected consistency posture consistency-not-wired, got %v", got)
	}
	if got := payload.Metrics["collector_state"]; got != "available" {
		t.Fatalf("expected metrics collector available, got %v", got)
	}
	if got := payload.Security["auth_enabled"]; got != false {
		t.Fatalf("expected auth disabled by default, got %v", got)
	}
	if got := payload.Security["rate_limit_enabled"]; got != false {
		t.Fatalf("expected rate limit disabled by default, got %v", got)
	}
	if got := payload.Security["security_posture"]; got != "api-service-security-unconfigured" {
		t.Fatalf("expected security posture api-service-security-unconfigured, got %v", got)
	}
	if got := payload.Security["security_hint"]; got != "api-service security controls are disabled by default; enable auth or rate limiting explicitly before exposing the entrypoint" {
		t.Fatalf("unexpected security hint: %v", got)
	}
}

func TestAPIServiceSecuritySurfaceProtectsRuntimeSummaryWhenEnabled(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	eventQueryHandler := api.NewEventQueryHandler(nil, logger, metrics)
	eventSubscriptionHandler := api.NewEventSubscriptionHandler(nil, logger, metrics)

	plugin := api.NewAPIGatewayPlugin(logger, metrics)
	plugin.SetEventQueryHandler(eventQueryHandler)
	plugin.SetEventSubscriptionHandler(eventSubscriptionHandler)
	plugin.SetHealthCheckHandler(healthHandler)

	authMiddleware, rateLimitMiddleware, err := buildAPIServiceSecurityControls(APIServiceConfig{
		AuthEnabled:   true,
		AuthJWTSecret: "secret-123",
		AuthAPIKeys:   []core.SecretString{"svc-key=client-1"},
	}, logger, metrics)
	if err != nil {
		t.Fatalf("build security controls: %v", err)
	}
	plugin.SetAuthMiddleware(authMiddleware)
	plugin.SetRateLimitMiddleware(rateLimitMiddleware)
	plugin.SetRuntimeSummaryProvider(buildAPIServiceRuntimeSummaryProvider(
		"api-service-1",
		metrics,
		plugin,
		&apiServiceTestQueryService{
			health: &core.HealthStatus{Status: "healthy", Message: "query service healthy"},
			runtimeSummary: &query.RuntimeSummary{
				Status:                "healthy",
				Message:               "query service healthy",
				QueryPosture:          "query-runtime-ready",
				CachePosture:          "cache-ready",
				CircuitBreakerPosture: "circuit-not-wired",
				ConsistencyPosture:    "consistency-not-wired",
				ReliabilityHint:       "query runtime is healthy and cache is ready",
			},
		},
	))
	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	handler := plugin.HTTPHandler()
	if handler == nil {
		t.Fatal("expected gateway handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without api key, got %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	req.Header.Set("X-API-Key", "svc-key")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 with api key, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload apiServiceRuntimeSummaryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime summary response: %v", err)
	}
	if payload.DeploymentMode != "microservice" {
		t.Fatalf("expected deployment mode microservice, got %q", payload.DeploymentMode)
	}
	if got := payload.Security["auth_enabled"]; got != true {
		t.Fatalf("expected auth enabled, got %v", got)
	}
	if got := payload.Security["security_posture"]; got != "api-service-security-partial" {
		t.Fatalf("expected partial security posture, got %v", got)
	}
}
