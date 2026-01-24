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
	"chainpulse/pkg/plugins/mq"
	"chainpulse/pkg/services/query"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║      ChainPulse - Event Processor Service                  ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadEventProcessorConfig()

	// Print configuration
	fmt.Println("Configuration Loaded:")
	fmt.Printf("  Service Port:       %d\n", config.Port)
	fmt.Printf("  Instance ID:        %s\n", config.InstanceID)
	fmt.Printf("  Kafka Brokers:      %v\n", config.KafkaBrokers)
	fmt.Printf("  Consumer Group:     %s\n", config.ConsumerGroup)
	fmt.Printf("  Input Topics:       %v\n", config.InputTopics)
	fmt.Printf("  Output Topics:      %v\n", config.OutputTopics)
	fmt.Printf("  Batch Size:         %d\n", config.BatchSize)
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

	// Initialize Event Store components
	fmt.Println("Initializing Event Store Components:")
	eventStoreConfig := &query.EventStoreConfig{
		CollectionName: "events",
		TTLDays:        config.EventTTLDays,
		BatchSize:      config.BatchSize,
		IndexTimeout:   10 * time.Second,
	}
	eventStore := query.NewMongoDBEventStore(dbManager, logger, metrics, eventStoreConfig)
	metadataStore := query.NewPostgreSQLEventMetadataStore(dbManager, logger, metrics)

	initCtx, cancel = context.WithTimeout(context.Background(), dbConfig.GetTimeout())
	if err := eventStore.Initialize(initCtx); err != nil {
		cancel()
		logger.Error("Failed to initialize event store", "error", err.Error())
		os.Exit(1)
	}
	cancel()
	fmt.Println("  [1/2] Event store initialized")

	initCtx, cancel = context.WithTimeout(context.Background(), dbConfig.GetTimeout())
	if err := metadataStore.Initialize(initCtx); err != nil {
		cancel()
		logger.Error("Failed to initialize metadata store", "error", err.Error())
		os.Exit(1)
	}
	cancel()
	fmt.Println("  [2/2] Metadata store initialized")
	fmt.Println()

	// Initialize Kafka Message Queue Plugin
	fmt.Println("Initializing Message Queue:")
	kafkaMQ := mq.NewKafkaMQPlugin(
		"kafka-mq",
		"1.0.0",
		&core.Config{
			APIType:  "kafka",
			LogLevel: config.LogLevel,
		},
		logger,
		metrics,
		nil, // eventBus - can be nil for now
		config.KafkaBrokers,
		config.ConsumerGroup,
	)
	if err := kafkaMQ.Initialize(); err != nil {
		logger.Error("Failed to initialize Kafka MQ", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ Kafka Message Queue initialized")
	fmt.Println()

	// Start all services
	fmt.Println("Starting Services:")
	var wg sync.WaitGroup

	// Start Kafka Message Queue
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := kafkaMQ.Start(); err != nil {
			logger.Error("Kafka MQ error", "error", err.Error())
		}
	}()
	fmt.Println("  [1/1] Kafka Message Queue started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("Event Processor available at: http://localhost:%d\n", config.Port)
	fmt.Printf("Health Check available at: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Metrics available at: http://localhost:%d/metrics\n", config.Port)
	fmt.Printf("Instance ID: %s\n", config.InstanceID)
	fmt.Printf("Consuming from topics: %v\n", config.InputTopics)
	fmt.Printf("Publishing to topics: %v\n", config.OutputTopics)
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

	// Stop Kafka Message Queue
	if err := kafkaMQ.Stop(); err != nil {
		logger.Error("Error stopping Kafka MQ", "error", err.Error())
	}
	fmt.Println("  [1/4] Kafka Message Queue stopped")

	// Close Event Store
	if err := eventStore.Close(shutdownCtx); err != nil {
		logger.Error("Error closing event store", "error", err.Error())
	}
	fmt.Println("  [2/4] Event store closed")

	// Close Metadata Store
	if err := metadataStore.Close(shutdownCtx); err != nil {
		logger.Error("Error closing metadata store", "error", err.Error())
	}
	fmt.Println("  [3/4] Metadata store closed")

	// Close database manager
	if err := dbManager.Close(shutdownCtx); err != nil {
		logger.Error("Error closing database manager", "error", err.Error())
	}
	fmt.Println("  [4/4] Database manager closed")

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

// EventProcessorConfig represents the Event Processor configuration
type EventProcessorConfig struct {
	Port          int
	InstanceID    string
	KafkaBrokers  []string
	ConsumerGroup string
	InputTopics   []string
	OutputTopics  []string
	BatchSize     int
	EventTTLDays  int
	LogLevel      string
}

// loadEventProcessorConfig loads configuration from environment variables
func loadEventProcessorConfig() EventProcessorConfig {
	instanceID := getEnv("HOSTNAME", "event-processor-1")
	if id := getEnv("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return EventProcessorConfig{
		Port:          getEnvInt("PROCESSOR_PORT", 8082),
		InstanceID:    instanceID,
		KafkaBrokers:  []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
		ConsumerGroup: "event-processor-consumers",
		InputTopics:   []string{"raw-events", "blockchain-events"},
		OutputTopics:  []string{"processed-events", "indexed-events"},
		BatchSize:     getEnvInt("BATCH_SIZE", 100),
		EventTTLDays:  getEnvInt("EVENT_TTL_DAYS", 30),
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
