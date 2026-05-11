# ADR-003: Reorg Handling Strategy — Rollback and Reindex

**Date**: 2026-03-25

### Status

Accepted

### Context

Blockchain reorganizations ("reorgs") occur when a previously accepted chain segment is replaced by a longer alternative chain. ChainPulse must handle reorgs to maintain data integrity — events from the old chain segment must be removed and replaced with events from the new chain.

Options considered:
1. **Tombstone marking** — Mark old events as `removed: true` instead of deleting. Preserves audit trail but complicates queries (every query must filter `removed: false`).
2. **Versioned events** — Keep both old and new events with version numbers. Doubles storage, requires merge logic on read.
3. **Rollback + reindex** — Delete events from the reorg block onward, then re-index from the new chain head. Simple, storage-efficient, but loses the old data.

### Decision

Use **rollback + reindex** (option 3). The `ReorgHandler` implements this in two phases:

**Phase 1 — Rollback**: `HandleReorg(ctx, reorgBlock)` → `RollbackEvents(ctx, reorgBlock)`:
1. `GetEventsByBlockRange(ctx, reorgBlock, ^uint64(0))` — find all events at or after the reorg block
2. `DeleteEventsByBlockRange(ctx, reorgBlock, ^uint64(0))` — delete them
3. Publish `"reorg-detected"` event via `EventBus` (if configured)

**Phase 2 — Reindex**: The puller automatically re-indexes blocks starting from the reorg point. No special reindex code is needed — the puller's normal polling loop picks up the new chain data.

Safety guards:
- `maxRollback` parameter prevents unbounded deletion (e.g., max 120 blocks)
- `reorgThreshold` parameter filters noise from small block number fluctuations
- Block cache is cleaned up to prevent false re-detection

```go
func (rh *ReorgHandler) HandleReorg(ctx context.Context, reorgBlock uint64) error {
    currentBlock, _ := rh.database.GetLatestBlock(ctx)
    blocksToRollback := currentBlock - reorgBlock + 1
    if blocksToRollback > rh.maxRollback {
        return fmt.Errorf("reorg too large: %d blocks (max: %d)", blocksToRollback, rh.maxRollback)
    }
    eventsRolledBack, _ := rh.RollbackEvents(ctx, reorgBlock)
    rh.cleanupBlockCache(reorgBlock)
    // publish event if bus is configured
}
```

### Consequences

- **Positive**: Simple implementation — delete + re-index is two operations, no version tracking needed
- **Positive**: Storage-efficient — no duplicate or tombstoned events
- **Positive**: Automatic reindex via existing puller — no special code path
- **Negative**: Old chain data is permanently lost — cannot audit what was rolled back (only the `ReorgEvent` metadata is retained in the event bus log)
- **Negative**: If reindex fails after rollback, there is a data gap until the puller catches up
- **Neutral**: `maxRollback` must be tuned per chain — Ethereum mainnet reorgs are typically < 7 blocks, but testnets can be deeper

### Amendments (2026-05-06)

**ReorgHandler wiring**: The original `BlockConfirmationTracker.detectReorgs()` was a no-op — reorg detection was never triggered. The tracker now maintains a `reorgHandlers map[string]*ReorgHandler` registry and `detectReorgs()` queries the `blocks` table for the latest block, calls `handler.DetectReorg()` to compare on-chain vs stored parent hashes, and invokes `handler.HandleReorg()` when a mismatch is found. This closes the detection-to-action gap.

**Depth limit on backward scan**: `findReorgBlock()` previously scanned backwards without bound, risking infinite loops on corrupted data. A `maxScanDepth=256` constant now limits the scan range, with context cancellation checks at each iteration. If the depth is exceeded, the scan returns an error rather than hanging.
