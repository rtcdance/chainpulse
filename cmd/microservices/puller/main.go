package main

//nolint:funlen // Command entrypoint is intentionally verbose.

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
	"github.com/rtcdance/chainpulse/pkg/infrastructure/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

//nolint:wsl,nlreturn // Command entrypoint is intentionally verbose.
func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       ChainPulse - Data Puller Service                     ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadPullerConfig()

	if err := validatePullerConfig(config); err != nil {
		fmt.Printf("config validation failed: %v\n", err)
		os.Exit(1)
	}

	runtimeProfile := bootstrap.RuntimeProfileFromEnv()
	if err := validatePullerProductionSecurity(config, runtimeProfile); err != nil {
		fmt.Printf("production security validation failed: %v\n", err)
		os.Exit(1)
	}

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
	fmt.Printf("  Security Auth:      %v\n", config.AuthEnabled)
	fmt.Printf("  Rate Limit:         %v\n", config.RateLimitEnabled)
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
		dbConfig.MongoDBURI,
		dbConfig.PostgresURL,
		dbConfig.PostgresSSLMode,
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

	progressTracker := &pullerLoopRuntimeProgress{}
	checkpointSource := newPullerRuntimeCheckpointSource()
	loopController := newPullerLoopController()
	executionRuntime := newPullerExecutionRuntime(logger, metrics, kafkaMQ, config.OutputTopics)
	authMiddleware, rateLimitMiddleware, err := bootstrap.BuildSecurityControls(bootstrap.SecurityControlsConfig{
		AuthEnabled:        config.AuthEnabled,
		AuthJWTSecret:      config.AuthJWTSecret,
		AuthAPIKeys:        config.AuthAPIKeys,
		RateLimitEnabled:   config.RateLimitEnabled,
		RateLimitPerMinute: config.RateLimitPerMinute,
		ServiceName:        "puller",
		EnvPrefix:          "PULLER",
	}, logger, metrics)
	if err != nil {
		logger.Error("Failed to build puller security controls", "error", err.Error())
		os.Exit(1)
	}

	rolloutHealthHandler, err := buildPullerRuntimeRolloutHealthHandler(
		context.Background(),
		config.InstanceID,
		logger,
		metrics,
		dbManager,
		kafkaMQ,
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     config.BlockchainRPCs,
			PollInterval:       config.PollInterval,
			CheckpointInterval: config.CheckpointInterval,
		},
		checkpointSource,
		progressTracker,
		executionRuntime,
	)
	if err != nil {
		logger.Error("Failed to initialize rollout health handler", "error", err.Error())
		os.Exit(1)
	}
	runtimeHTTPServer := newPullerRuntimeHTTPServer(
		config.Port,
		rolloutHealthHandler,
		metrics,
		func(r *http.Request) *pullerRuntimeSummaryResponse {
			state := buildPullerRuntimeRolloutState(
				r.Context(),
				dbManager,
				kafkaMQ,
				pullerRolloutRuntimeConfig{
					BlockchainRPCs:     config.BlockchainRPCs,
					PollInterval:       config.PollInterval,
					CheckpointInterval: config.CheckpointInterval,
				},
				checkpointSource,
				progressTracker,
				executionRuntime,
			)
			return buildPullerRuntimeSummary(state, metrics, time.Now(), config.AuthEnabled, config.RateLimitEnabled)
		},
		loopController,
	)
	if authMiddleware != nil || rateLimitMiddleware != nil {
		runtimeHTTPServer.Handler = wrapPullerRuntimeSecurityHandler(runtimeHTTPServer.Handler, authMiddleware, rateLimitMiddleware)
	}
	fmt.Println("  ✓ Rollout report producer initialized")
	fmt.Println("  ✓ Runtime HTTP health surface initialized")
	fmt.Println()

	// Initialize Multi-Chain Data Puller
	fmt.Println("Initializing Multi-Chain Data Puller:")
	multiChainPuller := pullers.NewMultiChainDataPuller(logger)
	registeredPullers, err := registerConfiguredPullers(multiChainPuller, config, logger, metrics)
	if err != nil {
		logger.Error("Failed to register configured pullers", "error", err.Error())
		os.Exit(1)
	}
	executionRuntime.SetConfiguredPullers(registeredPullers)
	fmt.Println("  ✓ Multi-Chain Data Puller initialized")
	fmt.Printf("  ✓ Configured Pullers Attached: %d\n", registeredPullers)
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
	fmt.Println("  [1/3] Kafka Message Queue started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runtimeHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Runtime HTTP server error", "error", err.Error())
		}
	}()
	fmt.Println("  [2/3] Runtime HTTP health surface started")

	// Start Multi-Chain Data Puller polling loop
	pullerCtx, pullerCancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		runPullerLoop(
			pullerCtx,
			multiChainPuller,
			config,
			logger,
			metrics,
			checkpointSource,
			progressTracker,
			loopController,
			executionRuntime,
		)
	}()
	fmt.Println("  [3/3] Multi-Chain Data Puller started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("Data Puller available at: http://localhost:%d\n", config.Port)
	fmt.Printf("Health Check available at: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Metrics available at: http://localhost:%d/metrics\n", config.Port)
	fmt.Printf("Runtime Summary available at: http://localhost:%d/runtime/summary\n", config.Port)
	fmt.Printf("Instance ID: %s\n", config.InstanceID)
	fmt.Printf("Polling from: %v\n", config.BlockchainRPCs)
	fmt.Printf("Publishing to topics: %v\n", config.OutputTopics)
	fmt.Println("Press Ctrl+C to shutdown gracefully")
	fmt.Println()

	// Setup signal handling
	_ = bootstrap.WaitForSignal()

	// Graceful shutdown
	fmt.Println("Shutting Down Services:")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Signal puller loop to stop
	pullerCancel()
	fmt.Println("  [0/3] Puller loop signaled to stop")

	// Stop Kafka Message Queue
	if err := runtimeHTTPServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error stopping runtime HTTP server", "error", err.Error())
	}
	fmt.Println("  [1/3] Runtime HTTP health surface stopped")

	if err := kafkaMQ.Stop(); err != nil {
		logger.Error("Error stopping Kafka MQ", "error", err.Error())
	}
	fmt.Println("  [2/3] Kafka Message Queue stopped")

	// Close database manager
	if err := dbManager.Close(shutdownCtx); err != nil {
		logger.Error("Error closing database manager", "error", err.Error())
	}
	fmt.Println("  [3/3] Database manager closed")

	bootstrap.ShutdownWithTimeout(&wg, 30*time.Second)

	fmt.Println()
	fmt.Println("Status: Shutdown complete")
	fmt.Println()
}

// PullerConfig represents the Data Puller configuration
type PullerConfig struct {
	Port                int
	InstanceID          string
	KafkaBrokers        []string
	ProducerGroup       string
	OutputTopics        []string
	BlockchainRPCs      []string
	PollInterval        int
	BlockConfirmation   int
	StateBackend        string
	CheckpointInterval  int
	ReorgDetectionDepth int
	BatchSize           int
	MaxRetries          int
	WorkerThreads       int
	LogLevel            string
	AuthEnabled         bool
	AuthJWTSecret       core.SecretString
	AuthAPIKeys         []string
	RateLimitEnabled    bool
	RateLimitPerMinute  int
}

// loadPullerConfig loads configuration from environment variables
func loadPullerConfig() PullerConfig {
	instanceID := env.Get("HOSTNAME", "puller-1")
	if id := env.Get("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return PullerConfig{
		Port:                env.GetInt("PULLER_PORT", 8083),
		InstanceID:          instanceID,
		KafkaBrokers:        env.GetCSV("KAFKA_BROKERS", []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"}),
		ProducerGroup:       env.Get("KAFKA_PRODUCER_GROUP", "data-puller-producers"),
		OutputTopics:        env.GetCSV("KAFKA_OUTPUT_TOPICS", []string{"raw-events", "blockchain-events"}),
		BlockchainRPCs:      env.GetCSV("BLOCKCHAIN_RPCS", []string{"http://ethereum-rpc:8545", "http://polygon-rpc:8545"}),
		PollInterval:        env.GetInt("POLL_INTERVAL", 12),
		BlockConfirmation:   env.GetInt("BLOCK_CONFIRMATION", 12),
		StateBackend:        env.Get("STATE_BACKEND", "redis"),
		CheckpointInterval:  env.GetInt("STATE_CHECKPOINT_INTERVAL", 100),
		ReorgDetectionDepth: env.GetInt("REORG_DETECTION_DEPTH", 256),
		BatchSize:           env.GetInt("BATCH_SIZE", 100),
		MaxRetries:          env.GetInt("MAX_RETRIES", 3),
		WorkerThreads:       env.GetInt("WORKER_THREADS", 4),
		LogLevel:            env.Get("LOG_LEVEL", "info"),
		AuthEnabled:         env.GetBool("PULLER_AUTH_ENABLED", false),
		AuthJWTSecret:       core.SecretString(env.Get("PULLER_AUTH_JWT_SECRET", "")),
		AuthAPIKeys:         env.GetCSV("PULLER_AUTH_API_KEYS", nil),
		RateLimitEnabled:    env.GetBool("PULLER_RATE_LIMIT_ENABLED", true),
		RateLimitPerMinute:  env.GetInt("PULLER_RATE_LIMIT", 100),
	}
}

func validatePullerConfig(c PullerConfig) error {
	if c.PollInterval < 1 {
		return fmt.Errorf("POLL_INTERVAL must be >= 1 second, got %d", c.PollInterval)
	}
	if c.WorkerThreads < 1 {
		return fmt.Errorf("WORKER_THREADS must be >= 1, got %d", c.WorkerThreads)
	}
	if c.BatchSize < 1 {
		return fmt.Errorf("BATCH_SIZE must be >= 1, got %d", c.BatchSize)
	}
	if c.MaxRetries < 1 {
		return fmt.Errorf("MAX_RETRIES must be >= 1, got %d", c.MaxRetries)
	}
	if c.BlockConfirmation < 1 {
		return fmt.Errorf("BLOCK_CONFIRMATION must be >= 1, got %d", c.BlockConfirmation)
	}
	if c.CheckpointInterval < 1 {
		return fmt.Errorf("STATE_CHECKPOINT_INTERVAL must be >= 1, got %d", c.CheckpointInterval)
	}
	if c.ReorgDetectionDepth < 1 {
		return fmt.Errorf("REORG_DETECTION_DEPTH must be >= 1, got %d", c.ReorgDetectionDepth)
	}
	if len(c.BlockchainRPCs) == 0 {
		return fmt.Errorf("BLOCKCHAIN_RPCS must not be empty")
	}
	if c.AuthEnabled && strings.TrimSpace(c.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("PULLER_AUTH_JWT_SECRET must not be empty when PULLER_AUTH_ENABLED is true")
	}
	return nil
}

func validatePullerProductionSecurity(c PullerConfig, runtimeProfile string) error {
	if runtimeProfile != "production" {
		return nil
	}
	if !c.AuthEnabled {
		return fmt.Errorf("production puller requires PULLER_AUTH_ENABLED=true")
	}
	if strings.TrimSpace(c.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("production puller requires non-empty PULLER_AUTH_JWT_SECRET")
	}
	if !c.RateLimitEnabled {
		return fmt.Errorf("production puller requires PULLER_RATE_LIMIT_ENABLED=true")
	}
	if c.RateLimitPerMinute <= 0 {
		return fmt.Errorf("production puller requires PULLER_RATE_LIMIT > 0")
	}
	return nil
}

// runPullerLoop runs the main polling loop for the data puller
func runPullerLoop(
	ctx context.Context,
	puller *pullers.MultiChainDataPuller,
	config PullerConfig,
	logger core.Logger,
	metrics core.MetricsCollector,
	checkpointSource pullerCheckpointSource,
	progress *pullerLoopRuntimeProgress,
	controller *pullerLoopController,
	execution *pullerExecutionRuntime,
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
			executePullerPollTick(ctx, puller, config, logger, metrics, checkpointSource, progress, controller, execution)
		}
	}
}
