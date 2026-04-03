package main

import "testing"

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
