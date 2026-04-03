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

	"chainpulse/pkg/application/bootstrap"
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

	// Build additive shared indexing runtime
	fmt.Println("Initializing Shared Indexing Runtime:")
	sharedIndexingRuntime, err := bootstrap.BuildMonolithicIndexingRuntime(logger, indexingDatabase, indexingCache, chains)
	if err != nil {
		logger.Error("Failed to build shared indexing runtime", "error", err.Error())
		os.Exit(1)
	}
	if err := sharedIndexingRuntime.Initialize(context.Background()); err != nil {
		logger.Error("Failed to initialize shared indexing runtime", "error", err.Error())
		os.Exit(1)
	}
	if err := sharedIndexingRuntime.Start(context.Background()); err != nil {
		logger.Error("Failed to start shared indexing runtime", "error", err.Error())
		os.Exit(1)
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
	)
	fmt.Printf("  ✓ Shared runtime started (%s)\n", sharedRuntimeStatus.State)
	fmt.Printf("  ✓ Shared runtime chains: %s\n", strings.Join(sharedRuntimeStatus.Chains, ","))
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
			nil, // eventBus
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

	// Initialize API Gateway Plugin
	fmt.Println("Initializing API Gateway:")
	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetDomainQueryService(runtimeWiring.DomainQueryService)
	gateway.SetEventQueryHandler(runtimeWiring.EventQueryHandler)
	gateway.SetEventSubscriptionHandler(runtimeWiring.EventSubscriptionHandler)
	gateway.SetHealthCheckHandler(runtimeWiring.HealthCheckHandler)
	gateway.SetRuntimeSummaryProvider(buildMonolithicRuntimeSummaryProvider(metrics, gateway, sharedIndexingRuntime, multiChainIndexer))
	if err := gateway.Initialize(*coreConfig); err != nil {
		logger.Error("Failed to initialize API Gateway", "error", err.Error())
		os.Exit(1)
	}
	fmt.Println("  ✓ API Gateway initialized")
	if gateway.IsDomainBridgeEnabled() {
		fmt.Println("  ✓ Domain query bridge configured")
	}
	if gateway.IsEventQueryHandlerEnabled() {
		fmt.Println("  ✓ Event query handler wired")
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

	// Start Query Service
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runtimeWiring.QueryService.Start(context.Background()); err != nil {
			logger.Error("Query Service error", "error", err.Error())
		}
	}()
	fmt.Println("  [1/2] Query Service started")

	// Start API Gateway
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := gateway.Start(); err != nil {
			logger.Error("API Gateway error", "error", err.Error())
		}
	}()
	fmt.Println("  [2/2] API Gateway started")
	fmt.Println()

	fmt.Println("✓ All services started successfully")
	fmt.Println()
	fmt.Println("Status: Running")
	fmt.Printf("GraphQL API available at: http://localhost:8080/graphql\n")
	fmt.Printf("GraphQL WebSocket available at: ws://localhost:8080/graphql\n")
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop API Gateway
	if err := gateway.Stop(); err != nil {
		logger.Error("Error stopping API Gateway", "error", err.Error())
	}
	fmt.Println("  [1/3] API Gateway stopped")

	if err := sharedIndexingRuntime.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping shared indexing runtime", "error", err.Error())
	}
	metrics.RecordGauge("indexing_runtime_started", 0, map[string]string{
		"service":   "monolithic",
		"operation": "shutdown",
	})
	logger.Info("Shared indexing runtime stopped", "service", "monolithic", "state", sharedIndexingRuntime.Status().State)
	fmt.Println("  [2/4] Shared indexing runtime stopped")

	if err := indexingCache.Stop(); err != nil {
		logger.Error("Error stopping indexing cache", "error", err.Error())
	}
	if err := indexingDatabase.Stop(); err != nil {
		logger.Error("Error stopping indexing database", "error", err.Error())
	}
	fmt.Println("  [3/5] Indexing storage stopped")

	if err := runtimeWiring.Close(shutdownCtx); err != nil {
		logger.Error("Error closing runtime wiring", "error", err.Error())
	}
	fmt.Println("  [4/5] Runtime wiring closed")

	// Close multi-chain indexer
	if err := multiChainIndexer.Close(); err != nil {
		logger.Error("Error closing multi-chain indexer", "error", err.Error())
	}
	fmt.Println("  [5/5] Multi-chain indexer closed")
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
	Chains             string
	DataPullerType     string
	BlockchainNodeURLs string
	MQType             string
	MQConnectionURL    string
	CacheType          string
	CacheConnectionURL string
	DatabaseType       string
	DatabaseURL        string
	APIType            string
	APIPort            string
	WorkerPoolSize     string
	BatchSize          string
	LogLevel           string
}

// loadConfiguration loads configuration from environment variables
func loadConfiguration() Configuration {
	return Configuration{
		Chains:             getEnv("CHAINS", "ethereum,polygon"),
		DataPullerType:     getEnv("DATA_PULLER_TYPE", "https-jsonrpc"),
		BlockchainNodeURLs: getEnv("BLOCKCHAIN_NODE_URLS", "http://localhost:8545,http://localhost:8546"),
		MQType:             getEnv("MQ_TYPE", "kafka"),
		MQConnectionURL:    getEnv("MQ_CONNECTION_URL", "localhost:9092"),
		CacheType:          getEnv("CACHE_TYPE", "redis"),
		CacheConnectionURL: getEnv("CACHE_CONNECTION_URL", "localhost:6379"),
		DatabaseType:       getEnv("DATABASE_TYPE", "postgres"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://localhost/chainpulse"),
		APIType:            getEnv("API_TYPE", "graphql"),
		APIPort:            getEnv("API_PORT", "8080"),
		WorkerPoolSize:     getEnv("WORKER_POOL_SIZE", "8"),
		BatchSize:          getEnv("BATCH_SIZE", "100"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}
