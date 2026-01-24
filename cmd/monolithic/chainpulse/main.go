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

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/indexing"
)

func main() {
	// Print header
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ChainPulse - Monolithic Indexer Service            ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadConfiguration()

	// Print configuration
	fmt.Println("Configuration Loaded:")
	fmt.Printf("  Chains:              %s\n", config.Chains)
	fmt.Printf("  Data Puller Type:    %s\n", config.DataPullerType)
	fmt.Printf("  Blockchain Nodes:    %s\n", config.BlockchainNodeURLs)
	fmt.Printf("  Message Queue Type:  %s\n", config.MQType)
	fmt.Printf("  Cache Type:          %s\n", config.CacheType)
	fmt.Printf("  Database Type:       %s\n", config.DatabaseType)
	fmt.Printf("  API Port:            %s\n", config.APIPort)
	fmt.Printf("  Worker Pool Size:    %s\n", config.WorkerPoolSize)
	fmt.Printf("  Batch Size:          %s\n", config.BatchSize)
	fmt.Printf("  Log Level:           %s\n", config.LogLevel)
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
		APIType:        "graphql",
		APIPort:        8080,
		LogLevel:       config.LogLevel,
		DatabaseType:   config.DatabaseType,
		DatabaseURL:    config.DatabaseURL,
		CacheType:      config.CacheType,
		FeatureFlags:   make(map[string]bool),
	}

	// Initialize Multi-Chain Indexer
	fmt.Println("Initializing Multi-Chain Indexer:")
	multiChainIndexer := indexing.NewMultiChainIndexer(logger, nil)
	chains := strings.Split(config.Chains, ",")
	for _, chainID := range chains {
		chainID = strings.TrimSpace(chainID)
		chainIndexer := indexing.NewDefaultChainIndexer(
			chainID,
			nil, // database
			nil, // cache
			logger,
			nil, // eventBus
		)
		if err := multiChainIndexer.RegisterChainIndexer(chainID, chainIndexer); err != nil {
			logger.Error("Failed to register chain indexer", "chain_id", chainID, "error", err.Error())
			os.Exit(1)
		}
		fmt.Printf("  ✓ Chain indexer registered: %s\n", chainID)
	}
	fmt.Println()

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
	fmt.Printf("GraphQL API available at: http://localhost:8080/graphql\n")
	fmt.Printf("GraphQL WebSocket available at: ws://localhost:8080/graphql\n")
	fmt.Printf("Health Check available at: http://localhost:8080/health\n")
	fmt.Printf("Metrics available at: http://localhost:8080/metrics\n")
	fmt.Printf("Indexed Chains: %s\n", config.Chains)
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
	fmt.Println("  [1/2] API Gateway stopped")

	// Close multi-chain indexer
	if err := multiChainIndexer.Close(); err != nil {
		logger.Error("Error closing multi-chain indexer", "error", err.Error())
	}
	fmt.Println("  [2/2] Multi-chain indexer closed")

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

// Configuration represents the application configuration
type Configuration struct {
	Chains                string
	DataPullerType        string
	BlockchainNodeURLs    string
	MQType                string
	MQConnectionURL       string
	CacheType             string
	CacheConnectionURL    string
	DatabaseType          string
	DatabaseURL           string
	APIType               string
	APIPort               string
	WorkerPoolSize        string
	BatchSize             string
	LogLevel              string
}

// loadConfiguration loads configuration from environment variables
func loadConfiguration() Configuration {
	return Configuration{
		Chains:                getEnv("CHAINS", "ethereum,polygon"),
		DataPullerType:        getEnv("DATA_PULLER_TYPE", "https-jsonrpc"),
		BlockchainNodeURLs:    getEnv("BLOCKCHAIN_NODE_URLS", "http://localhost:8545,http://localhost:8546"),
		MQType:                getEnv("MQ_TYPE", "kafka"),
		MQConnectionURL:       getEnv("MQ_CONNECTION_URL", "localhost:9092"),
		CacheType:             getEnv("CACHE_TYPE", "redis"),
		CacheConnectionURL:    getEnv("CACHE_CONNECTION_URL", "localhost:6379"),
		DatabaseType:          getEnv("DATABASE_TYPE", "postgres"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://localhost/chainpulse"),
		APIType:               getEnv("API_TYPE", "graphql"),
		APIPort:               getEnv("API_PORT", "8080"),
		WorkerPoolSize:        getEnv("WORKER_POOL_SIZE", "8"),
		BatchSize:             getEnv("BATCH_SIZE", "100"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
