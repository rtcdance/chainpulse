# Phase 201 - Event Processor Rollout Runtime Wiring

## Status
Status: Approved

## Why
- Phase 198 made the `event-processor` rollout producer health-aware, but that
  behavior still lived only in focused producer tests.
- The next step is to wire the producer into the service entrypoint so the
  rollout report path is built from actual runtime dependencies instead of test
  scaffolding alone.

## Scope
- Add a small runtime support helper for `event-processor`.
- Build a real `HealthCheckHandler` plus rollout producer from the service
  entrypoint.
- Keep the external rollout report contract unchanged.

## Implementation
- Add `buildEventProcessorRuntimeRolloutHealthHandler(...)` that:
  - initializes `HealthCheckHandler`
  - registers the `event-processor` rollout producer
  - derives runtime rollout state from:
    - Mongo/PostgreSQL database readiness
    - event store health
    - metadata store health
    - Kafka health
- Add focused runtime support tests for:
  - runtime state extraction
  - `/health/rollout` response through the constructed handler
- Wire the helper into `cmd/microservices/event-processor/main.go`.

## Validation
- Run focused `event-processor` rollout producer and runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `event-processor` service entrypoint now constructs a real rollout health
  handler with runtime-derived rollout state.
- Focused tests prove the wired handler can serve `/health/rollout`.
