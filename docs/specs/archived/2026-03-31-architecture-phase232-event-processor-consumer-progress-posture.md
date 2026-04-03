# Phase 232 - Event-Processor Consumer Progress Posture

## Status
Status: Approved

## Why
- `event-processor` rollout already exposes:
  - kafka activity
  - consumer lag
  - consumer offset
  - consumer progress state
- The next useful slice is to compress those consumer progress facts into a
  smaller operator-readable posture so execution progress is easier to scan at
  the same level as recent `puller` checkpoint posture work.

## Scope
- Keep existing consumer lag/offset/activity facts unchanged.
- Derive a lightweight compact consumer progress posture from:
  - consumer progress state
  - kafka activity state
  - lag
  - offset
- Thread that compact posture into `event-processor` runtime rollout reasons.

## Implementation
- Add a helper that derives one of:
  - `consumer-idle`
  - `consumer-backlog`
  - `consumer-advancing`
  - `consumer-active`
  - `consumer-watch`
  - `consumer-monitoring`
- Expose that result through `event-processor` runtime rollout state as:
  - `ConsumerProgressPosture`
- Append that result to advisory reasons as:
  - `consumer_progress_posture`

## Validation
- Run focused `event-processor` rollout/runtime-progress tests.

## Exit Criteria
- `event-processor` rollout reasons can now report both:
  - detailed consumer progress facts
  - a compact consumer progress posture
- The change stays additive and does not alter the existing rollout contract
  shape.
