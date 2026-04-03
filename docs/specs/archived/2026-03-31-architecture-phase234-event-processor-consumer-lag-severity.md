# Phase 234 - Event-Processor Consumer Lag Severity

## Status
Status: Approved

## Why
- `event-processor` rollout already exposes:
  - consumer lag
  - consumer progress posture
  - consumer progress state
- The next useful slice is to make backlog size easier to judge at a glance by
  adding a compact lag severity classification on top of the existing raw lag
  value.

## Scope
- Keep the existing raw `consumer_lag` fact unchanged.
- Derive a lightweight lag severity from current lag size.
- Thread that lag severity into `event-processor` runtime rollout reasons.

## Implementation
- Add a helper that classifies lag into:
  - `backlog-low`
  - `backlog-medium`
  - `backlog-high`
- Expose that result through `event-processor` runtime rollout state as:
  - `ConsumerLagSeverity`
- Append that result to advisory reasons as:
  - `consumer_lag_severity`

## Validation
- Run focused `event-processor` rollout/runtime-progress tests.

## Exit Criteria
- `event-processor` rollout reasons can now report both:
  - raw consumer lag
  - compact lag severity
- The change stays additive and preserves the existing rollout contract.
