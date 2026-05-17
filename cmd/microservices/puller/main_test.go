package main

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestLoadPullerConfigDefaultsToSecurityDisabled(t *testing.T) {
	t.Setenv("HOSTNAME", "")
	t.Setenv("INSTANCE_ID", "")
	t.Setenv("PULLER_AUTH_ENABLED", "")
	t.Setenv("PULLER_AUTH_JWT_SECRET", "")
	t.Setenv("PULLER_AUTH_API_KEYS", "")
	t.Setenv("PULLER_RATE_LIMIT_ENABLED", "")

	cfg := loadPullerConfig()
	if cfg.AuthEnabled {
		t.Fatal("expected auth to be disabled by default")
	}
	if !cfg.RateLimitEnabled {
		t.Fatal("expected rate limiting to be enabled by default (secure-by-default)")
	}
	if len(cfg.AuthAPIKeys) != 0 {
		t.Fatalf("expected no auth api keys by default, got %d", len(cfg.AuthAPIKeys))
	}
}

func TestLoadPullerConfigParsesSecurityOverrides(t *testing.T) {
	t.Setenv("PULLER_AUTH_ENABLED", "true")
	t.Setenv("PULLER_AUTH_JWT_SECRET", "secret-123")
	t.Setenv("PULLER_AUTH_API_KEYS", "svc-key=client-1, svc-key-2:client-2")
	t.Setenv("PULLER_RATE_LIMIT_ENABLED", "true")
	t.Setenv("PULLER_RATE_LIMIT", "42")

	cfg := loadPullerConfig()
	if !cfg.AuthEnabled {
		t.Fatal("expected auth to be enabled")
	}
	if got := cfg.AuthJWTSecret; got != "secret-123" {
		t.Fatalf("expected jwt secret to be parsed, got %q", got)
	}
	if len(cfg.AuthAPIKeys) != 2 {
		t.Fatalf("expected 2 api keys, got %d", len(cfg.AuthAPIKeys))
	}
	if !cfg.RateLimitEnabled {
		t.Fatal("expected rate limiting to be enabled")
	}
	if got := cfg.RateLimitPerMinute; got != 42 {
		t.Fatalf("expected rate limit 42, got %d", got)
	}
}

func TestBuildPullerSecurityControlsEnabled(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	authMiddleware, rateLimitMiddleware, err := buildPullerSecurityControls(PullerConfig{
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
