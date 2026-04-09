Title: M2 Monolithic Upstream Query Bridge
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2-5` made the monolithic gateway surface mode-aware, but under `microservice`
intent it currently falls back to a runtime/operator-only boundary. That is
honest, but it still stops short of a real cross-boundary transport slice.

The next most valuable `M2` step is to let monolithic mode intentionally bridge
read-only query traffic to `api-service` through the existing upstream query
bridge, instead of keeping all business routes absent.

## Scope

This slice will:

1. add a monolithic config seam for upstream query services
2. let `microservice` intent expose query routes through the existing upstream
   bridge instead of only runtime/operator routes
3. keep subscription ownership withheld for now
4. surface the monolithic upstream query bridge state in runtime summary

## Non-Goals

This slice will not:

1. add write/control RPCs between services
2. proxy subscriptions
3. claim full `M2` completion
4. change microservice entrypoints

## Selected Approach

Reuse the existing gateway upstream query bridge that already powers the
microservice API gateway. Under `microservice` intent, the monolithic gateway
will attach query routes through configured upstream query endpoints while still
withholding in-process query/subscription ownership.

This gives `M2` a real cross-boundary transport slice without pretending the
full dual-mode cutover is finished.

## Quality Gates

1. `go test -short ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-upstream-query-bridge.md`

## Decision

Approved for implementation as the next `M2` transport-boundary slice.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/m2_gateway_surface.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic gateway now supports a read-only upstream query bridge under
`microservice` intent and surfaces the bridge state through runtime summary.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-upstream-query-bridge.md`
