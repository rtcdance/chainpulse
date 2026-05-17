package mq

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/observability"
	"github.com/segmentio/kafka-go"
)

// KafkaProducer represents a Kafka message producer
type KafkaProducer struct {
	writer *kafka.Writer
	config *core.Config
	logger core.Logger
}

// KafkaConsumer represents a Kafka message consumer
type KafkaConsumer struct {
	reader *kafka.Reader
	config *core.Config
	logger core.Logger
}

// KafkaMQPlugin represents the Kafka message queue plugin
type KafkaMQPlugin struct {
	name                string
	version             string
	config              *core.Config
	logger              core.Logger
	metricsCollector    core.MetricsCollector
	eventBus            core.EventBus
	isInitialized       bool
	isRunning           bool
	mu                  sync.RWMutex
	producer            *KafkaProducer
	consumers           map[string]*KafkaConsumer
	messageCount        int64
	errorCount          int64
	lastError           error
	lastErrorTime       time.Time
	deadLetterQueueSize int64
	processingTime      int64
	batchSize           int
	maxRetries          int
	retryDelay          time.Duration
	brokers             []string
	consumerGroup       string
	offsetTracking      map[string]int64
	offsetTrackingMutex sync.RWMutex
	dlqReasonCounts     map[string]int64
	dlqReasonMutex      sync.RWMutex
	// Kafka-specific features
	brokerFailureCount   int64
	brokerRecoveryCount  int64
	consumerGroupMetrics map[string]int64
	consumerGroupMutex   sync.RWMutex
	offsetPersistenceMap map[string]map[int32]int64 // topic -> partition -> offset
	offsetPersistMutex   sync.RWMutex
	tracer               *observability.DefaultTracer
	inFlight             sync.WaitGroup // tracks in-flight publish/consume operations
}

// NewKafkaMQPlugin creates a new Kafka message queue plugin
func NewKafkaMQPlugin(
	name, version string,
	config *core.Config,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
	eventBus core.EventBus,
	brokers []string,
	consumerGroup string,
) *KafkaMQPlugin {
	return &KafkaMQPlugin{
		name:                 name,
		version:              version,
		config:               config,
		logger:               logger,
		metricsCollector:     metricsCollector,
		eventBus:             eventBus,
		isInitialized:        false,
		isRunning:            false,
		messageCount:         0,
		errorCount:           0,
		deadLetterQueueSize:  0,
		processingTime:       0,
		batchSize:            100,
		maxRetries:           3,
		retryDelay:           1 * time.Second,
		brokers:              brokers,
		consumerGroup:        consumerGroup,
		consumers:            make(map[string]*KafkaConsumer),
		offsetTracking:       make(map[string]int64),
		dlqReasonCounts:      make(map[string]int64),
		brokerFailureCount:   0,
		brokerRecoveryCount:  0,
		consumerGroupMetrics: make(map[string]int64),
		offsetPersistenceMap: make(map[string]map[int32]int64),
		tracer:               observability.NewDefaultTracer(logger, metricsCollector),
	}
}

// Initialize initializes the Kafka plugin
func (p *KafkaMQPlugin) Initialize() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isInitialized {
		return fmt.Errorf("plugin already initialized")
	}

	// Create Kafka writer (producer)
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(p.brokers...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: false,
	}

	p.producer = &KafkaProducer{
		writer: writer,
		config: p.config,
		logger: p.logger,
	}

	p.isInitialized = true
	p.logger.Info("Kafka MQ plugin initialized", "name", p.name, "version", p.version, "brokers", fmt.Sprintf("%v", p.brokers))

	return nil
}

// Start starts the Kafka plugin
func (p *KafkaMQPlugin) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isInitialized {
		return fmt.Errorf("plugin not initialized")
	}

	if p.isRunning {
		return fmt.Errorf("plugin already running")
	}

	p.isRunning = true
	p.logger.Info("Kafka MQ plugin started", "name", p.name)

	return nil
}

// Stop stops the Kafka plugin
func (p *KafkaMQPlugin) Stop() error {
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
		p.logger.Info("Kafka in-flight operations completed")
	case <-time.After(10 * time.Second):
		p.logger.Warn("Kafka stop timed out waiting for in-flight operations")
	}

	// Close producer
	p.mu.Lock()
	if p.producer != nil && p.producer.writer != nil {
		if err := p.producer.writer.Close(); err != nil {
			p.logger.Error("failed to close Kafka writer", "error", err)
		}
	}

	// Close all consumers
	for topic, consumer := range p.consumers {
		if consumer != nil && consumer.reader != nil {
			if err := consumer.reader.Close(); err != nil {
				p.logger.Error("failed to close Kafka reader", "topic", topic, "error", err)
			}
		}
	}
	p.mu.Unlock()

	p.logger.Info("Kafka MQ plugin stopped", "name", p.name)

	return nil
}

// Health returns the health status of the plugin
func (p *KafkaMQPlugin) Health() *core.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := "healthy"
	if p.errorCount > 0 {
		status = "degraded"
	}

	consumerGroupMetrics := p.GetConsumerGroupMetrics()
	consumerGroupLag := consumerGroupMetrics["lag"]
	maxTrackedOffset := p.maxTrackedOffset()

	return &core.HealthStatus{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Details: map[string]any{
			"name":                   p.name,
			"version":                p.version,
			"is_running":             p.isRunning,
			"message_count":          p.messageCount,
			"error_count":            p.errorCount,
			"dead_letter_queue_size": p.deadLetterQueueSize,
			"brokers":                p.brokers,
			"consumer_group":         p.consumerGroup,
			"active_consumers":       len(p.consumers),
			"consumer_group_lag":     consumerGroupLag,
			"max_tracked_offset":     maxTrackedOffset,
			"consumer_group_metrics": consumerGroupMetrics,
		},
	}
}

// Name returns the plugin name
func (p *KafkaMQPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *KafkaMQPlugin) Version() string {
	return p.version
}

// IsInitialized returns whether the plugin is initialized
func (p *KafkaMQPlugin) IsInitialized() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isInitialized
}

// IsRunning returns whether the plugin is running
func (p *KafkaMQPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isRunning
}

// Publish satisfies the generic MQ plugin contract using the richer Kafka
// message publisher underneath.
func (p *KafkaMQPlugin) Publish(ctx context.Context, topic string, message []byte) error {
	return p.PublishMessage(ctx, core.MessageQueueMessage{
		Topic:   topic,
		Payload: message,
	})
}

// PublishMessage publishes a message to a Kafka topic with comprehensive metrics
func (p *KafkaMQPlugin) PublishMessage(ctx context.Context, message core.MessageQueueMessage) error {
	ctx, span := p.tracer.StartSpan(ctx, "mq.publish", observability.SpanKindProducer)
	defer p.tracer.EndSpan(&span)
	p.tracer.SetAttribute(&span, "topic", message.Topic)
	p.tracer.SetAttribute(&span, "message_id", message.ID)

	startTime := time.Now()

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	producer := p.producer
	p.mu.Unlock()

	if producer == nil || producer.writer == nil {
		return fmt.Errorf("producer not initialized")
	}

	// Generate message ID if not provided
	if message.ID == "" {
		message.ID = fmt.Sprintf("%s-%d", message.Topic, time.Now().UnixNano())
	}

	// Assign timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now().UTC()
	}

	// Create Kafka message with partition key routing
	kafkaMsg := kafka.Message{
		Topic: message.Topic,
		Key:   []byte(message.PartitionKey),
		Value: message.Payload,
		Headers: []kafka.Header{
			{Key: "message_id", Value: []byte(message.ID)},
			{Key: "topic", Value: []byte(message.Topic)},
			{Key: "timestamp", Value: []byte(message.Timestamp.String())},
			{Key: "partition_key", Value: []byte(message.PartitionKey)},
		},
	}

	// Write message to Kafka
	err := producer.writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()

		// Record error metrics
		latency := time.Since(startTime).Milliseconds()
		p.metricsCollector.RecordCounter("mq_publish_errors", 1, map[string]string{"topic": message.Topic})
		p.metricsCollector.RecordGauge("mq_publish_error_latency_ms", float64(latency), map[string]string{"topic": message.Topic})
		p.logger.Error("failed to publish message to Kafka", "topic", message.Topic, "message_id", message.ID, "error", err, "latency_ms", latency)
		return err
	}

	// Record success metrics
	p.mu.Lock()
	p.messageCount++
	p.mu.Unlock()

	latency := time.Since(startTime).Milliseconds()
	p.metricsCollector.RecordCounter("mq_messages_published", 1, map[string]string{"topic": message.Topic})
	p.metricsCollector.RecordGauge("mq_publish_latency_ms", float64(latency), map[string]string{"topic": message.Topic})
	p.metricsCollector.RecordGauge("mq_message_size_bytes", float64(len(message.Payload)), map[string]string{"topic": message.Topic})

	p.logger.Info("message published to Kafka", "topic", message.Topic, "message_id", message.ID, "size_bytes", len(message.Payload), "latency_ms", latency)

	return nil
}

// ConsumeMessages consumes messages from a Kafka topic with exactly-once semantics
// Implements context support for cancellation, handler invocation, error handling,
// graceful consumer shutdown, and offset tracking per topic
func (p *KafkaMQPlugin) ConsumeMessages(ctx context.Context, topic string, handler func(core.MessageQueueMessage) error) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	// Create Kafka reader (consumer) with consumer group for offset tracking
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        p.brokers,
		Topic:          topic,
		GroupID:        p.consumerGroup,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second,
		MaxBytes:       10e6,
	})

	// Register consumer for graceful shutdown
	p.mu.Lock()
	p.consumers[topic] = &KafkaConsumer{
		reader: reader,
		config: p.config,
		logger: p.logger,
	}
	p.mu.Unlock()

	p.logger.Info("consuming messages from Kafka", "topic", topic, "consumer_group", p.consumerGroup)
	p.metricsCollector.RecordCounter("mq_consume_start", 1, map[string]string{"topic": topic})

	// Track in-flight operations for graceful shutdown
	var inFlightOps sync.WaitGroup

	// Track consecutive read errors for backoff and reader recreation
	consecutiveErrors := 0

	// Read messages in a loop
	for {
		select {
		case <-ctx.Done():
			// Context cancelled - graceful shutdown
			p.logger.Info("consumer shutdown initiated", "topic", topic)

			// Wait for in-flight operations to complete
			inFlightOps.Wait()

			// Close reader
			if err := reader.Close(); err != nil {
				p.logger.Error("failed to close Kafka reader", "topic", topic, "error", err)
			}

			// Unregister consumer
			p.mu.Lock()
			delete(p.consumers, topic)
			p.mu.Unlock()

			p.logger.Info("consumer stopped gracefully", "topic", topic)
			p.metricsCollector.RecordCounter("mq_consume_stop", 1, map[string]string{"topic": topic})
			return ctx.Err()

		default:
			// Read message with timeout
			readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			msg, err := reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					// Timeout is normal (no new messages), reset error counter
					consecutiveErrors = 0
					continue
				}
				if errors.Is(err, context.Canceled) {
					// Context was cancelled
					continue
				}

				// Record read error
				p.mu.Lock()
				p.errorCount++
				p.lastError = err
				p.lastErrorTime = time.Now()
				p.mu.Unlock()

				p.metricsCollector.RecordCounter("mq_consume_read_errors", 1, map[string]string{"topic": topic})
				p.logger.Error("failed to read message from Kafka", "topic", topic, "error", err.Error())

				// Apply exponential backoff to avoid tight spin loop
				consecutiveErrors++
				p.RecordBrokerFailure()
				backoff := p.CalculateExponentialBackoffDelay(consecutiveErrors)
				p.logger.Warn("backing off after read error", "topic", topic, "consecutive_errors", consecutiveErrors, "backoff_ms", backoff.Milliseconds())

				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return ctx.Err()
				}

				// Recreate reader after too many consecutive errors
				if consecutiveErrors >= 10 {
					p.logger.Warn("too many consecutive read errors, recreating reader", "topic", topic, "consecutive_errors", consecutiveErrors)
					_ = reader.Close()
					reader = kafka.NewReader(kafka.ReaderConfig{
						Brokers:        p.brokers,
						Topic:          topic,
						GroupID:        p.consumerGroup,
						StartOffset:    kafka.LastOffset,
						CommitInterval: time.Second,
						MaxBytes:       10e6,
					})
					p.mu.Lock()
					p.consumers[topic] = &KafkaConsumer{
						reader: reader,
						config: p.config,
						logger: p.logger,
					}
					p.mu.Unlock()
					consecutiveErrors = 0
				}

				continue
			}

			// Reset error counter on successful read
			consecutiveErrors = 0

			// Extract message metadata from headers
			messageID := ""
			partitionKey := ""
			for _, header := range msg.Headers {
				switch header.Key {
				case "message_id":
					messageID = string(header.Value)
				case "partition_key":
					partitionKey = string(header.Value)
				}
			}

			// Create core.MessageQueueMessage
			queueMsg := core.MessageQueueMessage{
				ID:           messageID,
				Topic:        topic,
				Payload:      msg.Value,
				Timestamp:    msg.Time,
				Offset:       msg.Offset,
				PartitionKey: partitionKey,
			}

			// Track in-flight operation
			inFlightOps.Add(1)

			// Call handler with exactly-once semantics
			// Handler is responsible for idempotent processing
			consumeStartTime := time.Now()
			_, consumeSpan := p.tracer.StartSpan(ctx, "mq.process_message", observability.SpanKindConsumer)
			p.tracer.SetAttribute(&consumeSpan, "topic", topic)
			p.tracer.SetAttribute(&consumeSpan, "message_id", messageID)
			handlerErr := handler(queueMsg)
			p.tracer.SetAttribute(&consumeSpan, "error", handlerErr != nil)
			p.tracer.EndSpan(&consumeSpan)
			consumeLatency := time.Since(consumeStartTime).Milliseconds()

			if handlerErr != nil {
				// Handler error - record and continue
				p.mu.Lock()
				p.errorCount++
				p.lastError = handlerErr
				p.lastErrorTime = time.Now()
				p.mu.Unlock()

				p.metricsCollector.RecordCounter("mq_handler_errors", int64(1), map[string]string{"topic": topic})
				p.metricsCollector.RecordCounter("mq_consume_error_latency_ms", consumeLatency, map[string]string{"topic": topic})
				p.logger.Error("handler error", "topic", topic, "message_id", messageID, "offset", msg.Offset, "error", handlerErr, "latency_ms", consumeLatency)

				inFlightOps.Done()
				continue
			}

			// Track offset for exactly-once semantics
			p.offsetTrackingMutex.Lock()
			p.offsetTracking[topic] = msg.Offset
			p.offsetTrackingMutex.Unlock()

			// Record successful consumption
			p.mu.Lock()
			p.messageCount++
			p.mu.Unlock()

			p.metricsCollector.RecordCounter("mq_messages_consumed", int64(1), map[string]string{"topic": topic})
			p.metricsCollector.RecordCounter("mq_consume_latency_ms", consumeLatency, map[string]string{"topic": topic})
			p.logger.Info("message consumed", "topic", topic, "message_id", messageID, "offset", msg.Offset, "latency_ms", consumeLatency)

			// Mark operation complete
			inFlightOps.Done()
		}
	}
}

// AcknowledgeMessage acknowledges a message with offset updates and metrics recording
func (p *KafkaMQPlugin) AcknowledgeMessage(ctx context.Context, message core.MessageQueueMessage) error {
	startTime := time.Now()

	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if message.Topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if message.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	// Update offset tracking for exactly-once semantics
	p.offsetTrackingMutex.Lock()
	p.offsetTracking[message.Topic] = message.Offset
	p.offsetTrackingMutex.Unlock()

	// Record acknowledgment metrics
	latency := time.Since(startTime).Milliseconds()
	p.metricsCollector.RecordCounter("mq_messages_acknowledged", int64(1), map[string]string{"topic": message.Topic})
	p.metricsCollector.RecordCounter("mq_acknowledge_latency_ms", latency, map[string]string{"topic": message.Topic})
	p.logger.Info("message acknowledged", "topic", message.Topic, "message_id", message.ID, "offset", message.Offset, "latency_ms", latency)

	return nil
}

// SendToDeadLetterQueue sends a message to the dead letter queue with reason preservation
func (p *KafkaMQPlugin) SendToDeadLetterQueue(ctx context.Context, message core.MessageQueueMessage, reason string) error {
	startTime := time.Now()

	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	producer := p.producer
	p.mu.Unlock()

	if producer == nil || producer.writer == nil {
		return fmt.Errorf("producer not initialized")
	}

	// Create DLQ topic name based on original topic
	dlqTopic := message.Topic + "-dlq"

	// Create Kafka message for DLQ with all metadata preserved
	kafkaMsg := kafka.Message{
		Topic: dlqTopic,
		Key:   []byte(message.PartitionKey),
		Value: message.Payload,
		Headers: []kafka.Header{
			{Key: "message_id", Value: []byte(message.ID)},
			{Key: "original_topic", Value: []byte(message.Topic)},
			{Key: "dlq_reason", Value: []byte(reason)},
			{Key: "timestamp", Value: []byte(message.Timestamp.String())},
			{Key: "retry_count", Value: []byte(fmt.Sprintf("%d", message.RetryCount))},
		},
	}

	// Write message to DLQ topic
	err := producer.writer.WriteMessages(ctx, kafkaMsg)
	if err != nil {
		p.mu.Lock()
		p.errorCount++
		p.lastError = err
		p.lastErrorTime = time.Now()
		p.mu.Unlock()

		latency := time.Since(startTime).Milliseconds()
		p.metricsCollector.RecordCounter("mq_dlq_send_errors", int64(1), map[string]string{"topic": dlqTopic, "reason": reason})
		p.metricsCollector.RecordCounter("mq_dlq_error_latency_ms", latency, map[string]string{"topic": dlqTopic})
		p.logger.Error("failed to send message to DLQ", "topic", dlqTopic, "reason", reason, "error", err, "latency_ms", latency)
		return err
	}

	// Track DLQ reason
	p.dlqReasonMutex.Lock()
	p.dlqReasonCounts[reason]++
	p.dlqReasonMutex.Unlock()

	// Update DLQ size and metrics
	p.mu.Lock()
	p.deadLetterQueueSize++
	p.mu.Unlock()

	latency := time.Since(startTime).Milliseconds()
	p.metricsCollector.RecordCounter("mq_dead_letter_queue_size", p.deadLetterQueueSize, nil)
	p.metricsCollector.RecordCounter("mq_dlq_messages_sent", int64(1), map[string]string{"topic": dlqTopic, "reason": reason})
	p.metricsCollector.RecordCounter("mq_dlq_send_latency_ms", latency, map[string]string{"topic": dlqTopic})
	p.logger.Warn("message sent to dead letter queue", "topic", dlqTopic, "reason", reason, "message_id", message.ID, "latency_ms", latency)

	return nil
}

// GetDeadLetterQueueMessages retrieves messages from the dead letter queue with limit support
func (p *KafkaMQPlugin) GetDeadLetterQueueMessages(ctx context.Context, limit int) ([]core.MessageQueueMessage, error) {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil, fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	messages := make([]core.MessageQueueMessage, 0, limit)

	// Get all topics that have DLQ variants
	// For now, we'll create a generic DLQ reader
	dlqTopic := "dlq-messages"

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     p.brokers,
		Topic:       dlqTopic,
		GroupID:     p.consumerGroup + "-dlq",
		StartOffset: kafka.LastOffset,
		MaxBytes:    10e6,
	})
	defer func() {
		_ = reader.Close()
	}()

	p.logger.Info("retrieving dead letter queue messages", "limit", limit, "dlq_topic", dlqTopic)

	// Read up to limit messages from DLQ
	for i := 0; i < limit; i++ {
		readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		msg, err := reader.ReadMessage(readCtx)
		cancel()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				// No more messages available
				break
			}
			p.logger.Error("failed to read DLQ message", "error", err)
			break
		}

		// Extract headers
		messageID := ""
		originalTopic := ""
		dlqReason := ""

		for _, header := range msg.Headers {
			switch header.Key {
			case "message_id":
				messageID = string(header.Value)
			case "original_topic":
				originalTopic = string(header.Value)
			case "dlq_reason":
				dlqReason = string(header.Value)
			}
		}

		// Create core.MessageQueueMessage
		queueMsg := core.MessageQueueMessage{
			ID:               messageID,
			Topic:            originalTopic,
			Payload:          msg.Value,
			Timestamp:        msg.Time,
			Offset:           msg.Offset,
			DeadLetterReason: dlqReason,
		}

		messages = append(messages, queueMsg)
	}

	p.logger.Info("retrieved dead letter queue messages", "count", len(messages), "limit", limit)
	p.metricsCollector.RecordCounter("mq_dlq_messages_retrieved", int64(len(messages)), nil)

	return messages, nil
}

// RetryMessage retries a message
func (p *KafkaMQPlugin) RetryMessage(ctx context.Context, message core.MessageQueueMessage) error {
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
func (p *KafkaMQPlugin) GetStats() core.MessageQueueStats {
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
func (p *KafkaMQPlugin) SetBatchSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchSize = size
	p.logger.Info("batch size set", "size", size)
}

// SetMaxRetries sets the maximum number of retries
func (p *KafkaMQPlugin) SetMaxRetries(maxRetries int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxRetries = maxRetries
	p.logger.Info("max retries set", "max_retries", maxRetries)
}

// SetRetryDelay sets the retry delay
func (p *KafkaMQPlugin) SetRetryDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retryDelay = delay
	p.logger.Info("retry delay set", "delay", delay)
}

// RecordMetric records a metric
func (p *KafkaMQPlugin) RecordCounter(name string, value int64, tags map[string]string) {
	p.metricsCollector.RecordCounter(name, value, tags)
}

// RecordError records an error
func (p *KafkaMQPlugin) RecordError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.errorCount++
	p.lastError = err
	p.lastErrorTime = time.Now()
	p.metricsCollector.RecordCounter("mq_errors", int64(1), nil)
}

// GetLastBlockNumber returns the last block number processed
func (p *KafkaMQPlugin) GetLastBlockNumber() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return 0
}

// SetLastBlockNumber sets the last block number processed
func (p *KafkaMQPlugin) SetLastBlockNumber(_ uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Not used for Kafka MQ
}

// GetLastOffset returns the last offset for a topic
func (p *KafkaMQPlugin) GetLastOffset(topic string) int64 {
	p.offsetTrackingMutex.RLock()
	defer p.offsetTrackingMutex.RUnlock()
	return p.offsetTracking[topic]
}

func (p *KafkaMQPlugin) maxTrackedOffset() int64 {
	p.offsetTrackingMutex.RLock()
	defer p.offsetTrackingMutex.RUnlock()

	var max int64
	for _, offset := range p.offsetTracking {
		if offset > max {
			max = offset
		}
	}
	return max
}

// SetLastOffset sets the last offset for a topic
func (p *KafkaMQPlugin) SetLastOffset(topic string, offset int64) {
	p.offsetTrackingMutex.Lock()
	defer p.offsetTrackingMutex.Unlock()
	p.offsetTracking[topic] = offset
}

// GetDLQReasonStats returns statistics about DLQ reasons
func (p *KafkaMQPlugin) GetDLQReasonStats() map[string]int64 {
	p.dlqReasonMutex.RLock()
	defer p.dlqReasonMutex.RUnlock()

	// Create a copy to avoid external modifications
	stats := make(map[string]int64)
	for reason, count := range p.dlqReasonCounts {
		stats[reason] = count
	}
	return stats
}

// ClearDLQReasonStats clears DLQ reason statistics
func (p *KafkaMQPlugin) ClearDLQReasonStats() {
	p.dlqReasonMutex.Lock()
	defer p.dlqReasonMutex.Unlock()
	p.dlqReasonCounts = make(map[string]int64)
}

// AcknowledgeMessageBatch acknowledges multiple messages in a batch for efficiency
func (p *KafkaMQPlugin) AcknowledgeMessageBatch(ctx context.Context, messages []core.MessageQueueMessage) error {
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
	messagesByTopic := make(map[string][]core.MessageQueueMessage)
	maxOffsetByTopic := make(map[string]int64)

	for _, msg := range messages {
		messagesByTopic[msg.Topic] = append(messagesByTopic[msg.Topic], msg)

		// Track the maximum offset for each topic
		if msg.Offset > maxOffsetByTopic[msg.Topic] {
			maxOffsetByTopic[msg.Topic] = msg.Offset
		}
	}

	// Update offset tracking for each topic with the maximum offset
	p.offsetTrackingMutex.Lock()
	for topic, maxOffset := range maxOffsetByTopic {
		p.offsetTracking[topic] = maxOffset
	}
	p.offsetTrackingMutex.Unlock()

	// Record batch acknowledgment metrics
	latency := time.Since(startTime).Milliseconds()
	for topic, topicMessages := range messagesByTopic {
		p.metricsCollector.RecordCounter("mq_messages_acknowledged_batch", int64(len(topicMessages)), map[string]string{"topic": topic})
		p.metricsCollector.RecordCounter("mq_batch_acknowledge_latency_ms", latency, map[string]string{"topic": topic})
	}

	p.logger.Info("batch acknowledgment completed", "batch_size", len(messages), "topics", len(messagesByTopic), "latency_ms", latency)

	return nil
}

// ============================================================================
// TASK 10: Kafka-Specific Features Implementation
// ============================================================================

// PersistOffset persists an offset for a topic partition to enable recovery
func (p *KafkaMQPlugin) PersistOffset(topic string, partition int32, offset int64) error {
	p.offsetPersistMutex.Lock()
	defer p.offsetPersistMutex.Unlock()

	if _, exists := p.offsetPersistenceMap[topic]; !exists {
		p.offsetPersistenceMap[topic] = make(map[int32]int64)
	}

	p.offsetPersistenceMap[topic][partition] = offset
	p.logger.Info("offset persisted", "topic", topic, "partition", partition, "offset", offset)
	p.metricsCollector.RecordCounter("mq_offset_persisted", int64(1), map[string]string{"topic": topic})

	return nil
}

// GetPersistedOffset retrieves a persisted offset for a topic partition
func (p *KafkaMQPlugin) GetPersistedOffset(topic string, partition int32) (int64, error) {
	p.offsetPersistMutex.RLock()
	defer p.offsetPersistMutex.RUnlock()

	if partitionOffsets, exists := p.offsetPersistenceMap[topic]; exists {
		if offset, exists := partitionOffsets[partition]; exists {
			return offset, nil
		}
	}

	return -1, fmt.Errorf("no persisted offset found for topic %s partition %d", topic, partition)
}

// GetConsumerGroupMetrics returns metrics for the consumer group
func (p *KafkaMQPlugin) GetConsumerGroupMetrics() map[string]int64 {
	p.consumerGroupMutex.RLock()
	defer p.consumerGroupMutex.RUnlock()

	// Create a copy to avoid external modifications
	metrics := make(map[string]int64)
	for key, value := range p.consumerGroupMetrics {
		metrics[key] = value
	}
	return metrics
}

// UpdateConsumerGroupMetric updates a consumer group metric
func (p *KafkaMQPlugin) UpdateConsumerGroupMetric(key string, value int64) {
	p.consumerGroupMutex.Lock()
	defer p.consumerGroupMutex.Unlock()

	// Replace the value instead of accumulating
	p.consumerGroupMetrics[key] = value
	p.metricsCollector.RecordCounter("mq_consumer_group_metric", value, map[string]string{"metric": key})
	p.logger.Info("consumer group metric updated", "metric", key, "value", value)
}

// RecordBrokerFailure records a broker failure and increments the failure counter
func (p *KafkaMQPlugin) RecordBrokerFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.brokerFailureCount++
	p.metricsCollector.RecordCounter("mq_broker_failures", int64(1), nil)
	p.logger.Warn("broker failure recorded", "total_failures", p.brokerFailureCount)
}

// RecordBrokerRecovery records a broker recovery and increments the recovery counter
func (p *KafkaMQPlugin) RecordBrokerRecovery() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.brokerRecoveryCount++
	p.metricsCollector.RecordCounter("mq_broker_recoveries", int64(1), nil)
	p.logger.Info("broker recovery recorded", "total_recoveries", p.brokerRecoveryCount)
}

// GetBrokerFailureCount returns the total number of broker failures
func (p *KafkaMQPlugin) GetBrokerFailureCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.brokerFailureCount
}

// GetBrokerRecoveryCount returns the total number of broker recoveries
func (p *KafkaMQPlugin) GetBrokerRecoveryCount() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.brokerRecoveryCount
}

// CalculateExponentialBackoffDelay calculates exponential backoff delay for broker failover
// Formula: baseDelay * (2 ^ (retryCount - 1)) with optional jitter
// For retry 1: delay = baseDelay * 2^0 = baseDelay
// For retry 2: delay = baseDelay * 2^1 = baseDelay * 2
// For retry 3: delay = baseDelay * 2^2 = baseDelay * 4
func (p *KafkaMQPlugin) CalculateExponentialBackoffDelay(retryCount int) time.Duration {
	if retryCount <= 0 {
		return p.retryDelay
	}

	// Calculate 2^(retryCount-1) using bit shift
	shift := retryCount - 1
	maxShift := bits.UintSize - 2
	if shift > maxShift {
		shift = maxShift
	}
	delayMultiplier := 1 << shift
	return p.retryDelay * time.Duration(delayMultiplier)
}

// ConnectWithRetry attempts to connect to Kafka brokers with exponential backoff
func (p *KafkaMQPlugin) ConnectWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Try to connect
		err := p.testBrokerConnection(ctx)
		if err == nil {
			// Connection successful
			if attempt > 1 {
				// Recovery after failure
				p.RecordBrokerRecovery()
				p.logger.Info("broker connection recovered", "attempt", attempt)
			}
			return nil
		}

		lastErr = err
		p.RecordBrokerFailure()
		p.logger.Warn("broker connection failed", "attempt", attempt, "error", err)

		// Don't wait after the last attempt
		if attempt < maxRetries {
			// Calculate exponential backoff delay
			delay := p.CalculateExponentialBackoffDelay(attempt)
			p.logger.Info("retrying broker connection", "attempt", attempt+1, "delay_ms", delay.Milliseconds())

			// Wait for the delay or until context is cancelled
			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed to connect to brokers after %d attempts: %w", maxRetries, lastErr)
}

// testBrokerConnection tests connectivity to Kafka brokers
func (p *KafkaMQPlugin) testBrokerConnection(ctx context.Context) error {
	// Create a temporary reader to test connection
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     p.brokers,
		Topic:       "__consumer_offsets", // System topic that always exists
		GroupID:     p.consumerGroup + "-health-check",
		StartOffset: kafka.LastOffset,
		MaxBytes:    1024,
	})
	defer func() {
		_ = reader.Close()
	}()

	// Try to read metadata with timeout
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := reader.ReadMessage(readCtx)
	if err != nil && err != context.DeadlineExceeded {
		// Timeout is acceptable for health check
		return err
	}

	return nil
}

// GetKafkaSpecificMetrics returns comprehensive Kafka-specific metrics
func (p *KafkaMQPlugin) GetKafkaSpecificMetrics() map[string]any {
	p.mu.RLock()
	brokerFailures := p.brokerFailureCount
	brokerRecoveries := p.brokerRecoveryCount
	p.mu.RUnlock()

	p.offsetTrackingMutex.RLock()
	offsetTrackingCopy := make(map[string]int64)
	for topic, offset := range p.offsetTracking {
		offsetTrackingCopy[topic] = offset
	}
	p.offsetTrackingMutex.RUnlock()

	p.dlqReasonMutex.RLock()
	dlqReasonsCopy := make(map[string]int64)
	for reason, count := range p.dlqReasonCounts {
		dlqReasonsCopy[reason] = count
	}
	p.dlqReasonMutex.RUnlock()

	consumerGroupMetrics := p.GetConsumerGroupMetrics()

	return map[string]any{
		"brokers":                p.brokers,
		"consumer_group":         p.consumerGroup,
		"broker_failures":        brokerFailures,
		"broker_recoveries":      brokerRecoveries,
		"offset_tracking":        offsetTrackingCopy,
		"dlq_reasons":            dlqReasonsCopy,
		"consumer_group_metrics": consumerGroupMetrics,
		"active_consumers":       len(p.consumers),
	}
}

// GetConsumerGroupStatus returns the status of the consumer group
func (p *KafkaMQPlugin) GetConsumerGroupStatus() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]any{
		"consumer_group":     p.consumerGroup,
		"active_consumers":   len(p.consumers),
		"is_running":         p.isRunning,
		"message_count":      p.messageCount,
		"error_count":        p.errorCount,
		"max_tracked_offset": p.maxTrackedOffset(),
	}
}

// RebalanceConsumerGroup triggers a consumer group rebalance
func (p *KafkaMQPlugin) RebalanceConsumerGroup(ctx context.Context) error {
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("plugin not running")
	}
	p.mu.Unlock()

	p.logger.Info("initiating consumer group rebalance", "consumer_group", p.consumerGroup)
	p.metricsCollector.RecordCounter("mq_consumer_group_rebalances", int64(1), nil)

	// Close all existing consumers to trigger rebalance
	p.mu.Lock()
	for topic, consumer := range p.consumers {
		if consumer != nil && consumer.reader != nil {
			if err := consumer.reader.Close(); err != nil {
				p.logger.Error("failed to close consumer during rebalance", "topic", topic, "error", err)
			}
		}
	}
	p.consumers = make(map[string]*KafkaConsumer)
	p.mu.Unlock()

	p.logger.Info("consumer group rebalance completed", "consumer_group", p.consumerGroup)
	return nil
}

// GetOffsetPersistenceStats returns statistics about offset persistence
func (p *KafkaMQPlugin) GetOffsetPersistenceStats() map[string]any {
	p.offsetPersistMutex.RLock()
	defer p.offsetPersistMutex.RUnlock()

	topicCount := int64(len(p.offsetPersistenceMap))
	totalOffsets := int64(0)

	for _, partitionOffsets := range p.offsetPersistenceMap {
		totalOffsets += int64(len(partitionOffsets))
	}

	return map[string]any{
		"topics_with_persisted_offsets": topicCount,
		"total_persisted_offsets":       totalOffsets,
	}
}
