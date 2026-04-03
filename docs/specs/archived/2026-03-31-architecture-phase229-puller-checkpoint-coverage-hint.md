# Phase 229 - Puller Checkpoint Coverage Hint

## Status
Status: Approved

## Why
- Phase 227-228 made `puller` checkpoint posture easier to scan with:
  - per-chain summaries
  - freshness labels
- The next useful slice is to add a compact coverage/count hint so rollout can
  quickly answer how many chains are currently:
  - recorded
  - at reorg risk
  - reconciled

## Scope
- Keep rollout contract shape stable.
- Reuse the existing checkpoint source snapshot and per-chain statuses.
- Add a lightweight checkpoint coverage hint to runtime rollout reasons.

## Implementation
- Derive a compact coverage summary from checkpoint source chain summaries:
  - `tracked`
  - `recorded`
  - `reorg_risk`
  - `reorg_reconciled`
- Thread that compact summary into `puller` runtime rollout state and advisory
  reasons as:
  - `checkpoint_coverage`

## Validation
- Run focused `puller` rollout/runtime-progress tests.

## Exit Criteria
- `puller` rollout reasons can report not only per-chain checkpoint posture,
  but also overall checkpoint coverage counts across tracked chains.
- The implementation stays lightweight and additive without introducing a new
  structured rollout payload in this phase.
