package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"chainpulse/pkg/core"
)

func TestLoadGatewayConfigDefaultsToLocalhostUpstream(t *testing.T) {
	t.Setenv("HOSTNAME", "")
	t.Setenv("INSTANCE_ID", "")
	t.Setenv("GATEWAY_UPSTREAM_SERVICES", "")

	cfg := loadGatewayConfig()
	if len(cfg.UpstreamServices) != 1 {
		t.Fatalf("expected 1 default upstream, got %d", len(cfg.UpstreamServices))
	}
	if got := cfg.UpstreamServices[0]; got != "http://localhost:8081" {
		t.Fatalf("expected localhost upstream default, got %q", got)
	}
}

func TestLoadGatewayConfigParsesUpstreamServicesOverride(t *testing.T) {
	t.Setenv("GATEWAY_UPSTREAM_SERVICES", "http://api-service-1:8081, http://api-service-2:8081 ,,http://api-service-3:8081")

	cfg := loadGatewayConfig()
	if len(cfg.UpstreamServices) != 3 {
		t.Fatalf("expected 3 upstreams, got %d", len(cfg.UpstreamServices))
	}
	if got := cfg.UpstreamServices[1]; got != "http://api-service-2:8081" {
		t.Fatalf("expected second upstream to be trimmed, got %q", got)
	}
}

func TestLoadGatewayConfigParsesRateLimitPerMinute(t *testing.T) {
	t.Setenv("GATEWAY_RATE_LIMIT_ENABLED", "true")
	t.Setenv("GATEWAY_RATE_LIMIT", "42")

	cfg := loadGatewayConfig()
	if !cfg.RateLimitEnabled {
		t.Fatal("expected rate limiting to be enabled")
	}
	if got := cfg.RateLimitPerMinute; got != 42 {
		t.Fatalf("expected rate limit 42 req/min, got %d", got)
	}
}

func TestBuildAPIGatewaySecurityControlsEnabled(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	authMiddleware, rateLimitMiddleware, err := buildAPIGatewaySecurityControls(GatewayConfig{
		AuthEnabled:        true,
		AuthJWTSecret:      "secret-123",
		AuthAPIKeys:        []string{"svc-key=client-1"},
		RateLimitEnabled:   true,
		RateLimitPerMinute: 120,
	}, logger, metrics)
	if err != nil {
		t.Fatalf("build security controls: %v", err)
	}
	if authMiddleware == nil {
		t.Fatal("expected auth middleware to be created")
	}
	if rateLimitMiddleware == nil {
		t.Fatal("expected rate limit middleware to be created")
	}
}

func TestBuildAPIGatewayMetricsProviderDefaultsToPrometheus(t *testing.T) {
	metrics := core.NewDefaultMetricsCollector()
	metrics.RecordCounter("gateway_test_counter", 1, nil)

	provider := buildAPIGatewayMetricsProvider(metrics)
	payload, ok := provider(httptest.NewRequest("GET", "/metrics", nil)).(string)
	if !ok {
		t.Fatalf("expected prometheus text payload, got %T", payload)
	}
	if !strings.Contains(payload, `chainpulse_gateway_test_counter{chain_id="global"} 1`) {
		t.Fatalf("expected prometheus payload to contain counter, got:\n%s", payload)
	}
}
