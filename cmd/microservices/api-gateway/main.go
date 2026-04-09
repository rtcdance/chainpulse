package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/application/bootstrap"
	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

//nolint:wsl,nlreturn // Command entrypoint is intentionally verbose.
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
	authMiddleware, rateLimitMiddleware, err := buildAPIGatewaySecurityControls(config, logger, metrics)
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Println()
	fmt.Printf("Received signal: %v\n", sig)
	fmt.Println()

	// Graceful shutdown
	fmt.Println("Shutting Down Services:")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop API Gateway
	if err := gateway.Stop(); err != nil {
		logger.Error("Error stopping API Gateway", "error", err.Error())
	}
	fmt.Println("  [1/1] API Gateway stopped")

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println()
		fmt.Println("✓ All services stopped successfully")
	case <-shutdownCtx.Done():
		fmt.Println()
		fmt.Println("⚠ Shutdown timeout exceeded")
	}

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
	AuthJWTSecret      string
	AuthAPIKeys        []string
	RateLimitEnabled   bool
	LogLevel           string
}

// loadGatewayConfig loads configuration from environment variables
func loadGatewayConfig() GatewayConfig {
	instanceID := getEnv("HOSTNAME", "api-gateway-1")
	if id := getEnv("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return GatewayConfig{
		InstanceID:         instanceID,
		Port:               getEnvInt("GATEWAY_PORT", 8080),
		TLSEnabled:         getEnvBool("GATEWAY_TLS_ENABLED", false),
		TLSCertPath:        getEnv("GATEWAY_TLS_CERT", ""),
		TLSKeyPath:         getEnv("GATEWAY_TLS_KEY", ""),
		UpstreamServices:   getEnvCSV("GATEWAY_UPSTREAM_SERVICES", []string{"http://localhost:8081"}),
		RateLimitPerMinute: getEnvInt("GATEWAY_RATE_LIMIT", 1000),
		AuthEnabled:        getEnvBool("GATEWAY_AUTH_ENABLED", false),
		AuthJWTSecret:      getEnv("GATEWAY_AUTH_JWT_SECRET", ""),
		AuthAPIKeys:        getEnvCSV("GATEWAY_AUTH_API_KEYS", nil),
		RateLimitEnabled:   getEnvBool("GATEWAY_RATE_LIMIT_ENABLED", false),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := parseIntSafe(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool gets an environment variable as boolean with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

//nolint:wsl,nlreturn // Command config parsing is intentionally dense.
func getEnvCSV(key string, defaultValues []string) []string {
	value := os.Getenv(key)
	if value == "" {
		result := make([]string, len(defaultValues))
		copy(result, defaultValues)
		return result
	}

	result := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		result = append(result, defaultValues...)
	}
	return result
}

//nolint:wsl,nlreturn // Security wiring stays centralized in the command entrypoint.
func buildAPIGatewaySecurityControls(config GatewayConfig, logger core.Logger, metrics core.MetricsCollector) (*api.AuthMiddleware, *api.RateLimitMiddleware, error) {
	if !config.AuthEnabled && !config.RateLimitEnabled {
		return nil, nil, nil
	}

	var authMiddleware *api.AuthMiddleware
	if config.AuthEnabled {
		if strings.TrimSpace(config.AuthJWTSecret) == "" {
			return nil, nil, fmt.Errorf("gateway auth is enabled but GATEWAY_AUTH_JWT_SECRET is empty")
		}

		tokenValidator := api.NewTokenValidator(config.AuthJWTSecret, logger, metrics)
		for _, entry := range config.AuthAPIKeys {
			apiKey, clientID, ok := parseKeyValuePair(entry)
			if !ok {
				return nil, nil, fmt.Errorf("invalid GATEWAY_AUTH_API_KEYS entry %q; expected key=clientID or key:clientID", entry)
			}
			if err := tokenValidator.RegisterAPIKey(apiKey, clientID); err != nil {
				return nil, nil, err
			}
		}

		rbacChecker := api.NewRBACChecker(logger, metrics)
		auditLogger := api.NewAuditLogger(logger, metrics)
		authMiddleware = api.NewAuthMiddleware(tokenValidator, rbacChecker, auditLogger, logger, metrics)
	}

	var rateLimitMiddleware *api.RateLimitMiddleware
	if config.RateLimitEnabled {
		rateLimiter := api.NewRateLimiter(logger, metrics, &api.RateLimitConfig{
			DefaultRequestsPerSecond: api.RequestsPerMinuteToPerSecond(config.RateLimitPerMinute),
			DefaultBurstSize:         api.BurstSizeFromRequestsPerMinute(config.RateLimitPerMinute),
			CleanupInterval:          5 * time.Minute,
		})
		rateLimitMiddleware = api.NewRateLimitMiddleware(rateLimiter, logger)
	}

	return authMiddleware, rateLimitMiddleware, nil
}

func validateGatewayProductionSecurity(config GatewayConfig, runtimeProfile string) error {
	if runtimeProfile != "production" {
		return nil
	}

	if !config.AuthEnabled {
		return fmt.Errorf("production gateway requires GATEWAY_AUTH_ENABLED=true")
	}
	if strings.TrimSpace(config.AuthJWTSecret) == "" {
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

//nolint:wsl,nlreturn // Command config parsing stays centralized and compact here.
func parseKeyValuePair(entry string) (string, string, bool) {
	for _, separator := range []string{"=", ":"} {
		if idx := strings.Index(entry, separator); idx > 0 && idx < len(entry)-1 {
			key := strings.TrimSpace(entry[:idx])
			clientID := strings.TrimSpace(entry[idx+1:])
			if key != "" && clientID != "" {
				return key, clientID, true
			}
		}
	}
	return "", "", false
}

//nolint:wsl,nlreturn // Tiny helper keeps the rate limit setup readable.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// parseIntSafe safely parses a string to int
func parseIntSafe(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
