# Phase 389 - Event Processor Control Feasibility Decision

## Status
Status: Approved

## Summary

Record a control-plane feasibility decision for `event-processor` after the
`puller` writable control pilot was established.

## Problem

After phase 387, the repository now has a real writable execution control
pilot on `puller`. The natural next question is whether the same pause/resume
control slice should be copied to `event-processor`.

The risk is that `event-processor` does not currently expose the same kind of
clearly owned execution loop:

- `puller` has a concrete polling ticker loop in `runPullerLoop(...)`
- `event-processor` currently wires runtime surfaces around dependency health
  and Kafka plugin lifecycle, but not around a single, clearly owned
  processor-run loop

Without that ownership boundary, copying the writable control slice would risk
creating a route that looks operationally meaningful while controlling an
unclear subset of runtime behavior.

## Decision

Record the current decision as:

- **do not yet mirror the writable pause/resume control slice into
  `event-processor`**

Treat `event-processor` writable control as blocked on a stronger execution
ownership boundary, such as:

- explicit processor-run wiring
- explicit consume/process lifecycle ownership
- a clearly scoped action target that is narrower than “the whole service”

## Scope

In scope:

- update the execution control-plane assessment in the coverage summary
- record the current no-go decision for `event-processor` writable control
- define reopen conditions for future `event-processor` control work

Out of scope:

- implementing new `event-processor` control routes
- redesigning Kafka MQ lifecycle semantics
- wiring the dormant processor service into the microservice entrypoint

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase389-event-processor-control-feasibility-decision.md`

## Exit Criteria

- The repository explicitly records why `puller` writable control is not yet
  being copied to `event-processor`.
- Future `event-processor` control-plane work is framed as a deliberate reopen
  tied to stronger execution ownership wiring.
