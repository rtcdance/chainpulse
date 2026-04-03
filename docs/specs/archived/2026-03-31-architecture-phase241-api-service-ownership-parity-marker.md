# Phase 241 - API Service Ownership Parity Marker

## Status
Status: Approved

## Why
- The rollout/control line is now paused at a strong pre-stage-complete
  boundary.
- The next high-value line is ownership/runtime parity.
- `api-service` is currently the most mature microservice rollout producer, but
  it still lacked one stable, explicit statement of the current truth:
  runtime wiring exists, while ownership-runtime parity with monolith does not.

## Scope
- Keep the existing rollout contract unchanged.
- Add an explicit ownership parity marker to `api-service` rollout semantics.
- Surface that marker through:
  - advisory reason
  - approval work item semantics

## Implementation
- Add an `ownership_parity_hint` to runtime-derived `api-service` advisory
  reason assembly.
- Make `api-service` approval work item review fields and reason explicitly
  reference ownership-runtime parity.
- Extend focused producer and route integration coverage accordingly.

## Validation
- Run focused `api-service` rollout tests.
- Run `pkg/plugins/api` tests.

## Exit Criteria
- `api-service` rollout now explicitly states that runtime wiring is present
  while ownership-runtime parity with monolith is still pending.
- That parity gap is visible through stable rollout semantics, not only through
  implicit zeroed ownership counters.
