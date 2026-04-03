# Phase 395 - Event Processor Control Target Proposal

## Status
Status: Approved

## Summary

Define the next honest writable-control target candidate for `event-processor`
after phases 391-394 strengthened execution ownership and added a real
consume/process seam.

## Problem

The repository now records that `event-processor` is closer to a future
control-plane reopen, but still not ready for default writable control.

That is a good boundary, but it still leaves an important ambiguity:

- what exactly should a future control action target?

Without answering that, future implementation work risks falling back into the
same anti-pattern we have been avoiding:

- copying `puller` pause/resume semantics
- controlling “the whole service”
- exposing a route whose operational effect is too broad or unclear

## Decision

Record the next recommended control target candidate as:

- **consume-loop gate, not processor-runtime stop/start**

Interpretation:

- future writable control, if reopened, should first target the
  topic-scoped consume loop boundary
- it should not directly target the processor lifecycle itself
- it should not present itself as a whole-service execution switch

Why this is the best target:

1. The consume loop is now a real, explicit seam in the microservice entrypoint.
2. Gating intake is narrower and safer than stopping the processor runtime
   itself.
3. It keeps control semantics close to “pause new work intake” rather than
   “tear down execution ownership”.
4. It aligns with the current service shape better than pretending there is one
   monolithic processor-run loop.

## Scope

In scope:

- document the preferred future control target for `event-processor`
- update coverage summary wording for the new targeted reopen path
- clarify why processor lifecycle should not be the first writable target

Out of scope:

- implementing writable routes
- designing full control payloads or auth policy
- declaring control-plane readiness complete

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase395-event-processor-control-target-proposal.md`

## Exit Criteria

- The repository explicitly records one preferred future control target for
  `event-processor`.
- Future reopen work is narrowed to consume-loop gating instead of generic
  pause/resume pressure.
- Processor lifecycle stop/start is explicitly documented as the wrong first
  writable target for this service.
