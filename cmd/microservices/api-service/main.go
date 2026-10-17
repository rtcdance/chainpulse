//nolint:funlen,wsl,nlreturn,godot // Transitional bootstrap; detailed refactor is tracked by phased architecture specs.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ChainPulse - API Service                           ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration from environment
	config := loadAPIServiceConfig()

	if err := validateAPIServiceConfig(config); err != nil {
		fmt.Printf("config validation failed: %v\n", err)
		os.Exit(1)
	}

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
	metrics := observability.Guarded(core.NewDefaultMetricsCollector())
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
	fmt.Printf("  MongoDB URI:        %s\n", core.RedactURL(runtimeWiring.DBConfig.MongoDBURI))
	fmt.Printf("  PostgreSQL URL:     %s\n", core.RedactURL(runtimeWiring.DBConfig.PostgresURL))
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
	if err := validateAPIServiceProductionSecurity(config, runtimeProfile); err != nil {
		logger.Error("API Service production security gate rejected startup", "profile", runtimeProfile, "error", err.Error())
		os.Exit(1)
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
	authMiddleware, rateLimitMiddleware, err := bootstrap.BuildSecurityControls(bootstrap.SecurityControlsConfig{
		AuthEnabled:        config.AuthEnabled,
		AuthJWTSecret:      config.AuthJWTSecret,
		AuthAPIKeys:        config.AuthAPIKeys,
		RateLimitEnabled:   config.RateLimitEnabled,
		RateLimitPerMinute: config.RateLimitPerMinute,
		ServiceName:        "api service",
		EnvPrefix:          "API_SERVICE",
	}, logger, metrics)
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

	// Wire stats handler for event statistics dashboard
	if runtimeWiring.EventRetrievalService != nil {
		eventReader := runtimeWiring.EventRetrievalService.GetEventReader()
		statsHandler := api.NewStatsHandler(logger, eventReader)
		service.SetStatsHandler(statsHandler)
	}

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

	// Query Start is nonblocking. Its critical initialization must succeed
	// before HTTP is exposed; the context belongs to the runtime lifetime.
	queryCtx, queryCancel := context.WithCancel(context.Background())
	if err := runtimeWiring.QueryService.Start(queryCtx); err != nil {
		queryCancel()
		logger.Error("Query Service startup failed", "error", err.Error())
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if closeErr := runtimeWiring.Close(cleanupCtx); closeErr != nil {
			logger.Error("Failed to close runtime after startup failure", "error", closeErr.Error())
		}
		cancel()
		os.Exit(1)
	}
	fmt.Println("  [1/2] Query Service started")

	var kafkaWG sync.WaitGroup
	pushCancel := func() {}
	if len(config.KafkaBrokers) > 0 && runtimeWiring.EventSubscriptionHandler != nil {
		pushCtx, cancelPush := context.WithCancel(queryCtx)
		pushCancel = cancelPush
		var ready func(context.Context) bool
		var pushErr error
		if source, ok := runtimeWiring.QueryService.(subscriptionQuerySource); ok {
			ready, pushErr = startEventPushConsumer(pushCtx, &kafkaWG, config, runtimeWiring.EventSubscriptionHandler, logger, metrics, source.SubscriptionStore(), source.SubscriptionChains())
		} else {
			pushErr = fmt.Errorf("native subscriptions require the canonical PostgreSQL query runtime")
		}
		if pushErr != nil {
			pushCancel()
			queryCancel()
			logger.Error("Subscription consumer initialization failed", "error", pushErr.Error())
			cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			if closeErr := runtimeWiring.Close(cleanupCtx); closeErr != nil {
				logger.Error("Close failed subscription startup", "error", closeErr.Error())
			}
			stop()
			os.Exit(1)
		}
		runtimeWiring.HealthCheckHandler.SetReadinessCheck(func(ctx context.Context) bool {
			health := runtimeWiring.QueryService.Health(ctx)
			return health != nil && health.Status == "healthy" && ready(ctx)
		})
		fmt.Println("  [2/3] Committed/reorg subscription consumer initialized")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := service.Start(); err != nil {
			logger.Error("API Service error", "error", err.Error())
		}
	}()
	fmt.Println("  [3/3] API Service started")
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
	_ = bootstrap.WaitForSignal()

	// Graceful shutdown
	fmt.Println("Shutting Down Services:")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pushCancel()
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
	case <-shutdownCtx.Done():
		logger.Error("Subscription consumers exceeded shutdown deadline")
		os.Exit(1)
	}
	fmt.Println("  [2/3] WebSocket push consumer stopped")

	queryCancel()
	if err := runtimeWiring.Close(shutdownCtx); err != nil {
		logger.Error("Error closing runtime wiring", "error", err.Error())
	}
	fmt.Println("  [3/3] Runtime wiring closed")

	bootstrap.ShutdownWithTimeout(&wg, 30*time.Second)

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
	CommittedTopic     string
	ReorgTopic         string
	LogLevel           string
	AuthEnabled        bool
	AuthJWTSecret      core.SecretString
	AuthAPIKeys        []core.SecretString
	RateLimitEnabled   bool
	RateLimitPerMinute int
}

// loadAPIServiceConfig loads configuration from environment variables
func loadAPIServiceConfig() APIServiceConfig {
	instanceID := env.Get("HOSTNAME", "api-service-1")
	if id := env.Get("INSTANCE_ID", ""); id != "" {
		instanceID = id
	}

	return APIServiceConfig{
		Port:               env.GetInt("API_SERVICE_PORT", 8081),
		InstanceID:         instanceID,
		DatabaseHost:       env.Get("API_SERVICE_DB_HOST", "postgres-primary"),
		DatabasePort:       env.GetInt("API_SERVICE_DB_PORT", 5432),
		RedisCluster:       []string{"redis-1:6379", "redis-2:6379", "redis-3:6379"},
		KafkaBrokers:       env.GetCSV("API_SERVICE_KAFKA_BROKERS", []string{"kafka:9092"}),
		ConsumerGroup:      env.Get("API_SERVICE_KAFKA_CONSUMER_GROUP", "api-service-consumers"),
		CommittedTopic:     env.Get("KAFKA_COMMITTED_TOPIC", "chainpulse.committed-blocks.v1"),
		ReorgTopic:         env.Get("KAFKA_REORG_TOPIC", "chainpulse.reorg.v1"),
		LogLevel:           env.Get("LOG_LEVEL", "info"),
		AuthEnabled:        env.GetBool("API_SERVICE_AUTH_ENABLED", false),
		AuthJWTSecret:      core.SecretString(env.Get("API_SERVICE_AUTH_JWT_SECRET", "")),
		AuthAPIKeys:        core.ToSecretStrings(env.GetCSV("API_SERVICE_AUTH_API_KEYS", nil)),
		RateLimitEnabled:   env.GetBool("API_SERVICE_RATE_LIMIT_ENABLED", true),
		RateLimitPerMinute: env.GetInt("API_SERVICE_RATE_LIMIT", 100),
	}
}

func validateAPIServiceConfig(c APIServiceConfig) error {
	if c.DatabasePort < 1 || c.DatabasePort > 65535 {
		return fmt.Errorf("API_SERVICE_DB_PORT must be between 1 and 65535, got %d", c.DatabasePort)
	}
	if c.RateLimitPerMinute < 1 {
		return fmt.Errorf("API_SERVICE_RATE_LIMIT must be >= 1, got %d", c.RateLimitPerMinute)
	}
	if c.AuthEnabled && strings.TrimSpace(c.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("API_SERVICE_AUTH_JWT_SECRET must not be empty when API_SERVICE_AUTH_ENABLED is true")
	}
	return nil
}

func validateAPIServiceProductionSecurity(c APIServiceConfig, runtimeProfile string) error {
	if runtimeProfile != "production" {
		return nil
	}
	if !c.AuthEnabled {
		return fmt.Errorf("production api service requires API_SERVICE_AUTH_ENABLED=true")
	}
	if strings.TrimSpace(c.AuthJWTSecret.Value()) == "" {
		return fmt.Errorf("production api service requires non-empty API_SERVICE_AUTH_JWT_SECRET")
	}
	if !c.RateLimitEnabled {
		return fmt.Errorf("production api service requires API_SERVICE_RATE_LIMIT_ENABLED=true")
	}
	if c.RateLimitPerMinute < 1 {
		return fmt.Errorf("production api service requires API_SERVICE_RATE_LIMIT >= 1")
	}
	return nil
}
