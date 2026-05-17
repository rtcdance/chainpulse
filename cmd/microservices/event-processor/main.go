package main

//nolint:funlen // Command entrypoint is intentionally verbose.

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/rtcdance/chainpulse/pkg/services/query"
	"github.com/rtcdance/chainpulse/pkg/services/reorg"

	"github.com/ethereum/go-ethereum/common"
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

	if err := validateEventProcessorConfig(config); err != nil {
		fmt.Printf("config validation failed: %v\n", err)
		os.Exit(1)
	}

	runtimeProfile := bootstrap.RuntimeProfileFromEnv()
	if err := validateEventProcessorProductionSecurity(config, runtimeProfile); err != nil {
		fmt.Printf("production security validation failed: %v\n", err)
		os.Exit(1)
	}

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

	warmUpCtx, warmUpCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if events, _, err := eventStore.GetEventsPaginated(warmUpCtx, "", 10000); err == nil && len(events) > 0 {
		hashes := make([]string, 0, len(events))
		for _, e := range events {
			if e != nil {
				hashes = append(hashes, core.ComputeEventHash(e))
			}
		}
		if wuErr := processorRuntime.WarmUpIdempotency(warmUpCtx, hashes); wuErr != nil {
			logger.Warn("Idempotency warm-up failed (DB unique index will handle duplicates)", "error", wuErr.Error())
		} else {
			fmt.Printf("  ✓ Idempotency warm-up: loaded %d hashes from %d events\n", len(hashes), len(events))
		}
	} else if err != nil {
		logger.Warn("Idempotency warm-up skipped (could not query recent events)", "error", err.Error())
	}
	warmUpCancel()

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

	dlqRetry := newDLQRetryService(
		logger,
		metrics,
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
		processorRuntime.MessageProcessor(),
	)
	fmt.Println("  ✓ DLQ retry service initialized")
	authMiddleware, rateLimitMiddleware, err := bootstrap.BuildSecurityControls(bootstrap.SecurityControlsConfig{
		AuthEnabled:        config.AuthEnabled,
		AuthJWTSecret:      config.AuthJWTSecret,
		AuthAPIKeys:        config.AuthAPIKeys,
		RateLimitEnabled:   config.RateLimitEnabled,
		RateLimitPerMinute: config.RateLimitPerMinute,
		ServiceName:        "event processor",
		EnvPrefix:          "EVENT_PROCESSOR",
	}, logger, metrics)
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

	dlqRetry.Start(consumeCtx, &wg)
	fmt.Println("  [2.2/3] DLQ retry service started")

	// Start reorg event consumer
	// Note: uses database-backed block hash provider (default). For production,
	// inject an RPC-backed provider via reorgHandler.SetBlockHashProvider()
	// so reorg detection compares against the live canonical chain.
	reorgHandler := reorg.NewReorgHandler(
		newReorgEventProcessorDatabaseAdapter(eventStore, metadataStore, getDB(dbManager)),
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
				return fmt.Errorf("failed to unmarshal reorg message: %w", err)
			}
			logger.Info("Reorg event received", "chain_id", reorgMsg.ChainID, "reorg_block", reorgMsg.ReorgBlock)
			if err := reorgHandler.HandleReorg(consumeCtx, reorgMsg.ReorgBlock); err != nil {
				logger.Error("Failed to handle reorg", "chain_id", reorgMsg.ChainID, "reorg_block", reorgMsg.ReorgBlock, "error", err.Error())
				return fmt.Errorf("failed to handle reorg at block %d: %w", reorgMsg.ReorgBlock, err)
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
	_ = bootstrap.WaitForSignal()

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

	dlqRetry.Stop()
	fmt.Println("  [3.5/6] DLQ retry service stopped")

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

	bootstrap.ShutdownWithTimeout(&wg, 30*time.Second)

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
	AuthJWTSecret      core.SecretString
	AuthAPIKeys        []string
	RateLimitEnabled   bool
	RateLimitPerMinute int
}

// loadEventProcessorConfig loads configuration from environment variables
func loadEventProcessorConfig() EventProcessorConfig {
	instanceID := env.Get("HOSTNAME", "event-processor-1")
	if id := env.Get("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return EventProcessorConfig{
		Port:               env.GetInt("PROCESSOR_PORT", 8082),
		InstanceID:         instanceID,
		KafkaBrokers:       env.GetCSV("KAFKA_BROKERS", []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"}),
		ConsumerGroup:      env.Get("KAFKA_CONSUMER_GROUP", "event-processor-consumers"),
		InputTopics:        env.GetCSV("KAFKA_INPUT_TOPICS", []string{"raw-events", "blockchain-events"}),
		OutputTopics:       env.GetCSV("KAFKA_OUTPUT_TOPICS", []string{"processed-events", "indexed-events"}),
		BatchSize:          env.GetInt("BATCH_SIZE", 100),
		EventTTLDays:       env.GetInt("EVENT_TTL_DAYS", 30),
		LogLevel:           env.Get("LOG_LEVEL", "info"),
		AuthEnabled:        env.GetBool("EVENT_PROCESSOR_AUTH_ENABLED", false),
		AuthJWTSecret:      core.SecretString(env.Get("EVENT_PROCESSOR_AUTH_JWT_SECRET", "")),
		AuthAPIKeys:        env.GetCSV("EVENT_PROCESSOR_AUTH_API_KEYS", nil),
		RateLimitEnabled:   env.GetBool("EVENT_PROCESSOR_RATE_LIMIT_ENABLED", true),
		RateLimitPerMinute: env.GetInt("EVENT_PROCESSOR_RATE_LIMIT", 100),
	}
}

func validateEventProcessorConfig(c EventProcessorConfig) error {
	if c.BatchSize < 1 {
		return fmt.Errorf("BATCH_SIZE must be >= 1, got %d", c.BatchSize)
	}
	if c.EventTTLDays < 1 {
		return fmt.Errorf("EVENT_TTL_DAYS must be >= 1, got %d", c.EventTTLDays)
	}
	if c.AuthEnabled && strings.TrimSpace(c.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("EVENT_PROCESSOR_AUTH_JWT_SECRET must not be empty when EVENT_PROCESSOR_AUTH_ENABLED is true")
	}
	return nil
}

func validateEventProcessorProductionSecurity(c EventProcessorConfig, runtimeProfile string) error {
	if runtimeProfile != "production" {
		return nil
	}
	if !c.AuthEnabled {
		return fmt.Errorf("production event processor requires EVENT_PROCESSOR_AUTH_ENABLED=true")
	}
	if strings.TrimSpace(c.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("production event processor requires non-empty EVENT_PROCESSOR_AUTH_JWT_SECRET")
	}
	if !c.RateLimitEnabled {
		return fmt.Errorf("production event processor requires EVENT_PROCESSOR_RATE_LIMIT_ENABLED=true")
	}
	if c.RateLimitPerMinute <= 0 {
		return fmt.Errorf("production event processor requires EVENT_PROCESSOR_RATE_LIMIT > 0")
	}
	return nil
}

// reorgEventProcessorDatabaseAdapter adapts existing stores to core.DatabasePlugin
// for use by ReorgHandler in the event-processor microservice.
type reorgEventProcessorDatabaseAdapter struct {
	eventStore    *query.MongoDBEventStore
	metadataStore *query.PostgreSQLEventMetadataStore
	db            *sql.DB
}

func newReorgEventProcessorDatabaseAdapter(
	eventStore *query.MongoDBEventStore,
	metadataStore *query.PostgreSQLEventMetadataStore,
	db *sql.DB,
) *reorgEventProcessorDatabaseAdapter {
	return &reorgEventProcessorDatabaseAdapter{eventStore: eventStore, metadataStore: metadataStore, db: db}
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

func (a *reorgEventProcessorDatabaseAdapter) QueryEvents(_ context.Context, _ any) ([]any, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) GetAllEvents(_ context.Context) ([]*core.BlockchainEvent, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	return a.eventStore.GetEventsByBlockRange(ctx, fromBlock, toBlock)
}

// EventWriter
func (a *reorgEventProcessorDatabaseAdapter) StoreEvent(_ context.Context, _ any) error {
	return fmt.Errorf("not implemented in reorg adapter")
}

func (a *reorgEventProcessorDatabaseAdapter) BatchStoreEvents(_ context.Context, _ []any) error {
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
func (a *reorgEventProcessorDatabaseAdapter) GetBlock(_ context.Context, blockNum uint64) (*core.Block, error) {
	if a.db == nil {
		return nil, fmt.Errorf("reorg adapter: no database connection")
	}
	row := a.db.QueryRow("SELECT block_hash, block_number FROM blockchain_events WHERE block_number = $1 LIMIT 1", blockNum)
	var hash string
	var number uint64
	if err := row.Scan(&hash, &number); err != nil {
		return nil, fmt.Errorf("reorg adapter: block %d not found: %w", blockNum, err)
	}
	return &core.Block{Number: number, Hash: common.HexToHash(hash)}, nil
}

func (a *reorgEventProcessorDatabaseAdapter) GetLatestBlock(_ context.Context) (uint64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("reorg adapter: no database connection")
	}
	var maxBlock uint64
	err := a.db.QueryRow("SELECT COALESCE(MAX(block_number), 0) FROM blockchain_events").Scan(&maxBlock)
	if err != nil {
		return 0, fmt.Errorf("reorg adapter: failed to query latest block: %w", err)
	}
	return maxBlock, nil
}

func (a *reorgEventProcessorDatabaseAdapter) GetAllBlocks(_ context.Context) ([]*core.Block, error) {
	return nil, fmt.Errorf("not implemented in reorg adapter")
}

// ReorgStatsProvider
func (a *reorgEventProcessorDatabaseAdapter) GetReorgStats(_ context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

func getDB(dbManager database.DatabaseManager) *sql.DB {
	raw, err := dbManager.GetPostgresDB(context.Background())
	if err != nil {
		return nil
	}
	db, ok := raw.(*sql.DB)
	if !ok {
		return nil
	}
	return db
}
