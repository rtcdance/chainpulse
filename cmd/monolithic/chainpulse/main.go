package main

//nolint:funlen // Main function is inherently complex bootstrap; refactor would go beyond minimal fix.

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"chainpulse/pkg/application/bootstrap"
	"chainpulse/pkg/core"
	"chainpulse/pkg/observability"
	"chainpulse/pkg/plugins/api"
	sharedhttp "chainpulse/pkg/infrastructure/http"
	"chainpulse/pkg/services/indexing"
)

//nolint:gocyclo // Monolithic entrypoint orchestrates many subsystems; keep linear startup/shutdown flow for ops visibility.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	fmt.Printf("  Deployment Mode:     %s\n", config.DeploymentMode)
	fmt.Printf("  Adapter Profile:     %s\n", config.AdapterProfile)
	fmt.Printf("  Transport Boundary:  %s\n", config.TransportAdapterBoundary)
	fmt.Printf("  Chains:              %s\n", config.Chains)
	fmt.Printf("  Data Puller Type:    %s\n", config.DataPullerType)
	fmt.Printf("  Blockchain Nodes:    %s\n", config.BlockchainNodeURLs)
	fmt.Printf("  Message Queue Type:  %s\n", config.MQType)
	fmt.Printf("  Cache Type:          %s\n", config.CacheType)
	fmt.Printf("  Database Type:       %s\n", config.DatabaseType)
	fmt.Printf("  DLQ Retention:       %s\n", config.DLQRetention)
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

	logFormat := getEnv("LOG_FORMAT", "slog")
	var logger core.Logger
	if logFormat == "legacy" {
		logger = core.NewDefaultLogger(logLevel)
	} else {
		logger = core.NewSlogLogger(logLevel, logFormat)
	}
	metrics := core.NewDefaultMetricsCollector()
	registry := core.NewPluginRegistry(logger)

	// Create shared observability provider (single TracerProvider)
	obsProvider, err := observability.NewObservabilityProvider(
		observability.ObservabilityConfig{ServiceName: "chainpulse-monolithic"},
		logger,
	)
	if err != nil {
		logger.Warn("Observability provider initialization failed, tracing disabled", "error", err.Error())
	}

	fmt.Println("  [1/4] Logger initialized")
	fmt.Println("  [2/4] Metrics collector initialized")
	fmt.Println("  [3/4] Plugin registry initialized")
	fmt.Println("  [4/4] Observability provider initialized")
	fmt.Println()

	// Build shared runtime wiring
	fmt.Println("Building Runtime Wiring:")
	runtimeWiring, err := bootstrap.BuildRuntimeWiring(ctx, logger, metrics)
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
	coreCfg := bootstrap.NewMonolithicCoreConfig(
		config.LogLevel,
		config.DatabaseType,
		config.DatabaseURL,
		config.CacheType,
	)
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

	// Validate the resolved core configuration
	configManager := core.NewConfigManager(logger)
	if err := configManager.Validate(*coreConfig); err != nil {
		logger.Error("Core configuration validation failed", "error", err.Error())
		os.Exit(1)
	}

	chains, err := parseChains(config.Chains)
	if err != nil {
		logger.Error("Failed to parse chain configuration", "error", err.Error())
		os.Exit(1)
	}

	// Build monolithic indexing storage
	fmt.Println("Initializing Indexing Storage:")
	indexingDatabase, indexingCache, err := bootstrap.BuildMonolithicIndexingStorage(logger, *coreConfig)
	if err != nil {
		logger.Error("Failed to build monolithic indexing storage", "error", err.Error())
		os.Exit(1)
	}
	logger.Info(
		"Monolithic indexing storage started",
		"service", "monolithic",
		"database", indexingDatabase.Name(),
		"cache", indexingCache.Name(),
	)
	fmt.Printf("  ✓ Database plugin started: %s\n", indexingDatabase.Name())
	fmt.Printf("  ✓ Cache plugin started: %s\n", indexingCache.Name())
	fmt.Println()

	// Build indexing-backed monolithic query surface
	fmt.Println("Initializing Monolithic Query Surface:")
	monolithicQuerySurface, err := resolveMonolithicQuerySurface(ctx, config, runtimeWiring, indexingDatabase, logger, metrics)
	if err != nil {
		logger.Error("Failed to build monolithic query surface", "error", err.Error())
		os.Exit(1)
	}
	if monolithicQuerySurface.domainQuery != nil {
		runtimeWiring.DomainQueryService = monolithicQuerySurface.domainQuery
	}
	if monolithicQuerySurface.eventRetrievalService != nil {
		runtimeWiring.EventRetrievalService = monolithicQuerySurface.eventRetrievalService
	}
	if monolithicQuerySurface.eventQueryHandler != nil {
		runtimeWiring.EventQueryHandler = monolithicQuerySurface.eventQueryHandler
	}
	if monolithicQuerySurface.eventSubscriptionHandler != nil {
		runtimeWiring.EventSubscriptionHandler = monolithicQuerySurface.eventSubscriptionHandler
	}
	if config.DeploymentMode == deploymentModeMicroservice {
		fmt.Println("  ✓ Managed-db/shared runtime query surface retained for microservice intent")
	} else {
		fmt.Println("  ✓ Indexing-backed event retrieval initialized")
		fmt.Println("  ✓ Monolithic event query handler aligned to indexing storage")
	}
	fmt.Println()

	// Build additive shared indexing runtime
	fmt.Println("Initializing Shared Indexing Runtime:")

	dlqRetention, err := parseMonolithicDLQRetention(config.DLQRetention)
	if err != nil {
		logger.Error("Failed to parse monolithic DLQ retention", "error", err.Error())
		os.Exit(1)
	}

	sharedIndexingRuntime, err := bootstrap.BuildMonolithicIndexingRuntimeWithOptions(
		logger,
		indexingDatabase,
		indexingCache,
		chains,
		bootstrap.InMemoryIndexingRuntimeOptions{DLQRetention: dlqRetention},
	)
	if err != nil {
		logger.Error("Failed to build shared indexing runtime", "error", err.Error())
		os.Exit(1)
	}
	if err := sharedIndexingRuntime.Initialize(ctx); err != nil {
		logger.Error("Failed to initialize shared indexing runtime", "error", err.Error())
		os.Exit(1)
	}
	if err := sharedIndexingRuntime.Start(ctx); err != nil {
		logger.Error("Failed to start shared indexing runtime", "error", err.Error())
		os.Exit(1)
	}
	for _, chainID := range chains {
		if err := sharedIndexingRuntime.RecoverChain(ctx, chainID); err != nil {
			logger.Warn("Shared indexing runtime recovery probe failed", "service", "monolithic", "chain_id", chainID, "error", err.Error())
		}
	}
	sharedRuntimeStatus := sharedIndexingRuntime.Status()
	metrics.RecordGauge("indexing_runtime_started", 1, map[string]string{
		"service":   "monolithic",
		"operation": "startup",
	})
	metrics.RecordGauge("indexing_runtime_chain_count", float64(len(sharedRuntimeStatus.Chains)), map[string]string{
		"service":   "monolithic",
		"operation": "startup",
	})
	logger.Info(
		"Shared indexing runtime started",
		"service", "monolithic",
		"state", sharedRuntimeStatus.State,
		"chains", strings.Join(sharedRuntimeStatus.Chains, ","),
		"recovery_state", sharedRuntimeStatus.RecoveryState,
		"dlq_retention", dlqRetention.String(),
	)
	fmt.Printf("  ✓ Shared runtime started (%s)\n", sharedRuntimeStatus.State)
	fmt.Printf("  ✓ Shared runtime chains: %s\n", strings.Join(sharedRuntimeStatus.Chains, ","))
	fmt.Printf("  ✓ Shared runtime recovery: %s\n", sharedRuntimeStatus.RecoveryState)
	fmt.Printf("  ✓ Shared runtime DLQ retention: %s\n", dlqRetention.String())
	fmt.Println()

	// Initialize Multi-Chain Indexer
	fmt.Println("Initializing Multi-Chain Indexer:")
	multiChainIndexer := indexing.NewMultiChainIndexer(logger, nil)
	for _, chainID := range chains {
		chainIndexer := indexing.NewDefaultChainIndexer(
			chainID,
			indexingDatabase,
			indexingCache,
			logger,
			nil,
		)
		chainIndexer.SetSharedRuntime(sharedIndexingRuntime, metrics)
		if err := multiChainIndexer.RegisterChainIndexer(chainID, chainIndexer); err != nil {
			logger.Error("Failed to register chain indexer", "chain_id", chainID, "error", err.Error())
			os.Exit(1)
		}
		fmt.Printf("  ✓ Chain indexer registered: %s\n", chainID)
	}
	runtimeWiring.HealthCheckHandler.SetRuntimeComponentProvider(func(ctx context.Context) *api.ComponentStatus {
		_ = ctx
		return buildOwnershipHealthComponent(multiChainIndexer.GetStatus(), time.Now())
	})
	runtimeWiring.HealthCheckHandler.SetReadinessDetailsProvider(func(ctx context.Context) map[string]interface{} {
		_ = ctx
		return buildOwnershipReadinessDetails(multiChainIndexer.GetStatus())
	})
	runtimeWiring.HealthCheckHandler.SetRolloutReportProducer(api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx
		return buildOwnershipRolloutSummary(multiChainIndexer.GetStatus()).reportDetails()
	}))
	fmt.Println()

	// Initialize monolithic puller runtime closure
	fmt.Println("Initializing Monolithic Puller Runtime:")

	monolithicPullerRuntime, err := newMonolithicPullerRuntime(ctx, *coreConfig, config.BlockchainNodeURLs, chains, logger, metrics, indexingDatabase, multiChainIndexer)
	if err != nil {
		logger.Error("Failed to initialize monolithic puller runtime", "error", err.Error())
		os.Exit(1)
	}

	fmt.Printf("  ✓ Event bus initialized (%d subscribers)\n", monolithicPullerRuntime.SubscriberCount())
	fmt.Printf("  ✓ Pullers initialized: %d\n", monolithicPullerRuntime.PullerCount())
	fmt.Println()

	// Initialize API Gateway Plugin
	fmt.Println("Initializing API Gateway:")
	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gatewaySurface := resolveMonolithicGatewaySurface(config)
	applyMonolithicGatewaySurface(gateway, gatewaySurface, gatewayRuntimeWiring{
		domainQueryService:       runtimeWiring.DomainQueryService,
		eventQueryHandler:        runtimeWiring.EventQueryHandler,
		eventSubscriptionHandler: runtimeWiring.EventSubscriptionHandler,
		healthCheckHandler:       runtimeWiring.HealthCheckHandler,
		upstreamQueryEndpoints:   config.UpstreamQueryServices,
	})
	gateway.SetRuntimeMetricsProvider(buildMonolithicMetricsProvider(metrics, nil))
	gateway.SetRuntimeSummaryProvider(buildMonolithicRuntimeSummaryProvider(metrics, gateway, sharedIndexingRuntime, multiChainIndexer, monolithicPullerRuntime, monolithicPullerRuntime, monolithicQuerySurface, config))
	gateway.SetRuntimeControlProvider(monolithicPullerRuntime.HandleRuntimeControl)
	gateway.SetRuntimeReplayProvider(newMonolithicDLQReplayHandler(sharedIndexingRuntime))
	if runtimeWiring.GraphQLHandler != nil {
		gateway.SetGraphQLHandler(runtimeWiring.GraphQLHandler)
	}
	if config.RateLimitEnabled {
		rateLimiter := api.NewRateLimiter(logger, metrics, &api.RateLimitConfig{
			DefaultRequestsPerSecond: api.RequestsPerMinuteToPerSecond(config.RateLimitPerMinute),
			DefaultBurstSize:         api.BurstSizeFromRequestsPerMinute(config.RateLimitPerMinute),
			CleanupInterval:          5 * time.Minute,
		})
		rateLimitMiddleware := api.NewRateLimitMiddleware(rateLimiter, logger)
		gateway.SetRateLimitMiddleware(rateLimitMiddleware)
		logger.Info("Rate limit middleware enabled", "requests_per_minute", config.RateLimitPerMinute)
	}
	if err := gateway.Initialize(*coreConfig); err != nil {
		logger.Error("Failed to initialize API Gateway", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ API Gateway initialized")
	fmt.Printf("  ✓ Gateway Surface Mode: %s\n", gatewaySurface.SurfaceMode)
	if len(config.UpstreamQueryServices) > 0 && gatewaySurface.SurfaceMode == "upstream-query-bridge" {
		fmt.Printf("  ✓ Upstream query bridge endpoints: %d\n", len(config.UpstreamQueryServices))
	}
	if gateway.IsDomainBridgeEnabled() {
		fmt.Println("  ✓ Domain query bridge configured")
	}
	if gateway.IsEventQueryHandlerEnabled() {
		fmt.Println("  ✓ Event query handler wired")
	}

	if config.TLSEnabled && config.TLSCertPath != "" && config.TLSKeyPath != "" {
		logger.Info("TLS enabled", "cert", config.TLSCertPath)
		fmt.Println("  ✓ TLS configured")
	}

	if config.JWTSecret != "" {
		if len(config.JWTSecret) < 32 {
			logger.Warn("JWT secret too short, minimum 32 characters recommended")
		}
		fmt.Println("  ✓ JWT authentication configured")
	} else {
		logger.Info("JWT secret not set, auth disabled")
	}

	// Build and inject auth middleware if authentication is enabled
	if config.AuthEnabled && config.JWTSecret != "" {
		tokenValidator := api.NewTokenValidator(config.JWTSecret, logger, metrics)
		for _, entry := range config.AuthAPIKeys {
			apiKey, clientID, ok := parseMonolithicKeyPair(entry)
			if !ok {
				logger.Warn("invalid CHAINPULSE_AUTH_API_KEYS entry; expected key=clientID or key:clientID", "entry", entry)
				continue
			}
			if err := tokenValidator.RegisterAPIKey(apiKey, clientID, "operator"); err != nil {
				logger.Warn("failed to register API key", "error", err.Error())
			}
		}
		rbacChecker := api.NewRBACChecker(logger, metrics)
		if err := rbacChecker.RegisterDefaultRoles(); err != nil {
			logger.Warn("failed to register default RBAC roles", "error", err.Error())
		}
		auditLogger := api.NewAuditLogger(logger, metrics)
		authMiddleware := api.NewAuthMiddleware(tokenValidator, rbacChecker, auditLogger, logger, metrics)
		gateway.SetAuthMiddleware(authMiddleware)
		logger.Info("Auth middleware enabled for monolithic mode")
		fmt.Println("  ✓ Auth middleware wired to API Gateway")
	}

	if gateway.IsRuntimeRoutesEnabled() {
		fmt.Println("  ✓ Runtime route composition enabled")
	}
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

	pullerCtx, stopPullers := context.WithCancel(ctx)
	defer stopPullers()

	// Start Query Service
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runtimeWiring.QueryService.Start(ctx); err != nil {
			logger.Error("Query Service error", "error", err.Error())
		}
	}()
	fmt.Println("  [1/2] Query Service started")

	// Start API Gateway
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := core.StartPlugin(ctx, gateway); err != nil {
			logger.Error("API Gateway error", "error", err.Error())
		}
	}()
	fmt.Println("  [2/2] API Gateway started")

	if err := monolithicPullerRuntime.Start(pullerCtx, &wg); err != nil {
		logger.Error("Failed to start monolithic puller runtime", "error", err.Error())
		stopPullers()
		os.Exit(1)
	}

	fmt.Printf("  [3/%d] Monolithic pullers started\n", monolithicPullerRuntime.PullerCount()+2)
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	if gateway.IsEventQueryHandlerEnabled() {
		fmt.Printf("GraphQL API available at: http://localhost:8080/graphql\n")
	}
	if gateway.IsEventSubscriptionHandlerEnabled() {
		fmt.Printf("GraphQL WebSocket available at: ws://localhost:8080/graphql\n")
	}
	fmt.Printf("Health Check available at: http://localhost:8080/health\n")
	fmt.Printf("Metrics available at: http://localhost:8080/metrics\n")
	if gateway.IsRuntimeRoutesEnabled() {
		fmt.Printf("Runtime Summary available at: http://localhost:8080/runtime/summary\n")
	}
	fmt.Printf("Indexed Chains: %s\n", config.Chains)
	rolloutSummary := buildOwnershipRolloutSummary(multiChainIndexer.GetStatus())
	emitOwnershipRolloutSummaryMetrics(metrics, rolloutSummary, "running")
	logOwnershipRolloutSummary(logger, "startup", rolloutSummary)
	printOwnershipRolloutSummary(os.Stdout, rolloutSummary, "running")
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
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	stopPullers()

	// Stop API Gateway
	if err := core.StopPlugin(shutdownCtx, gateway); err != nil {
		logger.Error("Error stopping API Gateway", "error", err.Error())
	}
	fmt.Println("  [1/3] API Gateway stopped")

	if err := monolithicPullerRuntime.Stop(); err != nil {
		logger.Error("Error stopping monolithic puller runtime", "error", err.Error())
	}

	fmt.Println("  [2/6] Monolithic puller runtime stopped")

	if err := sharedIndexingRuntime.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping shared indexing runtime", "error", err.Error())
	}
	metrics.RecordGauge("indexing_runtime_started", 0, map[string]string{
		"service":   "monolithic",
		"operation": "shutdown",
	})
	logger.Info("Shared indexing runtime stopped", "service", "monolithic", "state", sharedIndexingRuntime.Status().State)
	fmt.Println("  [3/6] Shared indexing runtime stopped")

	if err := indexingCache.Stop(); err != nil {
		logger.Error("Error stopping indexing cache", "error", err.Error())
	}
	if err := indexingDatabase.Stop(); err != nil {
		logger.Error("Error stopping indexing database", "error", err.Error())
	}

	fmt.Println("  [4/6] Indexing storage stopped")

	if err := runtimeWiring.Close(shutdownCtx); err != nil {
		logger.Error("Error closing runtime wiring", "error", err.Error())
	}

	fmt.Println("  [5/6] Runtime wiring closed")

	// Close multi-chain indexer
	if err := multiChainIndexer.Close(); err != nil {
		logger.Error("Error closing multi-chain indexer", "error", err.Error())
	}

	fmt.Println("  [6/6] Multi-chain indexer closed")

	// Shutdown observability provider (flush pending spans)
	if obsProvider != nil {
		if err := obsProvider.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error shutting down observability provider", "error", err.Error())
		}
		fmt.Println("  [7/7] Observability provider shut down")
	}

	// Close shared HTTP connection pool
	sharedhttp.DefaultSharedHTTPClient.CloseIdleConnections()
	fmt.Println("  Shared HTTP connection pool drained")

	finalRolloutSummary := buildOwnershipRolloutSummary(multiChainIndexer.GetStatus())
	emitOwnershipRolloutSummaryMetrics(metrics, finalRolloutSummary, "shutdown")
	logOwnershipRolloutSummary(logger, "shutdown", finalRolloutSummary)
	printOwnershipRolloutSummary(os.Stdout, finalRolloutSummary, "shutdown")

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
	DeploymentMode           string
	DeploymentPosture        string
	DeploymentHint           string
	AdapterProfile           string
	AdapterSelectionPosture  string
	AdapterSelectionHint     string
	IndexingStorageAdapter   string
	QueryRuntimeAdapter      string
	TransportAdapterBoundary string
	UpstreamQueryServices    []string
	Chains                   string
	DataPullerType           string
	BlockchainNodeURLs       string
	MQType                   string
	MQConnectionURL          string
	CacheType                string
	CacheConnectionURL       string
	DatabaseType             string
	DatabaseURL              string
	DatabaseSSLMode          string
	APIType                  string
	APIPort                  string
	WorkerPoolSize           string
	BatchSize                string
	LogLevel                 string
	DLQRetention             string
	RateLimitEnabled         bool
	RateLimitPerMinute       int
	TLSEnabled               bool
	TLSCertPath              string
	TLSKeyPath               string
	JWTSecret                string
	AuthEnabled              bool
	AuthAPIKeys              []string
}

// loadConfiguration loads configuration from environment variables
func loadConfiguration() Configuration {
	modeProfile := resolveDeploymentModeProfile(os.Getenv("DEPLOYMENT_MODE"))
	upstreamQueryServices := getEnvCSV("MONOLITHIC_UPSTREAM_QUERY_SERVICES", []string{"http://localhost:8081"})
	adapterProfile := resolveMonolithicAdapterProfile(modeProfile.Mode, upstreamQueryServices)

	return Configuration{
		DeploymentMode:           modeProfile.Mode,
		DeploymentPosture:        modeProfile.Posture,
		DeploymentHint:           modeProfile.ReliabilityHint,
		AdapterProfile:           adapterProfile.ProfileName,
		AdapterSelectionPosture:  adapterProfile.SelectionPosture,
		AdapterSelectionHint:     adapterProfile.ReliabilityHint,
		IndexingStorageAdapter:   adapterProfile.IndexingStorageAdapter,
		QueryRuntimeAdapter:      adapterProfile.QueryRuntimeAdapter,
		TransportAdapterBoundary: adapterProfile.TransportAdapterBoundary,
		UpstreamQueryServices:    upstreamQueryServices,
		Chains:                   getEnv("CHAINS", "ethereum,polygon"),
		DataPullerType:           getEnv("DATA_PULLER_TYPE", "https-jsonrpc"),
		BlockchainNodeURLs:       getEnv("BLOCKCHAIN_NODE_URLS", "http://localhost:8545,http://localhost:8546"),
		MQType:                   getEnv("MQ_TYPE", "kafka"),
		MQConnectionURL:          getEnv("MQ_CONNECTION_URL", "localhost:9092"),
		CacheType:                getEnv("CACHE_TYPE", "redis"),
		CacheConnectionURL:       getEnv("CACHE_CONNECTION_URL", "localhost:6379"),
		DatabaseType:             getEnv("DATABASE_TYPE", "postgres"),
		DatabaseURL:              getEnv("DATABASE_URL", "postgres://localhost/chainpulse"),
		DatabaseSSLMode:          getEnv("DATABASE_SSLMODE", "prefer"),
		APIType:                  getEnv("API_TYPE", "graphql"),
		APIPort:                  getEnv("API_PORT", "8080"),
		WorkerPoolSize:           getEnv("WORKER_POOL_SIZE", "8"),
		BatchSize:                getEnv("BATCH_SIZE", "100"),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		DLQRetention:             getEnv("MONOLITHIC_DLQ_RETENTION", "168h"),
		RateLimitEnabled:         getEnvBool("GATEWAY_RATE_LIMIT_ENABLED", false),
		RateLimitPerMinute:       getEnvInt("GATEWAY_RATE_LIMIT_PER_MINUTE", 60),
		TLSEnabled:               getEnvBool("GATEWAY_TLS_ENABLED", false),
		TLSCertPath:              getEnv("GATEWAY_TLS_CERT", ""),
		TLSKeyPath:               getEnv("GATEWAY_TLS_KEY", ""),
		JWTSecret:                getEnv("API_JWT_SECRET", ""),
		AuthEnabled:              getEnvBool("CHAINPULSE_AUTH_ENABLED", false),
		AuthAPIKeys:              getEnvCSV("CHAINPULSE_AUTH_API_KEYS", nil),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvCSV(key string, defaultValues []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		if defaultValues == nil {
			return nil
		}
		values := make([]string, len(defaultValues))
		copy(values, defaultValues)
		return values
	}

	values := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func parseChains(raw string) ([]string, error) {
	chains := make([]string, 0)
	for _, chainID := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(chainID)
		if trimmed == "" {
			continue
		}
		chains = append(chains, trimmed)
	}
	if len(chains) == 0 {
		return nil, fmt.Errorf("at least one chain is required")
	}
	return chains, nil
}

func parseMonolithicDLQRetention(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("dlq retention is required")
	}

	retention, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("parse monolithic dlq retention: %w", err)
	}

	if retention < 0 {
		return 0, fmt.Errorf("monolithic dlq retention must be non-negative")
	}

	return retention, nil
}

type ownershipSummary struct {
	ShadowOwnedEvents int64
	LegacyOwnedEvents int64
	Chains            int
}

func aggregateIndexerOwnership(status map[string]map[string]interface{}) ownershipSummary {
	summary := ownershipSummary{}
	for _, chainStatus := range status {
		summary.Chains++
		summary.ShadowOwnedEvents += int64Value(chainStatus["shadow_owned_events"])
		summary.LegacyOwnedEvents += int64Value(chainStatus["legacy_owned_events"])
	}
	return summary
}

func classifyOwnershipMode(summary ownershipSummary) string {
	switch {
	case summary.ShadowOwnedEvents == 0 && summary.LegacyOwnedEvents == 0:
		return "idle"
	case summary.ShadowOwnedEvents == 0 && summary.LegacyOwnedEvents > 0:
		return "legacy-only"
	case summary.ShadowOwnedEvents > 0 && summary.LegacyOwnedEvents == 0:
		return "runtime-owned"
	case summary.ShadowOwnedEvents > 0 && summary.LegacyOwnedEvents > 0:
		return "shadow"
	default:
		return "unknown"
	}
}

func ownershipModeCode(mode string) float64 {
	switch mode {
	case "idle":
		return 0
	case "legacy-only":
		return 1
	case "shadow":
		return 2
	case "runtime-owned":
		return 3
	default:
		return 9
	}
}

func buildOwnershipHealthComponent(status map[string]map[string]interface{}, now time.Time) *api.ComponentStatus {
	summary := aggregateIndexerOwnership(status)
	mode := classifyOwnershipMode(summary)
	componentStatus := "healthy"
	if mode == "unknown" {
		componentStatus = "degraded"
	}

	return &api.ComponentStatus{
		Name:      "Indexing Runtime",
		Status:    componentStatus,
		Timestamp: now.Unix(),
		Details: map[string]interface{}{
			"service":             "monolithic",
			"ownership_mode":      mode,
			"shadow_owned_events": summary.ShadowOwnedEvents,
			"legacy_owned_events": summary.LegacyOwnedEvents,
			"ownership_chains":    summary.Chains,
		},
	}
}

func buildOwnershipReadinessDetails(status map[string]map[string]interface{}) map[string]interface{} {
	return buildOwnershipRolloutSummary(status).readinessDetails()
}

func int64Value(value interface{}) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case uint64:
		if typed > math.MaxInt64 {
			return math.MaxInt64
		}

		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

// parseMonolithicKeyPair parses a "key=clientID" or "key:clientID" entry
func parseMonolithicKeyPair(entry string) (string, string, bool) {
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
