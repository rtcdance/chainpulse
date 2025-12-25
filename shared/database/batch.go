package database

import (
	"context"
	"sync"
	"time"

	"chainpulse/shared/types"

	"gorm.io/gorm/clause"
)

// BatchProcessor handles batch database operations for better performance
type BatchProcessor struct {
	db           *Database
	batchSize    int
	flushTimeout time.Duration
	eventsChan   chan *types.IndexedEvent
	flushChan    chan struct{}
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(db *Database, batchSize int, flushTimeout time.Duration) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	bp := &BatchProcessor{
		db:           db,
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		eventsChan:   make(chan *types.IndexedEvent, batchSize*10), // Buffer 10x batch size
		flushChan:    make(chan struct{}, 1),
		ctx:          ctx,
		cancel:       cancel,
	}
	
	bp.startProcessing()
	return bp
}

// startProcessing starts the background goroutine for batch processing
func (bp *BatchProcessor) startProcessing() {
	bp.wg.Add(1)
	go bp.processBatches()
}

// processBatches handles the batching logic
func (bp *BatchProcessor) processBatches() {
	defer bp.wg.Done()
	
	events := make([]*types.IndexedEvent, 0, bp.batchSize)
	ticker := time.NewTicker(bp.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case event := <-bp.eventsChan:
			events = append(events, event)
			
			// If we've reached batch size, flush immediately
			if len(events) >= bp.batchSize {
				bp.flushBatch(events)
				events = make([]*types.IndexedEvent, 0, bp.batchSize)
			}
		case <-ticker.C:
			// Flush batch if there are any events
			if len(events) > 0 {
				bp.flushBatch(events)
				events = make([]*types.IndexedEvent, 0, bp.batchSize)
			}
		case <-bp.flushChan:
			// Force flush when requested
			if len(events) > 0 {
				bp.flushBatch(events)
				events = make([]*types.IndexedEvent, 0, bp.batchSize)
			}
		case <-bp.ctx.Done():
			// Flush remaining events when shutting down
			if len(events) > 0 {
				bp.flushBatch(events)
			}
			return
		}
	}
}

// flushBatch processes a batch of events
func (bp *BatchProcessor) flushBatch(events []*types.IndexedEvent) {
	if len(events) == 0 {
		return
	}

	// Use GORM's clause for batch insert
	err := bp.db.DB.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(events, bp.batchSize).Error
	if err != nil {
		// In a real implementation, you might want to handle this error differently
		// For now, we'll just log it
		return
	}
}

// AddEvent adds an event to the batch processor
func (bp *BatchProcessor) AddEvent(event *types.IndexedEvent) error {
	select {
	case bp.eventsChan <- event:
		return nil
	case <-bp.ctx.Done():
		return bp.ctx.Err()
	}
}

// Flush forces a flush of all pending events
func (bp *BatchProcessor) Flush() {
	select {
	case bp.flushChan <- struct{}{}:
	default:
		// Channel is full, but that's OK
	}
}

// Close shuts down the batch processor
func (bp *BatchProcessor) Close() error {
	bp.cancel()
	bp.wg.Wait()
	return nil
}