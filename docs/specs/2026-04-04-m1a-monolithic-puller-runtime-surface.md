Title: M1a Monolithic Puller Runtime Surface
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The monolithic runtime now has a real puller execution path plus a minimal reorg rollback seam, but its puller runtime still lacks a truthful operator-facing surface. Today the monolith can start pullers, yet it does not expose a compact health posture, aggregated error state, or a clear control boundary for that runtime. This keeps M1a incomplete because one of the core monolithic execution components still behaves like an internal detail rather than a visible runtime slice.

## Scope

This slice will:

1. Add a compact monolithic puller runtime status snapshot over the existing per-chain pullers.
2. Surface puller runtime facts in monolithic `/runtime/summary`.
3. Add a read-only monolithic `/runtime/control` route that exposes the current puller control boundary using the shared runtime-control envelope.
4. Keep control semantics explicit: the monolithic puller runtime will expose read-only control status, not writable pause/resume actions, in this slice.
5. Add focused tests for:
   1. puller runtime summary surfacing
   2. monolithic `/runtime/control` route

## Non-Goals

This slice will not:

1. Redesign the puller polling loop.
2. Add writable pause/resume controls to monolithic mode.
3. Introduce distributed control semantics.
4. Add new microservice behavior.

## Selected Approach

Expose the monolithic puller runtime as a compact operator-facing surface rather than trying to force full writable control into the current monolithic polling loop. The runtime summary will report puller posture and reliability hints, while `/runtime/control` will return a shared control envelope with a clear read-only boundary for the polling target.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-puller-runtime-surface.md`

## Decision

Approved for implementation as the fifth M1a runtime-closure slice.

## Implementation Notes

Implemented with:

1. monolithic puller runtime status aggregation over existing per-chain pullers
2. puller runtime posture and read-only control boundary surfacing in monolithic `/runtime/summary`
3. a read-only monolithic `/runtime/control` route using the shared runtime-control envelope
4. additive gateway runtime-route composition support for optional runtime control providers

Primary changed files:

1. `cmd/monolithic/chainpulse/m1a_runtime_wiring.go`
2. `cmd/monolithic/chainpulse/runtime_summary.go`
3. `cmd/monolithic/chainpulse/runtime_control_test.go`
4. `pkg/plugins/api/gateway.go`
5. `pkg/plugins/api/gateway_router_integration.go`

## Verification Summary

Executed checks:

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-puller-runtime-surface.md`

Results:

1. focused monolithic tests passed
2. focused API/gateway tests passed
3. spec approval check passed
