# MQ Plugin Error Handling and Recovery Guide

## Overview

This guide explains the error handling and recovery mechanisms implemented in the ChainPulse MQ plugin system. These mechanisms ensure that transient failures don't cause system failures and that the system can gracefully degrade when permanent errors occur.

## Error Classification

The MQ error handler classifies errors into five categories:

### 1. Connection Errors
- **Examples**: Connection refused, connection reset, broken pipe, EOF
- **Behavior**: Retryable with exponential backoff
- **Recovery**: Automatic reconnection attempts

### 2. Timeout Errors
- **Examples**: Context deadline exceeded, I/O timeout
- **Behavior**: Retryable with exponential backoff
- **Recovery**: Automatic retry with increased timeout

### 3. Permanent Errors
- **Examples**: Invalid configuration, authentication failure
- **Behavior**: Not retryable
- **Recovery**: System enters degraded mode, operator alert required

### 4. Transient Errors
- **Examples**: Temporary network issues, temporary resource unavailability
- **Behavior**: Retryable with exponential backoff
- **Recovery**: Automatic retry

### 5. Unknown Errors
- **Behavior**: Treated as transient by default
- **Recovery**: Automatic retry

## Retry Strategy

### Exponential Backoff

The MQ error handler uses exponential backoff with jitter to calculate retry delays:

```
delay = min(baseDelay * (2 ^ (attempt - 1)), maxRetryDelay) + jitter
```

**Example delays** (with baseDelay = 100ms, maxRetryDelay = 30s):
- Attempt 1: ~100ms
- Attempt 2: ~200ms
- Attempt 3: ~400ms
- Attempt 4: ~800ms
- Attempt 5: ~1.6s
- ...
- Attempt 10+: ~30s (capped)

### Jitter

Jitter (±10% of delay) is added to prevent thundering herd problems when multiple clients retry simultaneously.

### Configuration

```go
handler := NewMQErrorHandler(
    logger,
    metricsCollector,
    maxRetries,        // Number of retry attempts (default: 3)
    baseRetryDelay,    // Base delay for exponential backoff (default: 1s)
)

// Optional configuration
handler.SetMaxRetries(5)
handler.SetMaxRetryDelay(30 * time.Second)
handler.SetTimeoutDuration(10 * time.Second)
```

## Error Handling Flow

```
┌─────────────────────┐
│ Operation Attempt   │
└──────────┬──────────┘
           │
           ▼
    ┌──────────────┐
    │ Success?     │
    └──┬───────┬───┘
       │ Yes   │ No
       │       ▼
       │   ┌──────────────┐
       │   │ Classify     │
       │   │ Error        │
       │   └──┬───────┬───┬───┬───┐
       │      │       │   │   │   │
       │      ▼       ▼   ▼   ▼   ▼
       │   Conn  Timeout Perm Trans Unknown
       │      │       │   │   │   │
       │      └───────┴───┘   │   │
       │          │           │   │
       │          ▼           ▼   ▼
       │      Retryable   Not Retryable
       │          │           │
       │          ▼           ▼
       │   ┌──────────────┐ ┌──────────────┐
       │   │ Retry Count  │ │ Degrade      │
       │   │ < Max?       │ │ System       │
       │   └──┬───────┬───┘ └──────────────┘
       │      │ Yes   │ No
       │      │       ▼
       │      │   ┌──────────────┐
       │      │   │ Send to DLQ  │
       │      │   └──────────────┘
       │      │
       │      ▼
       │   ┌──────────────┐
       │   │ Wait Delay   │
       │   └──────┬───────┘
       │          │
       │          ▼
       │   ┌──────────────┐
       │   │ Retry        │
       │   └──────┬───────┘
       │          │
       └──────────┴──────────┐
                             │
                             ▼
                        ┌─────────┐
                        │ Complete│
                        └─────────┘
```

## Degraded Mode

When permanent errors occur or retries are exhausted, the system enters **degraded mode**:

### Characteristics
- System continues operating but with reduced functionality
- Errors are logged and metrics are recorded
- Operators are alerted
- New operations may be rejected or queued

### Recovery from Degraded Mode
- Automatic recovery when operations succeed
- Manual recovery via operator intervention
- Health checks monitor recovery status

## Usage Examples

### Basic Error Handling

```go
handler := NewMQErrorHandler(logger, metricsCollector, 3, 100*time.Millisecond)

// Classify an error
errorType := handler.ClassifyError(err)

// Handle an error
shouldRetry, _ := handler.HandleError(ctx, err, "publish_message")
if shouldRetry {
    // Retry the operation
}
```

### Retry with Backoff

```go
handler := NewMQErrorHandler(logger, metricsCollector, 3, 100*time.Millisecond)

operation := func() error {
    return publishMessage(ctx, message)
}

err := handler.RetryWithBackoff(ctx, operation, "publish_message")
if err != nil {
    // Operation failed after all retries
    // Send to DLQ or handle error
}
```

### Custom Error Classification

```go
handler := NewMQErrorHandler(logger, metricsCollector, 3, 100*time.Millisecond)

// Register permanent errors
handler.RegisterPermanentError("invalid configuration")
handler.RegisterPermanentError("authentication failed")

// Register transient errors
handler.RegisterTransientError("temporary network issue")
handler.RegisterTransientError("broker temporarily unavailable")

// Now these errors will be classified correctly
errorType := handler.ClassifyError(errors.New("invalid configuration"))
// errorType == ErrorTypePermanent
```

### Monitoring Recovery

```go
handler := NewMQErrorHandler(logger, metricsCollector, 3, 100*time.Millisecond)

// Get recovery statistics
stats := handler.GetRecoveryStats()
fmt.Printf("Recovery attempts: %d\n", stats["recovery_attempts"])
fmt.Printf("Successful recoveries: %d\n", stats["successful_recoveries"])
fmt.Printf("Degraded mode: %v\n", stats["degraded_mode"])

// Check if in degraded mode
if handler.IsInDegradedMode() {
    // Take action to recover
}
```

## Integration with MQ Plugin

The error handler is integrated into the Kafka MQ plugin:

```go
// In KafkaMQPlugin
type KafkaMQPlugin struct {
    // ... other fields ...
    errorHandler *MQErrorHandler
}

// Initialize error handler
func (p *KafkaMQPlugin) Initialize() error {
    p.errorHandler = NewMQErrorHandler(
        p.logger,
        p.metricsCollector,
        p.maxRetries,
        p.retryDelay,
    )
    // ... rest of initialization ...
}

// Use error handler in operations
func (p *KafkaMQPlugin) PublishMessage(ctx context.Context, message MessageQueueMessage) error {
    operation := func() error {
        return p.publishToKafka(ctx, message)
    }
    
    return p.errorHandler.RetryWithBackoff(ctx, operation, "publish_message")
}
```

## Metrics and Monitoring

The error handler records comprehensive metrics:

### Counters
- `mq_errors`: Total errors encountered
- `mq_retry_attempts`: Total retry attempts
- `mq_successful_recovery`: Successful recoveries
- `mq_retries_exhausted`: Operations that failed after all retries
- `mq_permanent_errors`: Permanent errors encountered
- `mq_degraded_mode_entered`: Times system entered degraded mode
- `mq_degraded_mode_exited`: Times system exited degraded mode

### Gauges
- `mq_consecutive_errors`: Current consecutive error count
- `mq_degraded_mode`: Whether system is in degraded mode

### Example Monitoring

```go
// Get recovery statistics
stats := handler.GetRecoveryStats()

// Log to monitoring system
monitoring.RecordMetric("mq_recovery_attempts", stats["recovery_attempts"])
monitoring.RecordMetric("mq_successful_recoveries", stats["successful_recoveries"])
monitoring.RecordMetric("mq_consecutive_errors", stats["consecutive_errors"])
monitoring.RecordMetric("mq_degraded_mode", stats["degraded_mode"])
```

## Best Practices

### 1. Configure Appropriate Timeouts
```go
handler.SetTimeoutDuration(5 * time.Second)  // For fast operations
handler.SetTimeoutDuration(30 * time.Second) // For slow operations
```

### 2. Register Custom Error Codes
```go
// Register errors specific to your system
handler.RegisterPermanentError("database_corrupted")
handler.RegisterTransientError("cache_miss")
```

### 3. Monitor Degraded Mode
```go
if handler.IsInDegradedMode() {
    // Alert operators
    alerting.SendAlert("MQ system in degraded mode")
    
    // Reduce load
    limiter.SetMaxConcurrency(10)
}
```

### 4. Implement Graceful Degradation
```go
// When in degraded mode, queue operations instead of failing
if handler.IsInDegradedMode() {
    queue.Enqueue(operation)
} else {
    err := handler.RetryWithBackoff(ctx, operation, "operation_name")
}
```

### 5. Log Errors Appropriately
```go
// The error handler logs automatically, but you can add context
if err != nil {
    logger.Error("operation failed",
        "operation", "publish_message",
        "message_id", message.ID,
        "error", err,
        "degraded_mode", handler.IsInDegradedMode(),
    )
}
```

## Testing Error Handling

### Unit Tests

```go
func TestErrorHandling(t *testing.T) {
    handler := NewMQErrorHandler(logger, metricsCollector, 3, 10*time.Millisecond)
    
    // Test successful operation
    attempts := 0
    operation := func() error {
        attempts++
        if attempts < 3 {
            return errors.New("connection refused")
        }
        return nil
    }
    
    err := handler.RetryWithBackoff(context.Background(), operation, "test_op")
    assert.NoError(t, err)
    assert.Equal(t, 3, attempts)
}
```

### Integration Tests

```go
func TestErrorHandlingWithRealBroker(t *testing.T) {
    // Test with real Kafka broker
    plugin := NewKafkaMQPlugin(...)
    
    // Simulate broker failure
    stopBroker()
    
    // Verify error handling and recovery
    err := plugin.PublishMessage(ctx, message)
    assert.Error(t, err)
    
    // Restart broker
    startBroker()
    
    // Verify recovery
    err = plugin.PublishMessage(ctx, message)
    assert.NoError(t, err)
}
```

## Troubleshooting

### System Stuck in Degraded Mode
1. Check error logs for permanent errors
2. Verify broker connectivity
3. Check configuration validity
4. Manually trigger recovery: `handler.SetDegradedMode(false)`

### Excessive Retries
1. Increase `maxRetryDelay` to reduce retry frequency
2. Increase `maxRetries` if transient errors are common
3. Implement circuit breaker pattern for cascading failures

### High Error Rate
1. Check broker health
2. Check network connectivity
3. Review error metrics to identify error patterns
4. Implement rate limiting to reduce load

## References

- [Exponential Backoff And Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Graceful Degradation](https://en.wikipedia.org/wiki/Graceful_degradation)
