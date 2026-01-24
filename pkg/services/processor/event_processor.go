package processor

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/cache"
	"chainpulse/pkg/plugins/database"
)

// EventProcessor processes events from message queue and stores them
type EventProcessor interface {
	// Initialize initializes the event processor
	Initialize(config *core.Config) error

	// Start starts the event processor
	Start() error

	// Stop stops the event processor
	Stop() error

	// Health returns the health status of the processor
	Health() *core.HealthStatus

	// ProcessEvent processes a single event
	ProcessEvent(event *core.BlockchainEvent) error

	// ProcessBatch processes a batch of events
	ProcessBatch(events []*core.BlockchainEvent) error

	// GetProcessedCount returns the count of processed events
	GetProcessedCount() int64

	// GetFailedCount returns the count of failed events
	GetFailedCount() int64

	// GetDuplicateCount returns the count of duplicate events
	GetDuplicateCount() int64
}

// DefaultEventProcessor provides default event processing implementation
type DefaultEventProcessor struct {
	mu                  sync.RWMutex
	initialized         bool
	running             bool
	config              *core.Config
	logger              core.Logger
	metricsCollector    core.MetricsCollector
	idempotencyService  IdempotencyService
	cachePlugin         cache.CachePlugin
	databasePlugin      *database.DefaultInMemoryDatabasePlugin
	eventBus            core.EventBus
	processedCount      int64
	failedCount         int64
	duplicateCount      int64
	lastHealthCheck     *core.HealthStatus
	batchSize           int
	maxRetries          int
	retryDelay          time.Duration
}

// NewDefaultEventProcessor creates a new event processor
func NewDefaultEventProcessor(
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	idempotencyService IdempotencyService,
	cachePlugin cache.CachePlugin,
	databasePlugin *database.DefaultInMemoryDatabasePlugin,
	eventBus core.EventBus,
) *DefaultEventProcessor {
	return &DefaultEventProcessor{
		logger:             logger,
		metricsCollector:   metricsCollector,
		idempotencyService: idempotencyService,
		cachePlugin:        cachePlugin,
		databasePlugin:     databasePlugin,
		eventBus:           eventBus,
		batchSize:          100,
		maxRetries:         3,
		retryDelay:         time.Second,
	}
}

// Initialize initializes the event processor
func (p *DefaultEventProcessor) Initialize(config *core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("event processor already initialized")
	}

	if config == nil {
		return fmt.Errorf("config is required")
	}

	p.config = config
	p.initialized = true

	p.logger.Info("Event processor initialized", map[string]interface{}{
		"component": "event_processor",
	})

	return nil
}

// Start starts the event processor
func (p *DefaultEventProcessor) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return fmt.Errorf("event processor not initialized")
	}

	if p.running {
		return fmt.Errorf("event processor already running")
	}

	p.running = true
	p.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Event processor started",
	}

	p.logger.Info("Event processor started", map[string]interface{}{
		"component": "event_processor",
	})

	return nil
}

// Stop stops the event processor
func (p *DefaultEventProcessor) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("event processor not running")
	}

	p.running = false
	p.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Event processor stopped",
	}

	p.logger.Info("Event processor stopped", map[string]interface{}{
		"component": "event_processor",
	})

	return nil
}

// Health returns the health status of the processor
func (p *DefaultEventProcessor) Health() *core.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Event processor not initialized",
		}
	}

	if !p.running {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Event processor not running",
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "Event processor healthy",
	}
}

// ProcessEvent processes a single event
func (p *DefaultEventProcessor) ProcessEvent(event *core.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}

	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return fmt.Errorf("event processor not running")
	}
	p.mu.RUnlock()

	// Validate event
	if err := p.validateEvent(event); err != nil {
		p.mu.Lock()
		p.failedCount++
		p.mu.Unlock()

		p.metricsCollector.RecordCounter("event_processor_validation_failed", 1, map[string]string{
			"network": event.Network,
		})

		p.logger.Error("Event validation failed", map[string]interface{}{
			"error":   err.Error(),
			"network": event.Network,
		})

		return err
	}

	// Generate hash for idempotency
	hash, err := p.idempotencyService.GenerateHash(event)
	if err != nil {
		p.mu.Lock()
		p.failedCount++
		p.mu.Unlock()

		p.metricsCollector.RecordCounter("event_processor_hash_generation_failed", 1, map[string]string{
			"network": event.Network,
		})

		return err
	}

	// Check for duplicates
	isDuplicate, err := p.idempotencyService.IsDuplicate(hash)
	if err != nil {
		p.mu.Lock()
		p.failedCount++
		p.mu.Unlock()

		return err
	}

	if isDuplicate {
		p.mu.Lock()
		p.duplicateCount++
		p.mu.Unlock()

		p.metricsCollector.RecordCounter("event_processor_duplicate_detected", 1, map[string]string{
			"network": event.Network,
		})

		p.logger.Info("Duplicate event detected", map[string]interface{}{
			"hash":    hash,
			"network": event.Network,
		})

		return nil
	}

	// Store in database with retry logic
	err = p.storeEventWithRetry(event)
	if err != nil {
		p.mu.Lock()
		p.failedCount++
		p.mu.Unlock()

		p.metricsCollector.RecordCounter("event_processor_storage_failed", 1, map[string]string{
			"network": event.Network,
		})

		p.logger.Error("Event storage failed", map[string]interface{}{
			"error":   err.Error(),
			"network": event.Network,
		})

		return err
	}

	// Mark as processed
	err = p.idempotencyService.MarkProcessed(hash)
	if err != nil {
		p.logger.Error("Failed to mark event as processed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Update cache
	if p.cachePlugin != nil {
		cacheKey := fmt.Sprintf("event:%s:%d:%s", event.Network, event.BlockNumber, event.TransactionHash.Hex())
		// Serialize event to bytes (simplified - in production use proper serialization)
		eventBytes := []byte(fmt.Sprintf("%v", event))
		cacheEntry := &core.CacheEntry{
			Key:   cacheKey,
			Value: eventBytes,
			TTL:   3600, // 1 hour
		}

		err = p.cachePlugin.Set(cacheEntry)
		if err != nil {
			p.logger.Error("Failed to update cache", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	p.mu.Lock()
	p.processedCount++
	p.mu.Unlock()

	p.metricsCollector.RecordCounter("event_processor_event_processed", 1, map[string]string{
		"network": event.Network,
	})

	p.logger.Info("Event processed successfully", map[string]interface{}{
		"network":     event.Network,
		"blockNumber": event.BlockNumber,
	})

	return nil
}

// ProcessBatch processes a batch of events
func (p *DefaultEventProcessor) ProcessBatch(events []*core.BlockchainEvent) error {
	if len(events) == 0 {
		return nil
	}

	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return fmt.Errorf("event processor not running")
	}
	p.mu.RUnlock()

	successCount := 0
	failureCount := 0

	for _, event := range events {
		err := p.ProcessEvent(event)
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	p.metricsCollector.RecordCounter("event_processor_batch_processed", 1, map[string]string{
		"success": fmt.Sprintf("%d", successCount),
		"failure": fmt.Sprintf("%d", failureCount),
	})

	if failureCount > 0 {
		return fmt.Errorf("batch processing completed with %d failures", failureCount)
	}

	return nil
}

// GetProcessedCount returns the count of processed events
func (p *DefaultEventProcessor) GetProcessedCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.processedCount
}

// GetFailedCount returns the count of failed events
func (p *DefaultEventProcessor) GetFailedCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.failedCount
}

// GetDuplicateCount returns the count of duplicate events
func (p *DefaultEventProcessor) GetDuplicateCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.duplicateCount
}

// validateEvent validates event structure
func (p *DefaultEventProcessor) validateEvent(event *core.BlockchainEvent) error {
	if event.Network == "" {
		return fmt.Errorf("network is required")
	}

	if event.BlockNumber == 0 {
		return fmt.Errorf("block number is required")
	}

	if event.TransactionHash == (common.Hash{}) {
		return fmt.Errorf("transaction hash is required")
	}

	if event.ContractAddress == (common.Address{}) {
		return fmt.Errorf("contract address is required")
	}

	return nil
}

// storeEventWithRetry stores event with retry logic
func (p *DefaultEventProcessor) storeEventWithRetry(event *core.BlockchainEvent) error {
	var lastErr error

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(p.retryDelay * time.Duration(1<<uint(attempt-1))) // exponential backoff
		}

		err := p.databasePlugin.WriteEvent(event)
		if err == nil {
			return nil
		}

		lastErr = err

		p.logger.Warn("Event storage attempt failed", map[string]interface{}{
			"attempt": attempt + 1,
			"error":   err.Error(),
		})
	}

	return lastErr
}
