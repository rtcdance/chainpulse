# Phase 176 - Shared Rollout Surface Apply Helper

## Status
Status: Approved

## Why
- Monolith and `api-service` both build rollout report sections and then apply
  the exact same `surface` fields into `RolloutReportDetails`.
- That repeated write path is a good first shared body helper because it is
  purely structural and does not own any deployment-specific rollout logic.

## Scope
- Keep rollout report values unchanged.
- Extract a shared helper for applying `RolloutReportSurfaceSection` into
  `RolloutReportDetails`.

## Implementation
- Add `ApplyRolloutReportSurfaceSection(...)` to the shared rollout contract
  package.
- Update monolith and `api-service` section appliers to reuse that helper.
- Add contract-level coverage for the shared apply helper.

## Validation
- Run contract tests, monolith/api-service tests, and fast micro-loop.

## Exit Criteria
- Monolith and `api-service` no longer duplicate surface apply plumbing.
- Shared rollout surface values remain unchanged.
