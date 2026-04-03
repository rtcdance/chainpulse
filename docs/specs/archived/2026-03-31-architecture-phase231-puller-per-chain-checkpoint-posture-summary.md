# Phase 231 - Puller Per-Chain Checkpoint Posture Summary

## Status
Status: Approved

## Why
- Phase 227-230 made `puller` checkpoint rollout easier to scan with:
  - per-chain checkpoint summaries
  - freshness markers
  - compact global coverage counts
  - compact global coverage posture
- The next useful slice is to add a shorter per-chain posture summary so
  rollout can answer not only the detailed chain checkpoint facts, but also a
  faster operator-readable conclusion for each tracked chain.

## Scope
- Keep the existing detailed per-chain checkpoint summary unchanged.
- Derive a compact per-chain checkpoint posture summary from:
  - checkpoint status
  - freshness
- Thread that compact per-chain posture summary into `puller` runtime rollout
  reasons.

## Implementation
- Add a helper that maps each chain into a compact posture such as:
  - `recorded-healthy`
  - `recorded-stale`
  - `risk`
  - `risk-stale`
  - `reconciled`
  - `reconciled-stale`
- Expose the result through `puller` runtime rollout state as:
  - `CheckpointChainPostureSummary`
- Append that result to advisory reasons as:
  - `checkpoint_chain_posture_summary`

## Validation
- Run focused `puller` rollout/runtime-progress tests.

## Exit Criteria
- `puller` rollout reasons can now report both:
  - detailed per-chain checkpoint summaries
  - compact per-chain checkpoint posture summaries
- The change stays additive and does not replace the existing detailed
  checkpoint summary.
