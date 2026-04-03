# Phase 226 - Puller Checkpoint Reorg Reconciled

## Status
Status: Approved

## Why
- Phase 225 taught `puller` rollout progress to flag when a recorded checkpoint
  may be at reorg risk.
- That still left one important gap: rollout could not distinguish between:
  - a checkpoint that is still risky
  - a checkpoint that was risky, but has since been superseded by a newer
    recorded checkpoint
- The next useful slice is to add a minimal recovery-style posture so rollout
  can express that reorg risk has been reconciled.

## Scope
- Keep rollout contract shape stable.
- Extend the lightweight `pullerCheckpointSource` with a reconciled posture.
- Surface a shared rollout signal when a newer checkpoint supersedes an earlier
  reorg-risk state.

## Implementation
- Extend the runtime-backed checkpoint source snapshot with reconciled facts.
- When a checkpoint source has observed reorg risk and later records a newer
  checkpoint at or beyond the risky checkpoint boundary, mark that source as
  reconciled.
- Extend the shared reorg-checkpoint classifier so rollout can now express:
  - `reorg-clear`
  - `reorg-risk`
  - `reorg-reconciled`
- Thread that reconciled posture through:
  - `puller` runtime rollout state
  - shared execution-progress reason details
  - focused `puller` and shared API tests

## Validation
- Run focused `puller` rollout/runtime-progress tests.
- Run focused shared execution-progress tests in `pkg/plugins/api`.

## Exit Criteria
- `puller` rollout progress can distinguish active reorg risk from a checkpoint
  posture that has already been reconciled by a newer recorded checkpoint.
- The implementation stays additive and lightweight, without claiming that full
  persisted reorg recovery orchestration is already wired into the service.
