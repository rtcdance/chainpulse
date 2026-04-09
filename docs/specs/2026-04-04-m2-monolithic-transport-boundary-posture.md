Title: M2 Monolithic Transport Boundary Posture
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2-1` through `M2-5` made deployment mode affect parsing, adapter profile,
storage, query surface, and gateway exposure. The remaining gap is that the
monolithic `deployment` summary still reports only the intended transport
boundary string. It does not yet say whether the selected boundary is actually:

1. in-process and ready
2. runtime/operator-only by design
3. bridged but unavailable
4. bridged and healthy

That leaves the shared wiring / transport boundary story one step short of an
honest dual-mode baseline.

## Scope

This slice will:

1. classify the effective monolithic transport boundary posture
2. use real gateway bridge counts when `microservice` intent selects an
   upstream query bridge
3. surface `transport_boundary_posture` and `transport_boundary_hint` through
   the monolithic `deployment` summary
4. print the selected transport boundary in startup output

## Non-Goals

This slice will not:

1. declare `M2` completed
2. redesign cross-service transport contracts
3. add writable transport control
4. change microservice entrypoints

## Selected Approach

Add a small monolithic transport-boundary classifier that combines the selected
boundary with real gateway bridge attachment/availability facts. This keeps the
implementation narrow while making the dual-mode transport seam more truthful.

## Quality Gates

1. `GOCACHE=/tmp/chainpulse-go-build-cache go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-transport-boundary-posture.md`

## Decision

Approved for implementation as the next `M2` slice after the initial dual-mode
baseline assessment.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/m2_transport_boundary.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic `deployment` summary now exposes both the selected
`transport_adapter_boundary` and the effective `transport_boundary_posture`
derived from gateway bridge attachment and availability facts.

## Verification Summary

The following checks passed after implementation:

1. `GOCACHE=/tmp/chainpulse-go-build-cache go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-transport-boundary-posture.md`
