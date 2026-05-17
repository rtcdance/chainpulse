# ADR-001: Plugin Architecture — Monolith vs Microservices

**Date**: 2026-03-15

### Status

Accepted

### Context

ChainPulse needs to serve multiple operational modes:
1. **Development/local testing** — single process, in-memory storage, no external dependencies
2. **Acceptance testing** — Docker Compose with real Kafka and MongoDB but minimal replicas
3. **Production** — Kubernetes with dedicated microservices, high availability

A monolithic architecture simplifies development but cannot scale individual components independently. A pure microservices architecture adds operational complexity for local development.

We need a single codebase that supports both deployment models without forked business logic.

### Decision

Adopt a **plugin-based architecture** where each capability (database, event processing, API, pulling) is a `core.Plugin` implementation behind fine-grained interfaces (`EventReader`, `EventWriter`, `BlockReader`, `ReorgStatsProvider`). Deployment mode is determined by which plugin implementations are wired together at startup:

- **Monolith**: `cmd/monolithic/main.go` — in-memory adapters, single-process, no external deps
- **Microservices**: `cmd/microservices/*/main.go` — each service runs its own binary with MongoDB/Kafka adapters

Both modes share the same `pkg/domain/` business logic. The `core.DatabasePlugin` interface composes `Plugin + EventReader + EventWriter + BlockReader + ReorgStatsProvider` so consumers depend on the smallest interface they need.

```
pkg/core/plugin.go:
  Plugin             → Name(), Version(), Initialize(Config), Start(), Stop(), Health()
  EventReader        → GetEvent, QueryEvents, GetAllEvents, GetEventsByBlockRange
  EventWriter        → StoreEvent, BatchStoreEvents, DeleteEvent, DeleteEventsByBlockRange
  BlockReader        → GetBlock, GetLatestBlock, GetAllBlocks
  ReorgStatsProvider → GetReorgStats
  DatabasePlugin     → Plugin + EventReader + EventWriter + BlockReader + ReorgStatsProvider
```

### Consequences

- **Positive**: Same domain logic verified in both modes; in-memory mock enables fast unit tests without Docker; microservices can scale independently in production
- **Positive**: Fine-grained interfaces follow Interface Segregation Principle — query services only need `EventReader`
- **Negative**: Interface proliferation adds boilerplate; mock implementations must satisfy full `DatabasePlugin` even when tests only use a subset
- **Negative**: Two deployment topologies doubles the Docker Compose configuration surface
- **Neutral**: `cmd/monolithic/` and `cmd/microservices/` must stay in sync for env var names and config structure

### Amendments (2026-05-06)

**Dependency direction enforcement**: The layering rule `core → domain → services → infrastructure → plugins → cmd` is now enforced. Key corrections:
- `infrastructure/deployment/adapter_factory.go` refactored to plugin registration pattern (`RegisterFactory`) instead of direct plugin imports
- `services/processor/event_processor.go` uses local `CacheWriter` interface instead of importing `plugins/cache`
- `services/query/EventStore` consolidated as type alias for `domain/query.EventStore`

**ChainedDecoder**: A 3-strategy event decoder (`pkg/core/chained_decoder.go`) was added. Decode chain: runtime ABIs → static known ABIs → raw hex fallback. Unknown events are preserved with `_raw: true` flag instead of being silently dropped.

**Environment variable unification**: `getEnv()` in `pkg/core/config.go` now reads `CHAINPULSE_` prefixed keys first, falling back to bare names for backward compatibility.
