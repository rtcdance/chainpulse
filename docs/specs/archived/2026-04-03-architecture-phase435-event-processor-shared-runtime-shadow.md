Title: Phase 435 Event Processor Shared Runtime Shadow
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/event-processor, pkg/application/indexing, pkg/application/bootstrap

## Status

Status: Approved

## Context

The monolithic entrypoint now uses a more honest shared indexing runtime slice
with checkpoint, idempotency, failure routing, and replay wiring. The
`event-processor` microservice still owns its local consume/process seam but
does not yet reuse the shared indexing runtime contract.

## Problem Statement

`event-processor` currently exposes processor lifecycle and consume-loop
ownership, but not a shared indexing runtime contract. That weakens the
blueprint-aligned claim that monolith and microservice modes are converging on
one indexing core.

## Scope

- add an additive shared indexing runtime shadow to `event-processor`
- forward successfully processed events into the shared runtime without
  changing the primary processor behavior
- expose shared-runtime shadow status through `event-processor /runtime/summary`
- keep all changes additive and default-safe

## Non-Goals

- no replacement of the current processor runtime
- no change to the primary success/failure semantics of `ProcessEvent`
- no distributed replay or durable DLQ
- no change to the minimal runnable-app baseline

## Selected Approach

Wrap the existing processor runtime with a shadow adapter that:

- delegates primary event processing to the current processor
- lazily provisions per-chain shared runtimes using in-memory ports
- forwards successfully processed events into the shared runtime contract
- records additive shadow status and surfaces it in runtime summary

This keeps the microservice runtime behavior intact while beginning real
runtime convergence on the shared indexing core.

## Risks

- shared shadow runtime can drift from primary processor behavior if not kept
  explicitly additive
- chain identity inference must remain conservative when event chain fields are
  sparse

## Rollback Plan

- remove the shared runtime shadow wrapper from `event-processor`
- keep the existing processor lifecycle and consume/process seam unchanged
- remove the additive runtime summary section

## Test and Verification Plan

- run focused event-processor tests
- run focused shared application/bootstrap tests if helper wiring changes
- run `go test -short ./cmd/microservices/event-processor/... ./pkg/application/...`
- run the spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-03-architecture-phase435-event-processor-shared-runtime-shadow.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOCACHE=/tmp/chainpulse-go-build-cache; go test -short ./cmd/microservices/event-processor/... ./pkg/application/...`

## Review Notes

- Approved as the smallest additive microservice step toward a shared indexing
  runtime contract without destabilizing the current runnable path.

## Implementation Summary

`event-processor` now wraps its primary processor runtime with an additive
shared-runtime shadow adapter. Successfully processed events are lazily
forwarded into per-chain shared indexing runtimes backed by in-memory ports.
The runtime summary now exposes the shared-runtime shadow status and the local
processor runtime now uses a started in-memory database plugin so the primary
`ProcessEvent` path remains genuinely runnable.

## Final Verification

The following focused verification passed:

- `unset GOROOT; export GOCACHE=/tmp/chainpulse-go-build-cache; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; go test -short ./cmd/microservices/event-processor/...`
- `unset GOROOT; export GOCACHE=/tmp/chainpulse-go-build-cache; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; go test -short ./pkg/application/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-03-architecture-phase435-event-processor-shared-runtime-shadow.md`
