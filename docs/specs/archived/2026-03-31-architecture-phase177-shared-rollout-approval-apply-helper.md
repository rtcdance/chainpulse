# Phase 177 - Shared Rollout Approval Apply Helper

## Status
Status: Approved

## Why
- Phase 176 removed duplicated `surface` apply plumbing across monolith and
  `api-service`.
- The next safest shared body layer is `approval`, because monolith and
  microservice both already materialize it as the same typed contract section.

## Scope
- Keep rollout report values unchanged.
- Extract a shared helper for applying `RolloutReportApproval` into
  `RolloutReportDetails`.

## Implementation
- Add `ApplyRolloutReportApprovalSection(...)` to the shared rollout contract.
- Update monolith and `api-service` section appliers to reuse it.
- Add contract-level tests for the helper.

## Validation
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` no longer duplicate approval apply plumbing.
- Shared rollout approval values remain unchanged.
