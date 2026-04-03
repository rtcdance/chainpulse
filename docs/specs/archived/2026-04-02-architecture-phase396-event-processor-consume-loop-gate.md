# Phase 396 - Event Processor Consume Loop Gate

## Status
Status: Approved

## Summary

Implement the first narrow writable control slice for `event-processor` by
adding an intake-side consume-loop gate instead of broad processor-runtime or
whole-service pause semantics.

## Problem

Phase 395 established that the preferred first writable target for
`event-processor` should be consume-loop gating.

Without implementing that target, the repository still has:

- stronger execution ownership
- a real consume/process seam
- a clear preferred target

but no actual proof that the target can be controlled safely and honestly.

## Decision

Add a minimal writable control slice that:

- pauses new consume-loop intake
- resumes consume-loop intake
- reports the current intake-gate state through a dedicated runtime control
  route

Keep the scope narrow:

- do not stop the processor runtime
- do not present whole-service pause/resume semantics
- do not redesign Kafka offset or rebalance behavior

## Scope

In scope:

- consume-loop gate state in `event-processor`
- runtime control routes for intake pause/resume
- runtime summary/readiness integration for paused intake state
- focused tests for intake-gate behavior and routes

Out of scope:

- processor lifecycle stop/start control
- durable control coordination
- per-topic selective control policy

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase396-event-processor-consume-loop-gate.md`

## Exit Criteria

- `event-processor` exposes a real writable control slice for consume-loop
  intake.
- The control slice is clearly narrower than whole-service pause/resume.
- Runtime surfaces report paused intake state honestly and consistently.

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...` passed
