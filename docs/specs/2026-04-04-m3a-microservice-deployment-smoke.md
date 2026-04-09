Title: M3a Microservice Deployment Smoke
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/event-processor, cmd/microservices/puller

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M3a` already has an independent entrypoint verification script, but it still
needs a focused cross-entrypoint deployment smoke that proves the four-service
local slice can start together and expose the expected operator surfaces.

## Scope

This slice will:

1. add a repo-root script that starts the full local runnable microservice profile
2. reuse the existing full-profile verification entry
3. assert the key cross-entrypoint deployment facts after startup

## Non-Goals

This slice will not:

1. introduce docker-compose
2. claim full `M3a` completion
3. redesign the current local runnable scripts
4. add deeper end-to-end data assertions beyond the current runnable baseline

## Selected Approach

Add a thin deployment-smoke script that starts `run-local-runnable-app.sh
--profile full`, runs the existing full verification entry, and then checks the
most important runtime-summary facts across the four foreground services.

## Quality Gates

1. `bash -n scripts/verify-microservice-deployment-smoke.sh`
2. `bash scripts/verify-microservice-deployment-smoke.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3a-microservice-deployment-smoke.md`

## Decision

Approved for implementation as the second `M3a` slice.

## Implementation Notes

Implemented in:

- `scripts/verify-microservice-deployment-smoke.sh`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/MILESTONE_STATUS.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

The script keeps the implementation narrow by reusing the existing full-profile
startup and verification entries instead of creating a second orchestration path.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-microservice-deployment-smoke.sh`
2. `bash scripts/verify-microservice-deployment-smoke.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3a-microservice-deployment-smoke.md`
