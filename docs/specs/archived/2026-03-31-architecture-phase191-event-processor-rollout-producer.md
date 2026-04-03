# Phase 191 - Event Processor Rollout Producer

## Status
Status: Approved

## Why
- `api-service` now exposes a runtime-derived rollout report, but the rollout
  producer story is still too concentrated on API-facing services.
- `event-processor` is the next meaningful microservice to cover because it has
  a clear set of local dependency signals: database, event store, metadata
  store, and Kafka.

## Scope
- Add a dedicated `event-processor` rollout report producer.
- Keep the rollout report contract unchanged.
- Use focused handler-level coverage instead of full service HTTP wiring for
  now, because the service does not yet expose the shared health handler in its
  runtime entrypoint.

## Implementation
- Add an `event-processor` rollout producer with:
  - skeleton fallback semantics
  - runtime-derived dependency wiring semantics
- Reuse the shared rollout report sections contract and apply helpers.
- Add focused tests for:
  - skeleton producer
  - partially wired runtime-derived state
  - fully wired runtime-derived state
  - `/health/rollout` handler-level response

## Validation
- Run focused Go tests for the new producer files.
- Run `go test ./pkg/plugins/api/...`.
- Document any pre-existing package-level blockers that prevent full
  `go test ./cmd/microservices/event-processor/...` coverage.

## Exit Criteria
- `event-processor` has a real rollout report producer, not just documentation.
- The new producer uses the shared rollout report contract.
- Contract shape remains unchanged.
