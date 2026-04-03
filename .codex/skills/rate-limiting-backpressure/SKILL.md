---
name: "rate-limiting-backpressure"
description: "Handle RPC provider rate limits with graceful degradation, retry backoff, and flow control. Invoke when adding external API calls, modifying concurrency settings, implementing bulk operations, or integrating new RPC provider."
---

# Rate Limiting & Backpressure

## Purpose
Handle RPC provider rate limits with graceful degradation, retry backoff, and flow control.

## Trigger
- Adding external API calls
- Modifying concurrency settings
- Implementing bulk operations
- Integrating new RPC provider

## Must Do

### 1. Implement Rate Limiter
```go
type RateLimitedClient struct {
    client  RPCClient
    limiter *rate.Limiter  // golang.org/x/time/rate
}

func (r *RateLimitedClient) GetBlockByNumber(ctx context.Context, num uint64) (*Block, error) {
    if err := r.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    return r.client.GetBlockByNumber(ctx, num)
}
```

### 2. Add Exponential Backoff
```go
func (c *Client) CallWithRetry(ctx context.Context, fn func() error) error {
    backoff := time.Second
    for attempt := 0; attempt < 5; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        if isRateLimitError(err) {
            time.Sleep(backoff)
            backoff *= 2  // Exponential: 1s, 2s, 4s, 8s, 16s
            continue
        }
        return err
    }
    return errors.New("max retries exceeded")
}
```

### 3. Implement Backpressure
```go
type WorkQueue struct {
    tasks chan Task
    sem   chan struct{}  // Semaphore for concurrency control
}

func (q *WorkQueue) Submit(task Task) error {
    select {
    case q.tasks <- task:
        return nil
    case <-time.After(5 * time.Second):
        return errors.New("queue full, backpressure applied")
    }
}
```

### 4. Monitor Rate Limit Metrics
```go
var (
    rateLimitHits = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "rpc_rate_limit_hits_total",
    })
    retryAttempts = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "rpc_retry_attempts",
    })
)
```

## Exit Criteria
- [ ] Rate limiter configured per provider limits
- [ ] Exponential backoff on 429 errors
- [ ] Bounded concurrency (semaphore or worker pool)
- [ ] Backpressure prevents queue overflow
- [ ] Metrics track rate limit hits and retry counts

## Rate Limit Budgets
- Infura free tier: 100k requests/day (~1.15 req/s)
- Alchemy free tier: 300M compute units/month
- Self-hosted node: Unlimited but CPU-bound

## Anti-Patterns
- ❌ No rate limiting on free tier providers
- ❌ Immediate retry on 429 error
- ❌ Unbounded goroutine spawning
- ❌ No circuit breaker after repeated failures

## References
- `pkg/infrastructure/rate_limiter.go`
- `pkg/plugins/rpc/retry_client.go`
