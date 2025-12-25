package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chainpulse/services/blockchain/services"
	"chainpulse/shared/config"
	"chainpulse/shared/logger"
	"chainpulse/shared/metrics"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	appLogger := logger.NewLogger()

	// Initialize blockchain event processor
	bc, err := services.NewEventProcessor(cfg.EthereumNodeURL)
	if err != nil {
		appLogger.Error("Failed to connect to Ethereum node: %v", err)
		log.Fatal(err)
	}
	appLogger.Info("Connected to Ethereum node successfully")

	// Initialize metrics
	metricsClient := metrics.NewMetrics()

	// Initialize the blockchain service
	blockchainService := services.NewBlockchainService(bc, appLogger, metricsClient)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	appLogger.Info("Blockchain service started successfully")

	// Run the blockchain service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := blockchainService.Start(ctx); err != nil {
			appLogger.Error("Blockchain service error: %v", err)
		}
	}()

	<-quit
	appLogger.Info("Shutting down blockchain service...")

	// Close connections
	bc.Close()
	time.Sleep(2 * time.Second) // Allow for graceful shutdown
}