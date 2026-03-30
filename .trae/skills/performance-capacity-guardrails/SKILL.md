# Skill: performance-capacity-guardrails

## Trigger

Use this skill when changing hot paths (puller/indexer/query/cache/mq), concurrency, batching, or retry behavior.

## Must Do

1. Define expected impact on latency, throughput, and resource usage.
2. Add benchmark or load-test evidence for meaningful path changes.
3. Validate no severe regressions against baseline.
4. Expose/track key runtime signals:
   - queue lag
   - indexing delay
   - query latency
   - cache hit ratio
5. Document capacity assumptions and scaling knobs.

### Chain-Specific Capacity Planning

**EVM vs Non-EVM Chains**
```go
type ChainCapacity struct {
    MaxBlocksPerSecond  float64
    AvgEventsPerBlock   int
    PeakEventsPerBlock  int
}

var capacityProfiles = map[string]ChainCapacity{
    "ethereum": {MaxBlocksPerSecond: 0.083, AvgEventsPerBlock: 200, PeakEventsPerBlock: 2000},
    "bsc":      {MaxBlocksPerSecond: 0.33,  AvgEventsPerBlock: 150, PeakEventsPerBlock: 1500},
    "solana":   {MaxBlocksPerSecond: 2.5,   AvgEventsPerBlock: 5000, PeakEventsPerBlock: 50000},
}
```

**Event Density Benchmarks**
```go
func BenchmarkIndexer_HighDensityBlocks(b *testing.B) {
    // Simulate Uniswap V3 block (high event density)
    block := generateBlock(2000) // 2000 events

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        indexer.ProcessBlock(block)
    }
    // Target: <500ms per block
}
```

### RPC Cost Budget Tracking

**Cost Per Operation**
```go
var rpcCostEstimate = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "chainpulse_rpc_cost_estimate_usd",
        Help: "Estimated RPC cost in USD",
    },
    []string{"chain_id", "provider"},
)

// Track cost per indexing run
func (i *Indexer) IndexBlocks(start, end uint64) error {
    callCount := 0
    defer func() {
        costPerCall := 0.00001 // $0.01 per 1000 calls
        rpcCostEstimate.WithLabelValues(i.chainID, i.provider).Add(float64(callCount) * costPerCall)
    }()
    // ... indexing logic
}
```

**Cost Budget Alerts**
```yaml
- alert: HighRPCCost
  expr: rate(chainpulse_rpc_cost_estimate_usd[1h]) > 10
  annotations:
    summary: "RPC costs exceeding $10/hour"
```

## Must Not

- No performance-sensitive changes without measurement.
- No unbounded goroutines/retries/queues.
- No capacity planning without chain-specific profiles.

## Exit Criteria

- Performance impact is measured.
- Chain-specific capacity profiles defined (EVM vs non-EVM).
- Event density benchmarks added for high-load scenarios.
- RPC cost tracking implemented.
- Capacity controls and monitoring are defined.
