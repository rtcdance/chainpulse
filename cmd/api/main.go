package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chainpulse/services/api/handlers"
	"chainpulse/services/api/handlers/grpc"
	"chainpulse/services/blockchain/services"
	"chainpulse/shared/cache"
	"chainpulse/shared/config"
	"chainpulse/shared/database"
	"chainpulse/shared/datapuller"
	"chainpulse/shared/logger"
	"chainpulse/shared/metrics"
	"chainpulse/shared/migrations"
	"chainpulse/shared/service"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	appLogger := logger.NewLogger()

	// Initialize database
	db, err := database.NewDatabase(cfg.PostgreSQLURL)
	if err != nil {
		appLogger.Fatal("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	migrator := migrations.NewMigrator(db.DB)
	migrator.AddMigration(&migrations.InitialSchemaMigration{})
	migrator.AddMigration(&migrations.AddIndexesMigration{})
	
	if err := migrator.RunMigrations(); err != nil {
		appLogger.Fatal("Failed to run database migrations: %v", err)
	}

	appLogger.Info("Connected to database successfully")

	// Initialize cache
	cache, err := cache.NewCache(cfg.RedisURL)
	if err != nil {
		appLogger.Error("Failed to connect to cache: %v", err)
		log.Fatal(err)
	}
	appLogger.Info("Connected to cache successfully")

	// Initialize blockchain event processor
	bc, err := blockchain.NewEventProcessor(cfg.EthereumNodeURL)
	if err != nil {
		appLogger.Error("Failed to connect to Ethereum node: %v", err)
		log.Fatal(err)
	}
	appLogger.Info("Connected to Ethereum node successfully")

	// Initialize cached database
	cachedDB, err := database.NewCachedDatabase(cfg.PostgreSQLURL, cache)
	if err != nil {
		appLogger.Error("Failed to create cached database: %v", err)
		log.Fatal(err)
	}

	// Initialize resume service with regular database
	resumeService := service.NewResumeService(bc.Client, db)

	// Initialize metrics
	metrics := metrics.NewMetrics()

	// Initialize batch processor with cached database
	batchProcessor := database.NewBatchProcessor(cachedDB.DB, cfg.BatchSize, time.Duration(cfg.FlushTimeout)*time.Second)

	// Initialize reorg handler
	reorgHandler := service.NewReorgHandler(bc.Client, db, appLogger, 10, 100) // depth: 10, maxDepth: 100

	// Initialize idempotency service
	idempotencyService := service.NewIdempotencyService(cache, db, 24*time.Hour)

	// Initialize blockchain data puller with plugin architecture
	dataPuller := datapuller.NewBlockchainDataPuller()
	
	// Configure retry settings
	retryConfig := &datapuller.RetryConfig{
		MaxRetries:      3,
		BaseDelay:       time.Second,
		MaxDelay:        30 * time.Second,
		BackoffMultiplier: 2.0,
		EnableJitter:    true,
	}
	dataPuller.SetRetryConfig(retryConfig)
	
	// Configure data puller with plugin configurations
	pluginConfigs := map[string]map[string]interface{}{
		"https-jsonrpc": {
			"url": cfg.EthereumNodeURL, // Use the same Ethereum node URL for HTTPS JSON-RPC
		},
		"websocket-jsonrpc": {
			"url": cfg.EthereumNodeWSURL, // WebSocket URL for real-time data
		},
		"grpc": {
			"address": cfg.GRPCServerURL, // gRPC server address
		},
	}
	
	// Initialize the data puller with plugin configurations
	if err := dataPuller.Initialize(pluginConfigs); err != nil {
		appLogger.Error("Failed to initialize data puller: %v", err)
		log.Fatal(err)
	}

	// Initialize service
	indexerService := service.NewIndexerService(bc, cachedDB, batchProcessor, cache, resumeService, appLogger, metrics, reorgHandler, idempotencyService, dataPuller)

	// Initialize the API server
	server := api.NewServer(cfg)
	server.Service = indexerService
	server.Metrics = metrics

	// Define contract addresses to monitor (example addresses)
	contractAddresses := []common.Address{
		common.HexToAddress("0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"), // Bored Ape Yacht Club
		common.HexToAddress("0x60E4d786628Fea6478F785A6d7e704777c86a7c6"), // Mutant Ape Yacht Club
		// Add more contract addresses as needed
	}

	// Start the indexer service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := indexerService.StartIndexing(ctx, contractAddresses); err != nil {
			appLogger.Error("Failed to start indexing: %v", err)
		}
	}()

	// Start the REST server
	restPort := os.Getenv("PORT")
	if restPort == "" {
		restPort = "8080"
	}

	restSrv := &http.Server{
		Addr:    ":" + restPort,
		Handler: server.Router,
	}

	// Run REST server in a goroutine
	go func() {
		appLogger.Info("Starting chainpulse REST API server on port %s", restPort)
		if err := restSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("REST server error: %v", err)
		}
	}()

	// Start the gRPC server in a goroutine
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	go func() {
		appLogger.Info("Starting chainpulse gRPC server on port %s", grpcPort)
		if err := grpc.StartGRPCServer(indexerService, grpcPort, cfg.JWTSecret); err != nil {
			appLogger.Error("gRPC server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutting down servers...")

	// Shutdown the REST server with a timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := restSrv.Shutdown(ctx); err != nil {
		appLogger.Error("REST server forced to shutdown: %v", err)
	} else {
		appLogger.Info("REST server exited gracefully")
	}

	// Close connections
	bc.Close()
	cache.Close()
	batchProcessor.Close()
}