Title: M1b Monolithic Degraded Fault Semantics
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

After the first two `M1b` slices, the monolithic runtime now owns pull-loop
restart/backoff and startup checkpoint/recovery probes. But the top-level
`/runtime/summary` lifecycle classification still reports `healthy` whenever
the shared runtime is started, the gateway routes are enabled, and ownership is
present. That is no longer truthful enough: a monolith with pull loops in
backoff recovery or a failed recovery probe should not still present itself as
fully healthy.

## Scope

This slice will:

1. Add top-level monolithic fault posture classification.
2. Make `runtime_mode`, `runtime_posture`, and `component_state` sensitive to
   puller/recovery/reorg degraded states.
3. Add an additive top-level runtime reliability hint surface.
4. Lock the new semantics with focused tests.

## Non-Goals

This slice will not:

1. Change writable control semantics.
2. Redesign health handler endpoints outside `/runtime/summary`.
3. Introduce distributed fault coordination.
4. Add new transport protocols.

## Selected Approach

Keep the existing summary shape and add an additive top-level fault layer.
Classify monolithic runtime truth from:

1. shared indexing lifecycle,
2. gateway runtime wiring,
3. ownership presence,
4. puller posture,
5. recovery posture,
6. reorg posture.

Then derive a more honest top-level posture and component state:

- `ready` only when those surfaces are healthy,
- `watch` when recovery is in progress or partial,
- `degraded` when an active resilience seam reports failure.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-degraded-fault-semantics.md`

## Decision

Approved for implementation as the third `M1b` resilience slice.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic runtime summary now derives top-level lifecycle truth from the
real resilience surfaces instead of only from shared-runtime start state plus
gateway wiring.

Additive top-level fields now include:

- `fault_posture`
- `reliability_hint`

And `runtime_mode`, `runtime_posture`, and `component_state` now truthfully
degrade when puller, recovery, or reorg seams are in watch/degraded states.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-degraded-fault-semantics.md`
