package main

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
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

func buildAPIGatewaySecurityControls(cfg GatewayConfig, logger core.Logger, metrics core.MetricsCollector) (*api.AuthMiddleware, *api.RateLimitMiddleware, error) {
	return bootstrap.BuildSecurityControls(bootstrap.SecurityControlsConfig{
		AuthEnabled:        cfg.AuthEnabled,
		AuthJWTSecret:      cfg.AuthJWTSecret,
		AuthAPIKeys:        cfg.AuthAPIKeys,
		RateLimitEnabled:   cfg.RateLimitEnabled,
		RateLimitPerMinute: cfg.RateLimitPerMinute,
		ServiceName:        "api-gateway",
		EnvPrefix:          "GATEWAY",
	}, logger, metrics)
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
		AuthAPIKeys: []core.SecretString{core.SecretString("svc-key=client-1")},
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

func TestValidateGatewayProductionSecurity(t *testing.T) {
	base := GatewayConfig{
		AuthEnabled:        true,
		AuthJWTSecret:      "secret-123",
		RateLimitEnabled:   true,
		RateLimitPerMinute: 120,
	}

	testCases := []struct {
		name    string
		profile string
		cfg     GatewayConfig
		wantErr string
	}{
		{
			name:    "non production stays open",
			profile: "development",
			cfg:     GatewayConfig{},
		},
		{
			name:    "production happy path",
			profile: "production",
			cfg:     base,
		},
		{
			name:    "production requires auth",
			profile: "production",
			cfg: GatewayConfig{
				AuthEnabled:        false,
				RateLimitEnabled:   true,
				RateLimitPerMinute: 120,
			},
			wantErr: "GATEWAY_AUTH_ENABLED=true",
		},
		{
			name:    "production requires jwt secret",
			profile: "production",
			cfg: GatewayConfig{
				AuthEnabled:        true,
				RateLimitEnabled:   true,
				RateLimitPerMinute: 120,
			},
			wantErr: "GATEWAY_AUTH_JWT_SECRET",
		},
		{
			name:    "production requires rate limit",
			profile: "production",
			cfg: GatewayConfig{
				AuthEnabled:   true,
				AuthJWTSecret: "secret-123",
			},
			wantErr: "GATEWAY_RATE_LIMIT_ENABLED=true",
		},
		{
			name:    "production requires positive rate limit",
			profile: "production",
			cfg: GatewayConfig{
				AuthEnabled:        true,
				AuthJWTSecret:      "secret-123",
				RateLimitEnabled:   true,
				RateLimitPerMinute: 0,
			},
			wantErr: "GATEWAY_RATE_LIMIT > 0",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateGatewayProductionSecurity(testCase.cfg, testCase.profile)
			if testCase.wantErr == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if testCase.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", testCase.wantErr)
				}
				if !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("expected error containing %q, got %v", testCase.wantErr, err)
				}
			}
		})
	}
}
