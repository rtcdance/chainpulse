---
name: "chaos-resilience-testing"
description: "Verify recovery under RPC failures, timeouts, and network partitions. Inject failures and validate circuit breakers. Invoke when adding/modifying RPC client calls, implementing retry or fallback logic, cross-chain interaction changes, or external dependency integration."
---

# Chaos Resilience Testing

## Purpose
Verify system behavior under Web3 infrastructure failures: RPC timeouts, node forks, network partitions, inconsistent responses.

## Trigger
- Adding/modifying RPC client calls
- Implementing retry or fallback logic
- Cross-chain interaction changes
- External dependency integration

## Must Do

### 1. Identify Failure Modes
```go
// Document expected failures
// - RPC timeout (>30s)
// - Node out of sync (block height lag)
// - Rate limit (429)
// - Network partition (connection refused)
// - Inconsistent state (different nodes return different data)
```

### 2. Inject Failures
```go
// Use test doubles with failure injection
type ChaoticRPCClient struct {
    real RPCClient
    failureRate float64
    latencyMs int
}

func (c *ChaoticRPCClient) GetBlockByNumber(ctx context.Context, num uint64) (*Block, error) {
    if rand.Float64() < c.failureRate {
        return nil, errors.New("simulated RPC timeout")
    }
    time.Sleep(time.Duration(c.latencyMs) * time.Millisecond)
    return c.real.GetBlockByNumber(ctx, num)
}
```

### 3. Verify Recovery
- Circuit breaker opens after N failures
- Fallback to secondary RPC provider
- Graceful degradation (serve stale data with warning)
- Retry with exponential backoff
- No goroutine leaks or deadlocks

### 4. Add Chaos Tests
```go
func TestIndexer_RPCFailureRecovery(t *testing.T) {
    client := &ChaoticRPCClient{failureRate: 0.5}
    indexer := NewIndexer(client)

    // Should eventually succeed despite 50% failure rate
    err := indexer.IndexBlocks(ctx, 1000, 1100)
    require.NoError(t, err)

    // Verify circuit breaker metrics
    assert.True(t, indexer.CircuitBreakerTripped())
}
```

## Exit Criteria
- [ ] Failure modes documented in test comments
- [ ] At least 3 chaos scenarios tested (timeout, rate-limit, partition)
- [ ] Recovery verified: no panics, no infinite retries, metrics updated
- [ ] Circuit breaker or fallback logic present
- [ ] Test passes with 30%+ injected failure rate

## Anti-Patterns
- ❌ Only testing happy path
- ❌ Infinite retry without backoff
- ❌ No timeout on external calls
- ❌ Assuming RPC responses are always consistent

## References
- `test/helpers/chaotic_rpc.go` - Failure injection utilities
- `pkg/infrastructure/circuit_breaker.go` - Circuit breaker implementation
