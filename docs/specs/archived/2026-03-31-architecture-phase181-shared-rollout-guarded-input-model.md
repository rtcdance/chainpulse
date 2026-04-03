# Phase 181 - Shared Rollout Guarded Input Model

## Status
Status: Approved

## Why
- Phase 178 established a shared `guarded-cutover` apply helper, but monolith
  and `api-service` were still assembling guarded sections inline.
- After Phase 179 and Phase 180, the next safe reuse step is to promote the
  guarded section behind the same typed input model pattern.

## Scope
- Keep rollout report values unchanged.
- Add a shared typed input and builder for `RolloutReportGuarded`.

## Implementation
- Add:
  - `RolloutReportGuardedInput`
  - `BuildRolloutReportGuardedSection(...)`
- Update monolith and `api-service` guarded section builders to feed that
  shared input instead of constructing guarded sections inline.

## Validation
- Add contract-level coverage for the shared guarded builder.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both build rollout guarded sections through the
  same shared typed input model.
- External rollout payload values remain unchanged.
