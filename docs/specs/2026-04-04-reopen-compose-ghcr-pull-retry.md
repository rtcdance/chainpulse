Title: Reopen Compose GHCR Pull Retry
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: docker, scripts

## Status

Approved for implementation. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The microservice compose readiness flow depends on the Foundry `anvil` image from GHCR:

1. `docker/docker-compose.microservices.yml` starts `ghcr.io/foundry-rs/foundry:latest`
2. `scripts/verify-docker-compose-microservices-readiness.sh` brings the stack up directly

When GHCR returns a transient `EOF` during blob fetch, the readiness check fails before any service health check can run.

## Scope

This slice will:

1. make the `anvil` image reference overridable through an environment variable
2. add a pre-flight image pull step with bounded retries before compose startup
3. keep the existing readiness and cleanup flow unchanged after startup

## Non-Goals

This slice will not:

1. redesign the compose stack
2. change service runtime semantics
3. pin the Foundry image to a new long-term version strategy
4. add new integration tests that require Docker to be available in CI

## Selected Approach

Use a narrow resilience shim in the readiness script:

1. determine the image to pull from the compose default or `ANVIL_IMAGE`
2. pre-pull the image with a small bounded retry loop
3. log each retry so the failure mode is explicit
4. proceed with the existing compose startup once the image is present locally

This keeps the fix local to the orchestration path and avoids changing the actual service graph.

## Risks

1. A bad registry outage can still fail after the retry budget is exhausted.
2. Pre-pulling adds a small startup delay when the image is not already cached.
3. Overriding `ANVIL_IMAGE` incorrectly could point the stack at an incompatible image.

## Rollback

Rollback is straightforward:

1. remove the pre-pull helper and retry loop from the readiness script
2. restore the direct compose startup path
3. keep the env-var override if it remains useful, or revert it with the pull helper

## Test Strategy

1. `bash -n scripts/verify-docker-compose-microservices-readiness.sh`
2. `bash scripts/verify-docker-compose-microservices-readiness.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-reopen-compose-ghcr-pull-retry.md`

## Quality Gate Plan

1. run the shell syntax check on the readiness script
2. run the script help path
3. run the spec approval check
4. if Docker is available, exercise the readiness script through the compose startup path

## Decision

Approved as a narrow reliability fix for the compose readiness path.

## Implementation Notes

Implemented in:

- `scripts/verify-docker-compose-microservices-readiness.sh`
- `docker/docker-compose.microservices.yml`

The readiness script now pre-pulls the Foundry `anvil` image with bounded retries before it starts the compose stack. The compose file also exposes the image reference through `ANVIL_IMAGE` so the runtime image can be overridden when needed.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-docker-compose-microservices-readiness.sh`
2. `bash scripts/verify-docker-compose-microservices-readiness.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-reopen-compose-ghcr-pull-retry.md`
4. `docker compose -f docker/docker-compose.microservices.yml config --services`
