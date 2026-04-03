# Phase 394 - Event Processor Control Readiness Refresh

## Status
Status: Approved

## Summary

Refresh the writable-control readiness decision for `event-processor` after
phase 393 added a real consume/process seam to the microservice entrypoint.

## Problem

The repository now has a more mature `event-processor` execution boundary than
it had in phases 389-392:

- processor lifecycle is wired
- idempotency lifecycle is wired
- consume/process bridging now exists
- runtime surfaces expose processor and consume-loop ownership facts

Without a new assessment, the repository is likely to drift into one of two
bad interpretations:

- assuming the old no-go still applies unchanged
- assuming the new seam automatically makes pause/resume or similar writable
  control honest and ready

## Decision

Record the current control-readiness state for `event-processor` as:

- **approaching a narrower control target, but not yet control-ready**

Interpretation:

- the service now owns a materially more real execution seam than before
- that seam is good enough to justify a future control-plane reopen
- but the seam is still not yet shaped into one clearly owned, operator-safe,
  low-risk action target equivalent to the `puller` polling loop

Therefore:

- the line should move from “generic no-go” to “targeted reopen candidate”
- but writable control should still remain deferred until a narrower action
  boundary is intentionally selected and implemented

## Scope

In scope:

- refresh execution-control wording in the coverage summary
- record the post-phase-393 maturity level for `event-processor`
- define the next honest reopen condition for writable control work

Out of scope:

- implementing writable control routes
- redesigning Kafka rebalance or offset coordination
- claiming a shared execution-service control baseline

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase394-event-processor-control-readiness-refresh.md`

## Exit Criteria

- The repository explicitly records that `event-processor` is closer to a
  control target than before.
- The repository also explicitly records that writable control is still not yet
  honest at the current shape.
- Future reopen work is framed around selecting a narrower execution action
  target instead of blindly copying `puller` semantics.
