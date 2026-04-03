# Phase 221 - Puller Block Progress Carrier

## Status
Status: Approved

## Why
- The `puller` rollout report already exposed lightweight poll-loop activity,
  but that only answered whether the loop was alive.
- The next useful slice is to let the `puller` progress path carry minimal
  block-progress facts so later wiring can distinguish "alive" from
  "observing/processing block height".

## Scope
- Keep rollout semantics unchanged.
- Extend the shared poll-progress contract to carry optional block-progress
  facts.
- Extend the `puller` runtime progress tracker to record observed and processed
  block heights.
- Thread those facts into `puller` rollout reason details.

## Implementation
- Extend `RolloutPollProgressSnapshot` with:
  - `observed_block`
  - `processed_block`
  - `block_gap`
- Extend shared execution-progress reason helpers and tests to cover the new
  poll-progress fields.
- Extend `pullerLoopRuntimeProgress` with:
  - observed block tracking
  - processed block tracking
- Extend `buildPullerPollProgressSnapshot(...)` and runtime rollout state to
  carry the derived block gap.
- Add focused `puller` tests proving the carrier can surface block progress
  through rollout runtime state and reason coverage.

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- The shared poll-progress contract can carry minimal block-progress facts.
- `puller` rollout reason coverage can surface block-progress details without
  changing higher-level rollout policy or readiness semantics.
