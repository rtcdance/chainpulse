package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"chainpulse/shared/mq"
	"chainpulse/shared/types"
)

// EventProcessorService handles blockchain event processing
type EventProcessorService struct {
	mq     mq.MessageQueue
	db     *types.Database
}

// ProcessedEventMessage represents a message containing a processed event
type ProcessedEventMessage struct {
	Event types.IndexedEvent `json:"event"`
}

// NewEventProcessorService creates a new event processor service
func NewEventProcessorService(mq mq.MessageQueue, db *types.Database) *EventProcessorService {
	return &EventProcessorService{
		mq: mq,
		db: db,
	}
}

// Start begins processing events from the message queue
func (eps *EventProcessorService) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping event processor...")
		cancel()
	}()

	log.Println("Starting event processor service...")
	
	// Start consuming raw blockchain events
	if err := eps.mq.Consume(ctx, "blockchain.raw.events", eps.handleRawEvent); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

// handleRawEvent processes raw blockchain events from the queue
func (eps *EventProcessorService) handleRawEvent(data []byte) error {
	var rawEvent types.RawEvent
	if err := json.Unmarshal(data, &rawEvent); err != nil {
		return err
	}

	// Process the raw event and convert to indexed event
	indexedEvent := eps.processRawEvent(rawEvent)

	// Validate the event before storing
	if !eps.validateEvent(indexedEvent) {
		log.Printf("Invalid event detected: %+v", indexedEvent)
		return nil
	}

	// Check for idempotency - if already processed, skip
	if eps.isEventAlreadyProcessed(indexedEvent) {
		log.Printf("Event already processed, skipping: %s", indexedEvent.TxHash)
		return nil
	}

	// Store the processed event
	if err := eps.db.StoreEvent(indexedEvent); err != nil {
		return err
	}

	// Mark event as processed for idempotency
	if err := eps.markEventAsProcessed(indexedEvent); err != nil {
		log.Printf("Warning: failed to mark event as processed: %v", err)
	}

	// Publish processed event to next stage
	processedMsg := ProcessedEventMessage{
		Event: indexedEvent,
	}

	if err := eps.mq.Publish("blockchain.processed.events", processedMsg); err != nil {
		return err
	}

	log.Printf("Successfully processed event: %s", indexedEvent.TxHash)
	return nil
}

// processRawEvent converts a raw blockchain event to an indexed event
func (eps *EventProcessorService) processRawEvent(rawEvent types.RawEvent) types.IndexedEvent {
	// Parse and transform raw event data
	return types.IndexedEvent{
		ID:           0, // Will be set by database
		BlockNumber:  rawEvent.BlockNumber,
		BlockHash:    rawEvent.BlockHash,
		TxHash:       rawEvent.TxHash,
		EventName:    rawEvent.EventName,
		ContractAddr: rawEvent.ContractAddr,
		Data:         rawEvent.Data,
		Timestamp:    rawEvent.Timestamp,
		CreatedAt:    rawEvent.Timestamp,
		UpdatedAt:    rawEvent.Timestamp,
	}
}

// validateEvent performs validation on the event before processing
func (eps *EventProcessorService) validateEvent(event types.IndexedEvent) bool {
	// Add validation logic here
	return len(event.TxHash) > 0 && event.BlockNumber != 0
}

// isEventAlreadyProcessed checks if an event has already been processed
func (eps *EventProcessorService) isEventAlreadyProcessed(event types.IndexedEvent) bool {
	// Check if event exists in database
	existingEvent, err := eps.db.GetEventByTxHash(event.TxHash)
	if err != nil {
		// If there's an error, assume it doesn't exist and process
		return false
	}
	return existingEvent != nil
}

// markEventAsProcessed marks an event as processed for idempotency
func (eps *EventProcessorService) markEventAsProcessed(event types.IndexedEvent) error {
	// Store processed event ID or hash in a separate table for idempotency
	return eps.db.MarkProcessed(event.TxHash)
}

func main() {
	// Initialize metrics collector
	metricsCollector := mq.GlobalMetricsCollector

	// Initialize multi-protocol message queue
	multiMQ := mq.NewMultiProtocolMQ("kafka") // Use Kafka as default
	multiMQ.SetMetricsCollector(metricsCollector)

	// Configure plugin configurations
	pluginConfigs := map[string]map[string]interface{}{
		"kafka": {
			"brokers": []string{"localhost:9092"}, // This would come from config in real implementation
		},
		"redis": {
			"addr": "localhost:6379",
			"password": "",
			"db": 0,
		},
		"zeromq": {
			"publish_addr": "tcp://localhost:5555",
			"subscribe_addr": "tcp://localhost:5556",
		},
	}

	// Initialize the multi-protocol MQ with plugin configurations
	if err := multiMQ.Initialize(pluginConfigs); err != nil {
		log.Fatalf("Failed to initialize multi-protocol MQ: %v", err)
	}
	defer multiMQ.Close()

	// Initialize database (this would be a real database instance)
	var db *types.Database

	// Create and start event processor service
	service := NewEventProcessorService(multiMQ, db)
	
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start event processor service: %v", err)
	}
}