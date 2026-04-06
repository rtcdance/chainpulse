Title: M3a Microservice Entrypoint Verification
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/event-processor, cmd/microservices/puller

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2` completed the truthful dual-mode baseline, but `M3a` needs a more direct
way to verify that each microservice entrypoint can start independently and
expose its minimum operator surface.

The repository already has shared runnable-app scripts, but it still lacks a
focused verification entry for:

1. standalone `api-service`
2. standalone `api-gateway`
3. standalone `event-processor`
4. standalone `puller`

## Scope

This slice will:

1. add a repo-root script that starts selected microservice entrypoints one by one
2. verify each selected service exposes `/health`
3. verify each selected service exposes `/runtime/summary`
4. verify execution services also expose `/runtime/control`

## Non-Goals

This slice will not:

1. introduce docker-compose orchestration
2. claim full `M3a` completion
3. change service runtime behavior
4. replace the existing runnable-app startup flow

## Selected Approach

Add a narrow repo-root verification script that launches each service with
local-first defaults, waits for the relevant runtime routes, and then stops the
process before moving on to the next entrypoint.

## Quality Gates

1. `bash -n scripts/verify-microservice-entrypoints.sh`
2. `bash scripts/verify-microservice-entrypoints.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3a-microservice-entrypoint-verification.md`

## Decision

Approved for implementation as the first `M3a` slice.

## Implementation Notes

Implemented in:

- `scripts/verify-microservice-entrypoints.sh`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/MILESTONE_STATUS.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

The new verification entry focuses on independent microservice startup and
operator-surface reachability instead of multi-service orchestration.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-microservice-entrypoints.sh`
2. `bash scripts/verify-microservice-entrypoints.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3a-microservice-entrypoint-verification.md`
