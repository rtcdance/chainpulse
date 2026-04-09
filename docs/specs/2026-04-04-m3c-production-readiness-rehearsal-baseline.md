Title: M3c Production Readiness Rehearsal Baseline
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M3c` needs a single repo-root rehearsal entry that combines the already-built
deployment, observability, and alert-readiness baselines into one minimum
production-readiness drill.

Right now those checks exist, but they are still separate entrypoints.

## Scope

This slice will:

1. add a repo-root production-readiness rehearsal script
2. sequence the current deployment smoke
3. sequence the current observability baseline
4. sequence the current alert-readiness baseline

## Non-Goals

This slice will not:

1. introduce docker-compose or Kubernetes drills
2. claim final `M3c` completion
3. add new runtime assertions beyond the three existing baselines
4. introduce external paging or incident tooling

## Selected Approach

Add a thin orchestration script that simply runs the three existing baseline
verification entries in sequence, so the repository gains a single repeatable
production-readiness rehearsal command without duplicating existing checks.

## Quality Gates

1. `bash -n scripts/run-production-readiness-rehearsal.sh`
2. `bash scripts/run-production-readiness-rehearsal.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3c-production-readiness-rehearsal-baseline.md`

## Decision

Approved for implementation as the first `M3c` slice.

## Implementation Notes

Implemented in:

- `scripts/run-production-readiness-rehearsal.sh`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/MILESTONE_STATUS.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

The new script intentionally reuses existing verification entries instead of
introducing a second copy of the drill logic.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/run-production-readiness-rehearsal.sh`
2. `bash scripts/run-production-readiness-rehearsal.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3c-production-readiness-rehearsal-baseline.md`
