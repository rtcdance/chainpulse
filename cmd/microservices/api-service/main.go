package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/query"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ChainPulse - API Service                           ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadAPIServiceConfig()

	// Print configuration
	fmt.Println("Configuration Loaded:")
	fmt.Printf("  Service Port:       %d\n", config.Port)
	fmt.Printf("  Instance ID:        %s\n", config.InstanceID)
	fmt.Printf("  Database Host:      %s\n", config.DatabaseHost)
	fmt.Printf("  Redis Cluster:      %v\n", config.RedisCluster)
	fmt.Printf("  Kafka Brokers:      %v\n", config.KafkaBrokers)
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

	// Load database configuration
	fmt.Println("Loading Database Configuration:")
	dbConfig, err := database.LoadConfig()
	if err != nil {
		logger.Error("Failed to load database configuration", "error", err.Error())
		os.Exit(1)
	}
	fmt.Printf("  MongoDB URI:        %s\n", dbConfig.MongoDBURI)
	fmt.Printf("  PostgreSQL URL:     %s\n", dbConfig.PostgresURL)
	fmt.Printf("  Pool Size:          %d\n", dbConfig.PoolSize)
	fmt.Printf("  Timeout:            %dms\n", dbConfig.TimeoutMS)
	fmt.Println()

	// Initialize database manager
	fmt.Println("Initializing Database Manager:")
	dbManager := database.NewDatabaseManager(
		dbConfig.MongoDBURI,
		dbConfig.PostgresURL,
		dbConfig.PoolSize,
		dbConfig.GetTimeout(),
	)

	initCtx, cancel := context.WithTimeout(context.Background(), dbConfig.GetTimeout())
	if err := dbManager.Initialize(initCtx); err != nil {
		cancel()
		logger.Error("Failed to initialize database manager", "error", err.Error())
		os.Exit(1)
	}
	cancel()
	fmt.Println("  ✓ Database manager initialized")
	fmt.Println()

	// Initialize Query Service components
	fmt.Println("Initializing Query Service Components:")
	mongoAdapter := query.NewMongoDBAdapter(dbManager, logger, metrics)
	postgresAdapter := query.NewPostgreSQLAdapter(dbManager, logger, metrics)
	cacheService := query.NewCacheService(logger, metrics)
	queryService := query.NewQueryService(dbManager, mongoAdapter, postgresAdapter, cacheService, logger, metrics)
	fmt.Println("  [1/4] MongoDB adapter created")
	fmt.Println("  [2/4] PostgreSQL adapter created")
	fmt.Println("  [3/4] Cache service created")
	fmt.Println("  [4/4] Query service created")
	fmt.Println()

	// Initialize Query Service
	fmt.Println("Initializing Query Service:")
	initCtx, cancel = context.WithTimeout(context.Background(), dbConfig.GetTimeout())
	if err := queryService.Initialize(initCtx); err != nil {
		cancel()
		logger.Error("Failed to initialize query service", "error", err.Error())
		os.Exit(1)
	}
	cancel()
	fmt.Println("  ✓ Query service initialized")
	fmt.Println()

	// Convert configuration to core.Config
	coreConfig := &core.Config{
		APIType:      "service",
		APIPort:      config.Port,
		LogLevel:     config.LogLevel,
		FeatureFlags: make(map[string]bool),
	}

	// Initialize API Service Plugin
	fmt.Println("Initializing API Service:")
	service := api.NewAPIGatewayPlugin(logger, metrics)
	if err := service.Initialize(*coreConfig); err != nil {
		logger.Error("Failed to initialize API Service", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ API Service initialized")
	fmt.Println()

	// Register plugins with registry
	fmt.Println("Registering Plugins:")
	if err := registry.Register(service); err != nil {
		logger.Error("Failed to register API Service", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  [1/1] API Service registered")
	fmt.Println()

	// Start all services
	fmt.Println("Starting Services:")
	var wg sync.WaitGroup

	// Start Query Service
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := queryService.Start(context.Background()); err != nil {
			logger.Error("Query Service error", "error", err.Error())
		}
	}()
	fmt.Println("  [1/2] Query Service started")

	// Start API Service
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := service.Start(); err != nil {
			logger.Error("API Service error", "error", err.Error())
		}
	}()
	fmt.Println("  [2/2] API Service started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("API Service available at: http://localhost:%d\n", config.Port)
	fmt.Printf("Health Check available at: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Metrics available at: http://localhost:%d/metrics\n", config.Port)
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

	// Stop Query Service
	if err := queryService.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping Query Service", "error", err.Error())
	}
	fmt.Println("  [1/3] Query Service stopped")

	// Stop API Service
	if err := service.Stop(); err != nil {
		logger.Error("Error stopping API Service", "error", err.Error())
	}
	fmt.Println("  [2/3] API Service stopped")

	// Close database manager
	if err := dbManager.Close(shutdownCtx); err != nil {
		logger.Error("Error closing database manager", "error", err.Error())
	}
	fmt.Println("  [3/3] Database manager closed")

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

// APIServiceConfig represents the API Service configuration
type APIServiceConfig struct {
	Port          int
	InstanceID    string
	DatabaseHost  string
	DatabasePort  int
	RedisCluster  []string
	KafkaBrokers  []string
	ConsumerGroup string
	LogLevel      string
}

// loadAPIServiceConfig loads configuration from environment variables
func loadAPIServiceConfig() APIServiceConfig {
	instanceID := getEnv("HOSTNAME", "api-service-1")
	if id := getEnv("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return APIServiceConfig{
		Port:          getEnvInt("SERVICE_PORT", 8081),
		InstanceID:    instanceID,
		DatabaseHost:  getEnv("DB_HOST", "postgres-primary"),
		DatabasePort:  getEnvInt("DB_PORT", 5432),
		RedisCluster:  []string{"redis-1:6379", "redis-2:6379", "redis-3:6379"},
		KafkaBrokers:  []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
		ConsumerGroup: "api-service-consumers",
		LogLevel:      getEnv("LOG_LEVEL", "info"),
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
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
