# Phase 222 - Puller Runtime Block Progress Wiring

## Status
Status: Approved

## Why
- Phase 221 gave the `puller` rollout path a block-progress carrier, but that
  carrier still depended on direct test injection.
- The next useful slice is to wire the carrier to a real runtime source that
  already exists in the service shape: the registered pullers inside
  `MultiChainDataPuller`.

## Scope
- Keep rollout semantics unchanged.
- Add a minimal `MultiChainDataPuller` helper for highest observed and
  processed block tracking across registered pullers.
- Wire `runPullerLoop(...)` to capture those block-progress facts into the
  existing rollout runtime progress tracker.

## Implementation
- Add `MultiChainDataPuller` helpers for:
  - highest latest block
  - highest processed block
- Add focused `pullers` tests for those helpers.
- Extract `capturePullerBlockProgress(...)` into a dedicated helper file so it
  can be verified without pulling the full `main.go` dependency surface into
  focused tests.
- Update the polling loop to record:
  - observed block height
  - processed block height
  through the existing rollout runtime progress tracker.
- Add focused `puller` tests proving block progress is captured from a real
  `MultiChainDataPuller` source.

## Validation
- Run focused `pullers` file-based tests.
- Run focused `puller` rollout/runtime-progress tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `puller` block-progress carrier is no longer test-only.
- The polling loop can source observed and processed block facts from the real
  multi-chain runtime abstraction without changing higher-level rollout policy
  semantics.
