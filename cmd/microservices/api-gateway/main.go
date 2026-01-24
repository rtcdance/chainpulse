package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

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
	fmt.Printf("  TLS Enabled:        %v\n", config.TLSEnabled)
	fmt.Printf("  Rate Limit:         %d req/s\n", config.RateLimitPerSecond)
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

	// Initialize API Gateway Plugin
	fmt.Println("Initializing API Gateway:")
	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	if err := gateway.Initialize(*coreConfig); err != nil {
		logger.Error("Failed to initialize API Gateway", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ API Gateway initialized")
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
	Port                 int
	TLSEnabled           bool
	TLSCertPath          string
	TLSKeyPath           string
	UpstreamServices     []string
	RateLimitPerSecond   int
	AuthenticationToken  string
	LogLevel             string
}

// loadGatewayConfig loads configuration from environment variables
func loadGatewayConfig() GatewayConfig {
	return GatewayConfig{
		Port:               getEnvInt("GATEWAY_PORT", 8080),
		TLSEnabled:         getEnvBool("GATEWAY_TLS_ENABLED", false),
		TLSCertPath:        getEnv("GATEWAY_TLS_CERT", ""),
		TLSKeyPath:         getEnv("GATEWAY_TLS_KEY", ""),
		UpstreamServices:   []string{"http://api-service-1:8081", "http://api-service-2:8081", "http://api-service-3:8081"},
		RateLimitPerSecond: getEnvInt("GATEWAY_RATE_LIMIT", 1000),
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

// parseIntSafe safely parses a string to int
func parseIntSafe(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
