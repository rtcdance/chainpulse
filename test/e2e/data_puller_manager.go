package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DataPullerManager manages data puller integration testing
type DataPullerManager interface {
	// StartPuller starts the data puller service
	StartPuller(ctx context.Context, config DataPullerConfig) error

	// StopPuller stops the data puller service
	StopPuller(ctx context.Context) error

	// CollectEvents collects events from the blockchain
	CollectEvents(ctx context.Context, filter EventFilter) ([]*CollectedEvent, error)

	// ValidateEventCollection validates that all events were collected
	ValidateEventCollection(ctx context.Context, emitted []*EventEmission, collected []*CollectedEvent) error

	// SimulateRetry simulates retry logic with failures
	SimulateRetry(ctx context.Context, failureCount int) error

	// SimulateReorg simulates blockchain reorganization
	SimulateReorg(ctx context.Context, reorgDepth uint64) error

	// GetPullerMetrics returns data puller metrics
	GetPullerMetrics(ctx context.Context) DataPullerMetrics

	// WaitForEventCollection waits for events to be collected
	WaitForEventCollection(ctx context.Context, expectedCount int, timeout time.Duration) error

	// AddCollectedEvent adds a collected event to the manager
	AddCollectedEvent(ctx context.Context, event *CollectedEvent) error
}

// DataPullerConfig configures the data puller
type DataPullerConfig struct {
	BlockchainRPC     string
	ContractAddresses []string
	StartBlock        uint64
	MaxRetries        int
	RetryBackoff      time.Duration
	PollInterval      time.Duration
	Timeout           time.Duration
	ChainID           string
}

// CollectedEvent represents an event collected by the data puller
type CollectedEvent struct {
	ID              string
	ContractAddress string
	EventName       string
	TxHash          string
	BlockNumber     uint64
	LogIndex        uint32
	RawData         []byte
	CollectedAt     time.Time
	ChainID         string
	RetryCount      int
}

// DataPullerMetrics contains data puller performance metrics
type DataPullerMetrics struct {
	EventsCollected    int64
	EventsFailed       int64
	RetryCount         int64
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
	Throughput         float64
	ErrorRate          float64
	LastBlockProcessed uint64
}

// DefaultDataPullerManager implements DataPullerManager
type DefaultDataPullerManager struct {
	config              *DataPullerConfig
	mu                  sync.RWMutex
	isRunning           bool
	collectedEvents     []*CollectedEvent
	eventIndex          map[string]*CollectedEvent
	metrics             *DataPullerMetrics
	startTime           time.Time
	lastBlockProcessed  uint64
	retrySimulations    map[string]*RetrySimulation
	reorgSimulations    map[string]*ReorgSimulation
	eventCollectionChan chan *CollectedEvent
}

// RetrySimulation tracks retry simulation state
type RetrySimulation struct {
	FailureCount int
	RetryCount   int
	StartTime    time.Time
}

// ReorgSimulation tracks reorg simulation state
type ReorgSimulation struct {
	ReorgDepth  uint64
	AffectedTxs []string
	StartTime   time.Time
	IsActive    bool
}

// NewDataPullerManager creates a new data puller manager
func NewDataPullerManager() DataPullerManager {
	return &DefaultDataPullerManager{
		collectedEvents:     make([]*CollectedEvent, 0),
		eventIndex:          make(map[string]*CollectedEvent),
		retrySimulations:    make(map[string]*RetrySimulation),
		reorgSimulations:    make(map[string]*ReorgSimulation),
		eventCollectionChan: make(chan *CollectedEvent, 1000),
		metrics: &DataPullerMetrics{
			EventsCollected: 0,
			EventsFailed:    0,
			RetryCount:      0,
			AverageLatency:  0,
			MaxLatency:      0,
			MinLatency:      time.Hour, // Start with large value
			Throughput:      0,
			ErrorRate:       0,
		},
	}
}

// StartPuller starts the data puller service
func (dpm *DefaultDataPullerManager) StartPuller(ctx context.Context, config DataPullerConfig) error {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	if dpm.isRunning {
		return fmt.Errorf("data puller already running")
	}

	if config.BlockchainRPC == "" {
		return fmt.Errorf("blockchain RPC URL required")
	}

	if len(config.ContractAddresses) == 0 {
		return fmt.Errorf("at least one contract address required")
	}

	dpm.config = &config
	dpm.isRunning = true
	dpm.startTime = time.Now()
	dpm.lastBlockProcessed = config.StartBlock

	// Initialize metrics
	dpm.metrics = &DataPullerMetrics{
		EventsCollected:    0,
		EventsFailed:       0,
		RetryCount:         0,
		LastBlockProcessed: config.StartBlock,
	}

	return nil
}

// StopPuller stops the data puller service
func (dpm *DefaultDataPullerManager) StopPuller(ctx context.Context) error {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	if !dpm.isRunning {
		return fmt.Errorf("data puller not running")
	}

	dpm.isRunning = false
	close(dpm.eventCollectionChan)

	return nil
}

// AddCollectedEvent adds a collected event to the manager
func (dpm *DefaultDataPullerManager) AddCollectedEvent(ctx context.Context, event *CollectedEvent) error {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	if !dpm.isRunning {
		return fmt.Errorf("data puller not running")
	}

	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	dpm.collectedEvents = append(dpm.collectedEvents, event)
	dpm.eventIndex[event.ID] = event
	dpm.metrics.EventsCollected++

	if event.BlockNumber > dpm.metrics.LastBlockProcessed {
		dpm.metrics.LastBlockProcessed = event.BlockNumber
	}

	return nil
}

// CollectEvents collects events from the blockchain
func (dpm *DefaultDataPullerManager) CollectEvents(ctx context.Context, filter EventFilter) ([]*CollectedEvent, error) {
	dpm.mu.RLock()
	defer dpm.mu.RUnlock()

	if !dpm.isRunning {
		return nil, fmt.Errorf("data puller not running")
	}

	var result []*CollectedEvent

	for _, event := range dpm.collectedEvents {
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

		if filter.BlockRange != nil {
			if event.BlockNumber < filter.BlockRange.Start || event.BlockNumber > filter.BlockRange.End {
				continue
			}
		}

		result = append(result, event)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}

	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, nil
}

// ValidateEventCollection validates that all events were collected
func (dpm *DefaultDataPullerManager) ValidateEventCollection(ctx context.Context, emitted []*EventEmission, collected []*CollectedEvent) error {
	dpm.mu.RLock()
	defer dpm.mu.RUnlock()

	if len(emitted) != len(collected) {
		return fmt.Errorf("event count mismatch: emitted=%d, collected=%d", len(emitted), len(collected))
	}

	// Create map of collected events for quick lookup
	collectedMap := make(map[string]*CollectedEvent)
	for _, event := range collected {
		key := fmt.Sprintf("%s_%d_%d", event.TxHash, event.BlockNumber, event.LogIndex)
		collectedMap[key] = event
	}

	// Verify all emitted events were collected
	for _, emitted := range emitted {
		key := fmt.Sprintf("%s_%d_%d", emitted.TxHash, emitted.BlockNumber, emitted.LogIndex)
		if _, exists := collectedMap[key]; !exists {
			return fmt.Errorf("event not collected: %s", key)
		}
	}

	return nil
}

// SimulateRetry simulates retry logic with failures
func (dpm *DefaultDataPullerManager) SimulateRetry(ctx context.Context, failureCount int) error {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	if !dpm.isRunning {
		return fmt.Errorf("data puller not running")
	}

	simulationID := fmt.Sprintf("retry_%d", time.Now().UnixNano())

	simulation := &RetrySimulation{
		FailureCount: failureCount,
		RetryCount:   0,
		StartTime:    time.Now(),
	}

	dpm.retrySimulations[simulationID] = simulation

	// Simulate retries with exponential backoff
	backoff := dpm.config.RetryBackoff
	for attempt := 0; attempt < failureCount; attempt++ {
		select {
		case <-time.After(backoff):
			simulation.RetryCount++
			dpm.metrics.RetryCount++
			backoff *= 2
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// SimulateReorg simulates blockchain reorganization
func (dpm *DefaultDataPullerManager) SimulateReorg(ctx context.Context, reorgDepth uint64) error {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	if !dpm.isRunning {
		return fmt.Errorf("data puller not running")
	}

	simulationID := fmt.Sprintf("reorg_%d", time.Now().UnixNano())

	simulation := &ReorgSimulation{
		ReorgDepth:  reorgDepth,
		AffectedTxs: make([]string, 0),
		StartTime:   time.Now(),
		IsActive:    true,
	}

	// Mark events as affected by reorg
	if dpm.lastBlockProcessed >= reorgDepth {
		affectedBlockStart := dpm.lastBlockProcessed - reorgDepth
		for _, event := range dpm.collectedEvents {
			if event.BlockNumber > affectedBlockStart {
				simulation.AffectedTxs = append(simulation.AffectedTxs, event.TxHash)
			}
		}
	}

	dpm.reorgSimulations[simulationID] = simulation

	return nil
}

// GetPullerMetrics returns data puller metrics
func (dpm *DefaultDataPullerManager) GetPullerMetrics(ctx context.Context) DataPullerMetrics {
	dpm.mu.RLock()
	defer dpm.mu.RUnlock()

	metrics := *dpm.metrics

	// Calculate throughput
	if dpm.isRunning && !dpm.startTime.IsZero() {
		elapsed := time.Since(dpm.startTime).Seconds()
		if elapsed > 0 {
			metrics.Throughput = float64(metrics.EventsCollected) / elapsed
		}
	}

	// Calculate error rate
	total := metrics.EventsCollected + metrics.EventsFailed
	if total > 0 {
		metrics.ErrorRate = float64(metrics.EventsFailed) / float64(total)
	}

	return metrics
}

// WaitForEventCollection waits for events to be collected
func (dpm *DefaultDataPullerManager) WaitForEventCollection(ctx context.Context, expectedCount int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		dpm.mu.RLock()
		currentCount := len(dpm.collectedEvents)
		dpm.mu.RUnlock()

		if currentCount >= expectedCount {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for event collection: expected=%d, got=%d", expectedCount, currentCount)
		}

		select {
		case <-time.After(100 * time.Millisecond):
			// Continue polling
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
