package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/consensus"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/domain"
	"github.com/rtcdance/chainpulse/pkg/integrations/generic"
)

// DefaultChainIndexer implements ChainIndexer for a specific blockchain
type DefaultChainIndexer struct {
	chainID                string
	database               core.DatabasePlugin
	cache                  core.CachePlugin
	logger                 core.Logger
	genericIndexer         *generic.GenericContractIndexer
	sharedRuntime          domain.SharedBatchRuntime
	metrics                core.MetricsCollector
	confirmationTracker    *consensus.ConfirmationTracker
	mu                     sync.RWMutex
	lastIndexedBlock       uint64
	totalEventsIndexed     int64
	shadowOwnedEvents      int64
	legacyOwnedEvents      int64
	totalErrorsEncountered int64
	startTime              time.Time
}

// NewDefaultChainIndexer creates a new chain-specific indexer
func NewDefaultChainIndexer(
	chainID string,
	database core.DatabasePlugin,
	cache core.CachePlugin,
	logger core.Logger,
	genericIndexer *generic.GenericContractIndexer,
) *DefaultChainIndexer {
	return &DefaultChainIndexer{
		chainID:        chainID,
		database:       database,
		cache:          cache,
		logger:         logger,
		genericIndexer: genericIndexer,
		startTime:      time.Now(),
	}
}

// SetSharedRuntime configures additive shared runtime shadow batch forwarding.
func (dci *DefaultChainIndexer) SetSharedRuntime(runtime domain.SharedBatchRuntime, metrics core.MetricsCollector) {
	dci.mu.Lock()
	defer dci.mu.Unlock()
	dci.sharedRuntime = runtime
	dci.metrics = metrics
}

// SetConfirmationTracker configures the optional confirmation tracker for
// tracking events through the Pending → Confirmed → Finalized lifecycle.
func (dci *DefaultChainIndexer) SetConfirmationTracker(tracker *consensus.ConfirmationTracker) {
	dci.mu.Lock()
	defer dci.mu.Unlock()
	dci.confirmationTracker = tracker
}

// IndexEvents indexes events for this chain
func (dci *DefaultChainIndexer) IndexEvents(
	ctx context.Context,
	events []*blockchain.BlockchainEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	dci.logger.Debug("indexing events for chain", "chain_id", dci.chainID, "count", len(events))

	validEvents := make([]*blockchain.BlockchainEvent, 0, len(events))

	// Validate all events belong to this chain
	for _, event := range events {
		if event.ChainID != dci.chainID {
			dci.mu.Lock()
			dci.totalErrorsEncountered++
			dci.mu.Unlock()

			dci.logger.Warn("event chain ID mismatch", "expected", dci.chainID, "got", event.ChainID)
			continue
		}

		validEvents = append(validEvents, event)
	}

	dci.forwardShadowBatch(ctx, validEvents)

	for _, event := range validEvents {
		if consumeShadowWrite(event) {
			if dci.metrics != nil {
				dci.metrics.RecordCounter("indexing_runtime_shadow_owned_events_total", 1, map[string]string{
					"chain_id":  dci.chainID,
					"service":   core.DeploymentMonolithic,
					"operation": "shadow_owned_write",
				})
			}
			dci.mu.Lock()
			if event.BlockNumber > dci.lastIndexedBlock {
				dci.lastIndexedBlock = event.BlockNumber
			}
			dci.totalEventsIndexed++
			dci.shadowOwnedEvents++
			dci.mu.Unlock()

			// Track event in confirmation tracker
			if dci.confirmationTracker != nil {
				dci.confirmationTracker.Track(event.ID, event.BlockNumber, event.BlockHash.Hex())
			}
			continue
		}

		// Index event through generic indexer
		if err := dci.indexEvent(ctx, event); err != nil {
			dci.mu.Lock()
			dci.totalErrorsEncountered++
			dci.mu.Unlock()

			dci.logger.Error("failed to index event", "chain_id", dci.chainID, "error", err)
			continue
		}

		// Update block tracking
		dci.mu.Lock()
		if event.BlockNumber > dci.lastIndexedBlock {
			dci.lastIndexedBlock = event.BlockNumber
		}
		dci.totalEventsIndexed++
		dci.legacyOwnedEvents++
		dci.mu.Unlock()

		// Track event in confirmation tracker
		if dci.confirmationTracker != nil {
			dci.confirmationTracker.Track(event.ID, event.BlockNumber, event.BlockHash.Hex())
		}
	}

	// Advance block in confirmation tracker with the highest block number
	if dci.confirmationTracker != nil && len(validEvents) > 0 {
		dci.mu.RLock()
		highestBlock := dci.lastIndexedBlock
		dci.mu.RUnlock()
		dci.confirmationTracker.AdvanceBlock(highestBlock)
	}

	return nil
}

func (dci *DefaultChainIndexer) forwardShadowBatch(ctx context.Context, events []*blockchain.BlockchainEvent) {
	if dci.sharedRuntime == nil || len(events) == 0 {
		return
	}

	envelopes := make([]core.EventEnvelope, 0, len(events))
	for _, event := range events {
		envelopes = append(envelopes, toEventEnvelope(event))
	}

	if err := dci.sharedRuntime.ProcessBatch(ctx, dci.chainID, envelopes); err != nil {
		dci.logger.Warn(
			"shared runtime shadow batch failed",
			"chain_id", dci.chainID,
			"count", len(envelopes),
			"error", err.Error(),
		)
		if dci.metrics != nil {
			dci.metrics.RecordCounter("indexing_runtime_shadow_batch_errors_total", 1, map[string]string{
				"chain_id":  dci.chainID,
				"service":   core.DeploymentMonolithic,
				"operation": "shadow_batch",
			})
		}
	}
}

func toEventEnvelope(event *blockchain.BlockchainEvent) core.EventEnvelope {
	return core.EventEnvelope{
		EventKey:         event.ID,
		ChainID:          event.ChainID,
		BlockNumber:      event.BlockNumber,
		TransactionHash:  event.TransactionHash.Hex(),
		LogIndex:         event.LogIndex,
		Payload:          event,
		ReceivedAt:       event.CreatedAt,
		CheckpointCursor: fmt.Sprintf("%s:%d:%d", event.ChainID, event.BlockNumber, event.LogIndex),
	}
}

func cacheKeyForEvent(chainID string, event *blockchain.BlockchainEvent) string {
	return fmt.Sprintf(
		"event:%s:%s:%d:%d",
		chainID,
		event.TransactionHash.Hex(),
		event.BlockNumber,
		event.LogIndex,
	)
}

// indexEvent indexes a single event
func (dci *DefaultChainIndexer) indexEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// Store event in database
	if err := dci.database.StoreEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Cache event for quick retrieval
	cacheKey := cacheKeyForEvent(dci.chainID, event)

	// Convert event ID to bytes for caching
	eventIDBytes := []byte(event.ID)
	if err := dci.cache.Set(ctx, cacheKey, eventIDBytes, eventCacheTTLSeconds); err != nil {
		dci.logger.Warn("failed to cache event", "error", err)
		// Don't fail if caching fails
	}

	return nil
}

// GetChainID returns the chain ID
func (dci *DefaultChainIndexer) GetChainID() string {
	return dci.chainID
}

// GetStatus returns status information for this chain indexer
func (dci *DefaultChainIndexer) GetStatus() map[string]any {
	dci.mu.RLock()
	defer dci.mu.RUnlock()

	uptime := time.Since(dci.startTime).Seconds()

	eventsPerSecond := float64(0)
	if uptime > 0 {
		eventsPerSecond = float64(dci.totalEventsIndexed) / uptime
	}
	total := dci.totalEventsIndexed + dci.totalErrorsEncountered
	errorRate := float64(0)
	if total > 0 {
		errorRate = float64(dci.totalErrorsEncountered) / float64(total)
	}

	status := map[string]any{
		"chain_id":             dci.chainID,
		"last_indexed_block":   dci.lastIndexedBlock,
		"total_events_indexed": dci.totalEventsIndexed,
		"shadow_owned_events":  dci.shadowOwnedEvents,
		"legacy_owned_events":  dci.legacyOwnedEvents,
		"total_errors":         dci.totalErrorsEncountered,
		"uptime_seconds":       uptime,
		"events_per_second":    eventsPerSecond,
		"error_rate":           errorRate,
	}

	if dci.confirmationTracker != nil {
		status["confirmation_pending"] = dci.confirmationTracker.PendingCount()
		status["confirmation_confirmed"] = dci.confirmationTracker.ConfirmedCount()
		status["confirmation_finalized"] = dci.confirmationTracker.FinalizedCount()
	}

	return status
}

// Close closes the chain indexer
func (dci *DefaultChainIndexer) Close() error {
	dci.logger.Info("closing chain indexer", "chain_id", dci.chainID)
	return nil
}

// GetLastIndexedBlock returns the last indexed block number
func (dci *DefaultChainIndexer) GetLastIndexedBlock() uint64 {
	dci.mu.RLock()
	defer dci.mu.RUnlock()

	return dci.lastIndexedBlock
}

// GetTotalEventsIndexed returns total events indexed
func (dci *DefaultChainIndexer) GetTotalEventsIndexed() int64 {
	dci.mu.RLock()
	defer dci.mu.RUnlock()

	return dci.totalEventsIndexed
}

// GetTotalErrors returns total errors encountered
func (dci *DefaultChainIndexer) GetTotalErrors() int64 {
	dci.mu.RLock()
	defer dci.mu.RUnlock()

	return dci.totalErrorsEncountered
}

// ResetStats resets indexer statistics
func (dci *DefaultChainIndexer) ResetStats() {
	dci.mu.Lock()
	defer dci.mu.Unlock()

	dci.lastIndexedBlock = 0
	dci.totalEventsIndexed = 0
	dci.shadowOwnedEvents = 0
	dci.legacyOwnedEvents = 0
	dci.totalErrorsEncountered = 0
	dci.startTime = time.Now()
}
