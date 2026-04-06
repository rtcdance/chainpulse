Title: M1b Monolithic Resilience Assessment
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/application/indexing

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1b` was defined as the monolithic fault-tolerance and resilience milestone
that follows the foundational runtime closure from `M1a`. After three focused
implementation slices, the repository now needs a milestone-level assessment to
determine whether the monolithic resilience baseline is complete enough to stop
adding more `M1b` functionality and move into `M1c`.

## Assessment

The monolithic runtime now has the core resilience closure points required by
the `M1b` blueprint boundary:

1. bounded restart/backoff ownership for per-chain pull loops
2. runtime surfacing for pull-loop restart/failure/backoff state
3. a real shared-runtime checkpoint/replay recovery seam
4. per-chain startup recovery probes in monolithic mode
5. runtime surfacing for checkpoint/recovery posture and recent recovery facts
6. top-level fault-aware lifecycle semantics that no longer report healthy
   while puller, recovery, or reorg seams are in watch/degraded states

These changes move monolithic mode from “runnable and inspectable” into
“runnable with truthful minimal resilience ownership”, which is the intended
`M1b` boundary.

## Remaining Gaps

The following items still exist, but they no longer justify keeping work inside
`M1b`:

1. broader observability and gateway hardening
2. dual-mode switching across monolith and microservices
3. production deployment validation, alerting, and rehearsal

Those belong to `M1c`, `M2`, and `M3`, not to the monolithic resilience
milestone.

## Decision

`M1b` is now **stage-complete for the monolithic resilience baseline**.

The repository should stop adding new `M1b` resilience slices and switch the
active implementation focus to `M1c`.

## Verification Summary

This assessment is based on the focused `M1b` verification chain already
completed across the three implementation slices, including:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `go test -short ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-pull-loop-resilience.md`
4. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-checkpoint-recovery-closure.md`
5. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-degraded-fault-semantics.md`
