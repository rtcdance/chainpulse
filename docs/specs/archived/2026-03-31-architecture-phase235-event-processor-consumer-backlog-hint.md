# Phase 235 - Event-Processor Consumer Backlog Hint

## Status
Status: Approved

## Why
- `event-processor` rollout already exposes:
  - raw consumer lag
  - compact lag severity
  - compact consumer progress posture
- The next useful slice is to compress those signals into a more executable
  operator hint so rollout can answer not only what backlog looks like, but
  what kind of next step that backlog suggests.

## Scope
- Keep the existing raw lag, lag severity, and consumer progress posture
  unchanged.
- Derive a lightweight operator-facing backlog hint from:
  - consumer progress posture
  - consumer lag severity
- Thread that hint into `event-processor` runtime rollout reasons.

## Implementation
- Add a helper that derives compact hints such as:
  - continue observing drain progression
  - monitor drain rate
  - prioritize drain and investigate throughput
- Expose that result through `event-processor` runtime rollout state as:
  - `ConsumerBacklogHint`
- Append that result to advisory reasons as:
  - `consumer_backlog_hint`

## Validation
- Run focused `event-processor` rollout/runtime-progress tests.

## Exit Criteria
- `event-processor` rollout reasons can now report both:
  - backlog facts
  - a compact operator-facing backlog hint
- The change stays additive and preserves the current rollout contract.
