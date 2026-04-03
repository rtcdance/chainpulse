# Phase 180 - Shared Rollout Approval Input Model

## Status
Status: Approved

## Why
- Phase 177 established a shared `approval` apply helper, but monolith and
  `api-service` were still assembling approval sections inline.
- Phase 179 showed the safer next reuse step: promote stable cross-mode section
  fields behind a shared typed input model and builder.

## Scope
- Keep rollout report values unchanged.
- Add a shared typed input and builder for `RolloutReportApproval`.

## Implementation
- Add:
  - `RolloutReportApprovalInput`
  - `BuildRolloutReportApprovalSection(...)`
- Update monolith and `api-service` approval section builders to feed that
  shared input instead of constructing approval sections inline.

## Validation
- Add contract-level coverage for the shared approval builder.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both build rollout approval sections through the
  same shared typed input model.
- External rollout payload values remain unchanged.
