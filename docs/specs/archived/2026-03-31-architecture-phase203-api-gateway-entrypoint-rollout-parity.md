# Phase 203 - API Gateway Entrypoint Rollout Parity

## Status
Status: Approved

## Why
- `api-gateway` already had producer-level parity checks and a runtime support
  helper, but its wired `/health/rollout` route test still did not consume the
  shared microservice parity validators.
- The next low-risk tightening step is to apply the same handler-level parity
  guardrail already used on `puller` and `event-processor`.

## Scope
- Keep the rollout report contract unchanged.
- Reuse existing shared microservice rollout parity validators in the
  `api-gateway` runtime support route test.

## Implementation
- Update the wired `api-gateway` `/health/rollout` route test to validate:
  - shared metadata parity
  - shared runtime-derived posture parity
- Preserve service-specific reason text and deployment-specific details outside
  the shared parity boundary.

## Validation
- Run focused `api-gateway` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `api-gateway` wired `/health/rollout` handler path satisfies the same shared
  parity guardrails as the producer-level path.
- Entrypoint-level rollout parity now covers the microservices that already
  have real runtime support wiring.
