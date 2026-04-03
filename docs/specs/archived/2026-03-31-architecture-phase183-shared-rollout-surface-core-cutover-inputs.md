# Phase 183 - Shared Rollout Surface Core/Cutover Inputs

## Status
Status: Approved

## Why
- Phase 182 introduced a shared facade for assembling rollout sections, but the
  `surface` input itself still bundled stable rollout posture fields and
  cutover-specific fields into one deployment-local assembly step.
- The safest next reuse layer is to separate the stable `summary/advisory/
  policy/progression` cluster from the cutover fields while preserving the same
  external `surface` contract.

## Scope
- Keep rollout report values unchanged.
- Add shared builders for:
  - `surface core` input
  - `surface cutover` input
  - merged `surface` input

## Implementation
- Add:
  - `RolloutReportSurfaceCoreInput`
  - `RolloutReportSurfaceCutoverInput`
  - `BuildRolloutReportSurfaceInput(...)`
- Update monolith and `api-service` surface input builders to compose core and
  cutover inputs through the shared helper.

## Validation
- Add contract-level coverage for the shared surface input builder.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both assemble rollout surface inputs through the
  shared core/cutover input path.
- External rollout payload values remain unchanged.
