# Core Package

Shared kernel for ChainPulse: core abstractions, data models, and foundational implementations.

## Responsibility

This package is the **shared kernel** of ChainPulse. It contains:
- Type aliases for port interfaces (actual interface definitions are in `pkg/ports/`)
- Shared data models (blockchain events, config, health status)
- Foundational implementations tightly coupled to the shared types
  (event bus, config manager, logger, metrics collector, etc.)

**Important**: New business logic implementations should prefer `pkg/services/`
or `pkg/infrastructure/`. Only place implementations here if they are
fundamental cross-cutting utilities.

## Interface Definitions

All core interfaces (Plugin, EventBus, Config, Logger, Metrics, HealthChecker, etc.)
are defined in **`pkg/ports/`**. This package provides type aliases for backward
compatibility:

```go
// pkg/core/plugin.go
type Plugin = ports.Plugin
type Logger = ports.Logger

// pkg/core/eventbus_iface.go
type EventBus = ports.EventBus

// pkg/core/config_iface.go
type ConfigManager = ports.ConfigManager
```

New code should import `pkg/ports/` directly. The aliases in `pkg/core/` are
deprecated and will be removed in a future major version.

## Key Implementations

### Configuration
- **config.go** — DefaultConfigManager (env-based config, hot-reload)
- **config_validation.go** — Config validation rules
- **config_extensions.go** — Feature flags, multi-chain support
- **config_accessors.go** — Config accessor field definitions

### Event Bus
- **channel_eventbus.go** — Channel-based in-memory EventBus
- **typed_eventbus.go** — Typed event bus with topic routing

### Logging
- **slog_logger.go** — Slog-based Logger implementation

### Metrics
- **metrics.go** — In-memory metrics collector

### MQ (Message Queue)
- **mq_plugin.go** — MQ plugin definition
- **mq_messaging.go** — Messaging primitives
- **mq_batch.go** — Batch message processing
- **mq_lifecycle.go** — MQ lifecycle management
- **mq_metrics.go** — MQ metrics collection
- **mq_error_handler.go** — MQ error handling

### Core Data Structures
- **types.go** — Domain types and port type aliases
- **errors.go** — Error types and classification
- **health.go** — Health check models and implementations
- **health_iface.go** — HealthChecker interface alias
- **plugin.go** — Plugin interface alias and lifecycle helpers
- **eventbus_iface.go** — EventBus interface alias
- **config_iface.go** — Config interface aliases
- **secret.go** — Secret string type for credentials
- **constants.go** — Core constants
- **deployment_constants.go** — Deployment constants

### Web3 Data Structures
- **blockchain_validation.go** — Blockchain data validation
- **event_filter.go** — Event filtering logic
- **event_hash.go** — Event hashing utilities
- **solana_models.go** — Solana blockchain data structures
- **kzg.go** — KZG commitment verification
- **address_utils.go** — Address utility functions

### Registry & Runtime
- **registry.go** — Plugin registry
- **runtime_types.go** — Runtime type definitions
- **indexer_ops.go** — Indexer operations

## Important Rules

1. **Define new interfaces in `pkg/ports/`** — not in `pkg/core/` or `pkg/infrastructure/`
2. All implementations in `pkg/infrastructure/` must implement interfaces from `pkg/ports/`
3. Do NOT import `pkg/services/` or `pkg/infrastructure/` from this package
4. Use `pkg/ports/` for cross-package contracts

## Documentation

Inline doc comments in this package serve as the primary reference.
For architecture decisions, see `docs/specs/`.
