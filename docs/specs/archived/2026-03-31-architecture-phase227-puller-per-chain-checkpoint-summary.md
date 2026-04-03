# Phase 227 - Puller Per-Chain Checkpoint Summary

## Status
Status: Approved

## Why
- Phase 224-226 progressively improved `puller` checkpoint posture with:
  - persisted checkpoint state
  - reorg risk
  - reconciled checkpoint posture
- Those signals still only surfaced as a highest-block/global summary.
- The next useful slice is to add a lightweight per-chain checkpoint summary so
  rollout reasons can point to which chain is recorded, risky, or reconciled.

## Scope
- Keep rollout contract shape stable.
- Extend the lightweight checkpoint source snapshot with per-chain summaries.
- Append a stable per-chain checkpoint summary string to `puller` rollout
  reasons when runtime checkpoint state exists.

## Implementation
- Add per-chain checkpoint summaries to the checkpoint source snapshot.
- Derive stable per-chain statuses:
  - `checkpoint-recorded`
  - `reorg-risk`
  - `reorg-reconciled`
- Sort the per-chain summary by chain id for deterministic rollout output.
- Thread that summary into `puller` runtime rollout state and advisory reason
  details as:
  - `checkpoint_chain_summary`

## Validation
- Run focused `puller` rollout/runtime-progress tests.

## Exit Criteria
- `puller` rollout reasons no longer rely only on highest-block checkpoint
  posture.
- Operators can see which chain currently owns a recorded, risky, or
  reconciled checkpoint state from the rollout summary.
