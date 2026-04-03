---
name: "multi-chain-consistency"
description: "Handle chain-specific finality, reorg depth, and consensus differences. Use unified abstractions with chain-specific configs. Invoke when adding support for new blockchain, modifying consensus or finality logic, implementing cross-chain queries, or changing block confirmation thresholds."
---

# Multi-Chain Consistency

## Purpose
Handle chain-specific differences in finality, reorg depth, and consensus while maintaining unified abstractions.

## Trigger
- Adding support for new blockchain
- Modifying consensus or finality logic
- Implementing cross-chain queries
- Changing block confirmation thresholds

## Must Do

### 1. Define Chain Parameters
```go
type ChainConfig struct {
    ChainID          uint64
    FinalityBlocks   uint64  // Ethereum: 64, BSC: 15, Polygon: 128
    MaxReorgDepth    uint64  // Safe reorg handling depth
    BlockTime        time.Duration
    ConsensusType    string  // "PoW", "PoS", "PoA"
}

var chainConfigs = map[uint64]ChainConfig{
    1:     {ChainID: 1, FinalityBlocks: 64, MaxReorgDepth: 100, BlockTime: 12 * time.Second},
    56:    {ChainID: 56, FinalityBlocks: 15, MaxReorgDepth: 50, BlockTime: 3 * time.Second},
    137:   {ChainID: 137, FinalityBlocks: 128, MaxReorgDepth: 256, BlockTime: 2 * time.Second},
}
```

### 2. Abstract Chain Behavior
```go
type ChainAdapter interface {
    GetSafeBlock(ctx context.Context) (uint64, error)
    IsFinal(blockNum uint64) bool
    GetConfirmations(blockNum uint64) (uint64, error)
}
```

### 3. Test Cross-Chain Scenarios
```go
func TestIndexer_MultiChainConsistency(t *testing.T) {
    chains := []uint64{1, 56, 137}

    for _, chainID := range chains {
        cfg := chainConfigs[chainID]
        indexer := NewIndexer(chainID, cfg)

        // Verify finality threshold respected
        assert.Equal(t, cfg.FinalityBlocks, indexer.GetFinalityThreshold())

        // Verify reorg handling
        indexer.HandleReorg(cfg.MaxReorgDepth)
        assert.NoError(t, indexer.VerifyConsistency())
    }
}
```

## Exit Criteria
- [ ] Chain-specific config defined (finality, reorg depth, block time)
- [ ] Unified adapter interface for chain differences
- [ ] Tests cover at least 3 different chains
- [ ] No hardcoded chain assumptions in business logic
- [ ] Finality checks use chain-specific thresholds

## Anti-Patterns
- ❌ Hardcoding Ethereum-specific values (64 blocks)
- ❌ Assuming all chains have same reorg behavior
- ❌ Using fixed block time for all chains
- ❌ No chain-specific error handling

## References
- `pkg/domain/chain_config.go` - Chain parameter definitions
- `pkg/adapters/chain_adapter.go` - Chain abstraction layer
