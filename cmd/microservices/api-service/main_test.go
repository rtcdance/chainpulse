package main

import (
	"testing"

	"chainpulse/pkg/core"
)

func TestLoadAPIServiceConfigDefaultsToSecurityDisabled(t *testing.T) {
	t.Setenv("HOSTNAME", "")
	t.Setenv("INSTANCE_ID", "")
	t.Setenv("API_SERVICE_AUTH_ENABLED", "")
	t.Setenv("API_SERVICE_AUTH_JWT_SECRET", "")
	t.Setenv("API_SERVICE_AUTH_API_KEYS", "")
	t.Setenv("API_SERVICE_RATE_LIMIT_ENABLED", "")

	cfg := loadAPIServiceConfig()
	if cfg.AuthEnabled {
		t.Fatal("expected auth to be disabled by default")
	}
	if cfg.RateLimitEnabled {
		t.Fatal("expected rate limiting to be disabled by default")
	}
	if len(cfg.AuthAPIKeys) != 0 {
		t.Fatalf("expected no auth api keys by default, got %d", len(cfg.AuthAPIKeys))
	}
}

func TestLoadAPIServiceConfigParsesSecurityOverrides(t *testing.T) {
	t.Setenv("API_SERVICE_AUTH_ENABLED", "true")
	t.Setenv("API_SERVICE_AUTH_JWT_SECRET", "secret-123")
	t.Setenv("API_SERVICE_AUTH_API_KEYS", "svc-key=client-1, svc-key-2:client-2")
	t.Setenv("API_SERVICE_RATE_LIMIT_ENABLED", "true")
	t.Setenv("API_SERVICE_RATE_LIMIT", "42")

	cfg := loadAPIServiceConfig()
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
	if got := cfg.RateLimitPerSecond; got != 42 {
		t.Fatalf("expected rate limit 42, got %d", got)
	}
}

func TestBuildAPIServiceSecurityControlsEnabled(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	authMiddleware, rateLimitMiddleware, err := buildAPIServiceSecurityControls(APIServiceConfig{
		AuthEnabled:        true,
		AuthJWTSecret:      "secret-123",
		AuthAPIKeys:        []string{"svc-key=client-1"},
		RateLimitEnabled:   true,
		RateLimitPerSecond: 25,
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
