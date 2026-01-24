package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// BlockchainType represents a blockchain network type
type BlockchainType string

const (
	EVM    BlockchainType = "evm"
	Cosmos BlockchainType = "cosmos"
	Solana BlockchainType = "solana"
)

// BlockchainEvent represents a blockchain event
type BlockchainEvent struct {
	ID              string
	EventHash       string
	BlockNumber     uint64
	TransactionHash string
	LogIndex        uint
	ContractAddress string
	EventName       string
	EventData       map[string]interface{}
	ChainID         string
	Timestamp       time.Time
	ProcessedAt     time.Time
	Status          string
}

// DataPullerConfig represents data puller configuration
type DataPullerConfig struct {
	ChainType      BlockchainType
	ChainID        string
	BlockchainNode string
	StartBlock     uint64
	BatchSize      uint64
	PollInterval   time.Duration
}

// DataPuller pulls events from a blockchain
type DataPuller struct {
	config           DataPullerConfig
	currentBlock     uint64
	lastProcessedAt  time.Time
	running          bool
	mutex            sync.RWMutex
	metrics          *DataPullerMetrics
	eventChan        chan *BlockchainEvent
	errorChan        chan error
}

// NewDataPuller creates a new data puller
func NewDataPuller(config DataPullerConfig) *DataPuller {
	return &DataPuller{
		config:          config,
		currentBlock:    config.StartBlock,
		metrics:         NewDataPullerMetrics(),
		eventChan:       make(chan *BlockchainEvent, 100),
		errorChan:       make(chan error, 10),
	}
}

// Start starts the data puller
func (dp *DataPuller) Start(ctx context.Context) error {
	dp.mutex.Lock()
	if dp.running {
		dp.mutex.Unlock()
		return fmt.Errorf("data puller already running")
	}
	dp.running = true
	dp.mutex.Unlock()

	go dp.pullLoop(ctx)
	return nil
}

// Stop stops the data puller
func (dp *DataPuller) Stop() error {
	dp.mutex.Lock()
	defer dp.mutex.Unlock()

	if !dp.running {
		return fmt.Errorf("data puller not running")
	}

	dp.running = false
	close(dp.eventChan)
	close(dp.errorChan)
	return nil
}

// pullLoop continuously pulls events
func (dp *DataPuller) pullLoop(ctx context.Context) {
	ticker := time.NewTicker(dp.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dp.pullEvents(ctx)
		}
	}
}

// pullEvents pulls events from the blockchain
func (dp *DataPuller) pullEvents(ctx context.Context) {
	dp.mutex.Lock()
	if !dp.running {
		dp.mutex.Unlock()
		return
	}
	currentBlock := dp.currentBlock
	dp.mutex.Unlock()

	// Simulate pulling events
	events := dp.simulatePullEvents(currentBlock)

	for _, event := range events {
		select {
		case dp.eventChan <- event:
			dp.metrics.RecordEventPulled()
		case <-ctx.Done():
			return
		}
	}

	// Update current block
	dp.mutex.Lock()
	if len(events) > 0 {
		dp.currentBlock = events[len(events)-1].BlockNumber + 1
		dp.lastProcessedAt = time.Now()
	}
	dp.mutex.Unlock()
}

// simulatePullEvents simulates pulling events from blockchain
func (dp *DataPuller) simulatePullEvents(fromBlock uint64) []*BlockchainEvent {
	events := make([]*BlockchainEvent, 0)

	// Simulate pulling up to batch size events
	for i := uint64(0); i < dp.config.BatchSize; i++ {
		event := &BlockchainEvent{
			ID:              fmt.Sprintf("%s-%d-%d", dp.config.ChainID, fromBlock+i, i),
			EventHash:       fmt.Sprintf("hash-%s-%d-%d", dp.config.ChainID, fromBlock+i, i),
			BlockNumber:     fromBlock + i,
			TransactionHash: fmt.Sprintf("tx-%s-%d-%d", dp.config.ChainID, fromBlock+i, i),
			LogIndex:        uint(i),
			ContractAddress: "0x1234567890123456789012345678901234567890",
			EventName:       "Transfer",
			EventData: map[string]interface{}{
				"from":   "0x1111111111111111111111111111111111111111",
				"to":     "0x2222222222222222222222222222222222222222",
				"amount": "1000000000000000000",
			},
			ChainID:     dp.config.ChainID,
			Timestamp:   time.Now(),
			ProcessedAt: time.Now(),
			Status:      "pending",
		}
		events = append(events, event)
	}

	return events
}

// GetLatestBlock returns the latest processed block
func (dp *DataPuller) GetLatestBlock(ctx context.Context) (uint64, error) {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	if !dp.running {
		return 0, fmt.Errorf("data puller not running")
	}

	return dp.currentBlock, nil
}

// GetProcessedHeight returns the last processed block height
func (dp *DataPuller) GetProcessedHeight(ctx context.Context) (uint64, error) {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	if !dp.running {
		return 0, fmt.Errorf("data puller not running")
	}

	return dp.currentBlock - 1, nil
}

// GetEvents returns the event channel
func (dp *DataPuller) GetEvents() <-chan *BlockchainEvent {
	return dp.eventChan
}

// GetErrors returns the error channel
func (dp *DataPuller) GetErrors() <-chan error {
	return dp.errorChan
}

// Health returns the health status
func (dp *DataPuller) Health() core.HealthStatus {
	dp.mutex.RLock()
	defer dp.mutex.RUnlock()

	if dp.running {
		return core.HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now(),
		}
	}

	return core.HealthStatus{
		Status:    "unhealthy",
		Timestamp: time.Now(),
	}
}

// DataPullerMetrics tracks data puller metrics
type DataPullerMetrics struct {
	eventsPulled  int64
	eventsDropped int64
	errors        int64
	mutex         sync.RWMutex
}

// NewDataPullerMetrics creates new data puller metrics
func NewDataPullerMetrics() *DataPullerMetrics {
	return &DataPullerMetrics{}
}

// RecordEventPulled records an event pulled
func (dpm *DataPullerMetrics) RecordEventPulled() {
	dpm.mutex.Lock()
	defer dpm.mutex.Unlock()
	dpm.eventsPulled++
}

// RecordEventDropped records an event dropped
func (dpm *DataPullerMetrics) RecordEventDropped() {
	dpm.mutex.Lock()
	defer dpm.mutex.Unlock()
	dpm.eventsDropped++
}

// RecordError records an error
func (dpm *DataPullerMetrics) RecordError() {
	dpm.mutex.Lock()
	defer dpm.mutex.Unlock()
	dpm.errors++
}

// GetMetrics returns current metrics
func (dpm *DataPullerMetrics) GetMetrics() map[string]interface{} {
	dpm.mutex.RLock()
	defer dpm.mutex.RUnlock()

	return map[string]interface{}{
		"events_pulled":  dpm.eventsPulled,
		"events_dropped": dpm.eventsDropped,
		"errors":         dpm.errors,
	}
}

// MultiChainDataPuller manages data pullers for multiple chains
type MultiChainDataPuller struct {
	pullers map[string]*DataPuller
	mutex   sync.RWMutex
}

// NewMultiChainDataPuller creates a new multi-chain data puller
func NewMultiChainDataPuller() *MultiChainDataPuller {
	return &MultiChainDataPuller{
		pullers: make(map[string]*DataPuller),
	}
}

// AddPuller adds a data puller for a chain
func (mcdp *MultiChainDataPuller) AddPuller(chainID string, puller *DataPuller) error {
	mcdp.mutex.Lock()
	defer mcdp.mutex.Unlock()

	if _, exists := mcdp.pullers[chainID]; exists {
		return fmt.Errorf("puller already exists for chain: %s", chainID)
	}

	mcdp.pullers[chainID] = puller
	return nil
}

// RemovePuller removes a data puller for a chain
func (mcdp *MultiChainDataPuller) RemovePuller(chainID string) error {
	mcdp.mutex.Lock()
	defer mcdp.mutex.Unlock()

	if _, exists := mcdp.pullers[chainID]; !exists {
		return fmt.Errorf("puller not found for chain: %s", chainID)
	}

	delete(mcdp.pullers, chainID)
	return nil
}

// GetPuller gets a data puller for a chain
func (mcdp *MultiChainDataPuller) GetPuller(chainID string) (*DataPuller, error) {
	mcdp.mutex.RLock()
	defer mcdp.mutex.RUnlock()

	puller, exists := mcdp.pullers[chainID]
	if !exists {
		return nil, fmt.Errorf("puller not found for chain: %s", chainID)
	}

	return puller, nil
}

// StartAll starts all data pullers
func (mcdp *MultiChainDataPuller) StartAll(ctx context.Context) error {
	mcdp.mutex.RLock()
	pullers := make(map[string]*DataPuller)
	for chainID, puller := range mcdp.pullers {
		pullers[chainID] = puller
	}
	mcdp.mutex.RUnlock()

	for chainID, puller := range pullers {
		if err := puller.Start(ctx); err != nil {
			return fmt.Errorf("failed to start puller for chain %s: %w", chainID, err)
		}
	}

	return nil
}

// StopAll stops all data pullers
func (mcdp *MultiChainDataPuller) StopAll() error {
	mcdp.mutex.RLock()
	pullers := make(map[string]*DataPuller)
	for chainID, puller := range mcdp.pullers {
		pullers[chainID] = puller
	}
	mcdp.mutex.RUnlock()

	for chainID, puller := range pullers {
		if err := puller.Stop(); err != nil {
			return fmt.Errorf("failed to stop puller for chain %s: %w", chainID, err)
		}
	}

	return nil
}

// GetAllMetrics gets metrics from all pullers
func (mcdp *MultiChainDataPuller) GetAllMetrics() map[string]map[string]interface{} {
	mcdp.mutex.RLock()
	defer mcdp.mutex.RUnlock()

	metrics := make(map[string]map[string]interface{})
	for chainID, puller := range mcdp.pullers {
		metrics[chainID] = puller.metrics.GetMetrics()
	}

	return metrics
}

// HealthAll returns health status of all pullers
func (mcdp *MultiChainDataPuller) HealthAll() map[string]core.HealthStatus {
	mcdp.mutex.RLock()
	defer mcdp.mutex.RUnlock()

	health := make(map[string]core.HealthStatus)
	for chainID, puller := range mcdp.pullers {
		health[chainID] = puller.Health()
	}

	return health
}
