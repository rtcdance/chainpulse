# Phase 204 - Rollout Coverage Matrix Refresh

## Status
Status: Approved

## Why
- The microservice rollout coverage summary was written before the recent
  `api-gateway`, `puller`, and `event-processor` entrypoint-level rollout
  wiring and parity work landed.
- The current bottleneck is no longer “which services have producers,” but
  “which services are protected only at producer-level versus wired
  entrypoint-level.”

## Scope
- Refresh the rollout coverage summary document to reflect the current state of:
  - producer presence
  - runtime-derived signal depth
  - producer-level parity
  - entrypoint-level parity
  - route-level versus wired-handler exposure

## Implementation
- Update the service coverage table for:
  - `api-service`
  - `event-processor`
  - `puller`
  - `api-gateway`
- Add a clearer verification matrix that distinguishes:
  - producer tests
  - wired handler/route coverage
  - producer-level parity
  - entrypoint-level parity
- Refresh the “Remaining Gaps” and recommendation sections so they point at the
  highest-value remaining work instead of already-completed milestones.

## Validation
- Run spec approval for this summary refresh.

## Exit Criteria
- The coverage summary accurately reflects the current rollout refactor state.
- The document makes the next remaining gaps obvious without rereading the full
  phase history.
