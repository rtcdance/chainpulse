package core

import (
	"context"
	"fmt"
	"time"
)

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
	delayMultiplier := boundedShiftMultiplier(currentRetryCount)
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
