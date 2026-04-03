# Phase 228 - Puller Checkpoint Freshness Summary

## Status
Status: Approved

## Why
- Phase 227 added a deterministic per-chain checkpoint summary for `puller`.
- That summary still only answered:
  - which chain has a recorded checkpoint
  - which chain is risky or reconciled
- The next useful slice is to add a lightweight freshness/age signal so the
  rollout summary can also show whether a per-chain checkpoint posture looks
  recent or stale.

## Scope
- Keep rollout contract shape stable.
- Reuse the existing per-chain checkpoint summary rather than adding a new
  structured payload.
- Add lightweight freshness labels based on each chain's last checkpoint-source
  update time.

## Implementation
- Extend per-chain checkpoint summaries with their last-updated unix time.
- Add a lightweight freshness classifier for per-chain checkpoint summary
  entries:
  - `fresh`
  - `stale`
- Render per-chain checkpoint summaries as:
  - `<chain>=<status>:<freshness>@<block>`
- Thread the freshness-aware per-chain summary into `puller` runtime rollout
  state and advisory reasons.

## Validation
- Run focused `puller` rollout/runtime-progress tests.

## Exit Criteria
- `puller` rollout reasons can show not only per-chain checkpoint posture but
  also whether each chain's checkpoint status looks fresh or stale.
- The implementation remains lightweight and additive, without introducing a
  larger structured per-chain rollout contract in this phase.
