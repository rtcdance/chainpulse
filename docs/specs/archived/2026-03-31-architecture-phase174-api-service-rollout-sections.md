# Phase 174 - API Service Rollout Sections

## Status
Status: Approved

## Why
- `api-service` rollout production now has enough internal structure that it is
  worth aligning its implementation shape with the monolith rollout report path.
- We want future cross-mode body builder reuse to feel incremental instead of
  requiring another large rewrite later.

## Scope
- Keep `/health/rollout` payload values unchanged.
- Restructure `api-service` rollout production into:
  - `surface`
  - `approval`
  - `guarded-cutover`
  - section apply/assembly

## Implementation
- Add a small `apiServiceRolloutReportSections` type.
- Split skeleton and runtime-derived report construction into section builders.
- Route producer application through a shared section applier.

## Validation
- Add helper coverage for runtime-derived section assembly.
- Run api-service tests, shared tests, and the fast micro-loop gate.

## Exit Criteria
- `api-service` rollout producer no longer writes all surface/approval/guarded
  fields inline.
- External rollout payload values remain unchanged.
