package processor

import (
	"context"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
"github.com/rtcdance/chainpulse/pkg/logkeys"
	"github.com/rtcdance/chainpulse/pkg/core/eventsig"
	"github.com/rtcdance/chainpulse/pkg/observability"
)

// EventStorage persists processed events.
// Deprecated: Use domain/query.EventStore for full event storage capabilities.
type EventStorage interface {
	WriteEvent(ctx context.Context, event *blockchain.BlockchainEvent) error
}

// CacheWriter is the minimal cache interface needed by the event processor.
// This decouples the processor from the concrete plugins/cache package.
type CacheWriter interface {
	Set(entry *core.CacheEntry) error
}

// EventProcessor processes events from message queue and stores them.
type EventProcessor interface {
	// Initialize initializes the event processor
	Initialize(config *core.Config) error

	// Start starts the event processor
	Start() error

	// Stop stops the event processor
	Stop() error

	// Health returns the health status of the processor
	Health() *core.HealthStatus

	// ProcessEvent processes a single event.
	// The context.Context parameter enables cancellation during graceful shutdown
	// and timeout for long-running storage operations.
	ProcessEvent(ctx context.Context, event *blockchain.BlockchainEvent) error

	// ProcessBatch processes a batch of events.
	// The context.Context parameter enables cancellation of the entire batch.
	ProcessBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error

	// GetProcessedCount returns the count of processed events
	GetProcessedCount() int64

	// GetFailedCount returns the count of failed events
	GetFailedCount() int64

	// GetDuplicateCount returns the count of duplicate events
	GetDuplicateCount() int64
}

// DefaultEventProcessor provides default event processing implementation.
type DefaultEventProcessor struct {
	mu                 sync.RWMutex
	initialized        bool
	running            bool
	config             *core.Config
	logger             core.Logger
	metricsCollector   core.MetricsCollector
	idempotencyService IdempotencyService
	cachePlugin        CacheWriter
	databasePlugin     EventStorage
	eventBus           core.EventBus
	processedCount     int64
	failedCount        int64
	duplicateCount     int64
	lastHealthCheck    *core.HealthStatus
	batchSize          int
	maxRetries         int
	retryDelay         time.Duration
	tracer             *observability.DefaultTracer
}

// NewDefaultEventProcessor creates a new event processor.
func NewDefaultEventProcessor(
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	idempotencyService IdempotencyService,
	cachePlugin CacheWriter,
	databasePlugin EventStorage,
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
		tracer:             observability.NewDefaultTracer(logger, metricsCollector),
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

	p.logger.Info("Event processor initialized", logkeys.LogKeyComponent, "event_processor")

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

	if p.idempotencyService != nil {
		if err := p.idempotencyService.Initialize(p.config); err != nil {
			return err
		}
		if err := p.idempotencyService.Start(); err != nil {
			return err
		}
	}

	p.running = true
	p.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Event processor started",
	}

	p.logger.Info("Event processor started", logkeys.LogKeyComponent, "event_processor")

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

	p.logger.Info("Event processor stopped", logkeys.LogKeyComponent, "event_processor")

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

// ProcessEvent processes a single event.
//
// NOTE: The running check below is a best-effort guard (TOCTOU pattern). A full
// solution would require holding the lock through the entire operation, which
// would serialize all processing. In practice, Stop() and ProcessEvent() are
// called from different lifecycle phases and should not race. If stricter
// guarantees are needed, use a state machine.
//
//nolint:funlen // ProcessEvent has many statements for validation and processing steps.
func (p *DefaultEventProcessor) ProcessEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}

	ctx, span := p.tracer.StartSpan(ctx, "processor.process_event", observability.SpanKindInternal)
	defer p.tracer.EndSpan(&span)
	p.tracer.SetAttribute(&span, "event_id", event.ID)
	p.tracer.SetAttribute(&span, "chain_id", event.ChainID)

	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		p.logger.Warn("ProcessEvent rejected: processor not running", logkeys.LogKeyEventID, event.ID)
		return fmt.Errorf("event processor not running")
	}
	p.mu.RUnlock()

	// Re-resolve eventName from EventSignature if it looks like an unresolved hex hash
	if strings.HasPrefix(event.EventName, "0x") && event.EventSignature != (common.Hash{}) {
		if resolved := eventsig.ResolveEventNameFromTopic(event.EventSignature.Hex()); resolved != event.EventSignature.Hex() {
			event.EventName = resolved
		}
	}

	// Validate event
	if err := p.validateEvent(event); err != nil {
		p.mu.Lock()
		p.failedCount++
		p.mu.Unlock()

		p.metricsCollector.RecordCounter("event_processor_validation_failed", 1, map[string]string{
			"network": event.Network,
		})

		p.logger.Error("Event validation failed", logkeys.LogKeyError, err, logkeys.LogKeyNetwork, event.Network)

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

		p.logger.Error("Hash generation failed", logkeys.LogKeyError, err, logkeys.LogKeyNetwork, event.Network, logkeys.LogKeyEventID, event.ID)

		return err
	}

	// Check for duplicates
	isDuplicate, err := p.idempotencyService.IsDuplicate(ctx, hash)
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

		p.logger.Info("Duplicate event detected", logkeys.LogKeyHash, hash, logkeys.LogKeyNetwork, event.Network)

		return nil
	}

	// Store in database with retry logic (context-aware)
	err = p.storeEventWithRetry(ctx, event)
	if err != nil {
		p.mu.Lock()
		p.failedCount++
		p.mu.Unlock()

		p.metricsCollector.RecordCounter("event_processor_storage_failed", 1, map[string]string{
			"network": event.Network,
		})

		p.logger.Error("Event storage failed", logkeys.LogKeyError, err, logkeys.LogKeyNetwork, event.Network)

		return err
	}

	// Mark as processed with retry
	err = p.idempotencyService.MarkProcessed(ctx, hash)
	for attempt := 0; attempt < p.maxRetries && err != nil; attempt++ {
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			break
		case <-time.After(p.retryDelay):
		}
		err = p.idempotencyService.MarkProcessed(ctx, hash)
	}
	if err != nil {
		p.logger.Error("Failed to mark event as processed after retries",
			logkeys.LogKeyError, err, "attempts", p.maxRetries)
	}

	// Update cache
	if p.cachePlugin != nil {
		cacheKey := "event:" + event.Network + ":" + strconv.FormatUint(event.BlockNumber, 10) + ":" + event.TransactionHash.Hex()
		eventBytes := []byte(fmt.Sprintf("%v", event))
		cacheEntry := &core.CacheEntry{
			Key:   cacheKey,
			Value: eventBytes,
			TTL:   3600, // 1 hour
		}

		err = p.cachePlugin.Set(cacheEntry)
		if err != nil {
			p.logger.Error("Failed to update cache", logkeys.LogKeyError, err)
		}
	}

	p.mu.Lock()
	p.processedCount++
	p.mu.Unlock()

	p.metricsCollector.RecordCounter("event_processor_event_processed", 1, map[string]string{
		"network": event.Network,
	})

	p.logger.Info("Event processed successfully", logkeys.LogKeyNetwork, event.Network, logkeys.LogKeyBlockNumber, event.BlockNumber)

	return nil
}

// ProcessBatch processes a batch of events
func (p *DefaultEventProcessor) ProcessBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error {
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
		// Check context cancellation between events in the batch
		if err := ctx.Err(); err != nil {
			p.logger.Warn("Batch cancelled", logkeys.LogKeyError, err, logkeys.LogKeyBatchSize, len(events), logkeys.LogKeyProcessed, successCount+failureCount)
			return fmt.Errorf("batch processing cancelled: %w", err)
		}

		err := p.ProcessEvent(ctx, event)
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
func (p *DefaultEventProcessor) validateEvent(event *blockchain.BlockchainEvent) error {
	if event.Network == "" {
		return fmt.Errorf("network is required")
	}

	// Solana and other non-EVM networks use different address/transaction formats
	isNonEVM := event.Network == "solana"

	if event.BlockNumber == 0 {
		return fmt.Errorf("block number is required")
	}

	if event.TransactionHash == (common.Hash{}) && !isNonEVM {
		return fmt.Errorf("transaction hash is required")
	}

	if event.ContractAddress == (common.Address{}) && !isNonEVM {
		return fmt.Errorf("contract address is required")
	}

	return nil
}

// storeEventWithRetry stores event with retry logic.
// Uses context-aware backoff instead of time.Sleep for graceful shutdown support.
func (p *DefaultEventProcessor) storeEventWithRetry(ctx context.Context, event *blockchain.BlockchainEvent) error {
	if p.databasePlugin == nil {
		p.logger.Error("database plugin is required for event storage", logkeys.LogKeyEventID, event.ID)
		return fmt.Errorf("database plugin is required")
	}

	var lastErr error

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Check context before each attempt
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("store event cancelled: %w", err)
		}

		if attempt > 0 {
			backoff := p.retryDelay * time.Duration(boundedRetryMultiplier(attempt))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("store event cancelled during backoff: %w", ctx.Err())
			}
		}

		err := p.databasePlugin.WriteEvent(ctx, event)
		if err == nil {
			return nil
		}

		lastErr = err

		p.logger.Warn("Event storage attempt failed", logkeys.LogKeyAttempt, attempt+1, logkeys.LogKeyError, err)
	}

	return lastErr
}

func boundedRetryMultiplier(attempt int) int {
	shift := attempt - 1
	maxShift := bits.UintSize - 2

	if shift < 0 {
		shift = 0
	} else if shift > maxShift {
		shift = maxShift
	}

	return 1 << shift
}
