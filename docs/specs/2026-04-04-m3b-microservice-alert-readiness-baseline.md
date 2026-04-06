Title: M3b Microservice Alert Readiness Baseline
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/event-processor, cmd/microservices/puller

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M3b` now has an observability baseline verifier, but it still needs a focused
way to verify that the four foreground microservices expose advisory/rollout
signals in a shape that is usable as a minimum alert-readiness baseline.

The repository already exposes `/health/rollout`, but there is not yet a
single repo-root verification entry that checks those advisories together.

## Scope

This slice will:

1. add a repo-root alert-readiness verification script
2. start the full local runnable profile
3. verify that all four foreground microservices expose rollout advisory
   signals through `/health/rollout`

## Non-Goals

This slice will not:

1. introduce a real alert manager
2. add notification channels
3. claim full `M3b` completion
4. redesign rollout advisory payloads

## Selected Approach

Add a narrow script that reuses the full runnable profile and current full
verification entry, then asserts each service exposes:

1. advisory payload
2. advisory status
3. advisory readiness flag
4. rollout posture hint

## Quality Gates

1. `bash -n scripts/verify-microservice-alert-readiness.sh`
2. `bash scripts/verify-microservice-alert-readiness.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3b-microservice-alert-readiness-baseline.md`

## Decision

Approved for implementation as the second `M3b` slice.

## Implementation Notes

Implemented in:

- `scripts/verify-microservice-alert-readiness.sh`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/MILESTONE_STATUS.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

The script keeps the baseline narrow and operator-facing by validating rollout
advisories without introducing a real external alerting platform yet.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-microservice-alert-readiness.sh`
2. `bash scripts/verify-microservice-alert-readiness.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3b-microservice-alert-readiness-baseline.md`
