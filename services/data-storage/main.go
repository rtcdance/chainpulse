package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"chainpulse/shared/database"
	"chainpulse/shared/mq"
	"chainpulse/shared/types"
)

// DataStorageService handles data persistence for indexed events
type DataStorageService struct {
	mq   mq.MessageQueue
	db   *database.Database
}

// NewDataStorageService creates a new data storage service
func NewDataStorageService(mq mq.MessageQueue, db *database.Database) *DataStorageService {
	return &DataStorageService{
		mq: mq,
		db: db,
	}
}

// Start begins listening for processed events and storing them in the database
func (dss *DataStorageService) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping data storage service...")
		cancel()
	}()

	log.Println("Starting data storage service...")

	// Start consuming processed events
	if err := dss.mq.Consume(ctx, "blockchain.processed.events", dss.handleProcessedEvent); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

// handleProcessedEvent handles a processed event from the message queue
func (dss *DataStorageService) handleProcessedEvent(data []byte) error {
	var processedMsg event_processor.ProcessedEventMessage
	if err := json.Unmarshal(data, &processedMsg); err != nil {
		return err
	}

	event := processedMsg.Event

	// Check for duplicates before storing
	existingEvent, err := dss.db.GetEventByTxHash(event.TxHash)
	if err != nil {
		log.Printf("Error checking for existing event: %v", err)
		// Continue with storage even if check fails
	} else if existingEvent != nil {
		log.Printf("Event already exists in database, skipping: %s", event.TxHash)
		return nil
	}

	// Store the event in the database
	if err := dss.db.SaveEvent(&event); err != nil {
		return err
	}

	log.Printf("Successfully stored event in database: %s", event.TxHash)
	return nil
}

// StoreEvent provides a direct method to store an event (for API or other services)
func (dss *DataStorageService) StoreEvent(event *types.IndexedEvent) error {
	return dss.db.SaveEvent(event)
}

// GetEventByID retrieves an event by its ID
func (dss *DataStorageService) GetEventByID(id uint) (*types.IndexedEvent, error) {
	return dss.db.GetEventByID(id)
}

// GetEvents retrieves events based on filters
func (dss *DataStorageService) GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error) {
	return dss.db.GetEvents(filter)
}

// GetEventsByBlockRange retrieves events within a block range
func (dss *DataStorageService) GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error) {
	return dss.db.GetEventsByBlockRange(fromBlock, toBlock)
}

// GetLastProcessedBlock retrieves the last processed block number
func (dss *DataStorageService) GetLastProcessedBlock() (*big.Int, error) {
	return dss.db.GetLastProcessedBlock()
}

func main() {
	// Initialize message queue
	kafkaConfig := mq.KafkaConfig{
		Brokers: []string{"localhost:9092"}, // This would come from config in real implementation
	}
	
	mqInstance := mq.NewKafkaMQ(kafkaConfig)
	defer mqInstance.Close()

	// Initialize database
	db, err := database.NewDatabase("postgres://user:password@localhost:5432/chainpulse?sslmode=disable") // This would come from config
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create and start data storage service
	service := NewDataStorageService(mqInstance, db)
	
	if err := service.Start(); err != nil {
		if err != context.Canceled {
			log.Fatalf("Data storage service failed: %v", err)
		} else {
			log.Println("Data storage service stopped gracefully")
		}
	}
}