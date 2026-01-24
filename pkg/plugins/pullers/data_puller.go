package pullers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"chainpulse/pkg/core"
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
	isRunning         bool
	lastBlockNumber   uint64
	maxRetries        int
	retryBackoff      time.Duration
	connectionTimeout time.Duration
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

	p.isRunning = true

	if p.logger != nil {
		p.logger.Info("data puller plugin started", "name", p.name)
	}

	if p.metricsCollector != nil {
		p.metricsCollector.RecordCounter("plugin_starts", 1, map[string]string{"plugin": p.name})
	}

	return nil
}

// Stop stops the plugin
func (p *BaseDataPullerPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("plugin not running")
	}

	p.isRunning = false

	if p.logger != nil {
		p.logger.Info("data puller plugin stopped", "name", p.name)
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

// SetLastBlockNumber sets the last processed block number
func (p *BaseDataPullerPlugin) SetLastBlockNumber(blockNumber uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastBlockNumber = blockNumber
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
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if attempt < p.maxRetries-1 {
			select {
			case <-time.After(backoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// GetConfig returns the plugin configuration
func (p *BaseDataPullerPlugin) GetConfig() core.Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
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
	return nil
}

// GenerateEventHash generates a deterministic hash for an event
func (p *BaseDataPullerPlugin) GenerateEventHash(event core.BlockchainEvent) string {
	// Simple hash generation - in production, use SHA256
	return fmt.Sprintf("%s:%d:%d:%s:%s",
		event.TransactionHash,
		event.BlockNumber,
		event.LogIndex,
		event.ContractAddress,
		event.EventName,
	)
}
