# Phase 236 - Puller Checkpoint Recovery Hint

## Status
Status: Approved

## Why
- `puller` rollout already exposes:
  - checkpoint facts
  - compact checkpoint posture summaries
  - coverage posture
  - poll progress posture
- The next useful slice is to compress those checkpoint and poll posture facts
  into a more operator-facing recovery hint so rollout can answer not only what
  checkpoint posture looks like, but what recovery or observation action that
  posture suggests.

## Scope
- Keep existing checkpoint facts and posture fields unchanged.
- Derive a lightweight checkpoint recovery hint from:
  - reorg checkpoint state
  - persisted checkpoint state
  - checkpoint coverage posture
  - poll progress posture
- Thread that hint into `puller` runtime rollout reasons.

## Implementation
- Add a helper that derives compact recovery hints such as:
  - prioritize reconciliation before relying on persisted progress
  - continue observing catch-up and checkpoint advancement
  - continue observing fresh checkpoint coverage
- Expose that result through `puller` runtime rollout state as:
  - `CheckpointRecoveryHint`
- Append that result to advisory reasons as:
  - `checkpoint_recovery_hint`

## Validation
- Run focused `puller` rollout/runtime-progress tests.

## Exit Criteria
- `puller` rollout reasons can now report both:
  - checkpoint posture facts
  - a compact operator-facing recovery hint
- The change stays additive and preserves the current rollout contract.
