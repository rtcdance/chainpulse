# Phase 195 - API Gateway Rollout Producer Skeleton

## Status
Status: Approved

## Why
- The microservice rollout producer matrix now covers `api-service`,
  `event-processor`, and `puller`, but `api-gateway` still has no rollout
  producer entry.
- Adding a minimal `api-gateway` producer is the highest-value next step
  because it expands deployment-mode coverage without forcing full runtime route
  exposure in the same slice.

## Scope
- Add a dedicated `api-gateway` rollout report producer.
- Keep the rollout report contract unchanged.
- Use local gateway runtime wiring signals only.
- Use focused handler-level coverage instead of full service-entrypoint wiring.

## Implementation
- Add an `api-gateway` rollout producer with:
  - skeleton fallback semantics
  - runtime-derived gateway wiring semantics
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

## Exit Criteria
- `api-gateway` has a real rollout report producer entrypoint.
- The new producer uses the shared rollout report contract.
- Contract shape remains unchanged.
