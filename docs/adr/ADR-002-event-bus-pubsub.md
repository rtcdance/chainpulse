# ADR-002: Event Bus Publish/Subscribe Pattern

**Date**: 2026-03-20

### Status

Accepted

### Context

ChainPulse components need to communicate state changes (reorg detected, new block indexed, cache invalidated) without tight coupling. Direct method calls between services create circular dependencies and make testing difficult.

Options considered:
1. **Direct method calls** — simple but couples components
2. **Channel-based pub/sub** — Go-native but hard to debug, no topic discovery
3. **Event bus with topic-based routing** — decoupled, observable, testable

### Decision

Use `core.EventBus` — a synchronous, in-process, topic-based publish/subscribe event bus:

```go
type EventBus interface {
    Publish(ctx context.Context, topic string, event interface{}) error
    Subscribe(ctx context.Context, topic string, handler func(interface{})) error
    Unsubscribe(topic string, handler func(interface{})) error
}
```

`DefaultEventBus` implements this with a `map[string][]EventHandler` protected by `sync.RWMutex`. Topics are string-based (e.g., `"reorg-detected"`, `"block-indexed"`). Handlers are `func(interface{})` — consumers type-assert to the expected payload type.

Key design choices:
- **Synchronous dispatch**: `Publish` calls all handlers before returning. This ensures ordering and simplifies error handling. Async dispatch would require a goroutine pool and error aggregation.
- **No persistence**: Events are not durably stored. If a subscriber is not registered when an event is published, it is missed. This is acceptable because all critical state is persisted via `DatabasePlugin`.
- **Error isolation**: A panic in one handler does not affect others (recovered and logged).

### Consequences

- **Positive**: Components are decoupled — `ReorgHandler` publishes `"reorg-detected"` without knowing who consumes it
- **Positive**: Easy to test — subscribe in test, publish, assert received
- **Positive**: Observable — `GetSubscriberCount(topic)` and `GetTopics()` enable monitoring
- **Negative**: Synchronous dispatch means a slow handler blocks the publisher; if latency becomes an issue, handlers must offload to goroutines themselves
- **Negative**: No built-in retry or dead-letter; failed handlers are logged but the event is lost
- **Neutral**: Cross-process communication (between microservices) still requires Kafka — EventBus is intra-process only

### Amendments (2026-05-06)

**Backpressure and capacity control**: The original design spawned an unbounded goroutine per handler invocation in `Publish`. Under high event throughput this caused OOM risk and goroutine leaks. The implementation now uses a **16-slot worker pool** (`chan struct{}`) as a semaphore — each handler invocation acquires a slot, runs, then releases. When all 16 slots are occupied, new dispatches block until a slot frees, providing natural backpressure.

**Graceful shutdown**: A `sync.WaitGroup` tracks in-flight handler goroutines. `EventBus.Wait()` blocks until all handlers complete, enabling orderly shutdown. The `Stop()` methods of `KafkaMQ`, `RedisMQ`, and `BasePuller` now call `Wait()` with a 10-second timeout before closing resources, preventing data loss during deployment rollovers.
