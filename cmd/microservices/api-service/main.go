//nolint:funlen,wsl,nlreturn,godot // Transitional bootstrap; detailed refactor is tracked by phased architecture specs.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/application/bootstrap"
	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/plugins/mq"
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
	registry := core.NewPluginRegistry(logger)

	fmt.Println("  [1/3] Logger initialized")
	fmt.Println("  [2/3] Metrics collector initialized")
	fmt.Println("  [3/3] Plugin registry initialized")
	fmt.Println()

	// Build shared runtime wiring
	fmt.Println("Building Runtime Wiring:")
	runtimeWiring, err := bootstrap.BuildRuntimeWiring(context.Background(), logger, metrics)
	if err != nil {
		logger.Error("Failed to build runtime wiring", "error", err.Error())
		os.Exit(1)
	}
	fmt.Printf("  MongoDB URI:        %s\n", runtimeWiring.DBConfig.MongoDBURI)
	fmt.Printf("  PostgreSQL URL:     %s\n", runtimeWiring.DBConfig.PostgresURL)
	fmt.Printf("  Pool Size:          %d\n", runtimeWiring.DBConfig.PoolSize)
	fmt.Printf("  Timeout:            %dms\n", runtimeWiring.DBConfig.TimeoutMS)
	fmt.Println("  ✓ Query service initialized")
	fmt.Println("  ✓ Event query handler initialized")
	fmt.Println("  ✓ Event subscription handler initialized")
	fmt.Println("  ✓ Health check handler initialized")
	fmt.Println()

	// Convert configuration to core.Config
	coreCfg := bootstrap.NewAPIServiceCoreConfig(config.Port, config.LogLevel)
	envOverrides, err := bootstrap.ParseCoreConfigOverridesFromEnv()
	if err != nil {
		logger.Error("Failed to parse core config overrides", "error", err.Error())
		os.Exit(1)
	}
	cliOverrides, err := bootstrap.ParseCoreConfigOverridesFromCLI(os.Args[1:])
	if err != nil {
		logger.Error("Failed to parse CLI core config overrides", "error", err.Error())
		os.Exit(1)
	}
	coreOverrides := bootstrap.MergeCoreConfigOverrides(envOverrides, cliOverrides)
	runtimeProfile := bootstrap.RuntimeProfileFromEnv()
	policy := bootstrap.ResolveOverridePolicyRuntime(runtimeProfile)
	enforcementMode := bootstrap.ResolvePolicyEnforcementModeFromEnv()
	policyEval, err := bootstrap.ValidateCoreConfigOverridesWithMode(coreOverrides, runtimeProfile, policy, enforcementMode)
	if err != nil {
		logger.Error(
			"Core config overrides rejected by policy",
			"profile", runtimeProfile,
			"policy_enforcement", policyEval.EnforcementMode,
			"policy_code", bootstrap.PolicyErrorCode(err),
			"error", err.Error(),
		)
		os.Exit(1)
	}
	if policyEval.Violation {
		logger.Warn(
			"Core config overrides policy violation accepted in audit mode",
			"profile", runtimeProfile,
			"policy_enforcement", policyEval.EnforcementMode,
			"policy_code", policyEval.ViolationCode,
		)
	}
	coreCfg = bootstrap.ApplyCoreConfigOverrides(coreCfg, coreOverrides)
	logger.Info("Core config overrides applied", "overrides", bootstrap.SummarizeCoreConfigOverrides(coreOverrides))
	metricSchemaMode := bootstrap.ResolvePolicyMetricSchemaModeFromEnv()
	bootstrap.EmitPolicyOverrideMetrics(
		metrics,
		runtimeProfile,
		envOverrides,
		cliOverrides,
		coreOverrides,
		policy,
		policyEval,
		metricSchemaMode,
	)
	coreConfig := &coreCfg

	// Initialize API Service Plugin
	fmt.Println("Initializing API Service:")
	service := api.NewAPIGatewayPlugin(logger, metrics)
	authMiddleware, rateLimitMiddleware, err := buildAPIServiceSecurityControls(config, logger, metrics)
	if err != nil {
		logger.Error("Failed to build API Service security controls", "error", err.Error())
		os.Exit(1)
	}
	service.SetAuthMiddleware(authMiddleware)
	service.SetRateLimitMiddleware(rateLimitMiddleware)
	service.SetDomainQueryService(runtimeWiring.DomainQueryService)
	service.SetEventQueryHandler(runtimeWiring.EventQueryHandler)
	service.SetEventSubscriptionHandler(runtimeWiring.EventSubscriptionHandler)
	service.SetHealthCheckHandler(runtimeWiring.HealthCheckHandler)
	service.SetGraphQLHandler(runtimeWiring.GraphQLHandler)

	// Wire DLQ handler using PostgreSQL
	if runtimeWiring.DBManager != nil {
		if pgDB, err := runtimeWiring.DBManager.GetPostgresDB(context.Background()); err == nil {
			if sqlDB, ok := pgDB.(*sql.DB); ok {
				dlqHandler := api.NewDLQHandler(sqlDB, nil, logger, metrics)
				service.SetDLQHandler(dlqHandler)
			}
		}
	}
	service.SetRuntimeSummaryProvider(buildAPIServiceRuntimeSummaryProvider(config.InstanceID, metrics, service, runtimeWiring.QueryService))
	service.SetRuntimeMetricsProvider(buildAPIServiceMetricsProvider(metrics))
	if err := service.Initialize(*coreConfig); err != nil {
		logger.Error("Failed to initialize API Service", "error", err.Error())
		os.Exit(1)
	}
	runtimeWiring.HealthCheckHandler.SetRolloutReportProducer(newAPIServiceRolloutReportProducer(config.InstanceID, func() apiServiceRolloutRuntimeState {
		queryServiceHealth := runtimeWiring.QueryService.Health(context.Background())
		return apiServiceRolloutRuntimeState{
			DomainBridgeEnabled:      service.IsDomainBridgeEnabled(),
			EventQueryEnabled:        service.IsEventQueryHandlerEnabled(),
			EventSubscriptionEnabled: service.IsEventSubscriptionHandlerEnabled(),
			HealthCheckRoutesEnabled: service.IsHealthCheckHandlerEnabled(),
			QueryServiceMessage:      queryServiceHealth.Message,
			QueryServiceStatus:       queryServiceHealth.Status,
			RuntimeRoutesEnabled:     service.IsRuntimeRoutesEnabled(),
		}
	}))
	fmt.Println("  ✓ API Service initialized")
	fmt.Println("  ✓ Rollout report producer initialized")
	if service.IsDomainBridgeEnabled() {
		fmt.Println("  ✓ Domain query bridge configured")
	}
	if service.IsEventQueryHandlerEnabled() {
		fmt.Println("  ✓ Event query handler wired")
	}
	if service.IsRuntimeRoutesEnabled() {
		fmt.Println("  ✓ Runtime route composition enabled")
	}
	if service.IsAuthMiddlewareEnabled() {
		fmt.Println("  ✓ Security auth middleware enabled")
	}
	if service.IsRateLimitMiddlewareEnabled() {
		fmt.Println("  ✓ Security rate limit middleware enabled")
	}
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
		if err := runtimeWiring.QueryService.Start(context.Background()); err != nil {
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

	// Start WebSocket event push consumer (listens for processed events and pushes to WebSocket clients)
	var kafkaWG sync.WaitGroup
	pushCancel := func() {}
	if len(config.KafkaBrokers) > 0 && runtimeWiring.EventSubscriptionHandler != nil {
		pushCtx, cancel := context.WithCancel(context.Background())
		pushCancel = cancel
		startEventPushConsumer(pushCtx, &kafkaWG, config, runtimeWiring.EventSubscriptionHandler, logger, metrics)
		fmt.Println("  [3/3] WebSocket event push consumer started")
	} else {
		fmt.Println("  [3/3] WebSocket event push consumer skipped (no Kafka brokers)")
	}
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

	// Stop API Service (graceful shutdown with context)
	if err := service.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error("Error stopping API Service", "error", err.Error())
	}
	fmt.Println("  [1/3] API Service stopped")

	// Stop WebSocket push consumer
	pushCancel()
	pushDone := make(chan struct{})
	go func() {
		kafkaWG.Wait()
		close(pushDone)
	}()
	select {
	case <-pushDone:
	case <-time.After(5 * time.Second):
	}
	fmt.Println("  [2/3] WebSocket push consumer stopped")

	if err := runtimeWiring.Close(shutdownCtx); err != nil {
		logger.Error("Error closing runtime wiring", "error", err.Error())
	}
	fmt.Println("  [3/3] Runtime wiring closed")

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
	Port               int
	InstanceID         string
	DatabaseHost       string
	DatabasePort       int
	RedisCluster       []string
	KafkaBrokers       []string
	ConsumerGroup      string
	LogLevel           string
	AuthEnabled        bool
	AuthJWTSecret      string
	AuthAPIKeys        []string
	RateLimitEnabled   bool
	RateLimitPerMinute int
}

// loadAPIServiceConfig loads configuration from environment variables
func loadAPIServiceConfig() APIServiceConfig {
	instanceID := getEnv("HOSTNAME", "api-service-1")
	if id := getEnv("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return APIServiceConfig{
		Port:               getEnvInt("API_SERVICE_PORT", 8081),
		InstanceID:         instanceID,
		DatabaseHost:       getEnv("API_SERVICE_DB_HOST", "postgres-primary"),
		DatabasePort:       getEnvInt("API_SERVICE_DB_PORT", 5432),
		RedisCluster:       []string{"redis-1:6379", "redis-2:6379", "redis-3:6379"},
		KafkaBrokers:       parseCommaSeparatedList(getEnv("API_SERVICE_KAFKA_BROKERS", "kafka:9092")),
		ConsumerGroup:      getEnv("API_SERVICE_KAFKA_CONSUMER_GROUP", "api-service-consumers"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		AuthEnabled:        parseBoolEnv("API_SERVICE_AUTH_ENABLED", false),
		AuthJWTSecret:      getEnv("API_SERVICE_AUTH_JWT_SECRET", ""),
		AuthAPIKeys:        parseCommaSeparatedList(getEnv("API_SERVICE_AUTH_API_KEYS", "")),
		RateLimitEnabled:   parseBoolEnv("API_SERVICE_RATE_LIMIT_ENABLED", false),
		RateLimitPerMinute: getEnvInt("API_SERVICE_RATE_LIMIT", 100),
	}
}

func buildAPIServiceSecurityControls(config APIServiceConfig, logger core.Logger, metrics core.MetricsCollector) (*api.AuthMiddleware, *api.RateLimitMiddleware, error) {
	if !config.AuthEnabled && !config.RateLimitEnabled {
		return nil, nil, nil
	}

	var authMiddleware *api.AuthMiddleware
	if config.AuthEnabled {
		if strings.TrimSpace(config.AuthJWTSecret) == "" {
			return nil, nil, fmt.Errorf("api service auth is enabled but API_SERVICE_AUTH_JWT_SECRET is empty")
		}

		tokenValidator := api.NewTokenValidator(config.AuthJWTSecret, logger, metrics)
		for _, entry := range config.AuthAPIKeys {
			apiKey, clientID, ok := parseKeyValuePair(entry)
			if !ok {
				return nil, nil, fmt.Errorf("invalid API_SERVICE_AUTH_API_KEYS entry %q; expected key=clientID or key:clientID", entry)
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

func parseCommaSeparatedList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

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

// startEventPushConsumer starts a Kafka consumer that listens for processed events
// and pushes them to WebSocket subscribers via BroadcastEvent().
func startEventPushConsumer(
	ctx context.Context,
	wg *sync.WaitGroup,
	config APIServiceConfig,
	subscriptionHandler *api.EventSubscriptionHandler,
	logger core.Logger,
	metrics core.MetricsCollector,
) {
	kafkaMQ := mq.NewKafkaMQPlugin(
		"api-service-push-consumer",
		"1.0.0",
		&core.Config{
			APIType:  "kafka",
			LogLevel: config.LogLevel,
		},
		logger,
		metrics,
		nil, // eventBus not needed
		config.KafkaBrokers,
		config.ConsumerGroup,
	)
	if err := kafkaMQ.Initialize(); err != nil {
		logger.Error("Failed to initialize WebSocket push Kafka consumer", "error", err.Error())
		return
	}
	if err := kafkaMQ.Start(); err != nil {
		logger.Error("Failed to start WebSocket push Kafka consumer", "error", err.Error())
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer kafkaMQ.Stop() //nolint:errcheck // deferred stop

		topic := "processed-events"
		err := kafkaMQ.ConsumeMessages(ctx, topic, func(message core.MessageQueueMessage) error {
			var event core.BlockchainEvent
			if err := json.Unmarshal(message.Payload, &event); err != nil {
				logger.Warn("Failed to unmarshal event for WebSocket push", "error", err.Error())
				return nil // skip malformed messages
			}
			if err := subscriptionHandler.BroadcastEvent(ctx, &event); err != nil {
				logger.Warn("Failed to broadcast event to WebSocket clients", "eventId", event.ID, "error", err.Error())
			}
			return nil
		})
		if err != nil && ctx.Err() == nil {
			logger.Error("WebSocket push consumer stopped with error", "error", err.Error())
		}
	}()
}
