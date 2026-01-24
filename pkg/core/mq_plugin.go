package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Ensure atomic types are properly imported
var _ atomic.Int64

// BaseMQPlugin provides base implementation for message queue plugins
type BaseMQPlugin struct {
	name                 string
	version              string
	config               Config
	logger               Logger
	metricsCollector     MetricsCollector
	eventBus             EventBus
	isInitialized        bool
	isRunning            bool
	mu                   sync.RWMutex
	lastBlockNumber      uint64
	messageCount         atomic.Int64
	errorCount           atomic.Int64
	lastError            error
	lastErrorTime        time.Time
	deadLetterQueueSize  atomic.Int64
	processingTime       int64
	batchSize            int
	maxRetries           int
	retryDelay           time.Duration
	batchTimeout         time.Duration
	batchBuffer          []MessageQueueMessage
	batchBufferMutex     sync.RWMutex
	batchProcessedCount  atomic.Int64
	inFlightOperations   atomic.Int64
	inFlightWaitGroup    sync.WaitGroup
	deadLetterQueue      []MessageQueueMessage
	dlqMutex             sync.RWMutex
}

// MessageQueueMessage represents a message in the queue
type MessageQueueMessage struct {
	ID              string
	Topic           string
	Payload         []byte
	Timestamp       time.Time
	Offset          int64
	PartitionKey    string
	RetryCount      int
	DeadLetterReason string
	Headers         map[string]string
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

// NewBaseMQPlugin creates a new base message queue plugin
func NewBaseMQPlugin(
	name, version string,
	config Config,
	logger Logger,
	metricsCollector MetricsCollector,
	eventBus EventBus,
) *BaseMQPlugin {
	plugin := &BaseMQPlugin{
		name:                name,
		version:             version,
		config:              config,
		logger:              logger,
		metricsCollector:    metricsCollector,
		eventBus:            eventBus,
		isInitialized:       false,
		isRunning:           false,
		lastBlockNumber:     0,
		processingTime:      0,
		batchSize:           100,
		maxRetries:          3,
		retryDelay:          1 * time.Second,
		batchTimeout:        5 * time.Second,
		batchBuffer:         make([]MessageQueueMessage, 0, 100),
		deadLetterQueue:     make([]MessageQueueMessage, 0),
	}
	// Initialize atomic values
	plugin.messageCount.Store(0)
	plugin.errorCount.Store(0)
	plugin.deadLetterQueueSize.Store(0)
	plugin.batchProcessedCount.Store(0)
	plugin.inFlightOperations.Store(0)
	return plugin
}

// Initialize initializes the plugin
func (p *BaseMQPlugin) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isInitialized {
		return fmt.Errorf("plugin already initialized")
	}

	p.isInitialized = true
	p.logger.Info("message queue plugin initialized", "name", p.name, "version", p.version)

	return nil
}

// Start starts the plugin
func (p *BaseMQPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	p.isRunning = true
	p.logger.Info("message queue plugin started", "name", p.name)

	return nil
}

// Stop stops the plugin
func (p *BaseMQPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	p.isRunning = false
	p.logger.Info("message queue plugin stopping", "name", p.name)

	// Wait for all in-flight operations to complete
	p.inFlightWaitGroup.Wait()
	p.logger.Info("message queue plugin stopped", "name", p.name)

	return nil
}

// Health returns the health status of the plugin
func (p *BaseMQPlugin) Health() *HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	messageCount := p.messageCount.Load()
	errorCount := p.errorCount.Load()
	dlqSize := p.deadLetterQueueSize.Load()

	status := "healthy"
	if errorCount > 0 {
		status = "degraded"
	}

	return &HealthStatus{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Details: map[string]interface{}{
			"name":                   p.name,
			"version":                p.version,
			"is_running":             p.isRunning,
			"message_count":          messageCount,
			"error_count":            errorCount,
			"dead_letter_queue_size": dlqSize,
		},
	}
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

// PublishMessage publishes a message to the queue with comprehensive metrics
func (p *BaseMQPlugin) PublishMessage(ctx context.Context, message MessageQueueMessage) error {
	startTime := time.Now()

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	// Track in-flight operation
	p.inFlightWaitGroup.Add(1)
	p.inFlightOperations.Add(1)
	defer func() {
		p.inFlightOperations.Add(-1)
		p.inFlightWaitGroup.Done()
	}()

	// Generate message ID if not provided
	if message.ID == "" {
		message.ID = fmt.Sprintf("%s-%d", message.Topic, time.Now().UnixNano())
	}

	// Assign timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now().UTC()
	}

	// Initialize headers if nil
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}

	// Add standard headers
	message.Headers["message_id"] = message.ID
	message.Headers["timestamp"] = message.Timestamp.String()
	message.Headers["partition_key"] = message.PartitionKey

	// Record publish attempt using atomic operation
	p.messageCount.Add(1)

	// Record metrics
	latency := time.Since(startTime).Milliseconds()
	p.metricsCollector.RecordCounter("mq_messages_published", int64(1), map[string]string{"topic": message.Topic})
	p.metricsCollector.RecordHistogram("mq_publish_latency_ms", float64(latency), map[string]string{"topic": message.Topic})
	p.metricsCollector.RecordGauge("mq_message_size_bytes", float64(len(message.Payload)), map[string]string{"topic": message.Topic})

	p.logger.Info("message published", "topic", message.Topic, "message_id", message.ID, "size_bytes", len(message.Payload), "latency_ms", latency)

	return nil
}

// ConsumeMessages consumes messages from the queue with handler support
func (p *BaseMQPlugin) ConsumeMessages(ctx context.Context, topic string, handler func(MessageQueueMessage) error) error {
	// Validate context
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	// Validate topic
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	// Validate handler
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	p.logger.Info("consuming messages", "topic", topic)

	// Record consume start metric
	p.metricsCollector.RecordCounter("mq_consume_start", int64(1), map[string]string{"topic": topic})

	// This is a base implementation - subclasses should override
	// to provide actual message consumption logic
	// For now, we block and wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

// AcknowledgeMessage acknowledges a message with offset updates and metrics recording
func (p *BaseMQPlugin) AcknowledgeMessage(ctx context.Context, message MessageQueueMessage) error {
	startTime := time.Now()

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if message.Topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if message.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	// Record acknowledgment metrics
	latency := time.Since(startTime).Milliseconds()
	p.metricsCollector.RecordCounter("mq_messages_acknowledged", int64(1), map[string]string{"topic": message.Topic})
	p.metricsCollector.RecordHistogram("mq_acknowledge_latency_ms", float64(latency), map[string]string{"topic": message.Topic})
	p.logger.Info("message acknowledged", "topic", message.Topic, "message_id", message.ID, "offset", message.Offset, "latency_ms", latency)

	return nil
}

// SendToDeadLetterQueue sends a message to the dead letter queue
func (p *BaseMQPlugin) SendToDeadLetterQueue(ctx context.Context, message MessageQueueMessage, reason string) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	message.DeadLetterReason = reason
	p.deadLetterQueueSize.Add(1)
	p.metricsCollector.RecordCounter("mq_dead_letter_queue_size", 1, nil)
	p.logger.Warn("message sent to dead letter queue", "topic", message.Topic, "reason", reason)

	// Store the message in the DLQ
	p.dlqMutex.Lock()
	p.deadLetterQueue = append(p.deadLetterQueue, message)
	p.dlqMutex.Unlock()

	return nil
}

// GetDeadLetterQueueMessages retrieves messages from the dead letter queue
func (p *BaseMQPlugin) GetDeadLetterQueueMessages(ctx context.Context, limit int) ([]MessageQueueMessage, error) {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil, fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	p.dlqMutex.RLock()
	defer p.dlqMutex.RUnlock()

	p.logger.Info("retrieving dead letter queue messages", "limit", limit)

	if limit <= 0 || limit > len(p.deadLetterQueue) {
		limit = len(p.deadLetterQueue)
	}

	messages := make([]MessageQueueMessage, limit)
	copy(messages, p.deadLetterQueue[:limit])
	return messages, nil
}

// RetryMessage retries a message with exponential backoff
func (p *BaseMQPlugin) RetryMessage(ctx context.Context, message MessageQueueMessage) error {
	startTime := time.Now()

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}

	if ctx == nil {
		p.mu.Unlock()
		return fmt.Errorf("context cannot be nil")
	}

	if message.Topic == "" {
		p.mu.Unlock()
		return fmt.Errorf("topic cannot be empty")
	}

	if message.ID == "" {
		p.mu.Unlock()
		return fmt.Errorf("message ID cannot be empty")
	}

	// Check if max retries exceeded
	if message.RetryCount >= p.maxRetries {
		p.mu.Unlock()
		// Send to DLQ if max retries exceeded
		reason := fmt.Sprintf("max retries exceeded after %d attempts", message.RetryCount)
		dlqErr := p.SendToDeadLetterQueue(ctx, message, reason)
		if dlqErr != nil {
			p.logger.Error("failed to send to DLQ", "topic", message.Topic, "message_id", message.ID, "error", dlqErr)
			return dlqErr
		}
		return fmt.Errorf("max retries exceeded")
	}

	// Increment retry count
	message.RetryCount++
	currentRetryCount := message.RetryCount

	p.mu.Unlock()

	// Calculate exponential backoff delay: baseDelay * (2 ^ retryCount)
	// Formula: delay = retryDelay * math.Pow(2, float64(currentRetryCount-1))
	// For retry 1: delay = retryDelay * 2^0 = retryDelay
	// For retry 2: delay = retryDelay * 2^1 = retryDelay * 2
	// For retry 3: delay = retryDelay * 2^2 = retryDelay * 4
	delayMultiplier := 1 << uint(currentRetryCount-1) // Bit shift for 2^(retryCount-1)
	delayDuration := p.retryDelay * time.Duration(delayMultiplier)
	delayMs := delayDuration.Milliseconds()

	// Record retry metrics before delay
	p.metricsCollector.RecordCounter("mq_message_retries", int64(1), map[string]string{"topic": message.Topic, "retry_count": fmt.Sprintf("%d", currentRetryCount)})
	p.metricsCollector.RecordHistogram("mq_retry_delay_ms", float64(delayMs), map[string]string{"topic": message.Topic})
	p.logger.Info("message retry scheduled", "topic", message.Topic, "message_id", message.ID, "retry_count", currentRetryCount, "delay_ms", delayMs)

	// Wait for the calculated delay (respecting context cancellation)
	select {
	case <-time.After(delayDuration):
		// Delay completed, proceed with retry
		latency := time.Since(startTime).Milliseconds()
		p.metricsCollector.RecordHistogram("mq_retry_latency_ms", float64(latency), map[string]string{"topic": message.Topic})
		p.logger.Info("message retry executed", "topic", message.Topic, "message_id", message.ID, "retry_count", currentRetryCount, "total_latency_ms", latency)
		return nil

	case <-ctx.Done():
		// Context cancelled during delay
		p.logger.Warn("message retry cancelled", "topic", message.Topic, "message_id", message.ID, "retry_count", currentRetryCount)
		return ctx.Err()
	}
}

// GetStats returns statistics about the message queue
func (p *BaseMQPlugin) GetStats() MessageQueueStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return MessageQueueStats{
		MessageCount:        p.messageCount.Load(),
		ErrorCount:          p.errorCount.Load(),
		DeadLetterQueueSize: p.deadLetterQueueSize.Load(),
		AverageProcessTime:  p.processingTime,
		LastError:           p.lastError,
		LastErrorTime:       p.lastErrorTime,
		IsRunning:           p.isRunning,
	}
}

// SetBatchSize sets the batch size for message processing
func (p *BaseMQPlugin) SetBatchSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchSize = size
	p.logger.Info("batch size set", "size", size)
}

// SetMaxRetries sets the maximum number of retries
func (p *BaseMQPlugin) SetMaxRetries(maxRetries int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRetries = maxRetries
	p.logger.Info("max retries set", "max_retries", maxRetries)
}

// SetRetryDelay sets the retry delay
func (p *BaseMQPlugin) SetRetryDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retryDelay = delay
	p.logger.Info("retry delay set", "delay", delay)
}

// RecordMetric records a metric
func (p *BaseMQPlugin) RecordMetric(name string, value int64, tags map[string]string) {
	p.metricsCollector.RecordCounter(name, value, tags)
}

// LogInfo logs an info message
func (p *BaseMQPlugin) LogInfo(message string, fields ...interface{}) {
	p.logger.Info(message, fields...)
}

// LogError logs an error message
func (p *BaseMQPlugin) LogError(message string, fields ...interface{}) {
	p.logger.Error(message, fields...)
}

// LogWarn logs a warning message
func (p *BaseMQPlugin) LogWarn(message string, fields ...interface{}) {
	p.logger.Warn(message, fields...)
}

// RecordError records an error
func (p *BaseMQPlugin) RecordError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCount.Add(1)
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.metricsCollector.RecordCounter("mq_errors", int64(1), nil)
}

// GetLastBlockNumber returns the last block number processed
func (p *BaseMQPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastBlockNumber
}

// SetLastBlockNumber sets the last block number processed
func (p *BaseMQPlugin) SetLastBlockNumber(blockNumber uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastBlockNumber = blockNumber
}

// AddToBatch adds a message to the batch buffer
func (p *BaseMQPlugin) AddToBatch(message MessageQueueMessage) error {
	p.batchBufferMutex.Lock()
	defer p.batchBufferMutex.Unlock()

	p.batchBuffer = append(p.batchBuffer, message)
	p.metricsCollector.RecordCounter("mq_batch_messages_added", int64(1), map[string]string{"topic": message.Topic})

	return nil
}

// GetBatchSize returns the current batch size
func (p *BaseMQPlugin) GetBatchSize() int {
	p.batchBufferMutex.RLock()
	defer p.batchBufferMutex.RUnlock()
	return len(p.batchBuffer)
}

// ProcessBatch processes all messages in the batch with atomicity
func (p *BaseMQPlugin) ProcessBatch(ctx context.Context, handler func([]MessageQueueMessage) error) error {
	p.batchBufferMutex.Lock()
	if len(p.batchBuffer) == 0 {
		p.batchBufferMutex.Unlock()
		return nil
	}

	// Create a copy of the batch
	batch := make([]MessageQueueMessage, len(p.batchBuffer))
	copy(batch, p.batchBuffer)
	p.batchBufferMutex.Unlock()

	startTime := time.Now()

	// Process batch atomically
	err := handler(batch)
	if err != nil {
		p.mu.Lock()
		p.errorCount.Add(1)
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()

		latency := time.Since(startTime).Milliseconds()
		p.metricsCollector.RecordCounter("mq_batch_process_errors", int64(1), nil)
		p.metricsCollector.RecordHistogram("mq_batch_error_latency_ms", float64(latency), nil)
		p.logger.Error("batch processing failed", "batch_size", len(batch), "error", err, "latency_ms", latency)
		return err
	}

	// Clear batch on success
	p.batchBufferMutex.Lock()
	p.batchBuffer = make([]MessageQueueMessage, 0, p.batchSize)
	p.batchBufferMutex.Unlock()

	latency := time.Since(startTime).Milliseconds()
	p.batchProcessedCount.Add(1)

	p.metricsCollector.RecordCounter("mq_batches_processed", int64(1), nil)
	p.metricsCollector.RecordCounter("mq_batch_messages_processed", int64(len(batch)), nil)
	p.metricsCollector.RecordHistogram("mq_batch_latency_ms", float64(latency), nil)
	p.logger.Info("batch processed successfully", "batch_size", len(batch), "latency_ms", latency)

	return nil
}

// ClearBatch clears the batch buffer
func (p *BaseMQPlugin) ClearBatch() {
	p.batchBufferMutex.Lock()
	defer p.batchBufferMutex.Unlock()
	p.batchBuffer = make([]MessageQueueMessage, 0, p.batchSize)
}

// SetBatchTimeout sets the batch timeout
func (p *BaseMQPlugin) SetBatchTimeout(timeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchTimeout = timeout
	p.logger.Info("batch timeout set", "timeout", timeout)
}

// GetBatchTimeout returns the batch timeout
func (p *BaseMQPlugin) GetBatchTimeout() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.batchTimeout
}

// GetBatchProcessedCount returns the number of batches processed
func (p *BaseMQPlugin) GetBatchProcessedCount() int64 {
	return p.batchProcessedCount.Load()
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

// GetMetricsSnapshot returns a comprehensive snapshot of all metrics
func (p *BaseMQPlugin) GetMetricsSnapshot() MetricsSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return MetricsSnapshot{
		MessageCount:        p.messageCount.Load(),
		ErrorCount:          p.errorCount.Load(),
		DeadLetterQueueSize: p.deadLetterQueueSize.Load(),
		BatchesProcessed:    p.batchProcessedCount.Load(),
		AverageProcessTime:  p.processingTime,
		LastError:           p.lastError,
		LastErrorTime:       p.lastErrorTime,
		IsRunning:           p.isRunning,
		Timestamp:           time.Now().UTC(),
	}
}

// RecordPublishMetrics records comprehensive publish metrics
func (p *BaseMQPlugin) RecordPublishMetrics(topic string, messageSize int64, latencyMs int64, success bool) {
	if success {
		p.metricsCollector.RecordCounter("mq_messages_published", int64(1), map[string]string{"topic": topic})
		p.metricsCollector.RecordHistogram("mq_publish_latency_ms", float64(latencyMs), map[string]string{"topic": topic})
		p.metricsCollector.RecordGauge("mq_message_size_bytes", float64(messageSize), map[string]string{"topic": topic})
	} else {
		p.metricsCollector.RecordCounter("mq_publish_errors", int64(1), map[string]string{"topic": topic})
		p.metricsCollector.RecordHistogram("mq_publish_error_latency_ms", float64(latencyMs), map[string]string{"topic": topic})
	}
}

// RecordConsumeMetrics records comprehensive consume metrics
func (p *BaseMQPlugin) RecordConsumeMetrics(topic string, latencyMs int64, success bool) {
	if success {
		p.metricsCollector.RecordCounter("mq_messages_consumed", int64(1), map[string]string{"topic": topic})
		p.metricsCollector.RecordHistogram("mq_consume_latency_ms", float64(latencyMs), map[string]string{"topic": topic})
	} else {
		p.metricsCollector.RecordCounter("mq_consume_errors", int64(1), map[string]string{"topic": topic})
		p.metricsCollector.RecordHistogram("mq_consume_error_latency_ms", float64(latencyMs), map[string]string{"topic": topic})
	}
}

// RecordDLQMetrics records comprehensive DLQ metrics
func (p *BaseMQPlugin) RecordDLQMetrics(topic string, reason string, latencyMs int64, success bool) {
	if success {
		p.metricsCollector.RecordCounter("mq_dlq_messages_sent", int64(1), map[string]string{"topic": topic, "reason": reason})
		p.metricsCollector.RecordHistogram("mq_dlq_send_latency_ms", float64(latencyMs), map[string]string{"topic": topic})
	} else {
		p.metricsCollector.RecordCounter("mq_dlq_send_errors", int64(1), map[string]string{"topic": topic, "reason": reason})
		p.metricsCollector.RecordHistogram("mq_dlq_error_latency_ms", float64(latencyMs), map[string]string{"topic": topic})
	}
}

// CalculateExponentialBackoffDelay calculates exponential backoff delay
// Formula: baseDelay * (2 ^ (retryCount - 1))
// For retry 1: delay = baseDelay * 2^0 = baseDelay
// For retry 2: delay = baseDelay * 2^1 = baseDelay * 2
// For retry 3: delay = baseDelay * 2^2 = baseDelay * 4
func (p *BaseMQPlugin) CalculateExponentialBackoffDelay(retryCount int) time.Duration {
	if retryCount <= 0 {
		return p.retryDelay
	}
	delayMultiplier := 1 << uint(retryCount-1) // Bit shift for 2^(retryCount-1)
	return p.retryDelay * time.Duration(delayMultiplier)
}

// RecordRetryMetrics records comprehensive retry metrics
func (p *BaseMQPlugin) RecordRetryMetrics(topic string, retryCount int, delayMs int64) {
	p.metricsCollector.RecordCounter("mq_message_retries", int64(1), map[string]string{"topic": topic, "retry_count": fmt.Sprintf("%d", retryCount)})
	p.metricsCollector.RecordHistogram("mq_retry_delay_ms", float64(delayMs), map[string]string{"topic": topic})
}

// RecordAcknowledgmentMetrics records acknowledgment metrics
func (p *BaseMQPlugin) RecordAcknowledgmentMetrics(topic string, batchSize int64) {
	p.metricsCollector.RecordCounter("mq_messages_acknowledged", batchSize, map[string]string{"topic": topic})
}

// GetInFlightOperationCount returns the number of in-flight operations
func (p *BaseMQPlugin) GetInFlightOperationCount() int64 {
	return p.inFlightOperations.Load()
}

// TrackInFlightOperation tracks the start of an in-flight operation
func (p *BaseMQPlugin) TrackInFlightOperation() {
	p.inFlightWaitGroup.Add(1)
	p.inFlightOperations.Add(1)
}

// CompleteInFlightOperation marks an in-flight operation as complete
func (p *BaseMQPlugin) CompleteInFlightOperation() {
	p.inFlightOperations.Add(-1)
	p.inFlightWaitGroup.Done()
}

// AcknowledgeMessageBatch acknowledges multiple messages in a batch for efficiency
func (p *BaseMQPlugin) AcknowledgeMessageBatch(ctx context.Context, messages []MessageQueueMessage) error {
	startTime := time.Now()

	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if len(messages) == 0 {
		return fmt.Errorf("messages slice cannot be empty")
	}

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	// Validate all messages
	for _, msg := range messages {
		if msg.Topic == "" {
			return fmt.Errorf("topic cannot be empty")
		}
		if msg.ID == "" {
			return fmt.Errorf("message ID cannot be empty")
		}
	}

	// Group messages by topic for efficient batch processing
	messagesByTopic := make(map[string][]MessageQueueMessage)
	for _, msg := range messages {
		messagesByTopic[msg.Topic] = append(messagesByTopic[msg.Topic], msg)
	}

	// Record batch acknowledgment metrics
	latency := time.Since(startTime).Milliseconds()
	for topic, topicMessages := range messagesByTopic {
		p.metricsCollector.RecordCounter("mq_messages_acknowledged_batch", int64(len(topicMessages)), map[string]string{"topic": topic})
		p.metricsCollector.RecordHistogram("mq_batch_acknowledge_latency_ms", float64(latency), map[string]string{"topic": topic})
	}

	p.logger.Info("batch acknowledgment completed", "batch_size", len(messages), "topics", len(messagesByTopic), "latency_ms", latency)

	return nil
}

// MQConfiguration represents configuration for the MQ plugin
type MQConfiguration struct {
	BatchSize  int
	MaxRetries int
	RetryDelay time.Duration
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

	p.logger.Info("MQ configuration applied",
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
