# Error Handling Guide
## Microservices Data Layer - Phase 4

**Date:** January 12, 2026  
**Phase:** 4 - Error Handling and Resilience  
**Status:** Complete  
**Audience:** Developers, DevOps, SREs

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Error Classification](#error-classification)
3. [Retry Logic](#retry-logic)
4. [Circuit Breaker Pattern](#circuit-breaker-pattern)
5. [Data Consistency](#data-consistency)
6. [Graceful Degradation](#graceful-degradation)
7. [Error Recovery](#error-recovery)
8. [Monitoring and Metrics](#monitoring-and-metrics)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)

---

## Overview

The Event Processor data layer implements comprehensive error handling and resilience mechanisms to ensure reliable operation under various failure scenarios. The system automatically classifies errors, retries transient failures, prevents cascading failures, and recovers from errors.

### Key Components

- **Error Classifier** - Classifies errors into 4 types
- **Retry Policy** - Implements exponential backoff with jitter
- **Circuit Breaker** - Prevents cascading failures
- **Consistency Checker** - Verifies data integrity
- **Degradation Handler** - Continues with reduced functionality
- **Recovery Handler** - Automatic recovery procedures
- **Error Metrics** - Comprehensive observability

---

## Error Classification

### Overview

The error classifier analyzes errors and categorizes them into 4 types to determine the appropriate recovery strategy.

### Error Types

#### 1. Transient Errors
**Definition:** Temporary failures that may succeed on retry

**Examples:**
- Connection timeouts
- Temporary network issues
- Temporary resource unavailability
- I/O timeouts
- Connection refused (temporary)

**Recovery Strategy:**
- Retry with exponential backoff
- Jitter to prevent thundering herd
- Max attempts: 3 (configurable)
- Max duration: 30 seconds (configurable)

**Code Example:**
```go
classifier := query.NewErrorClassifier()
err := someOperation()

if classifier.IsTransient(err) {
    // Retry with backoff
    retryPolicy.Execute(ctx, operation)
}
```

#### 2. Permanent Errors
**Definition:** Failures that won't succeed on retry

**Examples:**
- Invalid input
- Constraint violations
- Duplicate key errors
- Authentication failures
- Authorization failures
- Schema mismatches

**Recovery Strategy:**
- Log error with context
- Alert if critical
- Fail fast
- No retry

**Code Example:**
```go
if classifier.IsPermanent(err) {
    logger.Error("Permanent error", map[string]interface{}{
        "error": err.Error(),
        "context": "operation_name",
    })
    return err
}
```

#### 3. Critical Errors
**Definition:** System-level failures requiring immediate attention

**Examples:**
- Out of memory
- Disk full
- Unrecoverable corruption
- System panic
- Resource exhaustion

**Recovery Strategy:**
- Alert immediately
- Log with full context
- Trigger incident response
- Graceful shutdown if necessary

**Code Example:**
```go
if classifier.IsCritical(err) {
    logger.Critical("Critical error", map[string]interface{}{
        "error": err.Error(),
        "action": "alert_ops",
    })
    metrics.RecordCriticalError(err)
}
```

#### 4. Unknown Errors
**Definition:** Errors that don't fit other categories

**Examples:**
- Unexpected error types
- Errors without clear classification
- Third-party library errors

**Recovery Strategy:**
- Log with full context
- Treat as permanent
- Monitor for patterns
- Investigate and classify

---

## Retry Logic

### Overview

The retry policy implements exponential backoff with jitter to safely retry transient errors without overwhelming the system.

### Configuration

```go
type RetryConfig struct {
    MaxAttempts       int           // Default: 3
    InitialBackoff    time.Duration // Default: 100ms
    MaxBackoff        time.Duration // Default: 10s
    BackoffMultiplier float64       // Default: 2.0
    JitterFraction    float64       // Default: 0.1
}
```

### Backoff Calculation

**Formula:** `backoff = min(initialBackoff * (multiplier ^ attempt), maxBackoff) * (1 ± jitter)`

**Example with defaults:**
- Attempt 1: 100ms ± 10ms
- Attempt 2: 200ms ± 20ms
- Attempt 3: 400ms ± 40ms
- Attempt 4: 800ms ± 80ms (capped at 10s)

### Usage

```go
retryPolicy := query.NewRetryPolicy(query.RetryConfig{
    MaxAttempts:       3,
    InitialBackoff:    100 * time.Millisecond,
    MaxBackoff:        10 * time.Second,
    BackoffMultiplier: 2.0,
    JitterFraction:    0.1,
})

result, err := retryPolicy.Execute(ctx, func() (interface{}, error) {
    return eventStore.GetEvent(ctx, eventID)
})
```

### Best Practices

1. **Use appropriate backoff values** - Too aggressive can overwhelm, too conservative wastes time
2. **Always use jitter** - Prevents thundering herd problem
3. **Set reasonable max attempts** - Usually 3-5 is sufficient
4. **Monitor retry rates** - High retry rates indicate systemic issues
5. **Log retry attempts** - Helps with debugging and monitoring

---

## Circuit Breaker Pattern

### Overview

The circuit breaker prevents cascading failures by stopping requests to failing services and allowing them time to recover.

### States

#### 1. Closed (Normal Operation)
- Requests pass through normally
- Failures are counted
- Transition to Open if threshold exceeded

#### 2. Open (Failure Detected)
- Requests are rejected immediately
- No calls to failing service
- Waits for timeout before transitioning

#### 3. Half-Open (Testing Recovery)
- Limited requests allowed
- Tests if service has recovered
- Transitions to Closed on success, Open on failure

### Configuration

```go
type CircuitBreakerConfig struct {
    FailureThreshold int           // Default: 5
    SuccessThreshold int           // Default: 2
    Timeout          time.Duration // Default: 30s
}
```

### State Transitions

```
Closed
  ↓ (failures >= threshold)
Open
  ↓ (timeout elapsed)
Half-Open
  ├─ (successes >= threshold) → Closed
  └─ (failure) → Open
```

### Usage

```go
breaker := query.NewCircuitBreaker(query.CircuitBreakerConfig{
    FailureThreshold: 5,
    SuccessThreshold: 2,
    Timeout:          30 * time.Second,
})

result, err := breaker.Execute(ctx, func() (interface{}, error) {
    return eventStore.GetEvent(ctx, eventID)
})

if err == query.ErrCircuitOpen {
    // Service is unavailable, use fallback
    return getCachedResult(eventID)
}
```

### Best Practices

1. **Set appropriate thresholds** - Too low causes unnecessary failures, too high delays detection
2. **Monitor state transitions** - Indicates service health issues
3. **Use with retry logic** - Retry for transient errors, circuit breaker for cascading failures
4. **Implement fallbacks** - Have alternative strategies when circuit is open
5. **Alert on state changes** - Helps with incident response

---

## Data Consistency

### Overview

The consistency checker verifies data integrity between MongoDB and PostgreSQL to detect and prevent data corruption.

### Consistency Checks

#### 1. Event Count Consistency
- Verify event count matches metadata count
- Detect missing or orphaned records
- Trigger recovery if inconsistent

#### 2. Metadata Completeness
- Verify all events have metadata
- Detect missing metadata records
- Trigger metadata rebuild if needed

#### 3. Orphaned Record Detection
- Detect metadata without corresponding events
- Detect events without metadata
- Quarantine orphaned records

#### 4. Data Integrity Validation
- Verify event data is not corrupted
- Verify metadata is not corrupted
- Detect partial writes

### Usage

```go
checker := query.NewConsistencyChecker(eventStore, metadataStore)

result, err := checker.Check(ctx)
if err != nil {
    logger.Error("Consistency check failed", map[string]interface{}{
        "error": err.Error(),
        "inconsistencies": result.Inconsistencies,
    })
}

if result.HasInconsistencies {
    // Trigger recovery
    err = checker.Repair(ctx)
}
```

### Metrics

- `consistency_checks_total` - Total consistency checks performed
- `consistency_errors_total` - Total inconsistencies found
- `consistency_repair_total` - Total repairs performed
- `consistency_repair_failures_total` - Total repair failures

### Best Practices

1. **Run checks periodically** - Detect issues early
2. **Monitor check results** - Alert on inconsistencies
3. **Implement repair procedures** - Automatically fix issues when possible
4. **Log all inconsistencies** - Helps with debugging
5. **Quarantine corrupted data** - Prevent further corruption

---

## Graceful Degradation

### Overview

Graceful degradation allows the system to continue operating with reduced functionality when components fail.

### Degradation Modes

#### 1. Normal Mode
- All stores available
- Full functionality
- Optimal performance

#### 2. MongoDB Unavailable
- Use PostgreSQL + Cache
- Reduced performance
- Full functionality

#### 3. PostgreSQL Unavailable
- Use MongoDB only
- Reduced performance
- Full functionality

#### 4. Both Unavailable
- Use Cache only
- Limited functionality
- Read-only operations

#### 5. Cache Unavailable
- Use stores only
- Reduced performance
- Full functionality

#### 6. Read-Only Mode
- All stores unavailable
- No write operations
- Read-only access to cache

### Fallback Strategies

#### 1. Cache-Only Strategy
- Retrieve from cache
- No database access
- Fast but limited data

#### 2. MongoDB-Only Strategy
- Retrieve from MongoDB
- No PostgreSQL access
- Full data but slower

#### 3. PostgreSQL-Only Strategy
- Retrieve from PostgreSQL
- No MongoDB access
- Full data but slower

#### 4. Hybrid Strategy
- Use available stores
- Combine results
- Best available performance

#### 5. Read-Only Strategy
- No operations allowed
- Return error for writes
- Prevent data corruption

### Usage

```go
handler := query.NewDegradationHandler(mongoStore, postgresStore, cacheStore)

// Automatically detects degradation mode
result, err := handler.GetEvent(ctx, eventID)

// Check degradation mode
mode := handler.GetDegradationMode(ctx)
if mode != query.ModeNormal {
    logger.Warn("System degraded", map[string]interface{}{
        "mode": mode.String(),
    })
}
```

### Best Practices

1. **Monitor degradation events** - Alert when system degrades
2. **Test fallback strategies** - Ensure they work correctly
3. **Implement graceful degradation** - Don't fail completely
4. **Log degradation transitions** - Helps with debugging
5. **Provide user feedback** - Inform users of reduced functionality

---

## Error Recovery

### Overview

The recovery handler implements automatic recovery procedures to restore system functionality after failures.

### Recovery Procedures

#### 1. Connection Recovery
- Automatic reconnection on connection loss
- Exponential backoff (100ms → 10s)
- Configurable retry attempts (default: 5)
- Thread-safe operations

#### 2. State Recovery
- Recover system state after restart
- Rebuild metadata if needed
- Verify data consistency
- Restore normal operation

#### 3. Data Synchronization
- Sync data between stores
- Rebuild missing metadata
- Repair corrupted data
- Verify consistency

### Recovery States

- **Healthy** - System operating normally
- **Recovering** - Recovery in progress
- **Failed** - Recovery failed, manual intervention needed

### Usage

```go
handler := query.NewRecoveryHandler(eventStore, metadataStore)

// Automatic recovery on connection loss
err := handler.RecoverConnection(ctx)
if err != nil {
    logger.Error("Connection recovery failed", map[string]interface{}{
        "error": err.Error(),
    })
}

// Check recovery state
state := handler.GetRecoveryState()
if state == query.StateFailed {
    // Manual intervention needed
    logger.Critical("Recovery failed, manual intervention required")
}
```

### Metrics

- `recovery_attempts_total` - Total recovery attempts
- `recovery_successes_total` - Successful recoveries
- `recovery_failures_total` - Failed recoveries
- `recovery_duration_seconds` - Recovery duration

### Best Practices

1. **Monitor recovery operations** - Alert on failures
2. **Implement recovery procedures** - Don't rely on manual intervention
3. **Test recovery procedures** - Ensure they work correctly
4. **Log recovery operations** - Helps with debugging
5. **Implement manual recovery** - For cases automatic recovery fails

---

## Monitoring and Metrics

### Overview

Comprehensive metrics collection provides visibility into error handling and resilience operations.

### Key Metrics

#### Error Metrics
- `event_store_errors_total` - Total errors by type
- `event_store_error_rate` - Error rate per second
- `event_store_error_duration_seconds` - Error recovery duration

#### Retry Metrics
- `event_store_retry_attempts_total` - Total retry attempts
- `event_store_retry_successes_total` - Successful retries
- `event_store_retry_failures_total` - Failed retries
- `event_store_retry_duration_seconds` - Retry duration

#### Circuit Breaker Metrics
- `event_store_circuit_breaker_state` - Current state (0=closed, 1=open, 2=half-open)
- `event_store_circuit_breaker_transitions_total` - State transitions
- `event_store_circuit_breaker_open_duration_seconds` - Time in open state

#### Consistency Metrics
- `event_store_consistency_checks_total` - Total consistency checks
- `event_store_consistency_errors_total` - Inconsistencies found
- `event_store_consistency_repair_total` - Repairs performed

#### Degradation Metrics
- `event_store_degradation_events_total` - Degradation events
- `event_store_degradation_mode` - Current degradation mode
- `event_store_degradation_duration_seconds` - Degradation duration

#### Recovery Metrics
- `event_store_recovery_attempts_total` - Recovery attempts
- `event_store_recovery_successes_total` - Successful recoveries
- `event_store_recovery_failures_total` - Failed recoveries

### Monitoring Dashboard

Key metrics to monitor:
1. Error rate and types
2. Retry success rate
3. Circuit breaker state
4. Data consistency status
5. Degradation events
6. Recovery success rate

### Alerting Rules

**Critical Alerts:**
- Circuit breaker open for > 5 minutes
- Data consistency errors detected
- Recovery failures
- Critical errors

**Warning Alerts:**
- High error rate (> 1% of operations)
- High retry rate (> 10% of operations)
- Degradation events
- Recovery in progress

---

## Troubleshooting

### Common Issues

#### 1. High Error Rate
**Symptoms:** Error rate > 1% of operations

**Diagnosis:**
- Check error types and sources
- Review error logs
- Check system resources
- Verify database connectivity

**Resolution:**
- Investigate root cause
- Scale resources if needed
- Check database performance
- Review application logs

#### 2. Circuit Breaker Open
**Symptoms:** Requests rejected, circuit breaker in open state

**Diagnosis:**
- Check failure threshold
- Review error logs
- Check service health
- Verify network connectivity

**Resolution:**
- Wait for timeout (default: 30s)
- Fix underlying issue
- Manually reset if needed
- Monitor recovery

#### 3. Data Inconsistency
**Symptoms:** Consistency check failures, orphaned records

**Diagnosis:**
- Run consistency check
- Review error logs
- Check for partial writes
- Verify data integrity

**Resolution:**
- Run repair procedure
- Verify data integrity
- Monitor for recurrence
- Investigate root cause

#### 4. Recovery Failures
**Symptoms:** Recovery attempts fail, system remains degraded

**Diagnosis:**
- Check recovery logs
- Verify connectivity
- Check resource availability
- Review error messages

**Resolution:**
- Manual intervention required
- Check system resources
- Verify connectivity
- Contact support if needed

### Debug Commands

```bash
# Check error metrics
curl http://localhost:9090/metrics | grep event_store_errors

# Check circuit breaker state
curl http://localhost:9090/metrics | grep circuit_breaker_state

# Check consistency status
curl http://localhost:9090/metrics | grep consistency_checks

# Check degradation mode
curl http://localhost:9090/metrics | grep degradation_mode

# Check recovery status
curl http://localhost:9090/metrics | grep recovery_attempts
```

### Log Analysis

**Error Classification:**
```
grep "error_type" application.log | sort | uniq -c
```

**Retry Patterns:**
```
grep "retry_attempt" application.log | tail -100
```

**Circuit Breaker Transitions:**
```
grep "circuit_breaker_state" application.log
```

**Recovery Operations:**
```
grep "recovery" application.log
```

---

## Best Practices

### 1. Error Handling
- Always classify errors before deciding on recovery strategy
- Log errors with full context
- Use appropriate error types
- Implement proper error propagation

### 2. Retry Logic
- Use exponential backoff with jitter
- Set reasonable max attempts
- Monitor retry rates
- Implement circuit breaker for cascading failures

### 3. Circuit Breaker
- Set appropriate failure thresholds
- Monitor state transitions
- Implement fallback strategies
- Alert on state changes

### 4. Data Consistency
- Run consistency checks periodically
- Implement repair procedures
- Monitor check results
- Alert on inconsistencies

### 5. Graceful Degradation
- Implement fallback strategies
- Test degradation scenarios
- Monitor degradation events
- Provide user feedback

### 6. Error Recovery
- Implement automatic recovery
- Test recovery procedures
- Monitor recovery operations
- Implement manual recovery for edge cases

### 7. Monitoring
- Collect comprehensive metrics
- Monitor key indicators
- Set up alerting rules
- Review metrics regularly

### 8. Testing
- Test error scenarios
- Test recovery procedures
- Test degradation modes
- Test concurrent failures

---

## Summary

The Event Processor data layer implements comprehensive error handling and resilience mechanisms to ensure reliable operation under various failure scenarios. By following this guide and best practices, you can build robust systems that gracefully handle failures and recover automatically.

**Key Takeaways:**
1. Classify errors to determine recovery strategy
2. Use exponential backoff with jitter for retries
3. Implement circuit breaker to prevent cascading failures
4. Verify data consistency to detect corruption
5. Implement graceful degradation for partial failures
6. Implement automatic recovery procedures
7. Monitor all operations with comprehensive metrics
8. Test error scenarios and recovery procedures

---

**For more information:**
- See `RESILIENCE_PATTERNS_GUIDE.md` for detailed resilience patterns
- See `MICROSERVICES_DATA_LAYER_PHASE_4_QUICK_REFERENCE.md` for quick reference
- See implementation files in `pkg/services/query/` for code examples
