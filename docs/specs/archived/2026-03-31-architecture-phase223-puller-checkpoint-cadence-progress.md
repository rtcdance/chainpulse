# Phase 223 - Puller Checkpoint Cadence Progress

## Status
Status: Approved

## Why
- Phase 222 wired `puller` block progress to the real multi-chain runtime
  abstraction.
- That still only exposed raw observed/processed block facts.
- The next useful slice is to derive a lightweight checkpoint-aware progress
  signal from the existing `STATE_CHECKPOINT_INTERVAL` configuration and the
  processed block height.

## Scope
- Keep rollout semantics unchanged.
- Extend shared poll-progress details with checkpoint cadence fields.
- Derive checkpoint cadence from:
  - processed block height
  - configured checkpoint interval
- Thread that derived checkpoint cadence into `puller` rollout state and reason
  details.

## Implementation
- Extend shared poll-progress details with:
  - `checkpoint_progress_state`
  - `blocks_until_checkpoint`
- Add a lightweight classifier for `puller` checkpoint cadence:
  - `checkpoint-uninitialized`
  - `checkpoint-pending`
  - `checkpoint-due`
- Extend `pullerRolloutRuntimeConfig` to carry `CheckpointInterval` into
  rollout-state assembly.
- Update focused `puller` tests to cover:
  - runtime rollout state
  - poll-progress snapshot assembly
  - checkpoint cadence classification
  - rollout reason coverage

## Validation
- Run focused `puller` rollout/runtime-progress tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `puller` rollout progress can describe checkpoint cadence in addition to raw
  block progress.
- The signal remains derived and lightweight, without pretending that a full
  persisted checkpoint backend is already wired into the service.
