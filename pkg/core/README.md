# Core Package

Shared kernel for ChainPulse: core abstractions, data models, and foundational implementations.

## Responsibility

This package is the **shared kernel** of ChainPulse. It defines:
- Cross-package interfaces (ports)
- Shared data models (blockchain events, config, health status)
- Foundational implementations that are tightly coupled to the shared types
  (event bus, config manager, confirmation tracker, MEV pipeline, etc.)

**Important**: New business logic implementations should prefer `pkg/services/`
or `pkg/infrastructure/`. Only place implementations here if they are
fundamental cross-cutting utilities or if they implement core interfaces
from this same package.

## Key Interfaces

### Plugin Interface
```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(ctx context.Context, config Config) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) HealthStatus
}
```

### EventBus Interface
```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(topic string, handler EventHandler) error
    Unsubscribe(topic string, handler EventHandler) error
}
```

### Config Interface
```go
type Config interface {
    Get(key string) any
    Set(key string, value any)
}
```

### Logger Interface
```go
type Logger interface {
    Debug(msg string, fields ...any)
    Info(msg string, fields ...any)
    Warn(msg string, fields ...any)
    Error(msg string, fields ...any)
}
```

### Metrics Interface
```go
type Metrics interface {
    Record(name string, value float64)
    Increment(name string)
}
```

### HealthChecker Interface
```go
type HealthChecker interface {
    Check(ctx context.Context) HealthStatus
}
```

## Key Implementations

### Configuration
- **config.go** — DefaultConfigManager (env-based config, hot-reload)
- **config_validation.go** — Config validation rules
- **config_extensions.go** — Feature flags, multi-chain support

### Event Bus
- **eventbus.go** — DefaultEventBus with bounded worker pool and panic recovery

### Confirmation Tracker
- **confirmation.go** — Pending → Confirmed → Finalized lifecycle

### Logging
- **slog_logger.go** — Slog-based Logger implementation

### Metrics
- **metrics.go** — In-memory metrics collector

### Web3 Utilities
- **blockchain_models.go** — Block, Transaction, Event, Log data structures
- **mev_pipeline.go**, **mev_builder.go**, **mev_flashbots.go** — MEV-Boost pipeline
- **aa_mempool.go**, **aa_bundler.go** — ERC-4337 Account Abstraction
- **l2_bridge.go** — L2→L1 message verification
- **defi_primitives.go** — AMM math, lending health factors
- **gas_estimator.go** — Gas estimation and history
- **consensus.go** — Consensus rules and validation

## Important Rules

1. **Do NOT define new interfaces** in `pkg/infrastructure` — define them here
2. All implementations in `pkg/infrastructure` must implement interfaces from here
3. Do NOT import `pkg/services/` or `pkg/infrastructure/` from this package
4. Use this package for cross-package contracts

## Documentation

Inline doc comments in this package serve as the primary reference.
For architecture decisions, see `docs/specs/`.
