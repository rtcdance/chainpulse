# Resilience Patterns Guide
## Microservices Data Layer - Phase 4

**Date:** January 12, 2026  
**Phase:** 4 - Error Handling and Resilience  
**Status:** Complete  
**Audience:** Architects, Senior Developers, DevOps

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Retry Pattern](#retry-pattern)
3. [Circuit Breaker Pattern](#circuit-breaker-pattern)
4. [Bulkhead Pattern](#bulkhead-pattern)
5. [Fallback Pattern](#fallback-pattern)
6. [Recovery Pattern](#recovery-pattern)
7. [Combining Patterns](#combining-patterns)
8. [Anti-Patterns](#anti-patterns)
9. [Performance Considerations](#performance-considerations)
10. [Real-World Examples](#real-world-examples)

---

## Overview

Resilience patterns are proven solutions for building systems that gracefully handle failures and recover automatically. This guide covers the key patterns implemented in the Event Processor data layer.

### Pattern Categories

- **Transient Failure Handling** - Retry, Exponential Backoff
- **Cascading Failure Prevention** - Circuit Breaker, Bulkhead
- **Graceful Degradation** - Fallback, Degradation
- **Automatic Recovery** - Recovery, Health Checks

---

## Retry Pattern

### Overview

The retry pattern automatically retries failed operations to handle transient failures.

### When to Use

- Transient failures (timeouts, temporary unavailability)
- Network issues
- Temporary resource exhaustion
- Temporary service degradation

### When NOT to Use

- Permanent failures (invalid input, authentication)
- Operations with side effects (without idempotency)
- Critical operations (without careful consideration)
- Operations that might cause cascading failures

### Implementation

```go
// Basic retry
for attempt := 0; attempt < maxAttempts; attempt++ {
    result, err := operation()
    if err == nil {
        return result, nil
    }
    
    if !isTransient(err) {
        return nil, err
    }
    
    if attempt < maxAttempts-1 {
        time.Sleep(calculateBackoff(attempt))
    }
}
```

### Best Practices

1. **Classify errors** - Only retry transient errors
2. **Use exponential backoff** - Avoid overwhelming the system
3. **Add jitter** - Prevent thundering herd
4. **Set max attempts** - Usually 3-5 is sufficient
5. **Set max duration** - Prevent indefinite retries
6. **Log attempts** - Help with debugging
7. **Monitor retry rates** - Indicate systemic issues

### Example

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

---

## Circuit Breaker Pattern

### Overview

The circuit breaker pattern prevents cascading failures by stopping requests to failing services.

### States

- **Closed** - Normal operation, requests pass through
- **Open** - Failures detected, requests rejected
- **Half-Open** - Testing recovery, limited requests allowed

### When to Use

- Preventing cascading failures
- Protecting against resource exhaustion
- Detecting service degradation
- Implementing fast-fail behavior

### When NOT to Use

- Single-threaded operations
- Operations without fallback
- Operations that must always succeed
- Low-traffic services

### Implementation

```go
type CircuitBreaker struct {
    state              State
    failureCount       int
    successCount       int
    lastFailureTime    time.Time
    failureThreshold   int
    successThreshold   int
    timeout            time.Duration
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    switch cb.state {
    case Closed:
        err := fn()
        if err != nil {
            cb.recordFailure()
            if cb.failureCount >= cb.failureThreshold {
                cb.state = Open
            }
        } else {
            cb.recordSuccess()
        }
        return err
    
    case Open:
        if time.Since(cb.lastFailureTime) > cb.timeout {
            cb.state = HalfOpen
            cb.successCount = 0
            return cb.Execute(fn)
        }
        return ErrCircuitOpen
    
    case HalfOpen:
        err := fn()
        if err != nil {
            cb.state = Open
            cb.lastFailureTime = time.Now()
        } else {
            cb.successCount++
            if cb.successCount >= cb.successThreshold {
                cb.state = Closed
                cb.failureCount = 0
            }
        }
        return err
    }
}
```

### Best Practices

1. **Set appropriate thresholds** - Balance between detection and false positives
2. **Monitor state transitions** - Indicate service health issues
3. **Implement fallbacks** - Have alternative strategies
4. **Use with retry logic** - Retry for transient errors, circuit breaker for cascading
5. **Alert on state changes** - Help with incident response
6. **Test state transitions** - Ensure correct behavior

### Example

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
    // Use fallback strategy
    return getCachedResult(eventID)
}
```

---

## Bulkhead Pattern

### Overview

The bulkhead pattern isolates failures by partitioning resources into independent groups.

### Concept

Like compartments in a ship, if one compartment floods, others remain intact. Similarly, if one service fails, others continue operating.

### Implementation Strategies

#### 1. Thread Pool Isolation
```go
// Separate thread pools for different operations
mongoPool := NewThreadPool(10)
postgresPool := NewThreadPool(10)
cachePool := NewThreadPool(20)

// Operations use their respective pools
mongoPool.Execute(mongoOperation)
postgresPool.Execute(postgresOperation)
cachePool.Execute(cacheOperation)
```

#### 2. Resource Isolation
```go
// Separate resources for different components
mongoConnections := NewConnectionPool(10)
postgresConnections := NewConnectionPool(10)
cacheConnections := NewConnectionPool(20)
```

#### 3. Timeout Isolation
```go
// Different timeouts for different operations
mongoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

postgresCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()

cacheCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
defer cancel()
```

### When to Use

- Multi-component systems
- Resource-constrained environments
- Services with different SLAs
- Preventing resource exhaustion

### Best Practices

1. **Isolate resources** - Prevent resource exhaustion
2. **Set appropriate limits** - Balance between isolation and efficiency
3. **Monitor resource usage** - Detect bottlenecks
4. **Implement fallbacks** - Have alternative strategies
5. **Test under load** - Ensure isolation works correctly

---

## Fallback Pattern

### Overview

The fallback pattern provides alternative strategies when primary operations fail.

### Fallback Strategies

#### 1. Cache Fallback
```go
result, err := primaryStore.Get(ctx, key)
if err != nil {
    // Fall back to cache
    result, err = cache.Get(ctx, key)
}
return result, err
```

#### 2. Default Value Fallback
```go
result, err := operation()
if err != nil {
    // Return default value
    return defaultValue, nil
}
return result, nil
```

#### 3. Alternative Service Fallback
```go
result, err := primaryService.Get(ctx, key)
if err != nil {
    // Fall back to alternative service
    result, err = alternativeService.Get(ctx, key)
}
return result, err
```

#### 4. Degraded Mode Fallback
```go
result, err := operation()
if err != nil {
    // Return degraded result
    return getDegradedResult(ctx, key), nil
}
return result, nil
```

### When to Use

- Multiple data sources available
- Acceptable to return stale data
- Acceptable to return partial results
- Acceptable to return default values

### When NOT to Use

- No alternative available
- Stale data unacceptable
- Partial results unacceptable
- Default values unacceptable

### Best Practices

1. **Prioritize fallbacks** - Use best available option
2. **Monitor fallback usage** - Indicate primary issues
3. **Test fallbacks** - Ensure they work correctly
4. **Document fallback behavior** - Inform users
5. **Implement graceful degradation** - Maintain partial functionality

---

## Recovery Pattern

### Overview

The recovery pattern implements automatic recovery procedures to restore system functionality.

### Recovery Procedures

#### 1. Connection Recovery
```go
func (h *RecoveryHandler) RecoverConnection(ctx context.Context) error {
    for attempt := 0; attempt < h.maxAttempts; attempt++ {
        err := h.reconnect(ctx)
        if err == nil {
            return nil
        }
        
        backoff := h.calculateBackoff(attempt)
        select {
        case <-time.After(backoff):
            continue
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return ErrRecoveryFailed
}
```

#### 2. State Recovery
```go
func (h *RecoveryHandler) RecoverState(ctx context.Context) error {
    // Rebuild system state
    state, err := h.rebuildState(ctx)
    if err != nil {
        return err
    }
    
    // Verify consistency
    err = h.verifyConsistency(ctx, state)
    if err != nil {
        return err
    }
    
    // Restore normal operation
    return h.restoreNormalOperation(ctx, state)
}
```

#### 3. Data Synchronization
```go
func (h *RecoveryHandler) SyncData(ctx context.Context) error {
    // Get data from primary source
    primaryData, err := h.getPrimaryData(ctx)
    if err != nil {
        return err
    }
    
    // Sync to secondary sources
    for _, secondary := range h.secondarySources {
        err = secondary.Sync(ctx, primaryData)
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### Recovery States

- **Healthy** - System operating normally
- **Recovering** - Recovery in progress
- **Failed** - Recovery failed, manual intervention needed

### When to Use

- Automatic recovery from transient failures
- Connection loss recovery
- State recovery after restart
- Data synchronization

### Best Practices

1. **Implement automatic recovery** - Don't rely on manual intervention
2. **Test recovery procedures** - Ensure they work correctly
3. **Monitor recovery operations** - Alert on failures
4. **Implement manual recovery** - For cases automatic recovery fails
5. **Log recovery operations** - Help with debugging

---

## Combining Patterns

### Overview

Resilience patterns are most effective when combined strategically.

### Pattern Combinations

#### 1. Retry + Circuit Breaker
```
Operation
  ↓
Retry (for transient errors)
  ├─ Success → Return
  └─ Failure → Circuit Breaker
      ├─ Closed → Retry again
      ├─ Open → Fallback
      └─ Half-Open → Test recovery
```

**Use Case:** Handling transient failures while preventing cascading failures

#### 2. Circuit Breaker + Fallback
```
Operation
  ↓
Circuit Breaker
  ├─ Closed → Execute
  ├─ Open → Fallback
  └─ Half-Open → Test recovery
```

**Use Case:** Graceful degradation when service fails

#### 3. Bulkhead + Circuit Breaker
```
Operation
  ↓
Bulkhead (isolate resources)
  ↓
Circuit Breaker (prevent cascading)
  ├─ Closed → Execute
  ├─ Open → Fallback
  └─ Half-Open → Test recovery
```

**Use Case:** Preventing resource exhaustion and cascading failures

#### 4. Retry + Circuit Breaker + Fallback + Recovery
```
Operation
  ↓
Retry (for transient errors)
  ├─ Success → Return
  └─ Failure → Circuit Breaker
      ├─ Closed → Retry again
      ├─ Open → Fallback
      │   ├─ Success → Return
      │   └─ Failure → Recovery
      └─ Half-Open → Test recovery
```

**Use Case:** Comprehensive resilience for critical operations

### Implementation Example

```go
// Comprehensive resilience pattern
func (s *Service) GetEventWithResilience(ctx context.Context, eventID string) (*Event, error) {
    // Try with circuit breaker
    result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
        // Try with retry
        return s.retryPolicy.Execute(ctx, func() (interface{}, error) {
            return s.eventStore.GetEvent(ctx, eventID)
        })
    })
    
    if err == nil {
        return result.(*Event), nil
    }
    
    // Circuit breaker open, try fallback
    if err == query.ErrCircuitOpen {
        result, err := s.cache.Get(ctx, eventID)
        if err == nil {
            return result.(*Event), nil
        }
    }
    
    // Fallback failed, trigger recovery
    s.recoveryHandler.RecoverConnection(ctx)
    
    return nil, err
}
```

---

## Anti-Patterns

### 1. Retry Without Classification
**Problem:** Retrying permanent errors wastes time and resources

**Solution:** Classify errors before retrying
```go
// Bad
for i := 0; i < 3; i++ {
    result, err := operation()
    if err == nil {
        return result
    }
}

// Good
if classifier.IsTransient(err) {
    return retryPolicy.Execute(ctx, operation)
}
```

### 2. Infinite Retries
**Problem:** System hangs waiting for recovery

**Solution:** Set max attempts and max duration
```go
// Bad
for {
    result, err := operation()
    if err == nil {
        return result
    }
}

// Good
retryPolicy := query.NewRetryPolicy(query.RetryConfig{
    MaxAttempts: 3,
    MaxBackoff:  10 * time.Second,
})
```

### 3. No Jitter in Backoff
**Problem:** Thundering herd problem when many clients retry simultaneously

**Solution:** Add jitter to backoff
```go
// Bad
backoff := initialBackoff * math.Pow(multiplier, float64(attempt))

// Good
backoff := initialBackoff * math.Pow(multiplier, float64(attempt))
jitter := backoff * jitterFraction * (2*rand.Float64() - 1)
backoff = backoff + jitter
```

### 4. Circuit Breaker Without Fallback
**Problem:** Requests fail immediately without alternative

**Solution:** Implement fallback strategy
```go
// Bad
result, err := circuitBreaker.Execute(operation)
if err != nil {
    return nil, err
}

// Good
result, err := circuitBreaker.Execute(operation)
if err == ErrCircuitOpen {
    return cache.Get(ctx, key)
}
```

### 5. No Monitoring
**Problem:** Can't detect issues or verify resilience

**Solution:** Implement comprehensive monitoring
```go
// Record metrics
metrics.RecordRetryAttempt()
metrics.RecordCircuitBreakerStateChange(state)
metrics.RecordFallbackUsage()
metrics.RecordRecoveryAttempt()
```

---

## Performance Considerations

### Latency Impact

#### Retry Pattern
- **Best case:** No retry, no latency impact
- **Worst case:** Max attempts × max backoff
- **Typical:** 1-2 retries, 100-500ms additional latency

#### Circuit Breaker Pattern
- **Closed state:** < 1% latency overhead
- **Open state:** < 1ms (immediate rejection)
- **Half-Open state:** 1-5% latency overhead

#### Fallback Pattern
- **Primary success:** No latency impact
- **Fallback used:** Depends on fallback strategy
- **Cache fallback:** < 10ms additional latency

### Resource Usage

#### Memory
- Circuit breaker state: ~1KB per breaker
- Retry policy: ~100 bytes per operation
- Metrics: ~1KB per metric

#### CPU
- Error classification: < 1% CPU overhead
- Retry logic: < 1% CPU overhead
- Circuit breaker: < 1% CPU overhead

### Optimization Tips

1. **Use appropriate backoff values** - Too aggressive wastes resources, too conservative wastes time
2. **Implement connection pooling** - Reduce connection overhead
3. **Use caching** - Reduce database load
4. **Implement bulkheads** - Prevent resource exhaustion
5. **Monitor performance** - Detect bottlenecks

---

## Real-World Examples

### Example 1: E-Commerce Order Processing

**Scenario:** Processing orders with multiple data stores

**Resilience Strategy:**
1. Retry transient errors (network timeouts)
2. Circuit breaker for database failures
3. Fallback to cache for read operations
4. Graceful degradation for partial failures

**Implementation:**
```go
func (s *OrderService) ProcessOrder(ctx context.Context, order *Order) error {
    // Try with retry and circuit breaker
    err := s.circuitBreaker.Execute(ctx, func() error {
        _, err := s.retryPolicy.Execute(ctx, func() (interface{}, error) {
            return nil, s.storeOrder(ctx, order)
        })
        return err
    })
    
    if err == nil {
        return nil
    }
    
    // Circuit breaker open, use fallback
    if err == query.ErrCircuitOpen {
        return s.storeOrderInCache(ctx, order)
    }
    
    return err
}
```

### Example 2: Real-Time Analytics

**Scenario:** Collecting and processing real-time events

**Resilience Strategy:**
1. Retry transient errors
2. Circuit breaker for event store failures
3. Fallback to in-memory buffer
4. Graceful degradation with reduced accuracy

**Implementation:**
```go
func (s *AnalyticsService) RecordEvent(ctx context.Context, event *Event) error {
    // Try with retry
    _, err := s.retryPolicy.Execute(ctx, func() (interface{}, error) {
        return nil, s.eventStore.Store(ctx, event)
    })
    
    if err == nil {
        return nil
    }
    
    // Fallback to in-memory buffer
    s.buffer.Add(event)
    
    // Trigger recovery
    go s.recoveryHandler.RecoverConnection(ctx)
    
    return nil
}
```

### Example 3: Microservices Communication

**Scenario:** Service-to-service communication with multiple services

**Resilience Strategy:**
1. Retry transient errors
2. Circuit breaker for service failures
3. Fallback to alternative service
4. Graceful degradation with cached responses

**Implementation:**
```go
func (s *ServiceClient) CallService(ctx context.Context, req *Request) (*Response, error) {
    // Try with retry and circuit breaker
    result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
        return s.retryPolicy.Execute(ctx, func() (interface{}, error) {
            return s.service.Call(ctx, req)
        })
    })
    
    if err == nil {
        return result.(*Response), nil
    }
    
    // Circuit breaker open, try alternative service
    if err == query.ErrCircuitOpen {
        return s.alternativeService.Call(ctx, req)
    }
    
    // Alternative failed, use cached response
    return s.cache.Get(ctx, req.ID)
}
```

---

## Summary

Resilience patterns are essential for building robust systems that gracefully handle failures. Key patterns include:

1. **Retry Pattern** - Retry transient errors with exponential backoff
2. **Circuit Breaker Pattern** - Prevent cascading failures
3. **Bulkhead Pattern** - Isolate failures
4. **Fallback Pattern** - Provide alternative strategies
5. **Recovery Pattern** - Automatic recovery procedures

By combining these patterns strategically, you can build systems that:
- Handle transient failures automatically
- Prevent cascading failures
- Gracefully degrade under partial failures
- Recover automatically from failures
- Maintain high availability and reliability

---

**For more information:**
- See `ERROR_HANDLING_GUIDE.md` for detailed error handling
- See `MICROSERVICES_DATA_LAYER_PHASE_4_QUICK_REFERENCE.md` for quick reference
- See implementation files in `pkg/services/query/` for code examples
