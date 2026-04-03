# Phase 230 - Puller Checkpoint Coverage Posture

## Status
Status: Approved

## Why
- Phase 229 added compact checkpoint coverage counts for `puller`:
  - `tracked`
  - `recorded`
  - `reorg_risk`
  - `reorg_reconciled`
- The next useful slice is to compress those counts into a smaller operator-ready
  conclusion so rollout can answer whether checkpoint coverage currently looks:
  - healthy
  - partial
  - risky
  - reconciled

## Scope
- Keep the existing compact coverage count string unchanged.
- Derive a lightweight checkpoint coverage posture from the current per-chain
  checkpoint statuses.
- Thread that posture into `puller` runtime rollout reasons.

## Implementation
- Add a small helper that derives one of:
  - `coverage-healthy`
  - `coverage-partial`
  - `coverage-risk`
  - `coverage-reconciled`
- Expose the result through `puller` runtime rollout state as:
  - `CheckpointCoveragePosture`
- Append that result to advisory reasons as:
  - `checkpoint_coverage_posture`

## Validation
- Run focused `puller` rollout/runtime-progress tests.

## Exit Criteria
- `puller` rollout reasons can report both:
  - detailed checkpoint coverage counts
  - a more compact checkpoint coverage posture
- The change remains additive and does not alter the existing rollout contract
  shape.
