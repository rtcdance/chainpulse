# Gas Cost Optimization

## Purpose
Minimize RPC call costs and optimize chain interaction patterns for cost-sensitive Web3 operations.

## Trigger
- Adding new on-chain read/write operations
- Modifying RPC call frequency or batch size
- Implementing event polling or log queries
- Cross-chain data aggregation

## Must Do

### 1. Batch Operations
```go
// ❌ Bad: N RPC calls
for _, addr := range addresses {
    balance, _ := client.BalanceAt(ctx, addr, nil)
}

// ✅ Good: 1 batched call
type BalanceRequest struct { Address common.Address }
results, _ := client.BatchCall(ctx, requests)
```

### 2. Cache Immutable Data
```go
// Cache block data (immutable after finality)
type BlockCache interface {
    Get(blockNum uint64) (*Block, bool)
    Set(blockNum uint64, block *Block)
}

// Only fetch if not in cache
if block, ok := cache.Get(num); ok {
    return block
}
```

### 3. Use Filters Over Polling
```go
// ❌ Bad: Poll every block
for blockNum := start; blockNum <= end; blockNum++ {
    logs, _ := client.FilterLogs(ctx, FilterQuery{FromBlock: blockNum, ToBlock: blockNum})
}

// ✅ Good: Single filter query
logs, _ := client.FilterLogs(ctx, FilterQuery{FromBlock: start, ToBlock: end})
```

### 4. Track Cost Metrics
```go
// Instrument RPC calls
type MeteredClient struct {
    client RPCClient
    callCount prometheus.Counter
    costEstimate prometheus.Histogram
}

func (m *MeteredClient) GetBlockByNumber(ctx context.Context, num uint64) (*Block, error) {
    m.callCount.Inc()
    m.costEstimate.Observe(0.001) // Estimate cost per call
    return m.client.GetBlockByNumber(ctx, num)
}
```

## Exit Criteria
- [ ] RPC calls batched where possible (>3 calls → 1 batch)
- [ ] Immutable data cached (blocks older than finality threshold)
- [ ] Filter queries used instead of per-block polling
- [ ] Cost metrics exposed (rpc_calls_total, estimated_cost_usd)
- [ ] No redundant calls in hot path (verified via trace logs)

## Cost Budget
- Indexing 1000 blocks: <100 RPC calls
- Real-time monitoring: <10 calls/minute per chain
- Historical query: Use archive node or local cache

## Anti-Patterns
- ❌ Fetching full block when only need header
- ❌ Re-fetching finalized blocks
- ❌ No rate limiting on RPC client
- ❌ Polling instead of event subscriptions

## References
- `pkg/plugins/rpc/batched_client.go` - Batch RPC implementation
- `pkg/plugins/cache/block_cache.go` - Block caching layer
