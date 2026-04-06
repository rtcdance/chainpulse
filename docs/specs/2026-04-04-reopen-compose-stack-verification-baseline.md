Title: Reopen Compose Stack Verification Baseline
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: docker, scripts, docs

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The completed milestone sequence established runnable, observable, and
rehearseable local baselines, but it did not yet connect those baselines back to
the repository's existing docker-compose stack definitions.

The repository already contains:

1. `docker/docker-compose.yml`
2. `docker/docker-compose.dev.yml`

But there is no repo-root verification entry that checks whether the expected
compose-layer infrastructure and observability services are still declared in a
way that matches the current runnable story.

## Scope

This slice will:

1. add a lightweight compose-stack verification script
2. verify the selected compose file contains the expected infra services
3. verify the selected compose file contains the expected observability services
4. verify `docker compose config --services` resolves the expected service set
5. verify repo-root runnable docs still describe the four foreground services
6. add a dedicated microservice compose profile for the four foreground services

## Non-Goals

This slice will not:

1. start containers
2. validate docker-compose runtime health
3. claim full compose orchestration completion
4. redesign compose manifests

## Selected Approach

Add a narrow verification script that checks the compose file, resolves the
declared service set through `docker compose config --services`, and verifies
the current repo-root runbook still describes the expected foreground services.
This keeps the first compose reopen slice lightweight and safe.

## Quality Gates

1. `bash -n scripts/verify-docker-compose-stack.sh`
2. `bash scripts/verify-docker-compose-stack.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-reopen-compose-stack-verification-baseline.md`

## Decision

Approved as the first slice of the docker-compose/platform-orchestration reopen
line.

## Implementation Notes

Implemented in:

- `scripts/verify-docker-compose-stack.sh`
- `docker/docker-compose.microservices.yml`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-docker-compose-stack.sh`
2. `bash scripts/verify-docker-compose-stack.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-reopen-compose-stack-verification-baseline.md`
