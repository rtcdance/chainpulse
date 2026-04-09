Title: Reopen Compose Microservices Readiness Smoke
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: docker, scripts

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The compose reopen line already has:

1. compose-file verification
2. compose service-set verification
3. a dedicated microservice compose profile

What it still lacks is a single readiness smoke that can:

1. bring the microservice compose profile up
2. wait for the four foreground services
3. reuse the current full runnable verification baseline

## Scope

This slice will:

1. add a docker-compose microservices readiness smoke script
2. verify Docker daemon reachability before startup
3. start the microservice compose profile with `up -d --build`
4. wait for the four foreground services to expose health/runtime summaries
5. reuse `verify-local-runnable-app.sh --profile full`

## Non-Goals

This slice will not:

1. claim full compose orchestration completion
2. introduce Kubernetes validation
3. add production-only container tuning
4. redesign the compose profile

## Selected Approach

Add a narrow readiness smoke script that layers on top of the dedicated
microservice compose profile and existing full runnable verification. This keeps
the implementation honest while moving from static compose validation into a
real startup check.

## Quality Gates

1. `bash -n scripts/verify-docker-compose-microservices-readiness.sh`
2. `bash scripts/verify-docker-compose-microservices-readiness.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-reopen-compose-microservices-readiness-smoke.md`

## Decision

Approved as the next slice of the docker-compose/platform-orchestration reopen
line.

## Implementation Notes

Implemented in:

- `scripts/verify-docker-compose-microservices-readiness.sh`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-docker-compose-microservices-readiness.sh`
2. `bash scripts/verify-docker-compose-microservices-readiness.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-reopen-compose-microservices-readiness-smoke.md`
