Title: M1b Monolithic Pull Loop Resilience
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/pullers

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1a` closed the monolithic foundational runtime path, but the puller loop still behaves as a single-shot execution. If `Poll(...)` exits with an unexpected error, the monolithic runtime only logs the failure and the chain poll loop dies. That is below the intended `M1b` fault-tolerance boundary, because the monolith has no restart/backoff ownership for one of its core execution loops.

## Scope

This slice will:

1. Add restart ownership to monolithic per-chain pull loops.
2. Apply bounded backoff before restarting a failed poll loop.
3. Surface restart/error/backoff facts in the monolithic puller runtime summary.
4. Add focused tests for restart tracking and resilience posture.

## Non-Goals

This slice will not:

1. Redesign the HTTPS puller itself.
2. Add distributed supervision across processes.
3. Change writable control semantics.
4. Implement full checkpoint-recovery orchestration.

## Selected Approach

Wrap the existing monolithic `Poll(...)` loop in a small restart supervisor owned by `monolithicPullerRuntime`. Unexpected poll exits will record runtime state, wait with bounded exponential backoff, and then restart the chain loop while the parent context is still active.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-pull-loop-resilience.md`

## Decision

Approved for implementation as the first `M1b` resilience slice.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/m1a_runtime_wiring.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/m1a_runtime_wiring_test.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic puller runtime now owns restart supervision for each per-chain
poll loop. Unexpected `Poll(...)` exits are classified as loop failures,
recorded into per-chain loop state, delayed with bounded exponential backoff,
and then restarted while the parent context is still alive.

The monolithic `/runtime/summary` puller section now exposes resilience facts:

- `backing_off_chains`
- `loop_restart_total`
- `loop_failure_total`
- `last_backoff_ms`
- `puller_posture=monolithic-puller-recovering` when at least one chain is in
  bounded backoff recovery

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-pull-loop-resilience.md`
