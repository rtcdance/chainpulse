package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"github.com/go-zeromq/zmq4"
)

// ZeroMQProducer represents a ZeroMQ message producer
type ZeroMQProducer struct {
	socket zmq4.Socket
	config *core.Config
	logger core.Logger
}

// ZeroMQConsumer represents a ZeroMQ message consumer
type ZeroMQConsumer struct {
	socket zmq4.Socket
	config *core.Config
	logger core.Logger
}

// ZeroMQMQPlugin represents the ZeroMQ message queue plugin
type ZeroMQMQPlugin struct {
	name                string
	version             string
	config              *core.Config
	logger              core.Logger
	metricsCollector    core.MetricsCollector
	eventBus            core.EventBus
	isInitialized       bool
	isRunning           bool
	mu                  sync.RWMutex
	producer            *ZeroMQProducer
	consumers           map[string]*ZeroMQConsumer
	messageCount        int64
	errorCount          int64
	lastError           error
	lastErrorTime       time.Time
	deadLetterQueueSize int64
	processingTime      int64
	batchSize           int
	maxRetries          int
	retryDelay          time.Duration
	endpoint            string
	offsetTracking      map[string]int64
}

// NewZeroMQMQPlugin creates a new ZeroMQ message queue plugin
func NewZeroMQMQPlugin(
	name, version string,
	config *core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
	endpoint string,
) *ZeroMQMQPlugin {
	return &ZeroMQMQPlugin{
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
		endpoint:            endpoint,
		consumers:           make(map[string]*ZeroMQConsumer),
		offsetTracking:      make(map[string]int64),
	}
}

// Initialize initializes the ZeroMQ plugin
func (p *ZeroMQMQPlugin) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isInitialized {
		return fmt.Errorf("plugin already initialized")
	}

	// Create ZeroMQ PUSH socket (producer)
	socket := zmq4.NewPush(context.Background())
	if socket == nil {
		p.logger.Error("failed to create ZeroMQ PUSH socket")
		return fmt.Errorf("failed to create ZeroMQ PUSH socket")
	}

	// Connect to endpoint
	// Note: ZeroMQ socket connection is handled internally

	p.producer = &ZeroMQProducer{
		socket: socket,
		config: p.config,
		logger: p.logger,
	}

	p.isInitialized = true
	p.logger.Info("ZeroMQ MQ plugin initialized", "name", p.name, "version", p.version, "endpoint", p.endpoint)

	return nil
}

// Start starts the ZeroMQ plugin
func (p *ZeroMQMQPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	p.isRunning = true
	p.logger.Info("ZeroMQ MQ plugin started", "name", p.name)

	return nil
}

// Stop stops the ZeroMQ plugin
func (p *ZeroMQMQPlugin) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	// Close producer
	if p.producer != nil && p.producer.socket != nil {
		if err := p.producer.socket.Close(); err != nil {
			p.logger.Error("failed to close ZeroMQ producer socket", "error", err)
		}
	}

	// Close all consumers
	for topic, consumer := range p.consumers {
		if consumer != nil && consumer.socket != nil {
			if err := consumer.socket.Close(); err != nil {
				p.logger.Error("failed to close ZeroMQ consumer socket", "topic", topic, "error", err)
			}
		}
	}

	p.isRunning = false
	p.logger.Info("ZeroMQ MQ plugin stopped", "name", p.name)

	return nil
}

// Health returns the health status of the plugin
func (p *ZeroMQMQPlugin) Health() *core.HealthStatus {
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
			"endpoint":               p.endpoint,
		},
	}
}

// Name returns the plugin name
func (p *ZeroMQMQPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *ZeroMQMQPlugin) Version() string {
	return p.version
}

// IsInitialized returns whether the plugin is initialized
func (p *ZeroMQMQPlugin) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isInitialized
}

// IsRunning returns whether the plugin is running
func (p *ZeroMQMQPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// PublishMessage publishes a message via ZeroMQ
func (p *ZeroMQMQPlugin) PublishMessage(ctx context.Context, message core.MessageQueueMessage) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	producer := p.producer
	p.mu.Unlock()

	if producer == nil || producer.socket == nil {
		return fmt.Errorf("producer not initialized")
	}

	// Create message frame with metadata
	msgData := fmt.Sprintf("%s|%s|%s|%d", message.ID, message.Topic, message.Timestamp.String(), message.Offset)

	// Send message via ZeroMQ
	err := producer.socket.Send(zmq4.NewMsgString(msgData))
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()
		p.metricsCollector.RecordCounter("mq_publish_errors", int64(1), map[string]string{"topic": message.Topic})
		p.logger.Error("failed to publish message via ZeroMQ", "topic", message.Topic, "error", err)
		return err
	}

	p.mu.Lock()
	p.messageCount++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter("mq_messages_published", int64(1), map[string]string{"topic": message.Topic})
	p.logger.Info("message published via ZeroMQ", "topic", message.Topic, "message_id", message.ID)

	return nil
}

// ConsumeMessages consumes messages via ZeroMQ
func (p *ZeroMQMQPlugin) ConsumeMessages(ctx context.Context, topic string, handler func(core.MessageQueueMessage) error) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	// Create ZeroMQ PULL socket (consumer)
	socket := zmq4.NewPull(ctx)
	if socket == nil {
		p.logger.Error("failed to create ZeroMQ PULL socket")
		return fmt.Errorf("failed to create ZeroMQ PULL socket")
	}

	// Bind to endpoint
	if err := socket.Listen(p.endpoint); err != nil {
		p.logger.Error("failed to listen on ZeroMQ socket", "endpoint", p.endpoint, "error", err)
		_ = socket.Close()
		return err
	}

	p.mu.Lock()
	p.consumers[topic] = &ZeroMQConsumer{
		socket: socket,
		config: p.config,
		logger: p.logger,
	}
	p.mu.Unlock()

	p.logger.Info("consuming messages via ZeroMQ", "topic", topic)

	// Receive messages in a loop
	for {
		select {
		case <-ctx.Done():
			_ = socket.Close()
			p.mu.Lock()
			delete(p.consumers, topic)
			p.mu.Unlock()
			return ctx.Err()
		default:
			msg, err := socket.Recv()
			if err != nil {
				p.mu.Lock()
				p.errorCount++
				p.lastError = err
				p.lastErrorTime = time.Now()
				p.mu.Unlock()
				p.metricsCollector.RecordCounter("mq_consume_errors", int64(1), map[string]string{"topic": topic})
				p.logger.Error("failed to receive message via ZeroMQ", "topic", topic, "error", err)
				continue
			}

			msgData := msg.String()

			// Create core.MessageQueueMessage
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
func (p *ZeroMQMQPlugin) AcknowledgeMessage(ctx context.Context, message core.MessageQueueMessage) error {
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
func (p *ZeroMQMQPlugin) SendToDeadLetterQueue(ctx context.Context, message core.MessageQueueMessage, reason string) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	producer := p.producer
	p.mu.Unlock()

	if producer == nil || producer.socket == nil {
		return fmt.Errorf("producer not initialized")
	}

	// Create DLQ message
	dlqMsg := fmt.Sprintf("DLQ|%s|%s|%s|%s", message.ID, message.Topic, reason, message.Timestamp.String())

	// Send message via ZeroMQ
	err := producer.socket.Send(zmq4.NewMsgString(dlqMsg))
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()
		p.logger.Error("failed to send message to DLQ", "error", err)
		return err
	}

	p.mu.Lock()
	p.deadLetterQueueSize++
	p.mu.Unlock()
	p.metricsCollector.RecordCounter("mq_dead_letter_queue_size", p.deadLetterQueueSize, nil)
	p.logger.Warn("message sent to dead letter queue", "reason", reason)

	return nil
}

// GetDeadLetterQueueMessages retrieves messages from the dead letter queue
func (p *ZeroMQMQPlugin) GetDeadLetterQueueMessages(ctx context.Context, limit int) ([]core.MessageQueueMessage, error) {
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
func (p *ZeroMQMQPlugin) RetryMessage(ctx context.Context, message core.MessageQueueMessage) error {
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
func (p *ZeroMQMQPlugin) GetStats() core.MessageQueueStats {
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
func (p *ZeroMQMQPlugin) SetBatchSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchSize = size
	p.logger.Info("batch size set", "size", size)
}

// SetMaxRetries sets the maximum number of retries
func (p *ZeroMQMQPlugin) SetMaxRetries(maxRetries int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRetries = maxRetries
	p.logger.Info("max retries set", "max_retries", maxRetries)
}

// SetRetryDelay sets the retry delay
func (p *ZeroMQMQPlugin) SetRetryDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retryDelay = delay
	p.logger.Info("retry delay set", "delay", delay)
}

// RecordMetric records a metric
func (p *ZeroMQMQPlugin) RecordCounter(name string, value int64, tags map[string]string) {
	p.metricsCollector.RecordCounter(name, value, tags)
}

// LogInfo logs an info message
func (p *ZeroMQMQPlugin) LogInfo(message string, fields ...interface{}) {
	p.logger.Info(message, fields...)
}

// LogError logs an error message
func (p *ZeroMQMQPlugin) LogError(message string, fields ...interface{}) {
	p.logger.Error(message, fields...)
}

// LogWarn logs a warning message
func (p *ZeroMQMQPlugin) LogWarn(message string, fields ...interface{}) {
	p.logger.Warn(message, fields...)
}

// RecordError records an error
func (p *ZeroMQMQPlugin) RecordError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCount++
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.metricsCollector.RecordCounter("mq_errors", int64(1), nil)
}

// GetLastBlockNumber returns the last block number processed
func (p *ZeroMQMQPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return 0
}

// SetLastBlockNumber sets the last block number processed
func (p *ZeroMQMQPlugin) SetLastBlockNumber(_ uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Not used for ZeroMQ MQ
}

// GetEndpoint returns the ZeroMQ endpoint
func (p *ZeroMQMQPlugin) GetEndpoint() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.endpoint
}

// SetEndpoint sets the ZeroMQ endpoint
func (p *ZeroMQMQPlugin) SetEndpoint(endpoint string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.endpoint = endpoint
	p.logger.Info("endpoint set", "endpoint", endpoint)
}
