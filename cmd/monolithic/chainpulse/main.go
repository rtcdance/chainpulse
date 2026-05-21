package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	sharedhttp "github.com/rtcdance/chainpulse/pkg/infrastructure/http"
	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func run() error {
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

	// Convert configuration to core.Config and apply overrides
	blockchainNodeURL := strings.Split(config.BlockchainNodeURLs, ",")[0]
	coreCfg := bootstrap.NewMonolithicCoreConfig(
		config.LogLevel,
		config.DatabaseType,
		config.DatabaseURL,
		config.CacheType,
		config.DataPullerType,
		blockchainNodeURL,
	)
	bootstrap.ApplyConfigOverrides(&coreCfg)
	coreConfig := &coreCfg

	// Print configuration header
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ChainPulse - Monolithic Indexer Service            ║")
	fmt.Println("║              Web3 Event Indexing System                    ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Println("Configuration Loaded:")
	fmt.Printf("  Deployment Mode:     %s\n", config.DeploymentMode)
	fmt.Printf("  Adapter Profile:     %s\n", config.AdapterProfile)
	fmt.Printf("  Transport Boundary:  %s\n", config.TransportAdapterBoundary)
	fmt.Printf("  Chains:              %s\n", config.Chains)
	fmt.Printf("  Data Puller Type:    %s\n", coreConfig.DataPullerType)
	fmt.Printf("  Blockchain Nodes:    %s\n", config.BlockchainNodeURLs)
	fmt.Printf("  Message Queue Type:  %s\n", coreConfig.MQType)
	fmt.Printf("  Cache Type:          %s\n", coreConfig.CacheType)
	fmt.Printf("  Database Type:       %s\n", coreConfig.DatabaseType)
	fmt.Printf("  DLQ Retention:       %s\n", config.DLQRetention)
	fmt.Printf("  API Port:            %d\n", coreConfig.APIPort)
	fmt.Printf("  Worker Pool Size:    %d\n", coreConfig.WorkerPoolSize)
	fmt.Printf("  Batch Size:          %d\n", coreConfig.BatchSize)
	fmt.Printf("  Log Level:           %s\n", coreConfig.LogLevel)
	fmt.Println()

	// Initialize core services
	fmt.Println("Initializing Core Services:")
	logLevel := core.LogLevelInfo
	if coreConfig.LogLevel == "debug" {
		logLevel = core.LogLevelDebug
	}

	logFormat := env.Get("LOG_FORMAT", "slog")
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

	// Production security gate: enforce auth + JWT + rate limit in production
	if err := validateMonolithicProductionSecurity(config, bootstrap.RuntimeProfileFromEnv()); err != nil {
		logger.Error("Monolithic production security gate rejected startup", "error", err.Error())
		return fmt.Errorf("production security validation failed: %w", err)
	}

	// Read filters from env
	if contractAddrs := os.Getenv("CONTRACT_ADDRESSES"); contractAddrs != "" {
		coreConfig.ContractAddresses = strings.Split(contractAddrs, ",")
	}
	if eventSigs := os.Getenv("EVENT_SIGNATURES"); eventSigs != "" {
		coreConfig.EventSignatures = strings.Split(eventSigs, ",")
	}

	// Log resolved config snapshot for debugging (mask credentials)
	maskedDBURL := coreConfig.DatabaseURL
	if idx := strings.Index(coreConfig.DatabaseURL, "://"); idx >= 0 {
		rest := coreConfig.DatabaseURL[idx+3:]
		if atIdx := strings.Index(rest, "@"); atIdx >= 0 {
			maskedDBURL = coreConfig.DatabaseURL[:idx+3] + "***" + rest[atIdx:]
		}
	}
	logger.Info("Core config resolved",
		"deployment_mode", coreConfig.DeploymentMode,
		"data_puller_type", coreConfig.DataPullerType,
		"blockchain_node_url", coreConfig.BlockchainNodeURL,
		"chain_id", coreConfig.ChainID,
		"block_chunk_size", coreConfig.BlockChunkSize,
		"database_type", coreConfig.DatabaseType,
		"database_url", maskedDBURL,
		"cache_type", coreConfig.CacheType,
		"worker_pool_size", coreConfig.WorkerPoolSize,
		"batch_size", coreConfig.BatchSize,
		"log_level", coreConfig.LogLevel,
		"skip_removed_logs", coreConfig.SkipRemovedLogs,
	)

	// Validate the resolved core configuration
	configManager := core.NewConfigManager(logger)
	configManager.Load()
	if err := configManager.Validate(); err != nil {
		return fmt.Errorf("core configuration validation failed: %w", err)
	}

	chains, err := parseChains(config.Chains)
	if err != nil {
		return fmt.Errorf("failed to parse chain configuration: %w", err)
	}

	// Build monolithic starter (assembles runtime wiring, indexing storage,
	// shared indexing runtime, and multi-chain indexer in one call)
	fmt.Println("Building Monolithic Runtime Components:")
	starter, err := bootstrap.BuildMonolithicStarter(ctx, *coreConfig, logger, metrics, chains)
	if err != nil {
		return fmt.Errorf("failed to build monolithic starter: %w", err)
	}
	runtimeWiring := starter.RuntimeWiring
	indexingDatabase := starter.IndexingDatabase
	indexingCache := starter.IndexingCache
	sharedIndexingRuntime := starter.SharedRuntime
	multiChainIndexer := starter.MultiChainIndexer

	fmt.Printf("  ✓ Database plugin started: %s\n", indexingDatabase.Name())
	fmt.Printf("  ✓ Cache plugin started: %s\n", indexingCache.Name())
	fmt.Printf("  ✓ Shared runtime started (%s)\n", sharedIndexingRuntime.Status().State)
	fmt.Printf("  ✓ Multi-chain indexer: %d chain(s)\n", len(chains))
	fmt.Println()

	// Wire health/readiness/rollout providers
	runtimeWiring.HealthCheckHandler.SetRuntimeComponentProvider(func(ctx context.Context) *api.ComponentStatus {
		_ = ctx
		return buildOwnershipHealthComponent(multiChainIndexer.GetStatus(), time.Now())
	})
	runtimeWiring.HealthCheckHandler.SetReadinessDetailsProvider(func(ctx context.Context) map[string]any {
		_ = ctx
		return buildOwnershipReadinessDetails(multiChainIndexer.GetStatus())
	})
	runtimeWiring.HealthCheckHandler.SetRolloutReportProducer(api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx
		return buildOwnershipRolloutSummary(multiChainIndexer.GetStatus()).reportDetails()
	}))
	fmt.Println()

	// Build monolithic query surface (used by gateway)
	monolithicQuerySurface, err := resolveMonolithicQuerySurface(ctx, config, runtimeWiring, indexingDatabase, logger, metrics)
	if err != nil {
		return fmt.Errorf("failed to build monolithic query surface: %w", err)
	}
	if monolithicQuerySurface.domainQuery != nil {
		runtimeWiring.DomainQueryService = monolithicQuerySurface.domainQuery
	}
	if monolithicQuerySurface.adminKeyHandler == nil {
		if pgRaw, pgErr := runtimeWiring.DBManager.GetPostgresDB(ctx); pgErr == nil {
			if sqlDB, ok := pgRaw.(*sql.DB); ok {
				monolithicQuerySurface.adminKeyHandler = api.NewAdminKeyHandler(sqlDB, logger)
			}
		}
	}

	// Initialize monolithic puller runtime closure
	fmt.Println("Initializing Monolithic Puller Runtime:")

	monolithicPullerRuntime, err := newMonolithicPullerRuntime(ctx, *coreConfig, config.BlockchainNodeURLs, chains, logger, metrics, indexingDatabase, multiChainIndexer)
	if err != nil {
		return fmt.Errorf("failed to initialize monolithic puller runtime: %w", err)
	}

	fmt.Printf("  ✓ Event bus initialized (%d subscribers)\n", monolithicPullerRuntime.SubscriberCount())
	fmt.Printf("  ✓ Pullers initialized: %d\n", monolithicPullerRuntime.PullerCount())
	fmt.Println()

	// Bridge EventBus "event:created" → push-based subscription handlers
	// This enables real-time WebSocket/GraphQL event push + webhook delivery.
	var webhookStore *api.WebhookStore
	if runtimeWiring.EventSubscriptionHandler != nil {
		_, subErr := monolithicPullerRuntime.EventBus().SubscribeNamed(
			ctx, "event:created", "monolithic-subscription-bridge",
			func(_ context.Context, payload any) error {
				if event, ok := payload.(*core.BlockchainEvent); ok && event != nil {
					runtimeWiring.EventSubscriptionHandler.BroadcastEvent(ctx, event)
					// Deliver webhooks for this event
					if webhookStore != nil {
						payloadMap := map[string]any{
							"id":              event.ID,
							"chainId":         event.ChainID,
							"blockNumber":     event.BlockNumber,
							"transactionHash": event.TransactionHash.Hex(),
							"eventName":       event.EventName,
							"contractAddress": event.ContractAddress.Hex(),
							"timestamp":       event.BlockTimestamp,
						}
						webhookStore.NotifyEvent(ctx, "event:created", payloadMap)
					}
				}
				return nil
			},
		)
		if subErr != nil {
			logger.Warn("Failed to wire push subscription bridge", "error", subErr.Error())
		} else {
			logger.Info("Push-based subscription bridge wired: EventBus → WebSocket")
			fmt.Println("  ✓ Push-based subscription bridge: EventBus → WebSocket + Webhooks")
		}
	}

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
	if monolithicQuerySurface.exportHandler != nil {
		gateway.SetExportHandler(monolithicQuerySurface.exportHandler)
	}
	if monolithicQuerySurface.statsHandler != nil {
		gateway.SetStatsHandler(monolithicQuerySurface.statsHandler)
	}
	if monolithicQuerySurface.adminKeyHandler != nil {
		gateway.SetAdminKeyHandler(monolithicQuerySurface.adminKeyHandler)
	}
	// Wire store-backed AdminAPIKeyHandler for CRUD at /admin/api-keys (if postgres is available)
	if runtimeWiring.DBManager != nil {
		if pgRaw, pgErr := runtimeWiring.DBManager.GetPostgresDB(ctx); pgErr == nil {
			if sqlDB, ok := pgRaw.(*sql.DB); ok {
				keyStore := api.NewAPIKeyStore(sqlDB, logger, metrics)
				adminAPIKeyHandler := api.NewAdminAPIKeyHandler(keyStore, logger)
				gateway.SetAdminAPIKeyHandler(adminAPIKeyHandler)
				logger.Info("Store-backed AdminAPIKeyHandler wired", "endpoint", "/admin/api-keys")

				// Wire webhook store for event-driven notifications
				webhookStore = api.NewWebhookStore(sqlDB, logger, metrics)
				logger.Info("Webhook store wired for event-driven delivery")
			}
		}
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
		return fmt.Errorf("failed to initialize API Gateway: %w", err)
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
		if len(config.JWTSecret.Value()) < 32 {
			logger.Warn("JWT secret too short, minimum 32 characters recommended")
		}
		fmt.Println("  ✓ JWT authentication configured")
	} else {
		logger.Info("JWT secret not set, auth disabled")
	}

	// Build and inject auth middleware if authentication is enabled
	var tokenValidator *api.TokenValidator
	if config.JWTSecret != "" {
		tokenValidator = api.NewTokenValidator(config.JWTSecret.Value(), logger, metrics)
		for _, entry := range config.AuthAPIKeys {
			apiKey, clientID, ok := parseMonolithicKeyPair(entry.Value())
			if !ok {
				logger.Warn("invalid CHAINPULSE_AUTH_API_KEYS entry; expected key=clientID or key:clientID", "entry", entry)
				continue
			}
			if err := tokenValidator.RegisterAPIKey(apiKey, clientID, "operator"); err != nil {
				logger.Warn("failed to register API key", "error", err.Error())
			}
		}

		if config.AuthEnabled {
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

		// Wire SIWE handler
		siweDomain := fmt.Sprintf("localhost:%d", coreConfig.APIPort)
		siweURI := "http://" + siweDomain + "/login"
		siweHandler := api.NewSIWEHandler(tokenValidator, siweDomain, siweURI, nil, logger, metrics)
		gateway.SetSIWEHandler(siweHandler)
		logger.Info("SIWE auth handler wired")
		fmt.Println("  ✓ SIWE authentication wired to API Gateway")
	}

	if gateway.IsRuntimeRoutesEnabled() {
		fmt.Println("  ✓ Runtime route composition enabled")
	}
	fmt.Println()

	// Register plugins with registry
	fmt.Println("Registering Plugins:")
	if err := registry.Register(gateway); err != nil {
		return fmt.Errorf("failed to register API Gateway: %w", err)
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
		stopPullers()
		return fmt.Errorf("failed to start monolithic puller runtime: %w", err)
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel()
	stopPullers()

	// Stop API Gateway
	if err := core.StopPlugin(shutdownCtx, gateway); err != nil {
		logger.Error("Error stopping API Gateway", "error", err.Error())
	}
	fmt.Println("  [1/7] API Gateway stopped")

	if err := monolithicPullerRuntime.Stop(); err != nil {
		logger.Error("Error stopping monolithic puller runtime", "error", err.Error())
	}

	fmt.Println("  [2/7] Monolithic puller runtime stopped")

	if err := sharedIndexingRuntime.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping shared indexing runtime", "error", err.Error())
	}
	metrics.RecordGauge("indexing_runtime_started", 0, map[string]string{
		"service":   "monolithic",
		"operation": "shutdown",
	})
	logger.Info("Shared indexing runtime stopped", "service", "monolithic", "state", sharedIndexingRuntime.Status().State)
	fmt.Println("  [3/7] Shared indexing runtime stopped")

	if p, ok := indexingCache.(core.LifecyclePlugin); ok {
		if err := p.Stop(shutdownCtx); err != nil {
			logger.Error("Error stopping indexing cache", "error", err.Error())
		}
	}
	if p, ok := indexingDatabase.(core.LifecyclePlugin); ok {
		if err := p.Stop(shutdownCtx); err != nil {
			logger.Error("Error stopping indexing database", "error", err.Error())
		}
	}

	fmt.Println("  [4/7] Indexing storage stopped")

	if err := runtimeWiring.Close(shutdownCtx); err != nil {
		logger.Error("Error closing runtime wiring", "error", err.Error())
	}

	fmt.Println("  [5/7] Runtime wiring closed")

	// Close multi-chain indexer
	if err := multiChainIndexer.Close(); err != nil {
		logger.Error("Error closing multi-chain indexer", "error", err.Error())
	}

	fmt.Println("  [6/7] Multi-chain indexer closed")

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

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "chainpulse: %v\n", err)
		os.Exit(1)
	}
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
	BlockchainNodeURLs       string
	CacheConnectionURL       string
	MQConnectionURL          string
	LogLevel                 string
	DatabaseType             string
	DatabaseURL              string
	DataPullerType           string
	CacheType                string
	DatabaseSSLMode          string
	DLQRetention             string
	RateLimitEnabled         bool
	RateLimitPerMinute       int
	TLSEnabled               bool
	TLSCertPath              string
	TLSKeyPath               string
	JWTSecret                core.SecretString
	AuthEnabled              bool
	AuthAPIKeys              []core.SecretString
}

// loadConfiguration loads configuration from environment variables.
// Only monolithic-specific deployment fields are loaded here.
// Common fields (worker pool, batch size, api port, etc.) are handled
// by bootstrap.NewMonolithicCoreConfig + ApplyConfigOverrides.
func loadConfiguration() Configuration {
	modeProfile := resolveDeploymentModeProfile(os.Getenv("DEPLOYMENT_MODE"))
	upstreamQueryServices := env.GetCSV("MONOLITHIC_UPSTREAM_QUERY_SERVICES", []string{"http://localhost:8081"})
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
		Chains:                   env.Get("CHAINS", "ethereum,polygon"),
		BlockchainNodeURLs:       env.Get("BLOCKCHAIN_NODE_URLS", "http://localhost:8545,http://localhost:8546"),
		CacheConnectionURL:       env.Get("CACHE_CONNECTION_URL", "localhost:6379"),
		MQConnectionURL:          env.Get("MQ_CONNECTION_URL", "localhost:9092"),
		DatabaseURL:              env.Get("DATABASE_URL", "postgres://localhost/chainpulse"),
		DatabaseSSLMode:          env.Get("DATABASE_SSLMODE", "prefer"),
		DLQRetention:             env.Get("MONOLITHIC_DLQ_RETENTION", "168h"),
		RateLimitEnabled:         env.GetBool("GATEWAY_RATE_LIMIT_ENABLED", false),
		RateLimitPerMinute:       env.GetInt("GATEWAY_RATE_LIMIT_PER_MINUTE", 60),
		TLSEnabled:               env.GetBool("GATEWAY_TLS_ENABLED", false),
		TLSCertPath:              env.Get("GATEWAY_TLS_CERT", ""),
		TLSKeyPath:               env.Get("GATEWAY_TLS_KEY", ""),
		JWTSecret:                core.SecretString(env.Get("API_JWT_SECRET", "")),
		AuthEnabled:              env.GetBool("CHAINPULSE_AUTH_ENABLED", false),
		AuthAPIKeys:              core.ToSecretStrings(env.GetCSV("CHAINPULSE_AUTH_API_KEYS", nil)),
	}
}

func validateMonolithicProductionSecurity(c Configuration, runtimeProfile string) error {
	if runtimeProfile != "production" {
		return nil
	}
	if !c.AuthEnabled {
		return fmt.Errorf("production monolithic requires CHAINPULSE_AUTH_ENABLED=true")
	}
	if strings.TrimSpace(c.JWTSecret.Value()) == "" {
		return fmt.Errorf("production monolithic requires non-empty API_JWT_SECRET")
	}
	if !c.RateLimitEnabled {
		return fmt.Errorf("production monolithic requires GATEWAY_RATE_LIMIT_ENABLED=true")
	}
	if c.RateLimitPerMinute <= 0 {
		return fmt.Errorf("production monolithic requires GATEWAY_RATE_LIMIT_PER_MINUTE > 0")
	}
	return nil
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

func aggregateIndexerOwnership(status map[string]map[string]any) ownershipSummary {
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

func buildOwnershipHealthComponent(status map[string]map[string]any, now time.Time) *api.ComponentStatus {
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
		Details: map[string]any{
			"service":             "monolithic",
			"ownership_mode":      mode,
			"shadow_owned_events": summary.ShadowOwnedEvents,
			"legacy_owned_events": summary.LegacyOwnedEvents,
			"ownership_chains":    summary.Chains,
		},
	}
}

func buildOwnershipReadinessDetails(status map[string]map[string]any) map[string]any {
	return buildOwnershipRolloutSummary(status).readinessDetails()
}

func int64Value(value any) int64 {
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
