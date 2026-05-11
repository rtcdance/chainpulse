package main

//nolint:funlen // Command entrypoint is intentionally verbose.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/plugins/mq"
	"chainpulse/pkg/services/query"
	"chainpulse/pkg/services/reorg"
)

//nolint:wsl,nlreturn // Command entrypoint is intentionally verbose.
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

	processorRuntime, err := newEventProcessorProcessingRuntimeWithStorage(
		config,
		logger,
		metrics,
		newPersistentEventProcessorStorage(eventStore, metadataStore),
	)
	if err != nil {
		logger.Error("Failed to initialize processor runtime", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ Processor runtime initialized")

	consumeRuntime := newEventProcessorConsumeRuntime(
		logger,
		metrics,
		kafkaMQ,
		processorRuntime.MessageProcessor(),
		kafkaMQ,
		config.InputTopics,
		config.OutputTopics,
		func() *sql.DB {
			raw, err := dbManager.GetPostgresDB(context.Background())
			if err != nil {
				return nil
			}
			db, ok := raw.(*sql.DB)
			if !ok {
				return nil
			}
			return db
		}(),
	)
	fmt.Println("  ✓ Consume/process seam initialized")
	authMiddleware, rateLimitMiddleware, err := buildEventProcessorSecurityControls(config, logger, metrics)
	if err != nil {
		logger.Error("Failed to build event processor security controls", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println()

	rolloutHealthHandler, err := buildEventProcessorRuntimeRolloutHealthHandler(
		context.Background(),
		config.InstanceID,
		logger,
		metrics,
		dbManager,
		eventStore,
		metadataStore,
		kafkaMQ,
		processorRuntime.Processor(),
		consumeRuntime,
	)
	if err != nil {
		logger.Error("Failed to initialize rollout health handler", "error", err.Error())
		os.Exit(1)
	}
	_ = rolloutHealthHandler
	runtimeHTTPServer := newEventProcessorRuntimeHTTPServer(
		config.Port,
		rolloutHealthHandler,
		metrics,
		func(r *http.Request) *eventProcessorRuntimeSummaryResponse {
			state := buildEventProcessorRuntimeRolloutState(
				r.Context(),
				dbManager,
				eventStore,
				metadataStore,
				kafkaMQ,
				processorRuntime.Processor(),
				consumeRuntime,
			)
			return buildEventProcessorRuntimeSummary(state, metrics, time.Now(), config.AuthEnabled, config.RateLimitEnabled)
		},
		consumeRuntime,
	)
	if authMiddleware != nil || rateLimitMiddleware != nil {
		runtimeHTTPServer.Handler = wrapEventProcessorRuntimeSecurityHandler(runtimeHTTPServer.Handler, authMiddleware, rateLimitMiddleware)
	}
	fmt.Println("  ✓ Rollout report producer initialized")
	fmt.Println("  ✓ Runtime HTTP health surface initialized")
	fmt.Println()

	// Start all services
	fmt.Println("Starting Services:")
	var wg sync.WaitGroup

	// Start Kafka Message Queue
	if err := kafkaMQ.Start(); err != nil {
		logger.Error("Kafka MQ error", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  [1/3] Kafka Message Queue started")

	consumeCtx, consumeCancel := context.WithCancel(context.Background())
	consumeRuntime.Start(consumeCtx, &wg)
	fmt.Println("  [2/3] Consume/process seam started")

	// Start reorg event consumer
	// Note: uses database-backed block hash provider (default). For production,
	// inject an RPC-backed provider via reorgHandler.SetBlockHashProvider()
	// so reorg detection compares against the live canonical chain.
	reorgHandler := reorg.NewReorgHandler(
		newReorgEventProcessorDatabaseAdapter(eventStore, metadataStore),
		logger,
		12,  // reorg threshold
		120, // max rollback
	)
	// Note: For production, inject an RPC-backed provider via
	// reorgHandler.SetBlockHashProvider() and wire the idempotency
	// service via reorgHandler.SetIdempotencyInvalidator() so that
	// re-indexed events after a reorg are not rejected as duplicates.
	wg.Add(1)
	go func() {
		defer wg.Done()
		reorgErr := kafkaMQ.ConsumeMessages(consumeCtx, "reorg-events", func(msg core.MessageQueueMessage) error {
			var reorgMsg core.ReorgDetectedMessage
			if err := json.Unmarshal(msg.Payload, &reorgMsg); err != nil {
				logger.Error("Failed to unmarshal reorg message", "error", err.Error())
				return err
			}
			logger.Info("Reorg event received", "chain_id", reorgMsg.ChainID, "reorg_block", reorgMsg.ReorgBlock)
			if err := reorgHandler.HandleReorg(consumeCtx, reorgMsg.ReorgBlock); err != nil {
				logger.Error("Failed to handle reorg", "chain_id", reorgMsg.ChainID, "reorg_block", reorgMsg.ReorgBlock, "error", err.Error())
				return err
			}
			logger.Info("Reorg handled successfully", "chain_id", reorgMsg.ChainID, "reorg_block", reorgMsg.ReorgBlock)
			return nil
		})
		if reorgErr != nil && reorgErr != context.Canceled {
			logger.Error("Reorg consumer stopped with error", "error", reorgErr.Error())
		}
	}()
	fmt.Println("  [2.5/3] Reorg event consumer started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runtimeHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Runtime HTTP server error", "error", err.Error())
		}
	}()
	fmt.Println("  [3/3] Runtime HTTP health surface started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("Event Processor available at: http://localhost:%d\n", config.Port)
	fmt.Printf("Health Check available at: http://localhost:%d/health\n", config.Port)
	fmt.Printf("Metrics available at: http://localhost:%d/metrics\n", config.Port)
	fmt.Printf("Runtime Summary available at: http://localhost:%d/runtime/summary\n", config.Port)
	fmt.Printf("Runtime Control available at: http://localhost:%d/runtime/control\n", config.Port)
	fmt.Printf("Instance ID: %s\n", config.InstanceID)
	fmt.Printf("Consuming from topics: %v\n", config.InputTopics)
	fmt.Printf("Publishing to topics: %v\n", config.OutputTopics)
	fmt.Println("Processor runtime ownership: lifecycle-wired (execution loop not yet service-owned)")
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
	consumeCancel()

	// Stop Kafka Message Queue
	if err := runtimeHTTPServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error stopping runtime HTTP server", "error", err.Error())
	}
	fmt.Println("  [1/5] Runtime HTTP health surface stopped")

	if err := kafkaMQ.Stop(); err != nil {
		logger.Error("Error stopping Kafka MQ", "error", err.Error())
	}
	fmt.Println("  [2/5] Kafka Message Queue stopped")

	if err := processorRuntime.Stop(); err != nil {
		logger.Error("Error stopping processor runtime", "error", err.Error())
	}
	fmt.Println("  [3/6] Processor runtime stopped")

	// Close Event Store
	if err := eventStore.Close(shutdownCtx); err != nil {
		logger.Error("Error closing event store", "error", err.Error())
	}
	fmt.Println("  [4/6] Event store closed")

	// Close Metadata Store
	if err := metadataStore.Close(shutdownCtx); err != nil {
		logger.Error("Error closing metadata store", "error", err.Error())
	}
	fmt.Println("  [5/6] Metadata store closed")

	// Close database manager
	if err := dbManager.Close(shutdownCtx); err != nil {
		logger.Error("Error closing database manager", "error", err.Error())
	}
	fmt.Println("  [6/6] Database manager closed")

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
	Port               int
	InstanceID         string
	KafkaBrokers       []string
	ConsumerGroup      string
	InputTopics        []string
	OutputTopics       []string
	BatchSize          int
	EventTTLDays       int
	LogLevel           string
	AuthEnabled        bool
	AuthJWTSecret      string
	AuthAPIKeys        []string
	RateLimitEnabled   bool
	RateLimitPerMinute int
}

// loadEventProcessorConfig loads configuration from environment variables
func loadEventProcessorConfig() EventProcessorConfig {
	instanceID := getEnv("HOSTNAME", "event-processor-1")
	if id := getEnv("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return EventProcessorConfig{
		Port:               getEnvInt("PROCESSOR_PORT", 8082),
		InstanceID:         instanceID,
		KafkaBrokers:       parseStringList(getEnv("KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092,kafka-3:9092")),
		ConsumerGroup:      getEnv("KAFKA_CONSUMER_GROUP", "event-processor-consumers"),
		InputTopics:        parseStringList(getEnv("KAFKA_INPUT_TOPICS", "raw-events,blockchain-events")),
		OutputTopics:       parseStringList(getEnv("KAFKA_OUTPUT_TOPICS", "processed-events,indexed-events")),
		BatchSize:          getEnvInt("BATCH_SIZE", 100),
		EventTTLDays:       getEnvInt("EVENT_TTL_DAYS", 30),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		AuthEnabled:        parseBoolEnv("EVENT_PROCESSOR_AUTH_ENABLED", false),
		AuthJWTSecret:      getEnv("EVENT_PROCESSOR_AUTH_JWT_SECRET", ""),
		AuthAPIKeys:        parseStringList(getEnv("EVENT_PROCESSOR_AUTH_API_KEYS", "")),
		RateLimitEnabled:   parseBoolEnv("EVENT_PROCESSOR_RATE_LIMIT_ENABLED", false),
		RateLimitPerMinute: getEnvInt("EVENT_PROCESSOR_RATE_LIMIT", 100),
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

//nolint:wsl,nlreturn // Small env parser stays centralized in the command entrypoint.
func parseBoolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

//nolint:wsl,nlreturn // Command config parsing stays centralized here.
func parseStringList(s string) []string {
	var result []string
	for _, item := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

//nolint:wsl,nlreturn // Security wiring stays centralized in the command entrypoint.
func buildEventProcessorSecurityControls(config EventProcessorConfig, logger core.Logger, metrics core.MetricsCollector) (*api.AuthMiddleware, *api.RateLimitMiddleware, error) {
	if !config.AuthEnabled && !config.RateLimitEnabled {
		return nil, nil, nil
	}

	var authMiddleware *api.AuthMiddleware
	if config.AuthEnabled {
		if strings.TrimSpace(config.AuthJWTSecret) == "" {
			return nil, nil, fmt.Errorf("event processor auth is enabled but EVENT_PROCESSOR_AUTH_JWT_SECRET is empty")
		}

		tokenValidator := api.NewTokenValidator(config.AuthJWTSecret, logger, metrics)
		for _, entry := range config.AuthAPIKeys {
			apiKey, clientID, ok := parseKeyValuePair(entry)
			if !ok {
				return nil, nil, fmt.Errorf("invalid EVENT_PROCESSOR_AUTH_API_KEYS entry %q; expected key=clientID or key:clientID", entry)
			}
			if err := tokenValidator.RegisterAPIKey(apiKey, clientID, "operator"); err != nil {
				return nil, nil, err
			}
		}

		rbacChecker := api.NewRBACChecker(logger, metrics)
		if err := rbacChecker.RegisterDefaultRoles(); err != nil {
			return nil, nil, fmt.Errorf("failed to register default RBAC roles: %w", err)
		}
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

//nolint:wsl,nlreturn // Command config parsing stays centralized here.
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

// reorgEventProcessorDatabaseAdapter adapts existing stores to core.DatabasePlugin
// for use by ReorgHandler in the event-processor microservice.
type reorgEventProcessorDatabaseAdapter struct {
	eventStore    *query.MongoDBEventStore
	metadataStore *query.PostgreSQLEventMetadataStore
}

func newReorgEventProcessorDatabaseAdapter(
	eventStore *query.MongoDBEventStore,
	metadataStore *query.PostgreSQLEventMetadataStore,
) *reorgEventProcessorDatabaseAdapter {
	return &reorgEventProcessorDatabaseAdapter{eventStore: eventStore, metadataStore: metadataStore}
}

func (a *reorgEventProcessorDatabaseAdapter) Name() string                   { return "reorg-adapter" }
func (a *reorgEventProcessorDatabaseAdapter) Version() string                { return "1.0.0" }
func (a *reorgEventProcessorDatabaseAdapter) Initialize(_ core.Config) error { return nil }
func (a *reorgEventProcessorDatabaseAdapter) Start() error                   { return nil }
func (a *reorgEventProcessorDatabaseAdapter) Stop() error                    { return nil }
func (a *reorgEventProcessorDatabaseAdapter) Health() error                  { return nil }

// EventReader
func (a *reorgEventProcessorDatabaseAdapter) GetEvent(_ context.Context, _ string) (*core.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) QueryEvents(_ context.Context, _ interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) GetAllEvents(_ context.Context) ([]*core.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	return a.eventStore.GetEventsByBlockRange(ctx, fromBlock, toBlock)
}

// EventWriter
func (a *reorgEventProcessorDatabaseAdapter) StoreEvent(_ context.Context, _ interface{}) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) BatchStoreEvents(_ context.Context, _ []interface{}) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) DeleteEvent(_ context.Context, _ string) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return a.eventStore.DeleteEventsByBlockRange(ctx, fromBlock, toBlock)
}

func (a *reorgEventProcessorDatabaseAdapter) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	// Delegate to DeleteEventsByBlockRange as fallback if the store doesn't support soft-delete
	return a.eventStore.DeleteEventsByBlockRange(ctx, fromBlock, toBlock)
}

// BlockReader
func (a *reorgEventProcessorDatabaseAdapter) GetBlock(_ context.Context, _ uint64) (*core.Block, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) GetLatestBlock(_ context.Context) (uint64, error) {
	return 0, fmt.Errorf("not implemented in reorg adapter — puller tracks block heights")
}

func (a *reorgEventProcessorDatabaseAdapter) GetAllBlocks(_ context.Context) ([]*core.Block, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

// ReorgStatsProvider
func (a *reorgEventProcessorDatabaseAdapter) GetReorgStats(_ context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}
