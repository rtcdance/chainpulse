Title: M2 Monolithic Gateway Surface Profile Selection
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2-1` through `M2-4` made deployment mode affect cmd parsing, adapter profile,
indexing storage, and query surface selection. The monolithic gateway exposure
surface still does not honor the same boundary: it is wired as if the monolith
always owns the full query/subscription API, even when `microservice` intent is
selected.

That leaves the transport/runtime boundary only partially real.

## Scope

This slice will:

1. Make monolithic gateway API surface selection depend on deployment mode.
2. Keep `monolithic` mode on the current full in-process gateway surface.
3. Let `microservice` intent retain health/runtime/operator routes while
   withholding monolithic query/subscription ownership.
4. Avoid registering query/subscription routes when the corresponding runtime
   handlers are absent.
5. Surface the selected gateway exposure posture through runtime summary.

## Non-Goals

This slice will not:

1. introduce real RPC forwarding from monolithic mode to microservices
2. claim full `M2` completion
3. change microservice entrypoints
4. redesign shared API contracts beyond the minimum route-registration boundary

## Selected Approach

Introduce a narrow cmd-layer gateway surface resolver and tighten route
registration so the runtime route inventory reflects actual handler ownership.
This makes the transport boundary honest without pretending the monolith now
delegates all traffic to external microservices.

## Quality Gates

1. `go test -short ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-gateway-surface-profile-selection.md`

## Decision

Approved for implementation as the fifth `M2` slice.

## Implementation Notes

Implemented in:

- `pkg/plugins/api/gateway_router_integration.go`
- `cmd/monolithic/chainpulse/m2_gateway_surface.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic gateway surface is now deployment-mode-aware. `monolithic` keeps
the full in-process query/runtime exposure, while `microservice` intent keeps a
runtime/operator-only boundary and stops overclaiming monolithic API ownership.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-gateway-surface-profile-selection.md`
