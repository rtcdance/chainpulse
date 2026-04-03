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

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

func TestLocalRunnableGatewayQuerySmoke(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("gateway_smoke_counter", 1, nil)

	upstreamTransport := smokeRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/health"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}, nil
		case r.URL.String() == "http://localhost:8081/events?limit=5":
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":[],"meta":{"source":"upstream-api-service"}}`)),
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`not found`)),
			}, nil
		}
	})

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetUpstreamQueryEndpoints([]string{"http://localhost:8081"})
	gateway.SetUpstreamQueryHTTPClient(&http.Client{Transport: upstreamTransport})
	gateway.SetUpstreamQueryHealthHTTPClient(&http.Client{Transport: upstreamTransport})

	_, _, _, err := buildAPIGatewayRuntimeRolloutComponents(
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

	handler := gateway.HTTPHandler()
	if handler == nil {
		t.Fatal("expected initialized gateway handler")
	}
	routerIntegration := gateway.GetRouterIntegration()
	if routerIntegration == nil {
		t.Fatal("expected initialized gateway router integration")
	}
	routerIntegration.GetRouter().SetHTTPClient(&http.Client{Transport: upstreamTransport})

	summaryReq := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	summaryReq.Header.Set("X-API-Key", "test-api-key")
	summaryRec := httptest.NewRecorder()
	handler(summaryRec, summaryReq)

	if summaryRec.Code != http.StatusOK {
		t.Fatalf("expected summary status 200, got %d body=%s", summaryRec.Code, summaryRec.Body.String())
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(summaryRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode runtime summary: %v", err)
	}
	gatewaySection, ok := summary["gateway"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gateway section, got %#v", summary["gateway"])
	}
	if got := gatewaySection["query_bridge_posture"]; got != "query-bridge-ready" {
		t.Fatalf("expected query bridge ready, got %v", got)
	}
	if got := gatewaySection["upstream_query_health_state"]; got != "query-upstream-healthy" {
		t.Fatalf("expected query upstream healthy, got %v", got)
	}

	queryReq := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	queryRec := httptest.NewRecorder()
	handler(queryRec, queryReq)

	if queryRec.Code != http.StatusOK {
		t.Fatalf("expected query status 200, got %d body=%s", queryRec.Code, queryRec.Body.String())
	}
	if got := queryRec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected json response, got %q", got)
	}
	if got := queryRec.Body.String(); got != `{"data":[],"meta":{"source":"upstream-api-service"}}` {
		t.Fatalf("unexpected forwarded body %q", got)
	}
}

func TestLocalRunnableGatewayQuerySmokeWithSecurityControls(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("gateway_smoke_counter", 1, nil)

	upstreamTransport := smokeRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/health"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}, nil
		case r.URL.String() == "http://localhost:8081/events?limit=5":
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":[],"meta":{"source":"upstream-api-service"}}`)),
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`not found`)),
			}, nil
		}
	})

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetUpstreamQueryEndpoints([]string{"http://localhost:8081"})
	gateway.SetUpstreamQueryHTTPClient(&http.Client{Transport: upstreamTransport})
	gateway.SetUpstreamQueryHealthHTTPClient(&http.Client{Transport: upstreamTransport})

	tokenValidator := api.NewTokenValidator("gateway-secret", logger, metrics)
	if err := tokenValidator.RegisterAPIKey("test-api-key", "test-client"); err != nil {
		t.Fatalf("register api key: %v", err)
	}
	gateway.SetAuthMiddleware(api.NewAuthMiddleware(
		tokenValidator,
		api.NewRBACChecker(logger, metrics),
		api.NewAuditLogger(logger, metrics),
		logger,
		metrics,
	))
	gateway.SetRateLimitMiddleware(api.NewRateLimitMiddleware(
		api.NewRateLimiter(logger, metrics, &api.RateLimitConfig{
			DefaultRequestsPerSecond: 1000,
			DefaultBurstSize:         100,
			CleanupInterval:          time.Minute,
		}),
		logger,
	))

	_, _, _, err := buildAPIGatewayRuntimeRolloutComponents(
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

	handler := gateway.HTTPHandler()
	if handler == nil {
		t.Fatal("expected initialized gateway handler")
	}
	routerIntegration := gateway.GetRouterIntegration()
	if routerIntegration == nil {
		t.Fatal("expected initialized gateway router integration")
	}
	routerIntegration.GetRouter().SetHTTPClient(&http.Client{Transport: upstreamTransport})

	unauthReq := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	unauthRec := httptest.NewRecorder()
	handler(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized response, got %d body=%s", unauthRec.Code, unauthRec.Body.String())
	}

	authReq := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	authReq.Header.Set("X-API-Key", "test-api-key")
	authRec := httptest.NewRecorder()
	handler(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("expected authorized query to pass, got %d body=%s", authRec.Code, authRec.Body.String())
	}

	summaryReq := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	summaryReq.Header.Set("X-API-Key", "test-api-key")
	summaryRec := httptest.NewRecorder()
	handler(summaryRec, summaryReq)
	if summaryRec.Code != http.StatusOK {
		t.Fatalf("expected summary status 200, got %d body=%s", summaryRec.Code, summaryRec.Body.String())
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(summaryRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode runtime summary: %v", err)
	}
	gatewaySection, ok := summary["gateway"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gateway section, got %#v", summary["gateway"])
	}
	if got := gatewaySection["security_posture"]; got != "gateway-security-ready" {
		t.Fatalf("expected security ready, got %v", got)
	}
}

type smokeRoundTripFunc func(*http.Request) (*http.Response, error)

func (f smokeRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
