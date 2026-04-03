# Phase 198 - Event Processor Runtime Health Advisory

## Status
Status: Approved

## Why
- Phase 191 and Phase 192 gave `event-processor` a runtime-derived rollout
  producer, but its fully wired posture still collapsed all processing
  dependency health into a single `runtime-wired` advisory.
- The next higher-value step is to let the rollout report distinguish healthy,
  degraded, unhealthy, and unknown processing dependency health without changing
  the external contract.

## Scope
- Keep the rollout report contract unchanged.
- Extend `event-processor` runtime-derived rollout state with processing
  dependency health signals.
- Refine fully wired advisory status and reason mapping from those health
  signals.

## Implementation
- Extend `eventProcessorRolloutRuntimeState` with health status and message
  fields for:
  - event store
  - metadata store
  - Kafka
- Add helper logic that:
  - normalizes dependency health statuses
  - maps fully wired runtime health into:
    - `runtime-wired`
    - `runtime-wired-degraded`
    - `runtime-wired-unhealthy`
    - `runtime-wired-health-unknown`
- Keep partially wired states on the existing
  `partial-runtime-wiring` contract.
- Append component-specific health status/message details and a health-aware
  posture hint to the advisory reason.

## Validation
- Add focused producer coverage for:
  - fully wired healthy
  - fully wired degraded
  - health-aware completeness classification
- Run focused `event-processor` rollout producer tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Fully wired `event-processor` rollout reports distinguish runtime health
  quality without changing the external payload shape.
- Partially wired rollout behavior remains unchanged.
