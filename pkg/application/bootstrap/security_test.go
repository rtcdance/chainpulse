package bootstrap

import (
	"testing"

	"chainpulse/pkg/core"
)

func TestBuildSecurityControls_BothDisabled(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:      false,
		RateLimitEnabled: false,
	}

	auth, rate, err := BuildSecurityControls(cfg, logger, metrics)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if auth != nil {
		t.Fatal("expected nil auth middleware when both disabled")
	}
	if rate != nil {
		t.Fatal("expected nil rate limit middleware when both disabled")
	}
}

func TestBuildSecurityControls_AuthEmptyJWTSecret(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:   true,
		AuthJWTSecret: "",
		ServiceName:   "test-svc",
		EnvPrefix:     "TEST_SVC",
	}

	_, _, err := BuildSecurityControls(cfg, logger, metrics)
	if err == nil {
		t.Fatal("expected error for empty JWT secret")
	}
}

func TestBuildSecurityControls_AuthEmptyJWTSecretWhitespace(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:   true,
		AuthJWTSecret: "   ",
		ServiceName:   "test-svc",
		EnvPrefix:     "TEST_SVC",
	}

	_, _, err := BuildSecurityControls(cfg, logger, metrics)
	if err == nil {
		t.Fatal("expected error for whitespace-only JWT secret")
	}
}

func TestBuildSecurityControls_InvalidAPIKeyFormat(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:   true,
		AuthJWTSecret: "secret123",
		AuthAPIKeys:   []string{"invalid-format-no-separator"},
		ServiceName:   "test-svc",
		EnvPrefix:     "TEST_SVC",
	}

	_, _, err := BuildSecurityControls(cfg, logger, metrics)
	if err == nil {
		t.Fatal("expected error for invalid API key format")
	}
}

func TestBuildSecurityControls_AuthOnly(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:      true,
		AuthJWTSecret:    "test-secret-key-min-32-chars-long!",
		AuthAPIKeys:      []string{"key-1=client-1"},
		RateLimitEnabled: false,
		ServiceName:      "test-svc",
		EnvPrefix:        "TEST_SVC",
	}

	auth, rate, err := BuildSecurityControls(cfg, logger, metrics)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if auth == nil {
		t.Fatal("expected non-nil auth middleware")
	}
	if rate != nil {
		t.Fatal("expected nil rate limit middleware when disabled")
	}
}

func TestBuildSecurityControls_RateLimitOnly(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:        false,
		RateLimitEnabled:   true,
		RateLimitPerMinute: 100,
		ServiceName:        "test-svc",
		EnvPrefix:          "TEST_SVC",
	}

	auth, rate, err := BuildSecurityControls(cfg, logger, metrics)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if auth != nil {
		t.Fatal("expected nil auth middleware when disabled")
	}
	if rate == nil {
		t.Fatal("expected non-nil rate limit middleware")
	}
}

func TestBuildSecurityControls_BothEnabled(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	cfg := SecurityControlsConfig{
		AuthEnabled:        true,
		AuthJWTSecret:      "test-secret-key-min-32-chars-long!",
		AuthAPIKeys:        []string{"key-1=client-1", "key-2=client-2"},
		RateLimitEnabled:   true,
		RateLimitPerMinute: 100,
		ServiceName:        "test-svc",
		EnvPrefix:          "TEST_SVC",
	}

	auth, rate, err := BuildSecurityControls(cfg, logger, metrics)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if auth == nil {
		t.Fatal("expected non-nil auth middleware")
	}
	if rate == nil {
		t.Fatal("expected non-nil rate limit middleware")
	}
}
