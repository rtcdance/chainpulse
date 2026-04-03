# Phase 184 - Shared Rollout Approval Flow/Work Item Inputs

## Status
Status: Approved

## Why
- Phase 183 split the rollout `surface` input into stable posture fields and
  cutover-specific fields.
- The next stable approval reuse step is similar: separate approval state flow
  fields from the approval work item payload while preserving the same external
  `approval` contract.

## Scope
- Keep rollout report values unchanged.
- Add shared builders for:
  - `approval flow` input
  - `approval work item` input
  - merged `approval` input

## Implementation
- Add:
  - `RolloutReportApprovalFlowInput`
  - `RolloutReportApprovalWorkItemInput`
  - `BuildRolloutReportApprovalInput(...)`
- Update monolith and `api-service` approval input builders to compose flow and
  work item inputs through the shared helper.

## Validation
- Add contract-level coverage for the shared approval input builder.
- Run contract tests, monolith/api-service tests, and the fast micro-loop gate.

## Exit Criteria
- Monolith and `api-service` both assemble rollout approval inputs through the
  shared flow/work-item input path.
- External rollout payload values remain unchanged.
