# Phase 200 - Puller Rollout Runtime Wiring

## Status
Status: Approved

## Why
- Phase 199 made the `puller` rollout producer health-aware, but that behavior
  still lived only in focused producer tests.
- The next step is to wire the producer into the real service entrypoint so the
  rollout report path is built from actual runtime dependencies instead of test
  scaffolding alone.

## Scope
- Add a small runtime support helper for `puller`.
- Build a real `HealthCheckHandler` plus rollout producer from the service
  entrypoint.
- Keep the external rollout report contract unchanged.

## Implementation
- Add `buildPullerRuntimeRolloutHealthHandler(...)` that:
  - initializes `HealthCheckHandler`
  - registers the `puller` rollout producer
  - derives runtime rollout state from:
    - PostgreSQL health
    - Kafka health
    - puller loop configuration
    - blockchain RPC configuration
- Add focused runtime support tests for:
  - runtime state extraction
  - `/health/rollout` response through the constructed handler
- Wire the helper into `cmd/microservices/puller/main.go`.

## Validation
- Run focused `puller` rollout producer and runtime support tests.
- Keep existing shared API rollout tests passing.

## Exit Criteria
- `puller` service entrypoint now constructs a real rollout health handler with
  runtime-derived rollout state.
- Focused tests prove the wired handler can serve `/health/rollout`.
