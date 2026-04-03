# Phase 178 - Shared Rollout Guarded Apply Helper

## Status
Status: Approved

## Why
- Phase 176 and Phase 177 already shared the `surface` and `approval` apply
  layers across monolith and `api-service`.
- The remaining duplicated apply plumbing sits in the `guarded-cutover`
  section, which is also a pure typed write path.

## Scope
- Keep rollout report values unchanged.
- Extract a shared helper for applying `RolloutReportGuarded` into
  `RolloutReportDetails`.

## Implementation
- Add `ApplyRolloutReportGuardedSection(...)` to the shared rollout contract.
- Update monolith and `api-service` section appliers to reuse it.
- Add contract-level tests for the helper.

## Validation
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` no longer duplicate guarded-cutover apply plumbing.
- Shared rollout guarded values remain unchanged.
