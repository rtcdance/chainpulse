Title: Phase 419 Local Dev Verification Entry
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Team
Related Modules: scripts, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor

## Status

Status: Approved

## Problem Statement

The repository now has a shared local/dev orchestration entry, but the acceptance
path is still partly manual. We need one small, repeatable verification entry so
the current runnable baseline can be checked consistently after startup.

## Scope

- Add one local/dev verification shell entry for the current runnable app slice.
- Support `minimal` and `full` verification profiles.
- Check the current baseline only:
  - health
  - runtime summary
  - query bridge
  - control endpoints for the full slice
- Update the shared quickstart/orchestration docs to point to the verification entry.

## Non-Goals

- No end-to-end dependency bootstrap.
- No container orchestration or deployment automation.
- No replacement for Go unit/integration tests.
- No broad protocol coverage expansion.

## Options Considered

### Option 1: Leave verification manual

Keep the current quickstart commands and rely on operators to run curls manually.

### Option 2: Add one focused verification shell entry

Provide one repository-root shell script that validates the current runnable
baseline after startup.

## Selected Approach

Choose Option 2.

Add `scripts/verify-local-runnable-app.sh` with:

- `minimal` profile:
  - `api-service /health`
  - `api-service /runtime/summary`
  - `api-gateway /health`
  - `api-gateway /runtime/summary`
  - `api-gateway /events?limit=5`
- `full` profile:
  - all of the above
  - `event-processor /health`
  - `event-processor /runtime/summary`
  - `event-processor /runtime/control`
  - `puller /health`
  - `puller /runtime/summary`
  - `puller /runtime/control`

## Risks

- String-based shell assertions can become brittle if response payloads drift.
- Local dependency failures may surface as verification failures even when the
  script itself is correct.

## Rollback

- Remove the verification script.
- Remove doc references to the shared verification entry.

## Test Strategy

- Validate shell syntax with `bash -n`.
- Validate help output path with `bash scripts/verify-local-runnable-app.sh --help`.
- Run spec approval check.

## Quality Gate Plan

- `bash -n scripts/verify-local-runnable-app.sh`
- `bash scripts/verify-local-runnable-app.sh --help`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase419-local-dev-verification-entry.md`

## Review Notes

- Chosen to keep the local/dev path practical and repeatable without drifting
  into broader orchestration or deployment work.
