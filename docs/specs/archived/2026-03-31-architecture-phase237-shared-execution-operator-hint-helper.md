# Phase 237 - Shared Execution Operator Hint Helper

## Status
Status: Approved

## Why
- Recent phases added operator-facing execution hints in two places:
  - `puller` checkpoint recovery hint
  - `event-processor` consumer backlog hint
- The next useful slice is to stop appending those operator hints with
  service-local reason keys and instead give execution services a shared helper
  for operator-facing execution hints.

## Scope
- Keep service-specific hint semantics unchanged.
- Add a shared execution operator-hint helper for:
  - poll-driven services
  - consumer-driven services
- Let `puller` and `event-processor` both use that shared helper when appending
  rollout reasons.

## Implementation
- Add a shared `RolloutExecutionOperatorHint` facade in
  `pkg/plugins/api/rollout_execution_progress_contract.go`.
- Add shared helpers:
  - `AppendRolloutExecutionOperatorHintReason(...)`
  - `ValidateRolloutExecutionOperatorHintReasonCoverage(...)`
- Update:
  - `puller` to publish its checkpoint recovery hint through the shared poll
    operator-hint path
  - `event-processor` to publish its backlog hint through the shared consumer
    operator-hint path

## Validation
- Run focused shared execution-progress helper tests.
- Run focused `puller` and `event-processor` rollout tests.

## Exit Criteria
- Shared execution operator-hint assembly exists in one place.
- `puller` and `event-processor` both use the shared operator-hint helper.
- Existing service-specific hint semantics remain unchanged.
