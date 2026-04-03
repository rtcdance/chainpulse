# Phase 199 - Puller Runtime Health Advisory

## Status
Status: Approved

## Why
- Phase 193 gave `puller` a runtime-derived rollout producer, but its fully
  wired posture still treated all local ingestion dependencies as a single
  `runtime-wired` state.
- The next practical improvement is to fold the two clearest real dependency
  health signals into that fully wired posture:
  - database health
  - Kafka health

## Scope
- Keep the rollout report contract unchanged.
- Extend `puller` runtime-derived rollout state with database and Kafka health
  signals.
- Refine fully wired advisory status and reasons from those health signals.

## Implementation
- Extend `pullerRolloutRuntimeState` with:
  - `database` health status/message
  - `kafka` health status/message
- Add helper logic that maps fully wired runtime health into:
  - `runtime-wired`
  - `runtime-wired-degraded`
  - `runtime-wired-unhealthy`
  - `runtime-wired-health-unknown`
- Keep partially wired rollout behavior on the existing
  `partial-runtime-wiring` contract.
- Append component-specific health status/message details and health-aware
  posture hints to advisory reasons.

## Validation
- Add focused producer coverage for:
  - fully wired healthy
  - fully wired degraded
  - health-aware completeness classification
- Run focused `puller` rollout producer tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Fully wired `puller` rollout reports distinguish dependency health quality
  without changing the external payload shape.
- Partially wired rollout behavior remains unchanged.
