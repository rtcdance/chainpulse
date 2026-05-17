package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

func TestMonolithicRuntimeMetricsRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("test_counter", 2, map[string]string{"service": "monolithic"})
	metrics.RecordGauge("test_gauge", 7, map[string]string{"service": "monolithic"})

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetEventQueryHandler(api.NewEventQueryHandler(nil, logger, metrics))
	gateway.SetEventSubscriptionHandler(api.NewEventSubscriptionHandler(nil, logger, metrics))
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	gateway.SetHealthCheckHandler(healthHandler)
	gateway.SetRuntimeMetricsProvider(buildMonolithicMetricsProvider(metrics, nil))
	gateway.SetRuntimeSummaryProvider(func(r *http.Request) any {
		return map[string]any{"service": "monolithic"}
	})

	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	gateway.HTTPHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain; version=0.0.4" {
		t.Fatalf("expected prometheus content type, got %q", got)
	}
	body := rr.Body.String()
	for _, expected := range []string{
		`# TYPE chainpulse_test_counter counter`,
		`chainpulse_test_counter{chain_id="global",service="monolithic"} 2`,
		`# TYPE chainpulse_test_gauge gauge`,
		`chainpulse_test_gauge{chain_id="global",service="monolithic"} 7`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", expected, body)
		}
	}
}
