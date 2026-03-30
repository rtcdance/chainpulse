# Data Retention & Archival

## Purpose
Manage blockchain data growth with clear hot/warm/cold storage strategy and archival policies.

## Trigger
- Adding new storage logic
- Modifying query time ranges
- Database size exceeds threshold
- Implementing historical data queries

## Must Do

### 1. Define Retention Policy
```go
type RetentionPolicy struct {
    HotDataDays   int  // Fast access: last 7 days
    WarmDataDays  int  // Slower access: 7-90 days
    ColdDataDays  int  // Archive: >90 days
    PurgeAfterDays int // Delete: >365 days (if applicable)
}
```

### 2. Implement Tiered Storage
```go
type TieredStorage interface {
    WriteHot(ctx context.Context, data BlockData) error
    MoveToWarm(ctx context.Context, beforeDate time.Time) error
    MoveToCold(ctx context.Context, beforeDate time.Time) error
}
```

### 3. Add Archival Tests
```go
func TestArchival_DataMovement(t *testing.T) {
    storage := NewTieredStorage()

    // Write to hot storage
    storage.WriteHot(ctx, block)

    // Move old data to warm
    storage.MoveToWarm(ctx, time.Now().Add(-7*24*time.Hour))

    // Verify data still accessible
    block, err := storage.Query(ctx, blockNum)
    assert.NoError(t, err)
}
```

## Exit Criteria
- [ ] Retention policy documented
- [ ] Hot/warm/cold boundaries defined
- [ ] Archival job implemented
- [ ] Query layer handles all tiers
- [ ] Metrics track storage by tier

## Anti-Patterns
- ❌ Keeping all data in hot storage
- ❌ No archival automation
- ❌ Deleting data without backup

## References
- `pkg/storage/tiered_storage.go`
- `scripts/archive-old-blocks.sh`
