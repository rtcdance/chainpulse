# Go Concurrency Patterns

## Purpose
Enforce Go-specific concurrency best practices: goroutine lifecycle, channel usage, context propagation.

## Trigger
- Spawning goroutines
- Using channels
- Implementing worker pools
- Adding context-aware operations

## Must Do

### 1. Bounded Goroutines
```go
// ❌ Bad: Unbounded goroutines
for _, item := range items {
    go process(item)
}

// ✅ Good: Worker pool
sem := make(chan struct{}, 10)
for _, item := range items {
    sem <- struct{}{}
    go func(i Item) {
        defer func() { <-sem }()
        process(i)
    }(item)
}
```

### 2. Context Propagation
```go
func (s *Service) Process(ctx context.Context, data Data) error {
    // Always pass context down
    return s.repo.Save(ctx, data)
}
```

### 3. Graceful Shutdown
```go
type Worker struct {
    done chan struct{}
}

func (w *Worker) Stop() {
    close(w.done)
}

func (w *Worker) Run(ctx context.Context) {
    for {
        select {
        case <-w.done:
            return
        case <-ctx.Done():
            return
        }
    }
}
```

## Exit Criteria
- [ ] No unbounded goroutine spawning
- [ ] Context passed to all blocking operations
- [ ] Graceful shutdown implemented
- [ ] No goroutine leaks (verified with tests)

## Anti-Patterns
- ❌ Goroutines without lifecycle management
- ❌ Ignoring context cancellation
- ❌ Channels without proper closing

## References
- `pkg/infrastructure/worker_pool.go`
