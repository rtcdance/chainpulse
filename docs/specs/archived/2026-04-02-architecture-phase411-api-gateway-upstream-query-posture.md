Title: Phase 411 - API Gateway Upstream Query Posture
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, pkg/plugins/api

## Problem Statement

Phase 410 added a minimal upstream query bridge from `api-gateway` to
`api-service`, but the gateway runtime summary still does not explicitly show
whether query upstreams are configured, attached, or effectively available.
That leaves the external entrypoint weaker than it should be for a runnable app
because operators still need to infer bridge readiness indirectly.

## Scope

- expose compact upstream query bridge facts through `api-gateway`
  `/runtime/summary`
- surface configured upstream count, attached upstream count, available
  upstream count, compact bridge posture, and a concise reliability hint
- add focused tests and update architecture docs

## Non-Goals

- no new health endpoint family
- no service discovery redesign
- no auth/policy changes
- no broader gateway proxy redesign

## Selected Approach

Extend the `api-gateway` runtime summary to include upstream-query bridge
status derived from the existing configured endpoints and attached route
handlers. Keep the posture compact and honest so the external entrypoint tells
operators whether the query bridge is absent, partially attached, or ready.

## Risks

- overstating readiness if attached handlers are treated as healthy without
  checking real upstream availability

## Rollback

- remove upstream query posture fields from gateway runtime summary
- keep the underlying upstream query bridge unchanged

## Test Strategy

- extend focused runtime summary tests for `api-gateway`
- run focused Go tests for `pkg/plugins/api` and
  `cmd/microservices/api-gateway`
- run race tests for `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase411-api-gateway-upstream-query-posture.md`

## Decision

- Add compact upstream query bridge posture to `api-gateway` runtime summary.

## Status

Status: Approved
