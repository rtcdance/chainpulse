package core

import (
	"context"
	"time"
)

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

// SetBatchSize sets the batch size for message processing
func (p *BaseMQPlugin) SetBatchSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.batchSize = size
	p.logger.Info("batch size set", "size", size)
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
