# Phase 225 - Puller Checkpoint Reorg Risk

## Status
Status: Approved

## Why
- Phase 224 separated derived checkpoint cadence from the latest recorded
  checkpoint source state.
- That still did not say whether a recorded checkpoint might already be unsafe
  because chain progress appears to have moved backwards.
- The next useful slice is to expose a minimal reorg-aware checkpoint risk
  signal without pretending that the full reorg handler is already wired into
  the puller main loop.

## Scope
- Keep rollout contract shape stable.
- Extend shared poll-progress details with checkpoint reorg-risk facts.
- Let the `puller` checkpoint source observe processed-block regressions.
- Surface a minimal rollout signal when recorded checkpoint state may have been
  invalidated by reorg-like progress.

## Implementation
- Extend shared poll-progress details with:
  - `reorg_checkpoint_state`
  - `reorg_checkpoint_block`
- Extend the runtime-backed `pullerCheckpointSource` so it can observe chain
  progress and mark a checkpoint as reorg-risk when processed block height
  regresses below the recorded checkpoint.
- Thread that signal into:
  - `puller` poll-progress snapshot assembly
  - runtime rollout state
  - rollout reason coverage

## Validation
- Run focused `puller` rollout/runtime-progress tests.
- Run focused shared execution-progress tests in `pkg/plugins/api`.

## Exit Criteria
- `puller` rollout progress can express whether a recorded checkpoint is:
  - clear
  - at reorg risk
- The implementation stays additive and lightweight, using current runtime
  progress facts instead of claiming full persisted reorg recovery wiring.
