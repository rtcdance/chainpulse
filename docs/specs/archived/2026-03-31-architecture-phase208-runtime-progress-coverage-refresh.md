# Phase 208 - Runtime Progress Coverage Refresh

## Status
Status: Approved

## Why
- Phase 206 and Phase 207 added the first two lightweight runtime-activity
  signals to execution-service rollout reports:
  - `puller` poll-loop progress
  - `event-processor` Kafka activity
- The current coverage summary still described those services mostly in terms of
  wiring and dependency health, so it no longer reflected the true state of the
  rollout refactor.

## Scope
- Refresh the microservice rollout coverage summary to explicitly distinguish:
  - wiring signals
  - dependency-health signals
  - lightweight runtime-activity/progress signals

## Implementation
- Update the service coverage table for:
  - `puller`
  - `event-processor`
- Add a dedicated runtime-activity coverage section.
- Refresh the “Remaining Gaps” and recommendation sections so the next step is
  framed as “stronger execution progress” rather than just “more health.”

## Validation
- Run spec approval for this summary refresh.

## Exit Criteria
- The coverage summary reflects the new lightweight runtime-activity signals.
- The document makes it clear that these signals are additive, but still not
  equivalent to full checkpoint/lag/progress semantics.
