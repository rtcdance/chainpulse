---
name: "indexer-state-consistency"
description: "State machine validation and checkpoint consistency. Recovery path verification and integrity checks. Invoke when modifying indexer state transitions, adding checkpoint logic, implementing recovery paths, or changing block processing flow."
---

# Indexer State Consistency

## Purpose
Ensure indexer state machine integrity and checkpoint consistency across restarts.

## Trigger
- Modifying indexer state transitions
- Adding checkpoint logic
- Implementing recovery paths
- Changing block processing flow

## Must Do

### 1. State Machine Validation
```go
type IndexerState int

const (
    StateStopped IndexerState = iota
    StateStarting
    StateRunning
    StateSyncing
    StateStopping
)

func (s *IndexerState) Transition(to IndexerState) error {
    validTransitions := map[IndexerState][]IndexerState{
        StateStopped:  {StateStarting},
        StateStarting: {StateRunning, StateStopped},
        StateRunning:  {StateSyncing, StateStopping},
        StateSyncing:  {StateRunning, StateStopping},
        StateStopping: {StateStopped},
    }

    if !contains(validTransitions[*s], to) {
        return fmt.Errorf("invalid transition: %v -> %v", *s, to)
    }
    *s = to
    return nil
}
```

### 2. Checkpoint Consistency
```go
type Checkpoint struct {
    BlockNumber uint64
    BlockHash   common.Hash
    StateRoot   common.Hash
    Timestamp   time.Time
}

func (i *Indexer) SaveCheckpoint(cp Checkpoint) error {
    // Atomic: save checkpoint + update state
    return i.db.Transaction(func(tx *sql.Tx) error {
        if err := saveCheckpoint(tx, cp); err != nil {
            return err
        }
        return updateIndexerState(tx, cp.BlockNumber)
    })
}
```

### 3. Recovery Verification
```go
func (i *Indexer) Recover() error {
    cp, err := i.LoadLastCheckpoint()
    if err != nil {
        return err
    }

    // Verify checkpoint integrity
    block, err := i.client.BlockByNumber(ctx, big.NewInt(int64(cp.BlockNumber)))
    if err != nil {
        return err
    }

    if block.Hash() != cp.BlockHash {
        return fmt.Errorf("checkpoint hash mismatch: reorg detected")
    }

    i.startBlock = cp.BlockNumber
    return nil
}
```

## Exit Criteria
- [ ] State transitions validated
- [ ] Checkpoint saves are atomic
- [ ] Recovery path tested with corrupted state
- [ ] State consistency verified after restart

## Anti-Patterns
- ❌ No state transition validation
- ❌ Non-atomic checkpoint saves
- ❌ No recovery verification

## References
- `pkg/services/indexing/state_machine.go`
- `pkg/services/indexing/checkpoint.go`
