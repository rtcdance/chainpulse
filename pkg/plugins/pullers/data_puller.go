package pullers

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
)

// BaseDataPullerPlugin provides common functionality for data puller plugins
type BaseDataPullerPlugin struct {
	mu                sync.RWMutex
	name              string
	version           string
	config            core.Config
	logger            core.Logger
	metricsCollector  core.MetricsCollector
	eventBus          core.EventBus
	checkpointStore   core.CheckpointStore
	isRunning         bool
	lastBlockNumber   uint64
	maxRetries        int
	retryBackoff      time.Duration
	connectionTimeout time.Duration
	inFlight          sync.WaitGroup // tracks in-flight operations
	shutdownTimeout   time.Duration  // max wait for in-flight ops on Stop()
}

// NewBaseDataPullerPlugin creates a new base data puller plugin
func NewBaseDataPullerPlugin(
	name string,
	version string,
	config core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
) *BaseDataPullerPlugin {
	return &BaseDataPullerPlugin{
		name:              name,
		version:           version,
		config:            config,
		logger:            logger,
		metricsCollector:  metricsCollector,
		eventBus:          eventBus,
		isRunning:         false,
		lastBlockNumber:   config.StartBlock,
		maxRetries:        config.MaxRetries,
		retryBackoff:      time.Duration(config.RetryBackoff) * time.Millisecond,
		connectionTimeout: 30 * time.Second,
	}
}

// Name returns the plugin name
func (p *BaseDataPullerPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BaseDataPullerPlugin) Version() string {
	return p.version
}

// Initialize initializes the plugin
func (p *BaseDataPullerPlugin) Initialize(config core.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config
	p.lastBlockNumber = config.StartBlock
	p.maxRetries = config.MaxRetries
	p.retryBackoff = time.Duration(config.RetryBackoff) * time.Millisecond

	if p.logger != nil {
		p.logger.Info("data puller plugin initialized", "name", p.name, "start_block", config.StartBlock)
	}

	return nil
}

// Start starts the plugin
func (p *BaseDataPullerPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	// Validate chain ID before starting to prevent cross-chain data contamination
	if p.config.ChainID == "" && p.config.ServiceName == "" {
		return fmt.Errorf("puller chainID is not configured: either ChainID or ServiceName must be set")
	}

	p.isRunning = true

	if p.logger != nil {
		p.logger.Info("data puller plugin started", "name", p.name)
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("plugin_starts", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// Stop stops the plugin and waits for in-flight operations to complete
func (p *BaseDataPullerPlugin) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.isRunning = false
	p.mu.Unlock()

	// Wait for in-flight operations with timeout
	done := make(chan struct{})
	go func() {
		p.inFlight.Wait()
		close(done)
	}()

	timeout := p.shutdownTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	select {
	case <-done:
		if p.logger != nil {
			p.logger.Info("data puller plugin stopped, all in-flight operations completed", "name", p.name)
		}
	case <-time.After(timeout):
		if p.logger != nil {
			p.logger.Warn("data puller plugin stop timed out waiting for in-flight operations", "name", p.name, "timeout", timeout)
		}
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("plugin_stops", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// Health checks the health of the plugin
func (p *BaseDataPullerPlugin) Health() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isRunning {
		return fmt.Errorf("plugin not running")
	}

	return nil
}

// IsRunning returns whether the plugin is running
func (p *BaseDataPullerPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// GetLastBlockNumber returns the last processed block number
func (p *BaseDataPullerPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastBlockNumber
}

// SetLastBlockNumber sets the last processed block number and persists it if checkpointStore is configured
func (p *BaseDataPullerPlugin) SetLastBlockNumber(blockNumber uint64) {
	p.mu.Lock()
	p.lastBlockNumber = blockNumber
	store := p.checkpointStore
	chainID := p.config.ChainID
	if chainID == "" {
		chainID = p.config.ServiceName
	}
	if chainID == "" {
		if p.logger != nil {
			p.logger.Warn("chainID is empty; checkpoint cannot be saved reliably", "block", blockNumber)
		}
	}
	p.mu.Unlock()

	// Persist checkpoint asynchronously
	if store != nil {
		p.inFlight.Add(1)
		go func() {
			defer p.inFlight.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := store.SaveLastIndexedBlock(ctx, chainID, blockNumber, ""); err != nil {
				if p.logger != nil {
					p.logger.Warn("failed to persist checkpoint", "error", err.Error(), "block", blockNumber)
				}
			}
		}()
	}
}

// SetCheckpointStore sets the checkpoint store for persisting indexing progress
func (p *BaseDataPullerPlugin) SetCheckpointStore(store core.CheckpointStore) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.checkpointStore = store
}

// LoadCheckpoint loads the last indexed block from the checkpoint store.
// Returns the stored block number, or 0 if no checkpoint exists or loading fails.
func (p *BaseDataPullerPlugin) LoadCheckpoint(ctx context.Context) uint64 {
	p.mu.RLock()
	store := p.checkpointStore
	chainID := p.config.ChainID
	if chainID == "" {
		chainID = p.config.ServiceName
	}
	if chainID == "" {
		if p.logger != nil {
			p.logger.Warn("chainID is empty; checkpoint cannot be loaded reliably")
		}
	}
	p.mu.RUnlock()

	if store == nil {
		return 0
	}

	blockNum, _, err := store.GetLastIndexedBlock(ctx, chainID)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("failed to load checkpoint", "error", err.Error())
		}
		return 0
	}
	return blockNum
}

// PublishEvent publishes an event to the event bus
func (p *BaseDataPullerPlugin) PublishEvent(ctx context.Context, event core.BlockchainEvent) error {
	if p.eventBus == nil {
		return fmt.Errorf("event bus not available")
	}

	event.ProcessedAt = time.Now().UTC()
	event.Status = "published"

	if err := p.eventBus.Publish(ctx, "blockchain-events", event); err != nil {
		if p.logger != nil {
			p.logger.Error("failed to publish event", "error", err.Error(), "block", event.BlockNumber)
		}
		return err
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("events_published", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// PublishEvents publishes multiple events to the event bus
func (p *BaseDataPullerPlugin) PublishEvents(ctx context.Context, events []core.BlockchainEvent) error {
	for _, event := range events {
		if err := p.PublishEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// RecordMetric records a metric
func (p *BaseDataPullerPlugin) RecordMetric(name string, value interface{}, tags map[string]string) {
	if p.metricsCollector == nil {
		return
	}

	if tags == nil {
		tags = make(map[string]string)
	}
	tags["plugin"] = p.name

	switch v := value.(type) {
	case int64:
		p.metricsCollector.RecordCounter(name, v, tags)
	case float64:
		p.metricsCollector.RecordGauge(name, v, tags)
	}
}

// LogInfo logs an info message
func (p *BaseDataPullerPlugin) LogInfo(msg string, fields ...interface{}) {
	if p.logger != nil {
		p.logger.Info(msg, fields...)
	}
}

// LogError logs an error message
func (p *BaseDataPullerPlugin) LogError(msg string, fields ...interface{}) {
	if p.logger != nil {
		p.logger.Error(msg, fields...)
	}
}

// LogWarn logs a warning message
func (p *BaseDataPullerPlugin) LogWarn(msg string, fields ...interface{}) {
	if p.logger != nil {
		p.logger.Warn(msg, fields...)
	}
}

// RetryWithBackoff retries an operation with exponential backoff
func (p *BaseDataPullerPlugin) RetryWithBackoff(ctx context.Context, operation func() error) error {
	backoff := p.retryBackoff
	var lastErr error

	if p.maxRetries <= 0 {
		p.maxRetries = 3
		p.retryBackoff = 1 * time.Second
		backoff = p.retryBackoff
	}

	p.logger.Debug("retry config", "maxRetries", p.maxRetries, "retryBackoff", backoff)

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		lastErr = err

		p.logger.Warn("retry attempt failed", "attempt", attempt+1, "maxRetries", p.maxRetries, "error", err)

		if attempt < p.maxRetries-1 {
			// Add jitter: randomize between 50%-100% of backoff to avoid thundering herd
			jitteredBackoff := time.Duration(float64(backoff) * (0.5 + rand.Float64()*0.5))
			select {
			case <-time.After(jitteredBackoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	p.logger.Error("all retries exhausted", "maxRetries", p.maxRetries, "lastError", lastErr)
	if lastErr != nil {
		return fmt.Errorf("max retries exceeded: last error: %w", lastErr)
	}
	return fmt.Errorf("max retries exceeded: unknown error")
}

// GetConfig returns the plugin configuration
func (p *BaseDataPullerPlugin) GetConfig() core.Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// ChainID returns the per-chain identifier for this puller instance.
// It checks Config.ChainID first, falls back to ServiceName for backward
// compatibility. Returns empty string if neither is configured — the caller
// must validate before use.
func (p *BaseDataPullerPlugin) ChainID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config.ChainID != "" {
		return p.config.ChainID
	}
	if p.config.ServiceName != "" {
		return p.config.ServiceName
	}
	return ""
}

// ValidateConfig validates that the puller has a valid chain ID configured.
// Returns an error if ChainID would be empty, which would cause data contamination.
func (p *BaseDataPullerPlugin) ValidateConfig() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config.ChainID == "" && p.config.ServiceName == "" {
		return fmt.Errorf("puller chainID is not configured: either ChainID or ServiceName must be set to prevent cross-chain data contamination")
	}
	return nil
}

// Network returns the network label for this puller's chain.
// Falls back to "mainnet" if not explicitly configured.
func (p *BaseDataPullerPlugin) Network() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config.Network != "" {
		return p.config.Network
	}
	return "mainnet"
}

// SetConnectionTimeout sets the connection timeout
func (p *BaseDataPullerPlugin) SetConnectionTimeout(timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connectionTimeout = timeout
}

// GetConnectionTimeout returns the connection timeout
func (p *BaseDataPullerPlugin) GetConnectionTimeout() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connectionTimeout
}

// ValidateEvent validates a blockchain event
func (p *BaseDataPullerPlugin) ValidateEvent(event core.BlockchainEvent) error {
	if event.BlockNumber == 0 {
		return fmt.Errorf("invalid block number")
	}
	if event.TransactionHash == (common.Hash{}) {
		return fmt.Errorf("invalid transaction hash")
	}
	if event.ContractAddress == (common.Address{}) {
		return fmt.Errorf("invalid contract address")
	}
	if event.EventName == "" {
		return fmt.Errorf("invalid event name")
	}

	// Client-side contract address allowlist validation.
	// The server-side eth_getLogs address filter is the primary filter,
	// but a compromised RPC could return events from unmonitored contracts.
	if addresses := p.GetConfig().ContractAddresses; len(addresses) > 0 {
		eventAddr := event.ContractAddress.Hex()
		found := false
		for _, addr := range addresses {
			if strings.EqualFold(eventAddr, addr) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("contract address %s not in monitored list", eventAddr)
		}
	}

	return nil
}

// GenerateEventHash generates a deterministic hash for an event using the
// canonical ComputeEventHash function from pkg/core.
func (p *BaseDataPullerPlugin) GenerateEventHash(event core.BlockchainEvent) string {
	return core.ComputeEventHash(&event)
}
