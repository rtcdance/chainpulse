# Phase 197 - Microservice Rollout Parity Guardrails

## Status
Status: Approved

## Why
- Four microservice rollout producers now exist:
  - `api-service`
  - `event-processor`
  - `puller`
  - `api-gateway`
- The next low-risk hardening step is to lock the shared contract boundaries so
  future rollout producer changes do not silently drift in metadata or core
  runtime-derived posture semantics.

## Scope
- Add shared parity validation helpers for microservice rollout producers.
- Apply those helpers in focused runtime-derived producer tests.
- Keep the rollout report contract unchanged.

## Implementation
- Add shared metadata parity validation for:
  - schema family
  - report version
  - report scope
  - report mode
  - deployment mode
  - service/report identity
- Add shared runtime-derived body parity validation for:
  - summary zeroes
  - advisory decision
  - policy posture
  - progression state
  - cutover dry-run action
  - approval posture
  - guarded-cutover posture
- Reuse the same helper in all four microservice producer tests.

## Validation
- Run focused Go tests for:
  - `pkg/plugins/api`
  - `api-service` rollout producer tests
  - `event-processor` focused rollout producer tests
  - `puller` focused rollout producer tests
  - `api-gateway` focused rollout producer/runtime tests

## Exit Criteria
- Shared metadata and runtime-derived body posture boundaries are enforced from
  one place.
- All four microservice rollout producers consume the same parity guardrails.
