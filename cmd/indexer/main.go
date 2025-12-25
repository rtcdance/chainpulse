package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chainpulse/services/blockchain/services"
	"chainpulse/shared/cache"
	"chainpulse/shared/config"
	"chainpulse/shared/database"
	"chainpulse/shared/datapuller"
	"chainpulse/shared/logger"
	"chainpulse/shared/metrics"
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

	// Initialize cache
	cacheClient, err := cache.NewCache(cfg.RedisURL)
	if err != nil {
		appLogger.Error("Failed to connect to cache: %v", err)
		log.Fatal(err)
	}
	appLogger.Info("Connected to cache successfully")

	// Initialize blockchain event processor
	bc, err := services.NewEventProcessor(cfg.EthereumNodeURL)
	if err != nil {
		appLogger.Error("Failed to connect to Ethereum node: %v", err)
		log.Fatal(err)
	}
	appLogger.Info("Connected to Ethereum node successfully")

	// Initialize cached database
	cachedDB, err := database.NewCachedDatabase(cfg.PostgreSQLURL, cacheClient)
	if err != nil {
		appLogger.Error("Failed to create cached database: %v", err)
		log.Fatal(err)
	}

	// Initialize resume service with regular database
	resumeService := service.NewResumeService(bc.Client, db)

	// Initialize metrics
	metricsClient := metrics.NewMetrics()

	// Initialize batch processor with cached database
	batchProcessor := database.NewBatchProcessor(cachedDB.DB, cfg.BatchSize, time.Duration(cfg.FlushTimeout)*time.Second)

	// Initialize reorg handler
	reorgHandler := service.NewReorgHandler(bc.Client, db, appLogger, 10, 100) // depth: 10, maxDepth: 100

	// Initialize idempotency service
	idempotencyService := service.NewIdempotencyService(cacheClient, db, 24*time.Hour)

	// Initialize metrics collector for data puller
	metricsCollector := datapuller.GlobalMetricsCollector
	
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
	
	// Wrap data puller with metrics after initialization
	dataPuller = datapuller.WithMetrics(dataPuller, metricsCollector)

	// Initialize indexer service
	indexerService := service.NewIndexerService(bc, cachedDB, batchProcessor, cacheClient, resumeService, appLogger, metricsClient, reorgHandler, idempotencyService, dataPuller)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Info("Indexer service started successfully")

	// Run the indexer service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Define contract addresses to monitor (example addresses)
	contractAddresses := []common.Address{
		common.HexToAddress("0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"), // Bored Ape Yacht Club
		common.HexToAddress("0x60E4d786628Fea6478F785A6d7e704777c86a7c6"), // Mutant Ape Yacht Club
		// Add more contract addresses as needed
	}

	go func() {
		if err := indexerService.StartIndexing(ctx, contractAddresses); err != nil {
			appLogger.Error("Failed to start indexing: %v", err)
		}
	}()

	<-quit
	appLogger.Info("Shutting down indexer service...")

	// Close connections
	bc.Close()
	cacheClient.Close()
	batchProcessor.Close()
	time.Sleep(2 * time.Second) // Allow for graceful shutdown
}