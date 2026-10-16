package pullers

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"golang.org/x/sync/errgroup"
)

type dataPullerLastProcessedBlockProvider interface {
	GetLastBlockNumber() uint64
}

// FinalityTargetFromChain preserves protocol-specific explicit policies while
// keeping legacy targets for adapters without that optional capability.
func (m *MultiChainDataPuller) FinalityTargetFromChain(ctx context.Context, chain string, latest, fallback uint64) (uint64, error) {
	m.mu.RLock()
	puller, ok := m.pullers[chain]
	m.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no puller registered for finality chain %s", chain)
	}
	if resolver, ok := puller.(interface {
		ResolveFinalityTarget(context.Context, uint64, uint64) uint64
	}); ok {
		target := resolver.ResolveFinalityTarget(ctx, latest, fallback)
		if target > latest {
			return 0, fmt.Errorf("finality target exceeds observed head")
		}
		return target, nil
	}
	return fallback, nil
}

// BlockHeaderFromChain exposes the verified header capability without leaking
// a mutable registered puller. Unsupported protocols fail explicitly.
func (m *MultiChainDataPuller) BlockHeaderFromChain(ctx context.Context, chainID string, height uint64) (*types.Header, error) {
	m.mu.RLock()
	source := m.pullers[chainID]
	m.mu.RUnlock()
	reader, ok := source.(interface {
		BlockHeader(context.Context, uint64) (*types.Header, error)
	})
	if !ok {
		return nil, fmt.Errorf("chain %s does not support complete EVM block headers", chainID)
	}
	header, err := reader.BlockHeader(ctx, height)
	if err != nil {
		return nil, fmt.Errorf("read block header for chain %s: %w", chainID, err)
	}
	return header, nil
}

type dataPullerLastProcessedBlockSetter interface {
	SetLastBlockNumber(uint64)
}

// defaultMaxConcurrency bounds the per-chain fan-out: the errgroup would
// otherwise spawn one goroutine per registered chain, most of which share the
// same rate-limited provider.
const defaultMaxConcurrency = 8

// MultiChainDataPuller manages data pulling from multiple blockchains
type MultiChainDataPuller struct {
	pullers map[string]portcontract.DataPullerPlugin
	mu      sync.RWMutex
	logger  portcontract.Logger
	// maxConcurrency bounds the errgroup fan-out; 0 means defaultMaxConcurrency.
	maxConcurrency int
}

// NewMultiChainDataPuller creates a new multi-chain data puller
func NewMultiChainDataPuller(logger portcontract.Logger) *MultiChainDataPuller {
	return &MultiChainDataPuller{
		pullers:        make(map[string]portcontract.DataPullerPlugin),
		logger:         logger,
		maxConcurrency: defaultMaxConcurrency,
	}
}

// SetMaxConcurrency sets the fixed-size worker-pool bound for the parallel
// per-chain fan-out. Values <= 0 are ignored, so callers can pass a config
// value without pre-validating it.
func (m *MultiChainDataPuller) SetMaxConcurrency(n int) {
	if n <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxConcurrency = n
}

func (m *MultiChainDataPuller) maxConcurrencyLimit() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.maxConcurrency <= 0 {
		return defaultMaxConcurrency
	}
	return m.maxConcurrency
}

// RegisterPuller registers a data puller for a specific blockchain
func (m *MultiChainDataPuller) RegisterPuller(chainID string, puller portcontract.DataPullerPlugin) error {
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
) ([]blockchain.BlockchainEvent, error) {
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
// using errgroup for consistent concurrency handling across the codebase.
func (m *MultiChainDataPuller) PullEventsFromAllChains(
	ctx context.Context,
	fromBlock, toBlock uint64,
) (map[string][]blockchain.BlockchainEvent, error) {
	m.mu.RLock()
	chains := make([]string, 0, len(m.pullers))
	for chainID := range m.pullers {
		chains = append(chains, chainID)
	}
	m.mu.RUnlock()

	results := make(map[string][]blockchain.BlockchainEvent)
	resultsMu := sync.Mutex{}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(m.maxConcurrencyLimit())

	for _, chainID := range chains {
		cid := chainID
		g.Go(func() error {
			events, err := m.PullEventsFromChain(gCtx, cid, fromBlock, toBlock)
			if err != nil {
				return fmt.Errorf("chain %s: %w", cid, err)
			}

			resultsMu.Lock()
			results[cid] = events
			resultsMu.Unlock()
			return nil
		})
	}

	// Wait for all chains; returns first error encountered
	if err := g.Wait(); err != nil {
		if m.logger != nil {
			m.logger.Warn("some chains failed to pull events", "error", err.Error())
		}
		// Return partial results along with the first error
		return results, err
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
// using errgroup for consistent concurrency handling across the codebase.
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

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(m.maxConcurrencyLimit())

	for _, chainID := range chains {
		cid := chainID
		g.Go(func() error {
			block, err := m.GetLatestBlockFromChain(gCtx, cid)
			if err != nil {
				if m.logger != nil {
					m.logger.Error("failed to get latest block", "chain_id", cid, "error", err.Error())
				}
				return fmt.Errorf("chain %s: %w", cid, err)
			}

			resultsMu.Lock()
			results[cid] = block
			resultsMu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if m.logger != nil {
			m.logger.Warn("some chains failed to get latest block", "error", err.Error())
		}
		return results, err
	}

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
func (m *MultiChainDataPuller) GetStats() map[string]map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]map[string]any)
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
