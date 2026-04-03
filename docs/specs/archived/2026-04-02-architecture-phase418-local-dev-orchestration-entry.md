Title: Phase 418 Local Dev Orchestration Entry
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Team
Related Modules: cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor, scripts

## Status

Status: Approved

## Problem Statement

The minimal runnable-app baseline is now stage-complete, but local startup still depends on manually
opening multiple terminals and copying service-specific commands from separate quickstarts. That slows
down validation against `docs/archive/ARCHITECTURE_v1.md` and makes the current runnable slice less
repeatable than it should be.

## Scope

- Add one local/dev orchestration entry for the current minimal runnable app.
- Support a smallest useful default slice and an explicit four-service slice.
- Keep the entry local-first and non-destructive.
- Update the most relevant quickstart docs to point to the shared entry.

## Non-Goals

- No Docker Compose redesign.
- No Kubernetes/deployment workflow changes.
- No process supervisor or daemonization framework.
- No attempt to solve missing local dependencies automatically.

## Options Considered

### Option 1: Documentation only

Keep separate quickstarts and add one more combined doc.

### Option 2: One local shell entry plus light doc alignment

Provide a small shell script that starts the current runnable slice with local-first defaults and
documents how to use it.

## Selected Approach

Choose Option 2.

Add a local orchestration shell script under `scripts/` that:

- starts `api-service` and `api-gateway` by default
- optionally starts `puller` and `event-processor`
- writes logs to a local temp directory
- waits for key health/runtime endpoints
- cleans up child processes on exit

Then align the gateway and API service quickstarts so the shared entry becomes the preferred local/dev
path.

## Risks

- Long-running processes started from one shell script can be brittle if cleanup is sloppy.
- Local dependency gaps (Kafka/PostgreSQL/Redis/RPC) may still prevent startup.
- Existing quickstarts may still contain older, more detailed instructions; wording needs to stay
  consistent.

## Rollback

- Remove the new script.
- Remove the quickstart references to the shared entry.

## Test Strategy

- Add focused tests for any new helper logic if implemented in Go; otherwise verify shell syntax.
- Run `bash -n` against the new script.
- Run spec approval check.

## Quality Gate Plan

- `bash -n scripts/run-local-runnable-app.sh`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase418-local-dev-orchestration-entry.md`

## Review Notes

- Chosen to keep scope tight around a repeatable local/dev entry.
- Explicitly avoids expanding into deployment-platform redesign.
