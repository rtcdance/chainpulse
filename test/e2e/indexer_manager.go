package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultIndexerManager implements IndexerManager
type DefaultIndexerManager struct {
	config         *IndexerConfig
	isRunning      bool
	mu             sync.RWMutex
	indexedEvents  map[string]*IndexedEvent
	metrics        IndexerMetrics
	startTime      time.Time
}

// NewIndexerManager creates a new indexer manager
func NewIndexerManager(config *IndexerConfig) (IndexerManager, error) {
	if config == nil {
		config = &IndexerConfig{
			Port:          8080,
			DatabaseURL:   "postgres://postgres:postgres@localhost:5432/chainpulse_test",
			BlockchainRPC: "http://localhost:8545",
			LogLevel:      "info",
			Timeout:       5 * time.Minute,
		}
	}

	return &DefaultIndexerManager{
		config:        config,
		indexedEvents: make(map[string]*IndexedEvent),
		metrics: IndexerMetrics{
			EventsProcessed: 0,
			EventsIndexed:   0,
			ErrorCount:      0,
		},
	}, nil
}

// StartIndexer starts the indexer service
func (im *DefaultIndexerManager) StartIndexer(ctx context.Context, config IndexerConfig) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.isRunning {
		return fmt.Errorf("indexer already running")
	}

	// Update config if provided
	if config.Port != 0 {
		im.config.Port = config.Port
	}
	if config.DatabaseURL != "" {
		im.config.DatabaseURL = config.DatabaseURL
	}
	if config.BlockchainRPC != "" {
		im.config.BlockchainRPC = config.BlockchainRPC
	}

	// In a real implementation, this would start the actual indexer service
	// For now, we'll just mark it as running
	im.isRunning = true
	im.startTime = time.Now()

	return nil
}

// StopIndexer stops the indexer
func (im *DefaultIndexerManager) StopIndexer(ctx context.Context) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.isRunning {
		return fmt.Errorf("indexer not running")
	}

	// In a real implementation, this would stop the actual indexer service
	im.isRunning = false

	return nil
}

// WaitForIndexing waits for events to be indexed
func (im *DefaultIndexerManager) WaitForIndexing(ctx context.Context, expectedCount int, timeout time.Duration) error {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if !im.isRunning {
		return fmt.Errorf("indexer not running")
	}

	// Wait for events to be indexed
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-time.After(time.Until(deadline)):
			return fmt.Errorf("timeout waiting for %d events to be indexed, got %d", expectedCount, len(im.indexedEvents))
		case <-ticker.C:
			if len(im.indexedEvents) >= expectedCount {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// GetIndexedEvents returns indexed events
func (im *DefaultIndexerManager) GetIndexedEvents(ctx context.Context, filter EventFilter) ([]*IndexedEvent, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if !im.isRunning {
		return nil, fmt.Errorf("indexer not running")
	}

	var results []*IndexedEvent

	for _, event := range im.indexedEvents {
		// Apply filters
		if filter.ContractAddress != "" && event.ContractAddress != filter.ContractAddress {
			continue
		}
		if filter.EventName != "" && event.EventName != filter.EventName {
			continue
		}
		if filter.ChainID != "" && event.ChainID != filter.ChainID {
			continue
		}

		results = append(results, event)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

// GetIndexerMetrics returns indexer performance metrics
func (im *DefaultIndexerManager) GetIndexerMetrics(ctx context.Context) IndexerMetrics {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.metrics
}

// AddIndexedEvent adds an indexed event (for testing)
func (im *DefaultIndexerManager) AddIndexedEvent(event *IndexedEvent) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.indexedEvents[event.ID] = event
	im.metrics.EventsIndexed++
}

// UpdateMetrics updates indexer metrics
func (im *DefaultIndexerManager) UpdateMetrics(metrics IndexerMetrics) {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.metrics = metrics
}

// ClearIndexedEvents clears all indexed events (for testing)
func (im *DefaultIndexerManager) ClearIndexedEvents() {
	im.mu.Lock()
	defer im.mu.Unlock()

	im.indexedEvents = make(map[string]*IndexedEvent)
	im.metrics.EventsIndexed = 0
}
