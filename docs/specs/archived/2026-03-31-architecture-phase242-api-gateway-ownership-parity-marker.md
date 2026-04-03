# Phase 242 - API Gateway Ownership Parity Marker

## Status
Status: Approved

## Why
- Phase 241 established the first ownership/runtime parity marker in
  `api-service`.
- The next useful parity slice is to avoid leaving `api-gateway` as a silent
  outlier that still implies the ownership gap only through zeroed ownership
  counters.
- `api-gateway` already has stable runtime-wiring rollout semantics, so it is a
  good second producer to carry an explicit ownership parity marker.

## Scope
- Keep the shared rollout contract unchanged.
- Add an explicit ownership parity marker to `api-gateway` rollout semantics.
- Surface that marker through:
  - advisory reason
  - approval work item semantics

## Implementation
- Add an `ownership_parity_hint` to runtime-derived `api-gateway` advisory
  reason assembly.
- Make `api-gateway` approval work item review fields and reason explicitly
  reference ownership-runtime parity.
- Extend focused producer and runtime-support coverage accordingly.

## Validation
- Run focused `api-gateway` rollout tests.
- Run `pkg/plugins/api` tests.

## Exit Criteria
- `api-gateway` rollout now explicitly states that runtime wiring is present
  while ownership-runtime parity with monolith is still pending.
- The ownership parity gap is now visible through stable rollout semantics
  instead of remaining implicit.
