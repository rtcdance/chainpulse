package pullers

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"chainpulse/pkg/core"
)

type dataPullerLastProcessedBlockProvider interface {
	GetLastBlockNumber() uint64
}

type dataPullerLastProcessedBlockSetter interface {
	SetLastBlockNumber(uint64)
}

// MultiChainDataPuller manages data pulling from multiple blockchains
type MultiChainDataPuller struct {
	pullers map[string]core.DataPullerPlugin
	mu      sync.RWMutex
	logger  core.Logger
}

// NewMultiChainDataPuller creates a new multi-chain data puller
func NewMultiChainDataPuller(logger core.Logger) *MultiChainDataPuller {
	return &MultiChainDataPuller{
		pullers: make(map[string]core.DataPullerPlugin),
		logger:  logger,
	}
}

// RegisterPuller registers a data puller for a specific blockchain
func (m *MultiChainDataPuller) RegisterPuller(chainID string, puller core.DataPullerPlugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pullers[chainID]; exists {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			fmt.Sprintf("puller already registered for chain %s", chainID),
			nil,
		)
	}

	m.pullers[chainID] = puller
	if m.logger != nil {
		m.logger.Info("puller registered", "chain_id", chainID)
	}
	return nil
}

// RegisteredChains returns the registered chain IDs in stable order.
func (m *MultiChainDataPuller) RegisteredChains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]string, 0, len(m.pullers))
	for chainID := range m.pullers {
		chains = append(chains, chainID)
	}
	sort.Strings(chains)
	return chains
}

// SetLastProcessedBlock updates the last processed block for a registered
// puller when that puller surfaces mutable progress state.
func (m *MultiChainDataPuller) SetLastProcessedBlock(chainID string, blockNumber uint64) error {
	m.mu.RLock()
	puller, exists := m.pullers[chainID]
	m.mu.RUnlock()

	if !exists {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeNotFound,
			fmt.Sprintf("no puller registered for chain %s", chainID),
			nil,
		)
	}

	setter, ok := puller.(dataPullerLastProcessedBlockSetter)
	if !ok {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeValidation,
			fmt.Sprintf("puller does not support progress updates for chain %s", chainID),
			nil,
		)
	}

	setter.SetLastBlockNumber(blockNumber)
	return nil
}

// PullEventsFromChain pulls events from a specific blockchain
func (m *MultiChainDataPuller) PullEventsFromChain(
	ctx context.Context,
	chainID string,
	fromBlock, toBlock uint64,
) ([]core.BlockchainEvent, error) {
	m.mu.RLock()
	puller, exists := m.pullers[chainID]
	m.mu.RUnlock()

	if !exists {
		return nil, core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeNotFound,
			fmt.Sprintf("no puller registered for chain %s", chainID),
			nil,
		)
	}

	events, err := puller.PullEvents(ctx, fromBlock, toBlock)
	if err != nil {
		if m.logger != nil {
			m.logger.Error("failed to pull events", "chain_id", chainID, "error", err.Error())
		}
		return nil, err
	}

	// Validate that all events carry the expected chain ID
	for i := range events {
		if events[i].ChainID != chainID {
			if m.logger != nil {
				m.logger.Warn("event chain ID mismatch — correcting",
					"expected", chainID, "got", events[i].ChainID, "event_hash", events[i].EventHash)
			}
			events[i].ChainID = chainID
		}
	}

	return events, nil
}

// PullEventsFromAllChains pulls events from all registered blockchains in parallel
func (m *MultiChainDataPuller) PullEventsFromAllChains(
	ctx context.Context,
	fromBlock, toBlock uint64,
) (map[string][]core.BlockchainEvent, error) {
	m.mu.RLock()
	chains := make([]string, 0, len(m.pullers))
	for chainID := range m.pullers {
		chains = append(chains, chainID)
	}
	m.mu.RUnlock()

	results := make(map[string][]core.BlockchainEvent)
	resultsMu := sync.Mutex{}
	errChan := make(chan error, len(chains))

	// Pull from all chains in parallel
	var wg sync.WaitGroup
	for _, chainID := range chains {
		wg.Add(1)
		go func(cid string) {
			defer wg.Done()

			events, err := m.PullEventsFromChain(ctx, cid, fromBlock, toBlock)
			if err != nil {
				errChan <- fmt.Errorf("chain %s: %w", cid, err)
				return
			}

			resultsMu.Lock()
			results[cid] = events
			resultsMu.Unlock()
		}(chainID)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 && m.logger != nil {
		m.logger.Warn("some chains failed to pull events", "error_count", len(errs))
	}

	return results, nil
}

// GetLatestBlockFromChain gets the latest block from a specific blockchain
func (m *MultiChainDataPuller) GetLatestBlockFromChain(
	ctx context.Context,
	chainID string,
) (uint64, error) {
	m.mu.RLock()
	puller, exists := m.pullers[chainID]
	m.mu.RUnlock()

	if !exists {
		return 0, core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeNotFound,
			fmt.Sprintf("no puller registered for chain %s", chainID),
			nil,
		)
	}

	return puller.GetLatestBlock(ctx)
}

// GetLatestBlocksFromAllChains gets the latest block from all blockchains in parallel
func (m *MultiChainDataPuller) GetLatestBlocksFromAllChains(
	ctx context.Context,
) (map[string]uint64, error) {
	m.mu.RLock()
	chains := make([]string, 0, len(m.pullers))
	for chainID := range m.pullers {
		chains = append(chains, chainID)
	}
	m.mu.RUnlock()

	results := make(map[string]uint64)
	resultsMu := sync.Mutex{}

	// Get latest blocks from all chains in parallel
	var wg sync.WaitGroup
	for _, chainID := range chains {
		wg.Add(1)
		go func(cid string) {
			defer wg.Done()

			block, err := m.GetLatestBlockFromChain(ctx, cid)
			if err != nil {
				if m.logger != nil {
					m.logger.Error("failed to get latest block", "chain_id", cid, "error", err.Error())
				}
				return
			}

			resultsMu.Lock()
			results[cid] = block
			resultsMu.Unlock()
		}(chainID)
	}

	wg.Wait()
	return results, nil
}

// GetHighestLatestBlock returns the highest latest block seen across all
// registered chains. When no chains are registered or no latest blocks can be
// retrieved, it returns 0 with a nil error.
func (m *MultiChainDataPuller) GetHighestLatestBlock(ctx context.Context) (uint64, error) {
	blocks, err := m.GetLatestBlocksFromAllChains(ctx)
	if err != nil {
		return 0, err
	}

	var highest uint64
	for _, block := range blocks {
		if block > highest {
			highest = block
		}
	}
	return highest, nil
}

// GetHighestProcessedBlock returns the highest processed block exposed by any
// registered puller. Pullers that do not surface last processed block state are
// ignored.
func (m *MultiChainDataPuller) GetHighestProcessedBlock() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var highest uint64
	for _, puller := range m.pullers {
		provider, ok := puller.(dataPullerLastProcessedBlockProvider)
		if !ok {
			continue
		}
		if block := provider.GetLastBlockNumber(); block > highest {
			highest = block
		}
	}
	return highest
}

// GetProcessedBlocksFromAllChains returns processed block heights exposed by
// registered pullers that surface last processed block state.
func (m *MultiChainDataPuller) GetProcessedBlocksFromAllChains() map[string]uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]uint64)
	for chainID, puller := range m.pullers {
		provider, ok := puller.(dataPullerLastProcessedBlockProvider)
		if !ok {
			continue
		}
		if block := provider.GetLastBlockNumber(); block > 0 {
			results[chainID] = block
		}
	}
	return results
}

// GetStats returns statistics for all pullers
func (m *MultiChainDataPuller) GetStats() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for chainID, puller := range m.pullers {
		stats[chainID] = puller.GetStats()
	}

	return stats
}

// GetRegisteredChains returns list of registered chain IDs
func (m *MultiChainDataPuller) GetRegisteredChains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]string, 0, len(m.pullers))
	for chainID := range m.pullers {
		chains = append(chains, chainID)
	}
	return chains
}

// UnregisterPuller unregisters a data puller for a specific blockchain
func (m *MultiChainDataPuller) UnregisterPuller(chainID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pullers[chainID]; !exists {
		return core.NewSystemError(
			core.ErrorTypePermanent,
			core.ErrorCodeNotFound,
			fmt.Sprintf("no puller registered for chain %s", chainID),
			nil,
		)
	}

	delete(m.pullers, chainID)
	if m.logger != nil {
		m.logger.Info("puller unregistered", "chain_id", chainID)
	}
	return nil
}
