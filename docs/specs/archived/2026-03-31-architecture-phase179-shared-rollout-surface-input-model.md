# Phase 179 - Shared Rollout Surface Input Model

## Status
Status: Approved

## Why
- Phase 176 established a shared `surface` apply helper, but monolith and
  `api-service` were still building their surface sections inline.
- The safest next reuse layer is a shared typed input model for surface section
  construction, because both deployment modes already populate the same
  contract fields there.

## Scope
- Keep rollout report values unchanged.
- Add a shared typed input and builder for `RolloutReportSurfaceSection`.

## Implementation
- Add:
  - `RolloutReportSurfaceInput`
  - `BuildRolloutReportSurfaceSection(...)`
- Update monolith and `api-service` surface builders to feed that shared input
  instead of constructing the surface section inline.

## Validation
- Add contract-level coverage for the shared surface builder.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both build rollout surface sections through the
  same shared typed input model.
- External rollout payload values remain unchanged.
