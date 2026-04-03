# Phase 224 - Puller Persisted Checkpoint Source

## Status
Status: Approved

## Why
- Phase 223 added a lightweight checkpoint cadence derived from processed block
  height and checkpoint interval.
- That still did not distinguish between:
  - a checkpoint that is merely due by cadence
  - a checkpoint that has actually been recorded by runtime state
- The next useful slice is to add a minimal checkpoint source abstraction so
  `puller` rollout can expose real checkpoint persistence posture without
  pretending a full external checkpoint backend is already wired.

## Scope
- Keep rollout contract shape stable.
- Add a minimal `puller` checkpoint source abstraction.
- Wire the polling loop to record checkpoint boundaries into that source.
- Extend shared poll-progress details with persisted-checkpoint facts.
- Keep the implementation local and lightweight for now.

## Implementation
- Add a dedicated `pullerCheckpointSource` with an in-process runtime-backed
  implementation built on the existing block-height tracker/recovery manager.
- Extend `MultiChainDataPuller` with per-chain processed block export so the
  polling loop can detect checkpoint-boundary chains.
- Persist checkpoints when a chain's processed block lands on a configured
  checkpoint interval boundary.
- Extend shared poll-progress details with:
  - `persisted_checkpoint_block`
  - `blocks_since_checkpoint`
  - `persisted_checkpoint_state`
- Thread those details into `puller` runtime rollout state and rollout reason
  coverage.

## Validation
- Run focused `puller` rollout/runtime-progress tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `puller` rollout progress can distinguish derived checkpoint cadence from
  recorded checkpoint state.
- The service exposes whether the latest runtime checkpoint is:
  - missing
  - present/current
  - behind processed progress
- The implementation remains additive and does not require wiring a new
  external checkpoint backend in this phase.
