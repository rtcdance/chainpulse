package processor

import (
	"context"
	"fmt"
	"math/bits"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/errgroup"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/observability"
)

const (
	metricValidationFailed     = "event_processor_validation_failed"
	metricHashGenFailed        = "event_processor_hash_generation_failed"
	metricDuplicateCheckFailed = "event_processor_duplicate_check_failed"
	metricDuplicateDetected    = "event_processor_duplicate_detected"
	metricStorageFailed        = "event_processor_storage_failed"
	metricEventProcessed       = "event_processor_event_processed"
	metricBatchProcessed       = "event_processor_batch_processed"
	metricBatchAtomic          = "event_processor_batch_atomic"

	tagNetwork = "network"
	tagSuccess = "success"
	tagFailure = "failure"
	tagSize    = "size"
)

// EventStorage persists processed events.
// Deprecated: Use domain/query.EventStore for full event storage capabilities.
// EventStorage provides write access to blockchain events for the processor.
type EventStorage interface {
	WriteEvent(ctx context.Context, event *core.BlockchainEvent) error
	// WriteBatch writes multiple events atomically if the underlying
	// storage supports transactions. If not, it falls back to individual
	// writes. Implementations should wrap all writes in a single DB
	// transaction when possible.
	WriteBatch(ctx context.Context, events []*core.BlockchainEvent) error
	DeleteEvent(ctx context.Context, eventID string) error
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
	ProcessEvent(ctx context.Context, event *core.BlockchainEvent) error

	// ProcessBatch processes a batch of events.
	// The context.Context parameter enables cancellation of the entire batch.
	ProcessBatch(ctx context.Context, events []*core.BlockchainEvent) error

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
	cacheEntryPool     sync.Pool
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
		cacheEntryPool: sync.Pool{
			New: func() any {
				return &core.CacheEntry{}
			},
		},
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

	p.logger.Info("Event processor initialized", core.LogKeyComponent, "event_processor")

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
			return fmt.Errorf("initialize idempotency service: %w", err)
		}
		if err := p.idempotencyService.Start(); err != nil {
			return fmt.Errorf("start idempotency service: %w", err)
		}
	}

	p.running = true
	p.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Event processor started",
	}

	p.logger.Info("Event processor started", core.LogKeyComponent, "event_processor")

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

	p.logger.Info("Event processor stopped", core.LogKeyComponent, "event_processor")

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

// ProcessEvent processes a single event through the full pipeline:
// validate → deduplicate → store → mark processed → cache → publish.
func (p *DefaultEventProcessor) ProcessEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}

	ctx, span := p.tracer.StartSpan(ctx, "processor.process_event", observability.SpanKindInternal)
	defer p.tracer.EndSpan(&span)
	p.tracer.SetAttribute(&span, "event_id", event.ID)
	p.tracer.SetAttribute(&span, "chain_id", event.ChainID)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled during event processing: %w", err)
	}

	if !p.isRunning() {
		p.logger.Warn("ProcessEvent rejected: processor not running", core.LogKeyEventID, event.ID)
		return fmt.Errorf("event processor not running")
	}

	if err := p.validateEvent(event); err != nil {
		p.recordFailureMetric(metricValidationFailed, tagNetwork, event.Network)
		p.logger.Error("Event validation failed", core.LogKeyError, err, core.LogKeyNetwork, event.Network)
		return fmt.Errorf("validate event %s: %w", event.ID, err)
	}

	hash, err := p.idempotencyService.GenerateHash(event)
	if err != nil {
		p.recordFailureMetric(metricHashGenFailed, tagNetwork, event.Network)
		p.logger.Error("Hash generation failed", core.LogKeyError, err, core.LogKeyNetwork, event.Network, core.LogKeyEventID, event.ID)
		return fmt.Errorf("generate hash for event %s: %w", event.ID, err)
	}

	if p.isDuplicateEvent(ctx, hash, event.Network) {
		return nil
	}

	if err := p.storeEventWithRetry(ctx, event); err != nil {
		p.recordFailureMetric(metricStorageFailed, tagNetwork, event.Network)
		p.logger.Error("Event storage failed", core.LogKeyError, err, core.LogKeyNetwork, event.Network)
		return fmt.Errorf("store event %s with retry: %w", event.ID, err)
	}

	p.markProcessedWithRollback(ctx, hash, event)

	p.updateCache(ctx, event)

	p.recordProcessedMetrics(event)

	p.publishToEventBus(ctx, event)

	p.logger.Info("Event processed successfully", core.LogKeyNetwork, event.Network, core.LogKeyBlockNumber, event.BlockNumber)
	return nil
}

// isRunning checks if the processor is in a running state.
func (p *DefaultEventProcessor) isRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// isDuplicateEvent checks the idempotency store for an existing hash.
// Returns true (and records metrics) if the event was already processed.
func (p *DefaultEventProcessor) isDuplicateEvent(ctx context.Context, hash, network string) bool {
	isDup, err := p.idempotencyService.IsDuplicate(ctx, hash)
	if err != nil {
		p.recordFailureMetric(metricDuplicateCheckFailed, tagNetwork, network)
		return false
	}
	if isDup {
		p.mu.Lock()
		p.duplicateCount++
		p.mu.Unlock()
		p.metricsCollector.RecordCounter(metricDuplicateDetected, 1, map[string]string{tagNetwork: network})
		p.logger.Info("Duplicate event detected", core.LogKeyHash, hash, core.LogKeyNetwork, network)
	}
	return isDup
}

// markProcessedWithRollback marks the event as processed in the idempotency store
// with retries. If all retries fail, it attempts to roll back by deleting the event.
func (p *DefaultEventProcessor) markProcessedWithRollback(ctx context.Context, hash string, event *core.BlockchainEvent) {
	err := p.idempotencyService.MarkProcessed(ctx, hash)
	for attempt := 0; attempt < p.maxRetries && err != nil; attempt++ {
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(p.retryDelay):
		}
		err = p.idempotencyService.MarkProcessed(ctx, hash)
	}
	if err != nil {
		p.logger.Error("Failed to mark event as processed after retries, attempting rollback",
			core.LogKeyError, err, "attempts", p.maxRetries)
		if delErr := p.deleteEvent(ctx, event); delErr != nil {
			p.logger.Error("Failed to rollback stored event after MarkProcessed failure",
				core.LogKeyError, delErr, "hash", hash)
		} else {
			p.logger.Info("Rolled back event after MarkProcessed failure", "hash", hash)
		}
	}
}

func (p *DefaultEventProcessor) writeCacheEntry(event *core.BlockchainEvent) {
	if p.cachePlugin == nil {
		return
	}
	cacheKey := "event:" + event.Network + ":" + strconv.FormatUint(event.BlockNumber, 10) + ":" + event.TransactionHash.Hex()
	eventBytes := []byte(fmt.Sprintf("%v", event))
	cacheEntry := p.cacheEntryPool.Get().(*core.CacheEntry)
	cacheEntry.Key = cacheKey
	cacheEntry.Value = eventBytes
	cacheEntry.TTL = 3600

	err := p.cachePlugin.Set(cacheEntry)
	cacheEntry.Key = ""
	cacheEntry.Value = nil
	p.cacheEntryPool.Put(cacheEntry)
	if err != nil {
		p.logger.Error("Failed to update cache", core.LogKeyError, err)
	}
}

func (p *DefaultEventProcessor) updateCache(_ context.Context, event *core.BlockchainEvent) {
	if p.cachePlugin == nil {
		return
	}
	p.writeCacheEntry(event)
}

// recordProcessedMetrics updates counters for a successfully processed event.
func (p *DefaultEventProcessor) recordProcessedMetrics(event *core.BlockchainEvent) {
	p.mu.Lock()
	p.processedCount++
	p.mu.Unlock()

	p.metricsCollector.RecordCounter(metricEventProcessed, 1, map[string]string{
		tagNetwork: event.Network,
	})
}

// publishToEventBus publishes the event to the event bus for push-based delivery.
func (p *DefaultEventProcessor) publishToEventBus(ctx context.Context, event *core.BlockchainEvent) {
	if p.eventBus == nil {
		return
	}
	if pubErr := p.eventBus.Publish(ctx, "event:created", event); pubErr != nil {
		p.logger.Error("Failed to publish event to event bus", core.LogKeyError, pubErr)
	}
}

// recordFailureMetric increments the failure counter and records a metric.
func (p *DefaultEventProcessor) recordFailureMetric(metricName, tagKey, tagValue string) {
	p.mu.Lock()
	p.failedCount++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter(metricName, 1, map[string]string{tagKey: tagValue})
}

// ProcessBatch processes a batch of events with bounded concurrency.
// When the underlying database supports it, events are written atomically
// via WriteBatch before idempotency marking, then each event's post-write
// steps (idempotency mark, cache update) proceed in parallel.
// If WriteBatch is not available or fails, falls back to per-event processing.
// Returns the first error encountered if any failures occur.
func (p *DefaultEventProcessor) ProcessBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	if len(events) == 0 {
		return nil
	}

	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return fmt.Errorf("event processor not running")
	}
	p.mu.RUnlock()

	// Attempt atomic batch write first
	if p.databasePlugin != nil {
		batchErr := p.databasePlugin.WriteBatch(ctx, events)
		if batchErr == nil {
			// Batch write succeeded. Now mark all as processed in parallel.
			return p.markBatchProcessed(ctx, events)
		}
		// Batch write failed; log and fall through to per-event processing
		p.logger.Warn(
			"WriteBatch failed, falling back to per-event processing",
			"batch_size", len(events),
			"error", batchErr.Error(),
		)
	}

	// Fallback: per-event processing via errgroup
	var successCount, failureCount atomic.Int64
	var firstErr atomic.Value

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, event := range events {
		event := event
		g.Go(func() error {
			if err := p.ProcessEvent(gCtx, event); err != nil {
				failureCount.Add(1)
				firstErr.CompareAndSwap(nil, err)
				return fmt.Errorf("process event in batch: %w", err)
			}
			successCount.Add(1)
			return nil
		})
	}

	batchErr := g.Wait()

	p.metricsCollector.RecordCounter(metricBatchProcessed, 1, map[string]string{
		tagSuccess: fmt.Sprintf("%d", successCount.Load()),
		tagFailure: fmt.Sprintf("%d", failureCount.Load()),
	})

	failure := failureCount.Load()
	if failure > 0 {
		if errVal := firstErr.Load(); errVal != nil {
			firstErrMsg := "unknown error"
			if e, ok := errVal.(error); ok {
				firstErrMsg = e.Error()
			}
			p.logger.Error(
				"batch processing completed with failures",
				"total", len(events),
				"success", successCount.Load(),
				"failure", failure,
				"first_error", firstErrMsg,
			)
		}
		if batchErr != nil {
			return fmt.Errorf("batch processing completed with %d/%d failures: %w", failure, len(events), batchErr)
		}
		return fmt.Errorf("batch processing completed with %d/%d failures", failure, len(events))
	}

	return nil
}

// markBatchProcessed marks all events in a batch as processed in the idempotency
// store after a successful atomic database write. This runs in parallel.
func (p *DefaultEventProcessor) markBatchProcessed(ctx context.Context, events []*core.BlockchainEvent) error {
	var failureCount atomic.Int64
	var firstErr atomic.Value

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, event := range events {
		event := event
		g.Go(func() error {
			hash, err := p.idempotencyService.GenerateHash(event)
			if err != nil {
				failureCount.Add(1)
				firstErr.CompareAndSwap(nil, err)
				return fmt.Errorf("generate hash for batch mark: %w", err)
			}

			if err := p.idempotencyService.MarkProcessed(gCtx, hash); err != nil {
				failureCount.Add(1)
				firstErr.CompareAndSwap(nil, err)
				return fmt.Errorf("mark processed in batch: %w", err)
			}

			// Update cache
			if p.cachePlugin != nil {
				p.writeCacheEntry(event)
			}

			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		p.logger.Error(
			"batch idempotency marking completed with failures",
			"batch_size", len(events),
			"failures", failureCount.Load(),
		)
		return fmt.Errorf("batch idempotency marking: %w", err)
	}

	p.mu.Lock()
	p.processedCount += int64(len(events))
	p.mu.Unlock()

	p.metricsCollector.RecordCounter(metricBatchAtomic, 1, map[string]string{
		tagSize: fmt.Sprintf("%d", len(events)),
	})

	return nil
}

// deleteEvent removes a stored event from the database when MarkProcessed fails.
// This prevents duplicate processing on replay. It uses the event's hash as the
// deletion key, falling back to network+tx_hash if the database supports it.
func (p *DefaultEventProcessor) deleteEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if p.databasePlugin == nil {
		return fmt.Errorf("database not configured for event rollback")
	}
	eventID := fmt.Sprintf("%s:%d:%s", event.Network, event.BlockNumber, event.TransactionHash.Hex())
	return p.databasePlugin.DeleteEvent(ctx, eventID)
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

	isNonEVM := event.NativeAddress != ""

	if !isNonEVM && event.TransactionHash == (common.Hash{}) {
		return fmt.Errorf("transaction hash is required")
	}

	if !isNonEVM && event.ContractAddress == (common.Address{}) {
		return fmt.Errorf("contract address is required")
	}

	return nil
}

// storeEventWithRetry stores event with retry logic.
// Uses context-aware backoff instead of time.Sleep for graceful shutdown support.
func (p *DefaultEventProcessor) storeEventWithRetry(ctx context.Context, event *core.BlockchainEvent) error {
	if p.databasePlugin == nil {
		p.logger.Error("database plugin is required for event storage", core.LogKeyEventID, event.ID)
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

		p.logger.Warn("Event storage attempt failed", core.LogKeyAttempt, attempt+1, core.LogKeyError, err)
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
