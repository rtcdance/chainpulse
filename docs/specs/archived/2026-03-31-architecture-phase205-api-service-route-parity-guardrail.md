# Phase 205 - API Service Route Parity Guardrail

## Status
Status: Approved

## Why
- `api-service` already had route-level `/health/rollout` integration coverage,
  but that real route path still validated shared parity only through manual
  field assertions.
- The next tightening step is to apply the same shared microservice rollout
  parity validators directly on that route-level payload.

## Scope
- Keep the rollout report contract unchanged.
- Reuse existing shared microservice rollout parity validators in the
  `api-service` `/health/rollout` route integration test.

## Implementation
- Update the `api-service` route integration test to validate:
  - shared metadata parity
  - shared runtime-derived posture parity
- Preserve service-specific reason text and selected route-level field
  assertions outside the shared parity boundary.

## Validation
- Run focused `api-service` rollout producer and route integration tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `api-service` route-level rollout exposure now consumes the same shared parity
- guardrails as the other microservice producer and wired-handler paths.
