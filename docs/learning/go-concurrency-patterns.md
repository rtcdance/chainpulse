# Go Concurrency Patterns in ChainPulse

ChainPulse uses Go's concurrency primitives for blockchain event processing. This guide explains the patterns used across the codebase.

## 1. CSP: Communicating Sequential Processes

The core concurrency model — goroutines communicate via channels:

```go
// pkg/core/channel_eventbus.go
type ChannelEventBus struct {
    subscribers map[string][]chan any
    mu          sync.RWMutex
}
```

**Key insight**: Channels carry data; mutexes protect subscriber lists. Two primitives, two purposes.

## 2. Event Bus (Publish/Subscribe)

```go
bus := core.NewChannelEventBus()

// Subscribe
subCh, subID := bus.Subscribe("events")
go func() {
    for event := range subCh {
        fmt.Printf("received: %v\n", event)
    }
}()

// Publish
bus.Publish(ctx, "events", myEvent)
```

**Pattern**: One publish, many subscribers. The EventBus uses:
- `sync.RWMutex` for subscriber list management (many readers, few writers)
- `atomic` for subscription ID generation
- buffered channels for event delivery
- `context.Context` for subscription cancellation

## 3. sync.RWMutex (Read-Optimized)

ChainPulse uses **166** `sync.RWMutex` vs **46** `sync.Mutex`:

```go
type DefaultEventProcessor struct {
    mu sync.RWMutex
    // ...
}

// Read path — many concurrent readers
func (p *DefaultEventProcessor) Health() *core.HealthStatus {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.lastHealthCheck
}

// Write path — serialized writers
func (p *DefaultEventProcessor) Start() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.running = true
    // ...
}
```

**When to use RWMutex**: When there are significantly more reads than writes (e.g., health checks, stats queries, cache lookups).

## 4. Semaphore Pattern

Used in the indexer to limit concurrent writes:

```go
// pkg/core/indexer_ops.go
type Semaphore struct {
    ch chan struct{}
}

func NewSemaphore(n int) *Semaphore {
    return &Semaphore{ch: make(chan struct{}, n)}
}

func (s *Semaphore) Acquire() { s.ch <- struct{}{} }
func (s *Semaphore) Release() { <-s.ch }
```

Usage:
```go
sem := NewSemaphore(10) // max 10 concurrent

for _, event := range events {
    sem.Acquire()
    go func(ev core.BlockchainEvent) {
        defer sem.Release()
        store(ctx, ev)
    }(event)
}
```

## 5. Goroutine Lifecycle

### errgroup (coordinated goroutine groups)

```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return indexer.Run(ctx) })
g.Go(func() error { return apiServer.Listen(ctx) })
if err := g.Wait(); err != nil {
    log.Fatal(err)
}
```

**errgroup advantage**: First goroutine's error cancels the context for all others.

### WaitGroup + Context

```go
// pkg/application/bootstrap/shutdown.go
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    server.ListenAndServe()
}()

// Graceful shutdown
sig := WaitForSignal()
server.Shutdown(ctx)
if ShutdownWithTimeout(&wg, 10*time.Second) {
    log.Println("clean shutdown")
}
```

## 6. sync.Pool (Object Reuse)

```go
// cmd/playground/pool_demo.go
var bufferPool = sync.Pool{
    New: func() any { return new(bytes.Buffer) },
}

buf := bufferPool.Get().(*bytes.Buffer)
defer func() {
    buf.Reset()
    bufferPool.Put(buf)
}()
```

**Use when**: Creating many short-lived objects of the same type (buffers, event structs).

## 7. Context Cancellation

Every long-running operation accepts `ctx context.Context`:

```go
func (p *DefaultEventProcessor) storeEventWithRetry(ctx context.Context, event *core.BlockchainEvent) error {
    for attempt := 0; attempt < p.maxRetries; attempt++ {
        if err := ctx.Err(); err != nil {
            return fmt.Errorf("store cancelled: %w", err)
        }
        // ... attempt
    }
}
```

**Pattern**: Check `ctx.Err()` at the top of each retry loop iteration and between batch items.

## 8. Channel Patterns

### Signal Channel
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
<-sigChan // blocks until signal
```

### Done Channel
```go
done := make(chan struct{})
go func() {
    wg.Wait()
    close(done)
}()

select {
case <-done:
    fmt.Println("all goroutines finished")
case <-time.After(timeout):
    fmt.Println("timeout")
}
```

### Ticker with Context
```go
ticker := time.NewTicker(1 * time.Second)
defer ticker.Stop()
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-ticker.C:
        // periodic work
    }
}
```

## 9. Atomic Counters

For counters that don't need mutex protection:

```go
import "sync/atomic"

type mockPuller struct {
    nextID   atomic.Uint64  // Go 1.19+
    blockNum atomic.Uint64
}

id := p.nextID.Add(1)
block := p.blockNum.Load()
```

## 10. Mutex vs Channel Decision Guide

| Scenario | Use | Why |
|----------|-----|-----|
| Protect shared state (config, cache data) | `sync.Mutex` / `sync.RWMutex` | Simple, clear ownership |
| Fan-out events to consumers | Channel (via EventBus) | CSP: many consumers receive copies |
| Limit concurrency | Semaphore (channel-based) | Buffered channel = token bucket |
| Signal completion | `done chan struct{}` | Zero-allocation signal |
| Timeout | `time.After` + `select` | One-shot, no cleanup needed |
| Periodic work | `time.Ticker` + `select` | Clean start/stop, context-aware |

## Reference

- [pkg/core/channel_eventbus.go](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/pkg/core/channel_eventbus.go) — pure-Go EventBus implementation
- [pkg/application/bootstrap/shutdown.go](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/pkg/application/bootstrap/shutdown.go) — graceful shutdown patterns
- [cmd/playground/pool_demo.go](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/cmd/playground/pool_demo.go) — sync.Pool demo
- [docs/learning/go-context-patterns.md](file:///Users/mingo/Applications/workspace/web3/project/chainpulse/docs/learning/go-context-patterns.md) — context patterns