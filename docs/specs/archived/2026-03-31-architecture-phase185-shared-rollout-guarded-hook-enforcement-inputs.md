# Phase 185 - Shared Rollout Guarded Hook/Enforcement Inputs

## Status
Status: Approved

## Why
- Phase 184 split the rollout `approval` input into stable flow and work-item
  layers.
- The guarded-cutover model has the same kind of natural boundary: current
  hook/policy posture versus future enforcement posture.

## Scope
- Keep rollout report values unchanged.
- Add shared builders for:
  - guarded hook input
  - guarded enforcement input
  - merged guarded input

## Implementation
- Add:
  - `RolloutReportGuardedHookInput`
  - `RolloutReportGuardedEnforcementInput`
  - `BuildRolloutReportGuardedInput(...)`
- Update monolith and `api-service` guarded input builders to compose hook and
  enforcement inputs through the shared helper.

## Validation
- Add contract-level coverage for the shared guarded input builder.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both assemble rollout guarded inputs through the
  shared hook/enforcement input path.
- External rollout payload values remain unchanged.
