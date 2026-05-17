# Testing Strategies in ChainPulse

ChainPulse uses a multi-layered testing strategy to ensure correctness at every level. This guide explains the available testing techniques and when to use each.

## Test Types

### 1. Unit Tests (`*_test.go`)

The foundation of all testing. Located alongside the source code.

```go
// pkg/services/processor/idempotency_test.go
func TestIdempotency_HashDeterministic(t *testing.T) {
    svc := NewDefaultIdempotencyService(logger, metrics)
    event1 := makeValidEvent()
    hash1, _ := svc.GenerateHash(event1)
    hash2, _ := svc.GenerateHash(event1)
    if hash1 != hash2 {
        t.Fatal("hash should be deterministic")
    }
}
```

**When to use**: For all Go code. Use `go test ./pkg/... -count=1` to run.

### 2. Table-Driven Tests

ChainPulse prefers table-driven tests for validating multiple cases:

```go
tests := []struct {
    name     string
    input    string
    expected bool
}{
    {"true literal", "true", true},
    {"yes literal", "yes", true},
    {"empty string", "", false},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        result := ParseBool(tc.input, false)
        if result != tc.expected {
            t.Errorf("got %v, want %v", result, tc.expected)
        }
    })
}
```

**Examples**: `pkg/env/env_test.go`, `pkg/core/errors_test.go`, `pkg/services/processor/idempotency_test.go`

### 3. Concurrency Tests

Tests that exercise concurrent safety with goroutines:

```go
func TestCache_ConcurrentReadWrite(t *testing.T) {
    c := NewInMemoryCache()
    _ = c.Start()
    defer c.Stop()

    var wg sync.WaitGroup
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < 50; j++ {
                c.Get(ctx, "shared")
                c.Set(ctx, "shared", []byte("v"), 3600)
            }
        }(i)
    }
    wg.Wait()
}
```

**Examples**: `pkg/plugins/cache/inmemory_cache_test.go`, `pkg/services/processor/idempotency_test.go`

### 4. Race Detection

Always run with race detection during development:

```bash
go test -race ./pkg/... -count=1
```

Race detector catches data races between goroutines. ChainPulse uses `sync.RWMutex` (166 occurrences) and `sync.Mutex` (46 occurrences) extensively.

### 5. Property-Based Testing

Uses `pgregory.net/rapid` to generate random inputs and verify invariants:

```go
func TestCacheProperty(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        c := NewInMemoryCache()
        keys := rapid.SliceOf(rapid.String()).Draw(t, "keys")
        // Verify: after Set(k, v), Get(k) returns v
        for _, k := range keys {
            c.Set(ctx, k, []byte("v"), 3600)
            val, _ := c.Get(ctx, k)
            if string(val) != "v" {
                t.Fatal("property violated")
            }
        }
    })
}
```

**Examples**: `pkg/plugins/cache/inmemory_cache_advanced_property_test.go`, `pkg/plugins/cache/redis_cache_property_test.go`

### 6. Fuzz Testing

Uses Go's native fuzzing (`go test -fuzz`):

```go
func FuzzEventHash(f *testing.F) {
    f.Add("ethereum", uint64(100), "0xabc")
    f.Fuzz(func(t *testing.T, network string, block uint64, tx string) {
        // Validate event processing with random inputs
    })
}
```

**Examples**: `pkg/plugins/api/auth_fuzz_test.go`, `pkg/plugins/api/input_validation_fuzz_test.go`

### 7. Benchmark Tests

Performance benchmarks for critical paths:

```go
func BenchmarkCache_Set(b *testing.B) {
    c := NewInMemoryCache()
    c.Start()
    defer c.Stop()

    ctx := context.Background()
    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        c.Set(ctx, fmt.Sprintf("k-%d", i), []byte("v"), 3600)
    }
}
```

Run with: `go test -bench=. -benchmem ./pkg/...`

**Examples**: `pkg/plugins/cache/inmemory_cache_bench_test.go`, `pkg/services/processor/event_processor_bench_test.go`, `test/performance/benchmark_test.go`

### 8. Parallel Tests

Many test files use `t.Parallel()` for faster execution:

```go
func TestSomething(t *testing.T) {
    t.Parallel()
    // test body
}
```

ChainPulse has **97** `t.Parallel()` calls across the codebase.

## Mocking Strategy

ChainPulse uses **hand-written mocks** rather than mock generators. The pattern:

```go
type mockStorage struct {
    writeErr error
    mu       sync.Mutex
    written  []*core.BlockchainEvent
}

func (m *mockStorage) WriteEvent(_ context.Context, e *core.BlockchainEvent) error {
    m.mu.Lock()
    m.written = append(m.written, e)
    m.mu.Unlock()
    return m.writeErr
}
```

Each mock tracks calls for assertion and can inject errors for failure paths.

Common mocks used across packages:
- `core.NewDefaultLogger(core.LogLevelError)` — silent logger
- `core.NewDefaultMetricsCollector()` — no-op metrics
- `NewDefaultIdempotencyService(logger, metrics)` — real idempotency with test isolation

## Test Organization

```
pkg/
├── core/
│   ├── errors.go          → errors_test.go
│   ├── generics.go        → generics_test.go
│   ├── replay_protection.go → replay_protection_test.go
├── plugins/
│   ├── cache/
│   │   ├── inmemory_cache.go          → inmemory_cache_test.go
│   │   ├── inmemory_cache_advanced.go → inmemory_cache_advanced_test.go
│   │   │                                inmemory_cache_advanced_property_test.go
│   │   └── redis_cache.go             → redis_cache_test.go
│   │                                    redis_cache_property_test.go
│   ├── api/
│   │   ├── rate_limiter.go    → rate_limiter_test.go
│   │   ├── auth_middleware.go → auth_middleware_test.go
│   │   └── auth_fuzz_test.go  (fuzz testing)
│   └── mq/
│       └── memory_mq.go       → memory_mq_test.go
├── services/
│   ├── processor/
│   │   ├── event_processor.go      → event_processor_test.go (unit)
│   │   ├── event_processor.go      → event_processor_bench_test.go (benchmark)
│   │   └── idempotency.go          → idempotency_test.go
│   └── indexing/
│       └── shadow_write_tracker.go → shadow_write_tracker_test.go
└── env/
    └── env.go → env_test.go
```

## Running Tests

| Command | Purpose |
|---------|---------|
| `go test ./pkg/...` | Run all unit tests |
| `go test -race ./pkg/...` | Run with race detection |
| `go test -v -run TestXxx ./pkg/...` | Run specific tests |
| `go test -bench=. -benchmem ./pkg/...` | Run benchmarks |
| `go test -fuzz=FuzzXxx -fuzztime=10s ./pkg/...` | Run fuzz tests |
| `go test -coverprofile=coverage.out ./pkg/...` | Generate coverage report |
| `go tool cover -html=coverage.out` | View coverage HTML |

## Best Practices

1. **Test isolation**: Each test creates fresh state; no shared global mutable state.
2. **Error paths first**: Always test validation failures before happy paths.
3. **Concurrent safety**: For types with locks, include a concurrent read/write test.
4. **Stop/Start lifecycle**: Plugin-style objects should test Initialize → Start → Stop sequences.
5. **Table-driven validation**: Use slices of structs for validation/parse tests.
6. **Benchmarks with ReportAllocs**: Always call `b.ReportAllocs()` to track memory pressure.