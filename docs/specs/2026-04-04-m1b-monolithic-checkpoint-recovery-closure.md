Title: M1b Monolithic Checkpoint Recovery Closure
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/application/indexing

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1b` slice 1 gave the monolithic pull loop restart/backoff ownership, but the
shared indexing runtime still only advertised checkpoint/replay capability as
static enablement flags. The monolith did not actually execute a recovery probe
on startup, and `/runtime/summary` could not distinguish between "recovery is
wired", "a checkpoint was loaded", "a replay batch was applied", or "recovery
failed".

## Scope

This slice will:

1. Add a real shared-runtime `RecoverChain(...)` seam.
2. Run per-chain recovery probes during monolithic startup after the shared
   indexing runtime starts.
3. Surface checkpoint-load / replay / recovery-error facts in monolithic
   `/runtime/summary`.
4. Add focused runtime and summary tests.

## Non-Goals

This slice will not:

1. Persist checkpoints across process restarts with a new storage backend.
2. Redesign replay sourcing beyond the existing in-memory failure journal.
3. Add distributed recovery orchestration.
4. Change writable runtime control semantics.

## Selected Approach

Extend `pkg/application/indexing.SharedRuntime` with an additive
`RecoverChain(...)` operation that:

1. loads the last checkpoint for one chain,
2. loads replay envelopes from the configured replay source,
3. reprocesses them through the existing batch path,
4. records runtime recovery status facts.

The monolithic entrypoint will call that recovery seam once per configured
chain during startup and publish the resulting posture through
`/runtime/summary`.

## Quality Gates

1. `go test -short ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-checkpoint-recovery-closure.md`

## Decision

Approved for implementation as the second `M1b` resilience slice.

## Implementation Notes

Implemented in:

- `pkg/application/indexing/runtime.go`
- `pkg/application/indexing/runtime_test.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The shared indexing runtime now exposes a real `RecoverChain(...)` seam that
loads the current checkpoint, pulls replay envelopes from the configured replay
source, reprocesses them through the existing batch path, and records recovery
status facts.

The monolithic startup path now runs one recovery probe per configured chain
after the shared indexing runtime starts. The monolithic `/runtime/summary`
indexing section now surfaces:

- `recovery_state`
- `recovery_run_total`
- `recovery_failure_total`
- `recovery_checkpoint_load_total`
- `recovery_replayed_events`
- `last_recovery_*`
- `recovery_posture`
- `recovery_reliability_hint`

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1b-monolithic-checkpoint-recovery-closure.md`
