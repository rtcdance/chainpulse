# Phase 206 - Puller Execution Progress Signal

## Status
Status: Approved

## Why
- `puller` rollout reports already reflect wiring and dependency health, but
  they still do not expose any direct evidence that the polling loop has
  actually executed.
- The next practical signal is lightweight execution progress:
  - poll count
  - last poll timestamp

## Scope
- Keep the rollout report contract unchanged.
- Add a small puller loop progress tracker.
- Fold execution progress into `puller` runtime-derived rollout reasons.

## Implementation
- Add a runtime progress tracker that records:
  - total poll count
  - last poll unix timestamp
- Wire the tracker into `runPullerLoop(...)`.
- Pass progress snapshots into the `puller` runtime rollout state builder.
- Append progress details to runtime-derived advisory reasons:
  - `poll_count`
  - `last_poll_unix`

## Validation
- Add focused runtime support coverage for progress snapshot ingestion.
- Add focused wired-handler coverage showing rollout reasons include execution
  progress details.
- Run focused `puller` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `puller` runtime-derived rollout reports expose lightweight execution
  progress without changing the external payload shape.
- The new signal remains additive and does not redefine existing rollout enums.
