Title: Phase 415 - Local Runnable Gateway Query Quickstart
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, cmd/microservices/api-service, docs

## Problem Statement

Recent phases made the `api-gateway -> api-service` query path locally runnable,
but the repository still does not provide one concise, current quickstart that
shows how to bring up that minimal two-service app on a developer machine.
Without that guide, the architecture is closer to runnable than the developer
experience suggests.

## Scope

- add a local quickstart for `api-gateway`
- document the minimal two-service startup order and local environment values
- update cross-links from existing quickstart docs

## Non-Goals

- no new runtime behavior
- no Docker Compose redesign
- no broader deployment guide rewrite

## Selected Approach

Add `cmd/microservices/api-gateway/QUICKSTART.md` describing the minimal local
startup path with `api-service` on `localhost:8081` and `api-gateway` on
`localhost:8080`. Update existing docs to link to the new quickstart and make
the runnable two-service query flow explicit.

## Risks

- docs can drift if future port defaults change

## Rollback

- remove the new quickstart doc
- remove the added cross-links

## Test Strategy

- spec approval check
- no code-path tests required because this phase is documentation-only

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase415-local-runnable-gateway-query-quickstart.md`

## Decision

- Add a dedicated local runnable quickstart for `api-gateway` and the minimal
  two-service query app.

## Status

Status: Approved
