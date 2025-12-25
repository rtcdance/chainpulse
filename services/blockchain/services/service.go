package blockchain

import (
	"context"
	"fmt"
	"sync"

	"chainpulse/shared/logger"
	"chainpulse/shared/metrics"
	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
)

type BlockchainService struct {
	EventProcessor *EventProcessor
	Logger         logger.Logger
	Metrics        *metrics.Metrics
	mu             sync.Mutex
}

func NewBlockchainService(bc *EventProcessor, logger logger.Logger, metrics *metrics.Metrics) *BlockchainService {
	return &BlockchainService{
		EventProcessor: bc,
		Logger:         logger,
		Metrics:        metrics,
	}
}

// Start starts the blockchain service
func (s *BlockchainService) Start(ctx context.Context) error {
	s.Logger.Info("Starting blockchain service...")

	// Example: Subscribe to specific events
	contractAddresses := []common.Address{} // This would be configured based on requirements

	// Start listening for events
	eventChan, errChan, err := s.EventProcessor.SubscribeToAllEvents(ctx, contractAddresses)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %v", err)
	}

	// Handle events in a goroutine
	go s.handleEvents(ctx, eventChan, errChan)

	return nil
}

func (s *BlockchainService) handleEvents(ctx context.Context, eventChan <-chan *types.IndexedEvent, errChan <-chan error) {
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				s.Logger.Warn("Blockchain event channel closed")
				return
			}
			go s.processEvent(event)
		case err, ok := <-errChan:
			if ok {
				s.Logger.Error("Blockchain event subscription error: %v", err)
			}
		case <-ctx.Done():
			s.Logger.Info("Blockchain event handler context cancelled")
			return
		}
	}
}

func (s *BlockchainService) processEvent(event *types.IndexedEvent) {
	s.Logger.Info("Processing blockchain event: %s, block %s", event.EventType, event.BlockNumber.String())

	// Perform blockchain-specific operations
	// This could include validation, verification, or other blockchain operations

	if s.Metrics != nil {
		s.Metrics.IncrementEventsProcessed()
	}

	s.Logger.Info("Successfully processed blockchain event: %s", event.TxHash)
}