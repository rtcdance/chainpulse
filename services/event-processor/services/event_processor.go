package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/services/blockchain/services"
	"chainpulse/shared/cache"
	"chainpulse/shared/database"
	"chainpulse/shared/metrics"
	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
)

type EventProcessorService struct {
	Blockchain     *blockchain.EventProcessor
	Database       *database.Database
	BatchProcessor *database.BatchProcessor
	Cache          *cache.Cache
	Logger         Logger
	Resume         *ResumeService
	Metrics        *metrics.Metrics
	mu             sync.Mutex
}

func NewEventProcessorService(bc *blockchain.EventProcessor, db *database.Database, batchProcessor *database.BatchProcessor, c *cache.Cache, resume *ResumeService, logger Logger, metrics *metrics.Metrics) *EventProcessorService {
	return &EventProcessorService{
		Blockchain:     bc,
		Database:       db,
		BatchProcessor: batchProcessor,
		Cache:          c,
		Resume:         resume,
		Logger:         logger,
		Metrics:        metrics,
	}
}

// Start starts the event processing service
func (s *EventProcessorService) Start(ctx context.Context) error {
	s.Logger.Info("Starting event processor service...")

	// Subscribe to all blockchain events
	contractAddresses := []common.Address{} // This would be configured based on requirements

	// Start processing events
	eventChan, errChan, err := s.Blockchain.SubscribeToAllEvents(ctx, contractAddresses)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %v", err)
	}

	// Handle events in a goroutine
	go s.handleEvents(ctx, eventChan, errChan)

	return nil
}

func (s *EventProcessorService) handleEvents(ctx context.Context, eventChan <-chan *types.IndexedEvent, errChan <-chan error) {
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				s.Logger.Warn("Event channel closed")
				return
			}
			go s.processEvent(event)
		case err, ok := <-errChan:
			if ok {
				s.Logger.Error("Event subscription error: %v", err)
			}
		case <-ctx.Done():
			s.Logger.Info("Event handler context cancelled")
			return
		}
	}
}

func (s *EventProcessorService) processEvent(event *types.IndexedEvent) {
	s.Logger.Info("Processing event: %s, block %s", event.EventType, event.BlockNumber.String())

	// Add to batch processor
	err := s.BatchProcessor.AddEvent(event)
	if err != nil {
		s.Logger.Error("Failed to add event to batch processor: %v", err)
		if s.Metrics != nil {
			s.Metrics.IncrementError("batch", "add_event_failed")
		}
		return
	}

	// Cache the event
	cacheKey := fmt.Sprintf("event:%s:%s:%s", event.EventType, event.Contract, event.TxHash)
	err = s.Cache.Set(context.Background(), cacheKey, event, 24*time.Hour)
	if err != nil {
		s.Logger.Warn("Failed to cache event: %v", err)
		if s.Metrics != nil {
			s.Metrics.IncrementError("cache", "set_failed")
		}
	}

	if s.Metrics != nil {
		s.Metrics.IncrementEventsProcessed()
	}

	s.Logger.Info("Successfully processed event: %s", event.TxHash)
}