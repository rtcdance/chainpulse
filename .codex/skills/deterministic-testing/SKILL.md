# Skill: deterministic-testing

## Trigger

Use this skill for any new/changed test, flaky test, property test, or CI instability issue.

## Must Do

1. Make tests replayable:
   - fixed random seed for randomized cases
   - stable clock/time injection where time matters
   - deterministic fixture inputs
2. Keep tests isolated:
   - no hidden dependency on local machine state
   - no implicit ordering dependency between tests
3. For failure reproduction:
   - record seed/input and replay path
4. For CI reliability:
   - keep tests deterministic under `-race` and repeated runs
5. Document deterministic assumptions in test setup when non-obvious.

### Blockchain-Specific Determinism

**Block Time Simulation**
```go
type MockClock struct {
    currentTime time.Time
}

func (m *MockClock) Now() time.Time {
    return m.currentTime
}

func (m *MockClock) AdvanceBlock() {
    m.currentTime = m.currentTime.Add(12 * time.Second) // Ethereum block time
}

func TestIndexer_BlockTiming(t *testing.T) {
    clock := &MockClock{currentTime: time.Unix(1000000, 0)}
    indexer := NewIndexer(WithClock(clock))

    // Deterministic block timestamps
    clock.AdvanceBlock()
    block := indexer.ProcessBlock(1)
    assert.Equal(t, int64(1000012), block.Timestamp)
}
```

**Deterministic Random for Addresses**
```go
func TestContract_RandomAddresses(t *testing.T) {
    seed := int64(12345) // Fixed seed
    rng := rand.New(rand.NewSource(seed))

    addr := generateRandomAddress(rng)
    // Always produces same address with same seed
    assert.Equal(t, "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb", addr.Hex())
}
```

**Chain State Fixtures**
```go
// Use deterministic block fixtures
func NewTestBlock(number uint64) *types.Block {
    return types.NewBlock(
        &types.Header{
            Number:     big.NewInt(int64(number)),
            Time:       1000000 + number*12, // Deterministic timestamp
            Difficulty: big.NewInt(1),
        },
        nil, nil, nil, trie.NewStackTrie(nil),
    )
}
```

## Must Not

- No unbounded randomness without seed capture.
- No tests depending on wall-clock timing only.
- No flaky tests left unresolved.
- No reliance on real blockchain time (use mock clock).
- No random addresses without fixed seed.

## Exit Criteria

- New/updated tests can be reproduced reliably.
- Flaky behavior is eliminated or explicitly blocked from merge.
- Block timestamps deterministic in tests.
- Random seeds logged for failure reproduction.
- Tests pass 100 consecutive runs: `go test -count=100`

