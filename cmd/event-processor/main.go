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
	"chainpulse/shared/logger"
	"chainpulse/shared/metrics"
	"chainpulse/shared/service"
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

	// Initialize resume service
	resumeService := service.NewResumeService(bc.Client, db)

	// Initialize metrics
	metricsClient := metrics.NewMetrics()

	// Initialize batch processor with configuration
	batchProcessor := database.NewBatchProcessor(db, cfg.BatchSize, time.Duration(cfg.FlushTimeout)*time.Second)

	// Initialize event processor service
	eventProcessorService := service.NewEventProcessorService(bc, db, batchProcessor, cacheClient, resumeService, appLogger, metricsClient)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Info("Event processor service started successfully")

	// Run the event processor service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := eventProcessorService.Start(ctx); err != nil {
			appLogger.Error("Event processor service error: %v", err)
		}
	}()

	<-quit
	appLogger.Info("Shutting down event processor service...")

	// Close connections
	bc.Close()
	cacheClient.Close()
	batchProcessor.Close()
	time.Sleep(2 * time.Second) // Allow for graceful shutdown
}