# Phase 202 - Entrypoint Rollout Parity Coverage

## Status
Status: Approved

## Why
- Phase 200 and Phase 201 moved `puller` and `event-processor` rollout
  producers from producer-only tests into real entrypoint-level health handler
  construction.
- The next hardening step is to ensure those wired handler paths still satisfy
  the same shared microservice rollout parity guarantees as the producer tests.

## Scope
- Keep the rollout report contract unchanged.
- Reuse existing shared microservice rollout parity validators in entrypoint
  runtime support tests for:
  - `puller`
  - `event-processor`

## Implementation
- Update `puller` wired handler `/health/rollout` tests to validate:
  - shared metadata parity
  - shared runtime-derived posture parity
- Update `event-processor` wired handler `/health/rollout` tests to validate:
  - shared metadata parity
  - shared runtime-derived posture parity
- Preserve service-specific reason text and deployment-specific details outside
  the shared parity boundary.

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run focused `event-processor` rollout producer/runtime support tests.

## Exit Criteria
- Both execution-service entrypoint rollout handlers now satisfy the shared
  microservice parity guardrails on their real wired handler path.
- Shared parity checks cover both producer-level and entrypoint-level rollout
  exposure.
