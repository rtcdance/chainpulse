Title: Phase 434 Monolithic Runtime Failure Replay Closure
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: pkg/application/indexing, pkg/application/bootstrap, cmd/monolithic/chainpulse

## Status

Status: Approved

## Context

The monolithic entrypoint now exposes a compact `/runtime/summary` route, but
the shared indexing runtime still relies on a no-op failure router and does not
configure a replay source. That means checkpoint, failure isolation, and replay
are still only partially closed in the first shared indexing runtime slice.

## Problem Statement

The runtime contract already includes:

- `CheckpointStore`
- `IdempotencyStore`
- `FailureRouter`
- `ReplaySource`

But the monolithic bootstrap currently wires only checkpoint/idempotency as
real in-memory adapters. Failure routing is a no-op and replay is absent.
That leaves the monolithic runtime weaker than the blueprint's minimum
indexer-operational expectations.

## Scope

- replace the monolithic no-op failure router with a real in-memory failure journal
- wire the same journal as a replay source for additive runtime recovery flow
- expose additive runtime status facts for checkpoint / idempotency / failure / replay wiring
- surface the new runtime facts through the monolithic runtime summary

## Non-Goals

- no durable DLQ storage
- no distributed replay orchestration
- no new microservice rewiring
- no behavior change to the existing legacy indexing path

## Selected Approach

Add a monolithic in-memory failure journal that implements both:

- `FailureRouter`
- `ReplaySource`

Then extend the shared runtime status so it can report whether checkpoint,
idempotency, failure routing, and replay are configured, plus the latest
checkpoint facts and duplicate-skip counters.

This keeps the change additive while turning the current shared runtime from a
partial skeleton into a more honest minimum operational slice.

## Risks

- the in-memory journal is still process-local and intentionally non-durable
- replay semantics remain additive and minimal, not a full production replay model

## Rollback Plan

- restore the monolithic no-op failure router
- remove replay source wiring from monolithic bootstrap
- keep runtime summary additive and read-only

## Test and Verification Plan

- run focused shared runtime tests
- run focused monolithic bootstrap/runtime summary tests
- run `go test -short ./pkg/application/... ./cmd/monolithic/chainpulse/...`
- run the spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-03-architecture-phase434-monolithic-runtime-failure-replay-closure.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOCACHE=/tmp/chainpulse-go-build-cache; go test -short ./pkg/application/... ./cmd/monolithic/chainpulse/...`

## Review Notes

- Approved as the smallest runtime-closure step that materially improves the
  shared indexing core without reopening a broader microservice rewrite.

## Implementation Summary

The monolithic shared indexing runtime now uses a real in-memory failure
journal for failure routing and reuses the same journal as a replay source.
Shared runtime status now reports checkpoint / idempotency / failure / replay
configuration facts, duplicate-skip counters, and the latest checkpoint facts.
The monolithic `/runtime/summary` route now exposes those runtime-closure facts
through the indexing section.

## Final Verification

The following focused verification passed:

- `unset GOROOT; export GOCACHE=/tmp/chainpulse-go-build-cache; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; go test -short ./pkg/application/...`
- `unset GOROOT; export GOCACHE=/tmp/chainpulse-go-build-cache; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; go test -short ./cmd/monolithic/chainpulse/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-03-architecture-phase434-monolithic-runtime-failure-replay-closure.md`
