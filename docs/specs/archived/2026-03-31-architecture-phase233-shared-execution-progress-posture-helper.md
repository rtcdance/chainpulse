# Phase 233 - Shared Execution Progress Posture Helper

## Status
Status: Approved

## Why
- Phase 231-232 added compact execution posture summaries in two separate
  places:
  - `puller` now exposes compact checkpoint and poll-facing posture signals
  - `event-processor` now exposes a compact consumer progress posture
- The next useful slice is to stop hand-assembling these compact progress
  posture hints inside individual services and move the generic execution
  progress posture layer into the shared rollout execution-progress helpers.

## Scope
- Keep the existing execution-progress raw fact contract unchanged.
- Add a shared compact execution-progress posture layer for:
  - poll progress
  - consumer progress
- Let `puller` and `event-processor` both consume that shared posture helper
  when appending rollout reasons.

## Implementation
- Add a shared `RolloutExecutionProgressPosture` facade in
  `pkg/plugins/api/rollout_execution_progress_contract.go`.
- Add:
  - `BuildRolloutExecutionProgressPosture(...)`
  - `AppendRolloutExecutionProgressPostureReason(...)`
  - `ValidateRolloutExecutionProgressPostureReasonCoverage(...)`
- Update:
  - `puller` rollout reason assembly to append shared poll progress posture
  - `event-processor` rollout reason assembly to append shared consumer
    progress posture

## Validation
- Run focused shared execution-progress tests.
- Run focused `puller` and `event-processor` rollout tests.

## Exit Criteria
- Shared execution-progress posture derivation exists in one place.
- `puller` and `event-processor` both consume that shared posture helper.
- The change remains additive and preserves existing service-specific rollout
  posture signals.
