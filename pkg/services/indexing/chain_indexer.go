package indexing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/integrations/generic"
)

// DefaultChainIndexer implements ChainIndexer for a specific blockchain
type DefaultChainIndexer struct {
	chainID              string
	database             core.DatabasePlugin
	cache                core.CachePlugin
	logger               core.Logger
	genericIndexer       *generic.GenericContractIndexer
	mu                   sync.RWMutex
	lastIndexedBlock     uint64
	totalEventsIndexed   int64
	totalErrorsEncountered int64
	startTime            time.Time
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

// IndexEvents indexes events for this chain
func (dci *DefaultChainIndexer) IndexEvents(
	ctx context.Context,
	events []*core.BlockchainEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	dci.logger.Debug("indexing events for chain", "chain_id", dci.chainID, "count", len(events))

	// Validate all events belong to this chain
	for _, event := range events {
		if event.ChainID != dci.chainID {
			dci.mu.Lock()
			dci.totalErrorsEncountered++
			dci.mu.Unlock()

			dci.logger.Warn("event chain ID mismatch", "expected", dci.chainID, "got", event.ChainID)
			continue
		}

		// Index event through generic indexer
		if err := dci.indexEvent(ctx, event); err != nil {
			dci.mu.Lock()
			dci.totalErrorsEncountered++
			dci.mu.Unlock()

			dci.logger.Error("failed to index event", "chain_id", dci.chainID, "error", err.Error())
			continue
		}

		// Update block tracking
		dci.mu.Lock()
		if event.BlockNumber > dci.lastIndexedBlock {
			dci.lastIndexedBlock = event.BlockNumber
		}
		dci.totalEventsIndexed++
		dci.mu.Unlock()
	}

	return nil
}

// indexEvent indexes a single event
func (dci *DefaultChainIndexer) indexEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// Store event in database
	if err := dci.database.StoreEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Cache event for quick retrieval
	cacheKey := fmt.Sprintf("event:%s:%s:%d:%d",
		dci.chainID,
		event.TransactionHash.Hex(),
		event.BlockNumber,
		event.LogIndex,
	)

	// Convert event ID to bytes for caching
	eventIDBytes := []byte(event.ID)
	if err := dci.cache.Set(ctx, cacheKey, eventIDBytes, 24*3600); err != nil {
		dci.logger.Warn("failed to cache event", "error", err.Error())
		// Don't fail if caching fails
	}

	return nil
}

// GetChainID returns the chain ID
func (dci *DefaultChainIndexer) GetChainID() string {
	return dci.chainID
}

// GetStatus returns status information for this chain indexer
func (dci *DefaultChainIndexer) GetStatus() map[string]interface{} {
	dci.mu.RLock()
	defer dci.mu.RUnlock()

	uptime := time.Since(dci.startTime).Seconds()

	return map[string]interface{}{
		"chain_id":                  dci.chainID,
		"last_indexed_block":        dci.lastIndexedBlock,
		"total_events_indexed":      dci.totalEventsIndexed,
		"total_errors":              dci.totalErrorsEncountered,
		"uptime_seconds":            uptime,
		"events_per_second":         float64(dci.totalEventsIndexed) / uptime,
		"error_rate":                float64(dci.totalErrorsEncountered) / float64(dci.totalEventsIndexed+dci.totalErrorsEncountered),
	}
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
	dci.totalErrorsEncountered = 0
	dci.startTime = time.Now()
}
