package core

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Ensure atomic types are properly imported
var _ atomic.Int64

// BaseMQPlugin provides base implementation for message queue plugins
type BaseMQPlugin struct {
	name                string
	version             string
	config              Config
	logger              Logger
	metricsCollector    MetricsCollector
	eventBus            EventBus
	isInitialized       bool
	isRunning           bool
	mu                  sync.RWMutex
	lastBlockNumber     uint64
	messageCount        atomic.Int64
	errorCount          atomic.Int64
	lastError           error
	lastErrorTime       time.Time
	deadLetterQueueSize atomic.Int64
	processingTime      int64
	batchSize           int
	maxRetries          int
	retryDelay          time.Duration
	batchTimeout        time.Duration
	batchBuffer         []MessageQueueMessage
	batchBufferMutex    sync.RWMutex
	batchProcessedCount atomic.Int64
	inFlightOperations  atomic.Int64
	inFlightWaitGroup   sync.WaitGroup
	deadLetterQueue     []MessageQueueMessage
	dlqMutex            sync.RWMutex
}

// MessageQueueMessage represents a message in the queue
type MessageQueueMessage struct {
	ID               string
	Topic            string
	Payload          []byte
	Timestamp        time.Time
	Offset           int64
	PartitionKey     string
	RetryCount       int
	DeadLetterReason string
	Headers          map[string]string
}

// MessageQueueStats represents statistics for a message queue
type MessageQueueStats struct {
	MessageCount        int64
	ErrorCount          int64
	DeadLetterQueueSize int64
	AverageProcessTime  int64
	LastError           error
	LastErrorTime       time.Time
	IsRunning           bool
}

// MetricsSnapshot represents a snapshot of all metrics
type MetricsSnapshot struct {
	MessageCount        int64
	ErrorCount          int64
	DeadLetterQueueSize int64
	BatchesProcessed    int64
	AverageProcessTime  int64
	LastError           error
	LastErrorTime       time.Time
	IsRunning           bool
	Timestamp           time.Time
}

// MQConfiguration represents configuration for the MQ plugin
type MQConfiguration struct {
	BatchSize  int
	MaxRetries int
	RetryDelay time.Duration
}

// NewBaseMQPlugin creates a new base message queue plugin
func NewBaseMQPlugin(
	name, version string,
	config Config,
	logger Logger,
	metricsCollector MetricsCollector,
	eventBus EventBus,
) *BaseMQPlugin {
	plugin := &BaseMQPlugin{
		name:             name,
		version:          version,
		config:           config,
		logger:           logger,
		metricsCollector: metricsCollector,
		eventBus:         eventBus,
		isInitialized:    false,
		isRunning:        false,
		lastBlockNumber:  0,
		processingTime:   0,
		batchSize:        100,
		maxRetries:       3,
		retryDelay:       1 * time.Second,
		batchTimeout:     5 * time.Second,
		batchBuffer:      make([]MessageQueueMessage, 0, 100),
		deadLetterQueue:  make([]MessageQueueMessage, 0),
	}
	// Initialize atomic values
	plugin.messageCount.Store(0)
	plugin.errorCount.Store(0)
	plugin.deadLetterQueueSize.Store(0)
	plugin.batchProcessedCount.Store(0)
	plugin.inFlightOperations.Store(0)
	return plugin
}

// Name returns the plugin name
func (p *BaseMQPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BaseMQPlugin) Version() string {
	return p.version
}

// IsInitialized returns whether the plugin is initialized
func (p *BaseMQPlugin) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isInitialized
}

// IsRunning returns whether the plugin is running
func (p *BaseMQPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// ValidateMQConfiguration validates MQ plugin configuration
// Property 9: Configuration Validation
// For any configuration provided to the MQ plugin, invalid configurations SHALL be rejected with clear error messages
func (p *BaseMQPlugin) ValidateMQConfiguration(config MQConfiguration) error {
	if config.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0, got %d", config.BatchSize)
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative, got %d", config.MaxRetries)
	}

	if config.RetryDelay < 0 {
		return fmt.Errorf("retry delay must be non-negative, got %v", config.RetryDelay)
	}

	return nil
}

// ApplyMQConfiguration applies validated configuration to the plugin
func (p *BaseMQPlugin) ApplyMQConfiguration(config MQConfiguration) error {
	// Validate first
	if err := p.ValidateMQConfiguration(config); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.batchSize = config.BatchSize
	p.maxRetries = config.MaxRetries
	p.retryDelay = config.RetryDelay

	p.logger.Info(
		"MQ configuration applied",
		"batch_size", config.BatchSize,
		"max_retries", config.MaxRetries,
		"retry_delay", config.RetryDelay,
	)

	return nil
}

// GetMQConfiguration returns the current MQ configuration
func (p *BaseMQPlugin) GetMQConfiguration() MQConfiguration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return MQConfiguration{
		BatchSize:  p.batchSize,
		MaxRetries: p.maxRetries,
		RetryDelay: p.retryDelay,
	}
}
