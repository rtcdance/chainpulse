# Phase 193 - Puller Rollout Producer

## Status
Status: Approved

## Why
- `api-service` and `event-processor` now expose rollout producers, but the
  puller path still has no rollout report producer despite being a core
  microservice in the ingestion pipeline.
- The next meaningful step is to give `puller` the same baseline rollout
  visibility using local dependency and wiring signals.

## Scope
- Add a dedicated `puller` rollout report producer.
- Keep the rollout report contract unchanged.
- Use focused handler-level coverage instead of full service HTTP wiring for
  now, because the service does not yet expose the shared health handler in its
  runtime entrypoint.

## Implementation
- Add a `puller` rollout producer with:
  - skeleton fallback semantics
  - runtime-derived dependency wiring semantics
  - compact rollout posture hints
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
  `go test ./cmd/microservices/puller/...` coverage.

## Exit Criteria
- `puller` has a real rollout report producer, not just documentation.
- The new producer uses the shared rollout report contract.
- Contract shape remains unchanged.
