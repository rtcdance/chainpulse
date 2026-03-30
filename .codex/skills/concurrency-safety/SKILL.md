# Skill: concurrency-safety

## Trigger

Use this skill when changing goroutines, channels, locks, worker pools, retries, or shared state.

## Must Do

1. Define goroutine lifecycle and cancellation via `context.Context`.
2. Ensure bounded concurrency:
   - worker pool size
   - queue/backpressure limits
   - retry caps
3. Protect shared mutable state explicitly (mutex/atomic/channel ownership).
4. Verify behavior under load and shutdown:
   - graceful stop
   - no goroutine leaks
5. Run race-focused verification for changed scope.

### Go-Specific Patterns

**Channel Closing**
```go
// ❌ Bad: Close from receiver
go func() {
    for msg := range ch {
        process(msg)
    }
    close(ch) // Wrong side!
}()

// ✅ Good: Close from sender
go func() {
    for _, msg := range messages {
        ch <- msg
    }
    close(ch)
}()
```

**Goroutine Leak Detection**
```go
func TestWorker_NoLeaks(t *testing.T) {
    before := runtime.NumGoroutine()

    w := NewWorker()
    w.Start()
    w.Stop()

    time.Sleep(100 * time.Millisecond)
    after := runtime.NumGoroutine()

    assert.Equal(t, before, after, "goroutine leak detected")
}
```

**WaitGroup Pattern**
```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        process(i)
    }(item)
}
wg.Wait()
```

## Must Not

- No fire-and-forget goroutines without ownership.
- No unbounded channel growth or retry loops.
- No lock patterns that can deadlock under normal failure paths.
- No closing channels from receiver side.
- No WaitGroup.Add() inside goroutine.

## Exit Criteria

- Concurrency model is explicit and bounded.
- Race/leak/deadlock risks are addressed and tested.
- Channel ownership and closing responsibility clear.
- Goroutine leak tests pass.
- `go test -race` passes on changed code.

