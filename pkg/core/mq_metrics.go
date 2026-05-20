package core

import (
	"fmt"
	"math/bits"
	"time"
)

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

// RecordMetric records a metric
func (p *BaseMQPlugin) RecordMetric(name string, value int64, tags map[string]string) {
	p.metricsCollector.RecordCounter(name, value, tags)
}

// LogInfo logs an info message
func (p *BaseMQPlugin) LogInfo(message string, fields ...any) {
	p.logger.Info(message, fields...)
}

// LogError logs an error message
func (p *BaseMQPlugin) LogError(message string, fields ...any) {
	p.logger.Error(message, fields...)
}

// LogWarn logs a warning message
func (p *BaseMQPlugin) LogWarn(message string, fields ...any) {
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

// CalculateExponentialBackoffDelay calculates exponential backoff delay
// Formula: baseDelay * (2 ^ (retryCount - 1))
// For retry 1: delay = baseDelay * 2^0 = baseDelay
// For retry 2: delay = baseDelay * 2^1 = baseDelay * 2
// For retry 3: delay = baseDelay * 2^2 = baseDelay * 4
func (p *BaseMQPlugin) CalculateExponentialBackoffDelay(retryCount int) time.Duration {
	if retryCount <= 0 {
		return p.retryDelay
	}
	delayMultiplier := boundedShiftMultiplier(retryCount)
	return p.retryDelay * time.Duration(delayMultiplier)
}

func boundedShiftMultiplier(count int) int {
	shift := count - 1
	maxShift := bits.UintSize - 2
	if shift < 0 {
		shift = 0
	} else if shift > maxShift {
		shift = maxShift
	}
	return 1 << shift
}
