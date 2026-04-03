Title: Phase 416 - Local Runnable Gateway Query Smoke
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, pkg/plugins/api

## Problem Statement

The repository now documents a minimal local runnable `api-gateway +
api-service` query path, but the current automated coverage still verifies the
pieces mostly in isolation. For the runnable-app goal, we need one focused
smoke slice that locks the combined external-entrypoint flow:

- gateway runtime summary reflects an attached healthy query bridge
- gateway query requests forward through the upstream bridge

## Scope

- add a focused smoke test for the minimal local runnable gateway query slice
- keep the test in-process and deterministic
- update architecture docs

## Non-Goals

- no real subprocess orchestration
- no Docker/Compose test harness
- no broader e2e framework work

## Selected Approach

Create a focused smoke test in `cmd/microservices/api-gateway` that wires the
gateway runtime composition with a deterministic in-memory HTTP transport for
the upstream `api-service`. The test should verify both `/runtime/summary` and
`/events` in the same slice.

## Risks

- smoke coverage may still be lighter than a true multi-process run

## Rollback

- remove the smoke test and docs references

## Test Strategy

- add the focused smoke test
- run focused Go tests for `cmd/microservices/api-gateway`
- run race tests for `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase416-local-runnable-gateway-query-smoke.md`

## Decision

- Add a focused automated smoke slice for the local runnable gateway query path.

## Status

Status: Approved
