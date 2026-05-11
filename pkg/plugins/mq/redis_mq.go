package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"github.com/go-redis/redis/v8"
)

// RedisMQPlugin represents the Redis message queue plugin
type RedisMQPlugin struct {
	name                string
	version             string
	config              *core.Config
	logger              core.Logger
	metricsCollector    core.MetricsCollector
	eventBus            core.EventBus
	isInitialized       bool
	isRunning           bool
	mu                  sync.RWMutex
	client              *redis.Client
	messageCount        int64
	errorCount          int64
	lastError           error
	lastErrorTime       time.Time
	deadLetterQueueSize int64
	processingTime      int64
	batchSize           int
	maxRetries          int
	retryDelay          time.Duration
	connectionURL       string
	offsetTracking      map[string]int64
	inFlight            sync.WaitGroup // tracks in-flight publish/consume operations
}

// NewRedisMQPlugin creates a new Redis message queue plugin
func NewRedisMQPlugin(
	name, version string,
	config *core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
	connectionURL string,
) *RedisMQPlugin {
	return &RedisMQPlugin{
		name:                name,
		version:             version,
		config:              config,
		logger:              logger,
		metricsCollector:    metricsCollector,
		eventBus:            eventBus,
		isInitialized:       false,
		isRunning:           false,
		messageCount:        0,
		errorCount:          0,
		deadLetterQueueSize: 0,
		processingTime:      0,
		batchSize:           100,
		maxRetries:          3,
		retryDelay:          1 * time.Second,
		connectionURL:       connectionURL,
		offsetTracking:      make(map[string]int64),
	}
}

// Initialize initializes the Redis plugin
func (p *RedisMQPlugin) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isInitialized {
		return fmt.Errorf("plugin already initialized")
	}

	// Parse Redis connection URL and create client
	opt, err := redis.ParseURL(p.connectionURL)
	if err != nil {
		p.logger.Error("failed to parse Redis URL", "url", p.connectionURL, "error", err)
		return err
	}

	p.client = redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.client.Ping(ctx).Err(); err != nil {
		p.logger.Error("failed to connect to Redis", "error", err)
		return err
	}

	p.isInitialized = true
	p.logger.Info("Redis MQ plugin initialized", "name", p.name, "version", p.version, "url", p.connectionURL)

	return nil
}

// Start starts the Redis plugin
func (p *RedisMQPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	p.isRunning = true
	p.logger.Info("Redis MQ plugin started", "name", p.name)

	return nil
}

// Stop stops the Redis plugin
func (p *RedisMQPlugin) Stop() error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.isRunning = false
	p.mu.Unlock()

	// Wait for in-flight operations with timeout
	done := make(chan struct{})
	go func() {
		p.inFlight.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Redis in-flight operations completed")
	case <-time.After(10 * time.Second):
		p.logger.Warn("Redis stop timed out waiting for in-flight operations")
	}

	// Close Redis connection
	p.mu.Lock()
	if p.client != nil {
		if err := p.client.Close(); err != nil {
			p.logger.Error("failed to close Redis connection", "error", err)
		}
	}
	p.mu.Unlock()

	p.logger.Info("Redis MQ plugin stopped", "name", p.name)

	return nil
}

// Health returns the health status of the plugin
func (p *RedisMQPlugin) Health() *core.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := "healthy"
	if p.errorCount > 0 {
		status = "degraded"
	}

	return &core.HealthStatus{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Details: map[string]interface{}{
			"name":                   p.name,
			"version":                p.version,
			"is_running":             p.isRunning,
			"message_count":          p.messageCount,
			"error_count":            p.errorCount,
			"dead_letter_queue_size": p.deadLetterQueueSize,
			"connection_url":         p.connectionURL,
		},
	}
}

// Name returns the plugin name
func (p *RedisMQPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *RedisMQPlugin) Version() string {
	return p.version
}

// IsInitialized returns whether the plugin is initialized
func (p *RedisMQPlugin) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isInitialized
}

// IsRunning returns whether the plugin is running
func (p *RedisMQPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// PublishMessage publishes a message to a Redis list
func (p *RedisMQPlugin) PublishMessage(ctx context.Context, message core.MessageQueueMessage) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// Use Redis list (LPUSH) for message queue
	queueKey := fmt.Sprintf("queue:%s", message.Topic)

	// Create message with metadata
	msgData := fmt.Sprintf("%s|%s|%d", message.ID, message.Timestamp.String(), message.Offset)

	// Push message to Redis list
	err := client.LPush(ctx, queueKey, msgData).Err()
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()
		p.metricsCollector.RecordCounter("mq_publish_errors", int64(1), map[string]string{"topic": message.Topic})
		p.logger.Error("failed to publish message to Redis", "topic", message.Topic, "error", err)
		return err
	}

	p.mu.Lock()
	p.messageCount++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter("mq_messages_published", int64(1), map[string]string{"topic": message.Topic})
	p.logger.Info("message published to Redis", "topic", message.Topic, "message_id", message.ID)

	return nil
}

// ConsumeMessages consumes messages from a Redis list
func (p *RedisMQPlugin) ConsumeMessages(ctx context.Context, topic string, handler func(core.MessageQueueMessage) error) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	queueKey := fmt.Sprintf("queue:%s", topic)
	p.logger.Info("consuming messages from Redis", "topic", topic)

	// Consume messages in a loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Use BRPOP to block and wait for messages
			result, err := client.BRPop(ctx, 1*time.Second, queueKey).Result()
			if err != nil {
				if err == redis.Nil {
					// Timeout, no message available
					continue
				}
				p.mu.Lock()
				p.errorCount++
				p.lastError = err
				p.lastErrorTime = time.Now()
				p.mu.Unlock()
				p.metricsCollector.RecordCounter("mq_consume_errors", int64(1), map[string]string{"topic": topic})
				p.logger.Error("failed to read message from Redis", "topic", topic, "error", err)
				continue
			}

			if len(result) < 2 {
				continue
			}

			msgData := result[1]

			// Parse message data
			queueMsg := core.MessageQueueMessage{
				ID:        msgData,
				Topic:     topic,
				Payload:   []byte(msgData),
				Timestamp: time.Now().UTC(),
			}

			// Call handler
			if err := handler(queueMsg); err != nil {
				p.mu.Lock()
				p.errorCount++
				p.lastError = err
				p.lastErrorTime = time.Now()
				p.mu.Unlock()
				p.metricsCollector.RecordCounter("mq_handler_errors", int64(1), map[string]string{"topic": topic})
				p.logger.Error("handler error", "topic", topic, "message_id", msgData, "error", err)
				continue
			}

			p.metricsCollector.RecordCounter("mq_messages_consumed", int64(1), map[string]string{"topic": topic})
		}
	}
}

// AcknowledgeMessage acknowledges a message
func (p *RedisMQPlugin) AcknowledgeMessage(ctx context.Context, message core.MessageQueueMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("plugin not running")
	}

	p.metricsCollector.RecordCounter("mq_messages_acknowledged", int64(1), map[string]string{"topic": message.Topic})
	p.logger.Info("message acknowledged", "topic", message.Topic, "message_id", message.ID)

	return nil
}

// SendToDeadLetterQueue sends a message to the dead letter queue
func (p *RedisMQPlugin) SendToDeadLetterQueue(ctx context.Context, message core.MessageQueueMessage, reason string) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// Create DLQ key
	dlqKey := fmt.Sprintf("dlq:%s", message.Topic)

	// Create DLQ message with reason
	dlqMsg := fmt.Sprintf("%s|%s|%s", message.ID, reason, message.Timestamp.String())

	// Push message to DLQ
	err := client.LPush(ctx, dlqKey, dlqMsg).Err()
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()
		p.logger.Error("failed to send message to DLQ", "topic", dlqKey, "error", err)
		return err
	}

	p.mu.Lock()
	p.deadLetterQueueSize++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter("mq_dead_letter_queue_size", p.deadLetterQueueSize, nil)
	p.logger.Warn("message sent to dead letter queue", "topic", dlqKey, "reason", reason)

	return nil
}

// GetDeadLetterQueueMessages retrieves messages from the dead letter queue
func (p *RedisMQPlugin) GetDeadLetterQueueMessages(ctx context.Context, limit int) ([]core.MessageQueueMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil, fmt.Errorf("plugin not running")
	}

	messages := make([]core.MessageQueueMessage, 0)
	p.logger.Info("retrieving dead letter queue messages", "limit", limit)

	return messages, nil
}

// RetryMessage retries a message
func (p *RedisMQPlugin) RetryMessage(ctx context.Context, message core.MessageQueueMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return fmt.Errorf("plugin not running")
	}

	if message.RetryCount >= p.maxRetries {
		return fmt.Errorf("max retries exceeded")
	}

	message.RetryCount++
	p.metricsCollector.RecordCounter("mq_message_retries", int64(1), map[string]string{"topic": message.Topic})
	p.logger.Info("message retry", "topic", message.Topic, "retry_count", message.RetryCount)

	return nil
}

// GetStats returns statistics about the message queue
func (p *RedisMQPlugin) GetStats() core.MessageQueueStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return core.MessageQueueStats{
		MessageCount:        p.messageCount,
		ErrorCount:          p.errorCount,
		DeadLetterQueueSize: p.deadLetterQueueSize,
		AverageProcessTime:  p.processingTime,
		LastError:           p.lastError,
		LastErrorTime:       p.lastErrorTime,
		IsRunning:           p.isRunning,
	}
}

// SetBatchSize sets the batch size for message processing
func (p *RedisMQPlugin) SetBatchSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchSize = size
	p.logger.Info("batch size set", "size", size)
}

// SetMaxRetries sets the maximum number of retries
func (p *RedisMQPlugin) SetMaxRetries(maxRetries int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRetries = maxRetries
	p.logger.Info("max retries set", "max_retries", maxRetries)
}

// SetRetryDelay sets the retry delay
func (p *RedisMQPlugin) SetRetryDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retryDelay = delay
	p.logger.Info("retry delay set", "delay", delay)
}

// RecordMetric records a metric
func (p *RedisMQPlugin) RecordCounter(name string, value int64, tags map[string]string) {
	p.metricsCollector.RecordCounter(name, value, tags)
}

// LogInfo logs an info message
func (p *RedisMQPlugin) LogInfo(message string, fields ...interface{}) {
	p.logger.Info(message, fields...)
}

// LogError logs an error message
func (p *RedisMQPlugin) LogError(message string, fields ...interface{}) {
	p.logger.Error(message, fields...)
}

// LogWarn logs a warning message
func (p *RedisMQPlugin) LogWarn(message string, fields ...interface{}) {
	p.logger.Warn(message, fields...)
}

// RecordError records an error
func (p *RedisMQPlugin) RecordError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCount++
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.metricsCollector.RecordCounter("mq_errors", int64(1), nil)
}

// GetLastBlockNumber returns the last block number processed
func (p *RedisMQPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return 0
}

// SetLastBlockNumber sets the last block number processed
func (p *RedisMQPlugin) SetLastBlockNumber(_ uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Not used for Redis MQ
}

// GetQueueDepth returns the depth of a queue
func (p *RedisMQPlugin) GetQueueDepth(ctx context.Context, topic string) (int64, error) {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return 0, fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	queueKey := fmt.Sprintf("queue:%s", topic)
	depth, err := client.LLen(ctx, queueKey).Result()
	if err != nil {
		p.logger.Error("failed to get queue depth", "topic", topic, "error", err)
		return 0, err
	}

	return depth, nil
}

// FlushQueue flushes all messages from a queue
func (p *RedisMQPlugin) FlushQueue(ctx context.Context, topic string) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	client := p.client
	p.mu.Unlock()

	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	queueKey := fmt.Sprintf("queue:%s", topic)
	err := client.Del(ctx, queueKey).Err()
	if err != nil {
		p.logger.Error("failed to flush queue", "topic", topic, "error", err)
		return err
	}

	p.logger.Info("queue flushed", "topic", topic)
	return nil
}
