package indexing

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"chainpulse/pkg/core"
	"golang.org/x/sync/errgroup"
)

// MultiChainIndexer orchestrates indexing across multiple blockchains
type MultiChainIndexer struct {
	indexers map[string]ChainIndexer
	mu       sync.RWMutex
	logger   core.Logger
	config   core.ConfigManager
}

// ChainIndexer defines the interface for chain-specific indexing
type ChainIndexer interface {
	IndexEvents(ctx context.Context, events []*core.BlockchainEvent) error
	GetChainID() string
	GetStatus() map[string]interface{}
	Close() error
}

// NewMultiChainIndexer creates a new multi-chain indexer
func NewMultiChainIndexer(logger core.Logger, config core.ConfigManager) *MultiChainIndexer {
	return &MultiChainIndexer{
		indexers: make(map[string]ChainIndexer),
		logger:   logger,
		config:   config,
	}
}

// RegisterChainIndexer registers an indexer for a specific blockchain
func (mci *MultiChainIndexer) RegisterChainIndexer(chainID string, indexer ChainIndexer) error {
	if chainID == "" {
		return fmt.Errorf("chain ID cannot be empty")
	}

	if indexer == nil {
		return fmt.Errorf("indexer cannot be nil")
	}

	mci.mu.Lock()
	defer mci.mu.Unlock()

	if _, exists := mci.indexers[chainID]; exists {
		return fmt.Errorf("indexer already registered for chain %s", chainID)
	}

	mci.indexers[chainID] = indexer
	mci.logger.Info("chain indexer registered", "chain_id", chainID)

	return nil
}

// IndexEventsFromChain indexes events from a specific blockchain
func (mci *MultiChainIndexer) IndexEventsFromChain(
	ctx context.Context,
	chainID string,
	events []*core.BlockchainEvent,
) error {
	if chainID == "" {
		return fmt.Errorf("chain ID cannot be empty")
	}

	if len(events) == 0 {
		return nil
	}

	mci.mu.RLock()
	indexer, exists := mci.indexers[chainID]
	mci.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no indexer registered for chain %s", chainID)
	}

	mci.logger.Debug("indexing events from chain", "chain_id", chainID, "count", len(events))

	if err := indexer.IndexEvents(ctx, events); err != nil {
		mci.logger.Error("failed to index events", "chain_id", chainID, "error", err.Error())
		return err
	}

	return nil
}

// IndexEventsFromAllChains indexes events from all registered blockchains in parallel
func (mci *MultiChainIndexer) IndexEventsFromAllChains(
	ctx context.Context,
	eventsByChain map[string][]*core.BlockchainEvent,
) error {
	if len(eventsByChain) == 0 {
		return nil
	}

	mci.mu.RLock()
	registeredChains := make([]string, 0, len(mci.indexers))
	for chainID := range mci.indexers {
		registeredChains = append(registeredChains, chainID)
	}
	mci.mu.RUnlock()

	// Validate all chains have indexers
	for chainID := range eventsByChain {
		found := false
		for _, registered := range registeredChains {
			if registered == chainID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no indexer registered for chain %s", chainID)
		}
	}

	// Index from all chains in parallel using errgroup
	g, gCtx := errgroup.WithContext(ctx)

	for chainID, events := range eventsByChain {
		cID, evts := chainID, events
		g.Go(func() error {
			return mci.IndexEventsFromChain(gCtx, cID, evts)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	return nil
}

// GetChainIndexer retrieves an indexer for a specific chain
func (mci *MultiChainIndexer) GetChainIndexer(chainID string) (ChainIndexer, error) {
	mci.mu.RLock()
	defer mci.mu.RUnlock()

	indexer, exists := mci.indexers[chainID]
	if !exists {
		return nil, fmt.Errorf("no indexer registered for chain %s", chainID)
	}

	return indexer, nil
}

// GetRegisteredChains returns list of registered chain IDs
func (mci *MultiChainIndexer) GetRegisteredChains() []string {
	mci.mu.RLock()
	defer mci.mu.RUnlock()

	chains := make([]string, 0, len(mci.indexers))
	for chainID := range mci.indexers {
		chains = append(chains, chainID)
	}

	return chains
}

// GetStatus returns status of all chain indexers
func (mci *MultiChainIndexer) GetStatus() map[string]map[string]interface{} {
	mci.mu.RLock()
	defer mci.mu.RUnlock()

	status := make(map[string]map[string]interface{})
	for chainID, indexer := range mci.indexers {
		status[chainID] = indexer.GetStatus()
	}

	return status
}

// Close closes all chain indexers
func (mci *MultiChainIndexer) Close() error {
	mci.mu.Lock()
	defer mci.mu.Unlock()

	var errs []error
	for chainID, indexer := range mci.indexers {
		if err := indexer.Close(); err != nil {
			mci.logger.Error("failed to close chain indexer", "chain_id", chainID, "error", err.Error())
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d indexers: %w", len(errs), errors.Join(errs...))
	}

	return nil
}

// IsMultiChain returns true if multiple chains are registered
func (mci *MultiChainIndexer) IsMultiChain() bool {
	mci.mu.RLock()
	defer mci.mu.RUnlock()

	return len(mci.indexers) > 1
}

// GetIndexerCount returns the number of registered indexers
func (mci *MultiChainIndexer) GetIndexerCount() int {
	mci.mu.RLock()
	defer mci.mu.RUnlock()

	return len(mci.indexers)
}
