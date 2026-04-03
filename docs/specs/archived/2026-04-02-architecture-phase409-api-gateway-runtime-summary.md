Title: Phase 409 - API Gateway Runtime Summary Surface
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, pkg/plugins/api

## Problem Statement

`ARCHITECTURE_v1.md` treats API Gateway as the external entrypoint of the
microservice app. The current repository already exposes richer read-only
runtime surfaces on `api-service`, `puller`, and `event-processor`, but
`api-gateway` still lacks a compact `/runtime/summary` route. That leaves the
outermost entrypoint weaker than the internal services and makes the current
runnable app harder to assess quickly from the operator side.

## Scope

- add a read-only `/runtime/summary` surface for `api-gateway`
- expose compact gateway runtime posture, rollout posture, and metrics summary
- wire the new summary provider through the existing gateway runtime route
  composition path
- add focused route/integration coverage and update architecture docs

## Non-Goals

- no new writable control plane on `api-gateway`
- no new protocol family or request/response redesign
- no attempt to claim deeper query-service ownership than current wiring

## Options Considered

### Option 1 - Do only a documentation gap assessment

Pros:
- zero code risk

Cons:
- leaves the external entrypoint weaker than the internal services
- does not move the runnable app closer to the v1 operator shape

### Option 2 - Add a small read-only gateway runtime summary

Pros:
- directly improves the runnable app surface at the external entrypoint
- reuses existing route composition and rollout wiring
- keeps scope small and aligned with recent runtime-summary work

Cons:
- still only a read-only operator surface

## Selected Approach

Choose Option 2.

Implement a compact `api-gateway` runtime summary provider in
`cmd/microservices/api-gateway`, wire it through the existing optional runtime
summary provider path in `pkg/plugins/api`, and keep the payload honest by
deriving posture only from current gateway runtime facts and rollout wiring
signals.

## Risks

- overstating gateway readiness if summary fields imply stronger semantics than
  current route wiring actually provides
- route-count drift in gateway integration tests when the optional summary route
  is enabled

## Rollback

- remove the gateway runtime summary provider wiring from `main.go`
- remove the new summary helper file and focused tests
- keep existing runtime route composition unchanged

## Test Strategy

- add/update focused tests for the `/runtime/summary` route
- run focused Go tests for `pkg/plugins/api` and `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase409-api-gateway-runtime-summary.md`

## Review Notes

- This phase is intentionally narrow so we keep momentum toward a more runnable
  app instead of opening another broad architecture branch.

## Decision

- Add a read-only `api-gateway` runtime summary now.

## Status

Status: Approved
