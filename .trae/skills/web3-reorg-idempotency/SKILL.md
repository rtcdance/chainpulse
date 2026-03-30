# Skill: web3-reorg-idempotency

## Trigger

Use this skill when modifying puller/indexer/query/reorg/consistency code paths.

## Must Do

1. Define event idempotency keys and enforce deduplication.
2. Handle reorg with rollback + replay strategy.
3. Preserve data correctness under finality uncertainty.
4. Add failure-path tests for reorg and duplicate events.
5. Emit chain-aware telemetry (`chain_id`, `block_height`, `operation`).

### Enhanced Reorg Scenarios

**Deep Reorg (>10 blocks)**
```go
func TestIndexer_DeepReorg(t *testing.T) {
    indexer.IndexBlocks(1000, 1050)

    // Simulate 20-block reorg
    indexer.HandleReorg(1030, 1050)

    // Verify: rolled back to 1030, replayed new chain
    assert.Equal(t, 1050, indexer.GetLatestBlock())
    assert.NoError(t, indexer.VerifyConsistency())
}
```

**Uncle Blocks**
```go
// Verify uncle blocks don't create duplicate events
func TestIndexer_UncleBlockHandling(t *testing.T) {
    // Index block with uncle
    indexer.IndexBlock(1000)

    // Uncle appears later
    indexer.ProcessUncle(1000, uncleHash)

    // Verify: no duplicate events emitted
    events := indexer.GetEvents(1000)
    assert.Equal(t, 1, len(events))
}
```

**MEV Reorg**
```go
// Test MEV-induced reorg (common on Ethereum)
func TestIndexer_MEVReorg(t *testing.T) {
    // Index block with high-value tx
    indexer.IndexBlock(1000)

    // MEV bot causes 1-block reorg
    indexer.HandleReorg(999, 1000)

    // Verify: tx order may change, but state consistent
    assert.NoError(t, indexer.VerifyStateRoot(1000))
}
```

### Finality Threshold Configuration

```go
type FinalityConfig struct {
    SafeBlocks      uint64  // Blocks before considering "safe"
    FinalizedBlocks uint64  // Blocks before considering "finalized"
}

// Chain-specific finality
var finalityConfigs = map[uint64]FinalityConfig{
    1:   {SafeBlocks: 32, FinalizedBlocks: 64},   // Ethereum PoS
    56:  {SafeBlocks: 10, FinalizedBlocks: 15},   // BSC
    137: {SafeBlocks: 64, FinalizedBlocks: 128},  // Polygon
}

func (i *Indexer) IsSafe(blockNum uint64) bool {
    latest := i.GetLatestBlock()
    cfg := finalityConfigs[i.chainID]
    return latest-blockNum >= cfg.SafeBlocks
}
```

## ChainPulse Pointers

- Reorg handler: `pkg/services/reorg/reorg_handler.go`
- Indexing flow: `pkg/services/indexing/*`
- Puller flow: `pkg/plugins/pullers/*`
- Query consistency helpers: `pkg/services/query/*`

## Must Not

- No best-effort-only rollback for reorg-sensitive data.
- No idempotency assumptions without tests.
- No silent drops of replay events.
- No hardcoded finality thresholds (use chain config).

## Exit Criteria

- Reorg/duplicate scenarios covered by tests.
- Deep reorg (>10 blocks), uncle blocks, MEV reorg tested.
- Finality thresholds configured per chain.
- Rollback + replay behavior documented.
- Metrics/logs expose chain-specific recovery signals.
