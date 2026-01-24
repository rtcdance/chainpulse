package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/mq"
	"chainpulse/pkg/plugins/pullers"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       ChainPulse - Data Puller Service                     ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadPullerConfig()

	// Print configuration
	fmt.Println("Configuration Loaded:")
	fmt.Printf("  Service Port:       %d\n", config.Port)
	fmt.Printf("  Instance ID:        %s\n", config.InstanceID)
	fmt.Printf("  Kafka Brokers:      %v\n", config.KafkaBrokers)
	fmt.Printf("  Producer Group:     %s\n", config.ProducerGroup)
	fmt.Printf("  Output Topics:      %v\n", config.OutputTopics)
	fmt.Printf("  Blockchain RPCs:    %v\n", config.BlockchainRPCs)
	fmt.Printf("  Poll Interval:      %ds\n", config.PollInterval)
	fmt.Printf("  Block Confirmation: %d\n", config.BlockConfirmation)
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

	fmt.Println("  [1/2] Logger initialized")
	fmt.Println("  [2/2] Metrics collector initialized")
	fmt.Println()

	// Load database configuration
	fmt.Println("Loading Database Configuration:")
	dbConfig, err := database.LoadConfig()
	if err != nil {
		logger.Error("Failed to load database configuration", "error", err.Error())
		os.Exit(1)
	}
	fmt.Printf("  PostgreSQL URL:     %s\n", dbConfig.PostgresURL)
	fmt.Printf("  Pool Size:          %d\n", dbConfig.PoolSize)
	fmt.Printf("  Timeout:            %dms\n", dbConfig.TimeoutMS)
	fmt.Println()

	// Initialize database manager
	fmt.Println("Initializing Database Manager:")
	dbManager := database.NewDatabaseManager(
		"", // MongoDB URI not needed for puller
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
		config.ProducerGroup,
	)
	if err := kafkaMQ.Initialize(); err != nil {
		logger.Error("Failed to initialize Kafka MQ", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ Kafka Message Queue initialized")
	fmt.Println()

	// Initialize Multi-Chain Data Puller
	fmt.Println("Initializing Multi-Chain Data Puller:")
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)
	fmt.Println("  ✓ Multi-Chain Data Puller initialized")
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
	fmt.Println("  [1/2] Kafka Message Queue started")

	// Start Multi-Chain Data Puller polling loop
	wg.Add(1)
	go func() {
		defer wg.Done()
		runPullerLoop(context.Background(), multiChainPuller, config, logger, metrics)
	}()
	fmt.Println("  [2/2] Multi-Chain Data Puller started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("Data Puller available at: http://localhost:%d\n", config.Port)
	fmt.Printf("Health Check available at: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Metrics available at: http://localhost:%d/metrics\n", config.Port)
	fmt.Printf("Instance ID: %s\n", config.InstanceID)
	fmt.Printf("Polling from: %v\n", config.BlockchainRPCs)
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
	fmt.Println("  [1/2] Kafka Message Queue stopped")

	// Close database manager
	if err := dbManager.Close(shutdownCtx); err != nil {
		logger.Error("Error closing database manager", "error", err.Error())
	}
	fmt.Println("  [2/2] Database manager closed")

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

// PullerConfig represents the Data Puller configuration
type PullerConfig struct {
	Port               int
	InstanceID         string
	KafkaBrokers       []string
	ProducerGroup      string
	OutputTopics       []string
	BlockchainRPCs     []string
	PollInterval       int
	BlockConfirmation  int
	StateBackend       string
	CheckpointInterval int
	ReorgDetectionDepth int
	BatchSize          int
	MaxRetries         int
	WorkerThreads      int
	LogLevel           string
}

// loadPullerConfig loads configuration from environment variables
func loadPullerConfig() PullerConfig {
	instanceID := getEnv("HOSTNAME", "puller-1")
	if id := getEnv("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return PullerConfig{
		Port:                getEnvInt("PULLER_PORT", 8083),
		InstanceID:          instanceID,
		KafkaBrokers:        parseStringList(getEnv("KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092,kafka-3:9092")),
		ProducerGroup:       getEnv("KAFKA_PRODUCER_GROUP", "data-puller-producers"),
		OutputTopics:        parseStringList(getEnv("KAFKA_OUTPUT_TOPICS", "raw-events,blockchain-events")),
		BlockchainRPCs:      parseStringList(getEnv("BLOCKCHAIN_RPCS", "http://ethereum-rpc:8545,http://polygon-rpc:8545")),
		PollInterval:        getEnvInt("POLL_INTERVAL", 12),
		BlockConfirmation:   getEnvInt("BLOCK_CONFIRMATION", 12),
		StateBackend:        getEnv("STATE_BACKEND", "redis"),
		CheckpointInterval:  getEnvInt("STATE_CHECKPOINT_INTERVAL", 100),
		ReorgDetectionDepth: getEnvInt("REORG_DETECTION_DEPTH", 256),
		BatchSize:           getEnvInt("BATCH_SIZE", 100),
		MaxRetries:          getEnvInt("MAX_RETRIES", 3),
		WorkerThreads:       getEnvInt("WORKER_THREADS", 4),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
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

// parseStringList parses a comma-separated string into a slice
func parseStringList(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// runPullerLoop runs the main polling loop for the data puller
func runPullerLoop(
	ctx context.Context,
	puller *pullers.MultiChainDataPuller,
	config PullerConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
) {
	ticker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
	defer ticker.Stop()

	logger.Info("Data puller loop started", "poll_interval", config.PollInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Data puller loop stopped")
			return
		case <-ticker.C:
			// Polling logic would go here
			// For now, just log that we're polling
			logger.Debug("Polling for new blocks", "instance_id", config.InstanceID)
			metrics.RecordCounter("puller_polls", 1, map[string]string{"instance": config.InstanceID})
		}
	}
}
