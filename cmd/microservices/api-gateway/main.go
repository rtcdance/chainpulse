package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

//nolint:wsl,nlreturn,funlen // Command entrypoint is intentionally verbose.
func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ChainPulse - API Gateway Service                   ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadGatewayConfig()

	// Print configuration
	fmt.Println("Configuration Loaded:")
	fmt.Printf("  Gateway Port:       %d\n", config.Port)
	fmt.Printf("  Instance ID:        %s\n", config.InstanceID)
	fmt.Printf("  TLS Enabled:        %v\n", config.TLSEnabled)
	fmt.Printf("  Rate Limit:         %d req/min\n", config.RateLimitPerMinute)
	fmt.Printf("  Auth Enabled:       %v\n", config.AuthEnabled)
	fmt.Printf("  Rate Limit Enabled: %v\n", config.RateLimitEnabled)
	fmt.Printf("  Upstream Services:  %v\n", config.UpstreamServices)
	fmt.Printf("  Log Level:          %s\n", config.LogLevel)
	fmt.Println()

	// Initialize core services
	fmt.Println("Initializing Core Services:")
	logLevel := core.LogLevelInfo
	if config.LogLevel == "debug" {
		logLevel = core.LogLevelDebug
	}
	logger := core.NewDefaultLogger(logLevel)
	metrics := core.NewDefaultMetricsCollector()
	registry := core.NewPluginRegistry(logger)

	fmt.Println("  [1/3] Logger initialized")
	fmt.Println("  [2/3] Metrics collector initialized")
	fmt.Println("  [3/3] Plugin registry initialized")
	fmt.Println()

	// Convert configuration to core.Config
	coreConfig := &core.Config{
		APIType:      "gateway",
		APIPort:      config.Port,
		LogLevel:     config.LogLevel,
		FeatureFlags: make(map[string]bool),
	}
	runtimeProfile := bootstrap.RuntimeProfileFromEnv()
	if err := validateGatewayProductionSecurity(config, runtimeProfile); err != nil {
		logger.Error("Gateway production security gate rejected startup", "profile", runtimeProfile, "error", err.Error())
		os.Exit(1)
	}

	// Initialize API Gateway Plugin
	fmt.Println("Initializing API Gateway:")
	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetUpstreamQueryEndpoints(config.UpstreamServices)
	if upstreamClient := buildGatewayUpstreamHTTPClient(config); upstreamClient != nil {
		gateway.SetUpstreamQueryHTTPClient(upstreamClient)
		gateway.SetUpstreamQueryHealthHTTPClient(upstreamClient)
	}
	if upstreamHealthHeaders := buildGatewayUpstreamHealthHeaders(config); len(upstreamHealthHeaders) > 0 {
		gateway.SetUpstreamQueryHealthHeaders(upstreamHealthHeaders)
	}
	authMiddleware, rateLimitMiddleware, err := bootstrap.BuildSecurityControls(bootstrap.SecurityControlsConfig{
		AuthEnabled:        config.AuthEnabled,
		AuthJWTSecret:      config.AuthJWTSecret,
		AuthAPIKeys:        config.AuthAPIKeys,
		RateLimitEnabled:   config.RateLimitEnabled,
		RateLimitPerMinute: config.RateLimitPerMinute,
		ServiceName:        "gateway",
		EnvPrefix:          "GATEWAY",
	}, logger, metrics)
	if err != nil {
		logger.Error("Failed to build API Gateway security controls", "error", err.Error())
		os.Exit(1)
	}
	if authMiddleware != nil {
		gateway.SetAuthMiddleware(authMiddleware)
	}
	if rateLimitMiddleware != nil {
		gateway.SetRateLimitMiddleware(rateLimitMiddleware)
	}
	if _, _, _, err := buildAPIGatewayRuntimeRolloutComponents(context.Background(), config.InstanceID, logger, metrics, gateway); err != nil {
		logger.Error("Failed to build API Gateway rollout runtime components", "error", err.Error())
		os.Exit(1)
	}
	gateway.SetRuntimeSummaryProvider(buildAPIGatewayRuntimeSummaryProvider(config.InstanceID, metrics, gateway))
	gateway.SetRuntimeMetricsProvider(buildAPIGatewayMetricsProvider(metrics))
	if err := gateway.Initialize(*coreConfig); err != nil {
		logger.Error("Failed to initialize API Gateway", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ API Gateway initialized")
	fmt.Println("  ✓ Rollout report producer initialized")
	fmt.Println()

	// Register plugins with registry
	fmt.Println("Registering Plugins:")
	if err := registry.Register(gateway); err != nil {
		logger.Error("Failed to register API Gateway", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  [1/1] API Gateway registered")
	fmt.Println()

	// Start all services
	fmt.Println("Starting Services:")
	var wg sync.WaitGroup

	// Start API Gateway
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := gateway.Start(); err != nil {
			logger.Error("API Gateway error", "error", err.Error())
		}
	}()
	fmt.Println("  [1/1] API Gateway started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("API Gateway available at: http://localhost:%d\n", config.Port)
	fmt.Printf("Health Check available at: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Metrics available at: http://localhost:%d/metrics\n", config.Port)
	fmt.Printf("Runtime Summary available at: http://localhost:%d/runtime/summary\n", config.Port)
	fmt.Printf("Instance ID: %s\n", config.InstanceID)
	fmt.Println("Press Ctrl+C to shutdown gracefully")
	fmt.Println()

	// Setup signal handling
	_ = bootstrap.WaitForSignal()

	// Graceful shutdown
	fmt.Println("Shutting Down Services:")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop API Gateway (graceful shutdown with context)
	if err := gateway.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error("Error stopping API Gateway", "error", err.Error())
	}
	fmt.Println("  [1/1] API Gateway stopped")

	bootstrap.ShutdownWithTimeout(&wg, 30*time.Second)

	fmt.Println()
	fmt.Println("Status: Shutdown complete")
	fmt.Println()
}

// GatewayConfig represents the API Gateway configuration
type GatewayConfig struct {
	InstanceID         string
	Port               int
	TLSEnabled         bool
	TLSCertPath        string
	TLSKeyPath         string
	UpstreamServices   []string
	RateLimitPerMinute int
	AuthEnabled        bool
	AuthJWTSecret      core.SecretString
	AuthAPIKeys        []core.SecretString
	RateLimitEnabled   bool
	UpstreamAuthAPIKey core.SecretString
	LogLevel           string
}

// loadGatewayConfig loads configuration from environment variables
func loadGatewayConfig() GatewayConfig {
	instanceID := env.Get("HOSTNAME", "api-gateway-1")
	if id := env.Get("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return GatewayConfig{
		InstanceID:         instanceID,
		Port:               env.GetInt("GATEWAY_PORT", 8080),
		TLSEnabled:         env.GetBool("GATEWAY_TLS_ENABLED", false),
		TLSCertPath:        env.Get("GATEWAY_TLS_CERT", ""),
		TLSKeyPath:         env.Get("GATEWAY_TLS_KEY", ""),
		UpstreamServices:   env.GetCSV("GATEWAY_UPSTREAM_SERVICES", []string{"http://localhost:8081"}),
		RateLimitPerMinute: env.GetInt("GATEWAY_RATE_LIMIT", 1000),
		AuthEnabled:        env.GetBool("GATEWAY_AUTH_ENABLED", false),
		AuthJWTSecret:      core.SecretString(env.Get("GATEWAY_AUTH_JWT_SECRET", "")),
		AuthAPIKeys:        toSecretStrings(env.GetCSV("GATEWAY_AUTH_API_KEYS", nil)),
		RateLimitEnabled:   env.GetBool("GATEWAY_RATE_LIMIT_ENABLED", true),
		UpstreamAuthAPIKey: core.SecretString(env.Get("GATEWAY_UPSTREAM_AUTH_API_KEY", "")),
		LogLevel:           env.Get("LOG_LEVEL", "info"),
	}
}

func buildGatewayUpstreamHTTPClient(config GatewayConfig) *http.Client {
	if strings.TrimSpace(config.UpstreamAuthAPIKey.Value()) == "" {
		return nil
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: gatewayUpstreamAuthTransport{
			apiKey: config.UpstreamAuthAPIKey.Value(),
			base:   http.DefaultTransport,
		},
	}
}

func buildGatewayUpstreamHealthHeaders(config GatewayConfig) map[string]string {
	if strings.TrimSpace(config.UpstreamAuthAPIKey.Value()) == "" {
		return nil
	}

	return map[string]string{
		"X-API-Key": config.UpstreamAuthAPIKey.Value(),
	}
}

type gatewayUpstreamAuthTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t gatewayUpstreamAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	if cloned.Header == nil {
		cloned.Header = make(http.Header)
	}
	cloned.Header.Set("X-API-Key", t.apiKey)

	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(cloned)
}

func validateGatewayProductionSecurity(config GatewayConfig, runtimeProfile string) error {
	if runtimeProfile != "production" {
		return nil
	}

	if !config.AuthEnabled {
		return fmt.Errorf("production gateway requires GATEWAY_AUTH_ENABLED=true")
	}
	if strings.TrimSpace(config.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("production gateway requires non-empty GATEWAY_AUTH_JWT_SECRET")
	}
	if !config.RateLimitEnabled {
		return fmt.Errorf("production gateway requires GATEWAY_RATE_LIMIT_ENABLED=true")
	}
	if config.RateLimitPerMinute <= 0 {
		return fmt.Errorf("production gateway requires GATEWAY_RATE_LIMIT > 0")
	}

	return nil
}

func toSecretStrings(strs []string) []core.SecretString {
	result := make([]core.SecretString, len(strs))
	for i, s := range strs {
		result[i] = core.SecretString(s)
	}
	return result
}

//nolint:wsl,nlreturn // Gateway TLS initialization stays centralized here.
