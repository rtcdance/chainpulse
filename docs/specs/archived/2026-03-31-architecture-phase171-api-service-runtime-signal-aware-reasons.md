# Phase 171 - API Service Runtime-Signal-Aware Rollout Reasons

## Status
Status: Approved

## Why
- Phase 169 made `api-service` rollout state more truthful by switching from a
  pure skeleton to runtime-derived local wiring posture.
- The remaining gap is operator readability: a `hold` or `observe` posture is
  more actionable when the report states which runtime signals are enabled and
  which are still missing.

## Scope
- Keep rollout states and shared `/health/rollout` contract unchanged.
- Improve `api-service` rollout reasons so they enumerate enabled and missing
  runtime wiring signals.

## Implementation
- Add a small helper that derives a stable reason string from:
  - `runtime_routes_enabled`
  - `event_query_enabled`
  - `domain_bridge_enabled`
- Reuse that reason across advisory, policy, progression, approval, and guarded
  cutover fields for runtime-derived api-service rollout reports.

## Validation
- Extend producer tests to assert fully wired reason contents.
- Extend route integration tests to assert partial wiring reason contents.
- Run api-service package tests, shared package tests, and fast micro-loop.

## Exit Criteria
- Runtime-derived api-service rollout reasons explicitly list enabled signals.
- Partial runtime-derived api-service rollout reasons explicitly list missing
  signals.
- No rollout state or contract field names change.
