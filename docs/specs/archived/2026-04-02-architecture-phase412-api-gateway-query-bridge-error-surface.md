Title: Phase 412 - API Gateway Query Bridge Error Surface
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: pkg/plugins/api, cmd/microservices/api-gateway

## Problem Statement

`api-gateway` now forwards read-only `/events*` routes to configured
`api-service` upstreams, but upstream forwarding failures still degrade to a
plain `Bad Gateway` text response. That is too weak for the current runnable
app goal because the external entrypoint does not expose a compact,
machine-readable failure reason for query-bridge outages.

## Scope

- add a structured JSON error response for upstream query bridge failures
- keep the scope limited to `api-gateway` read-only query forwarding failures
- add focused tests and update architecture docs

## Non-Goals

- no subscription failure redesign
- no auth/policy changes
- no gateway-wide error envelope migration

## Selected Approach

Return a compact JSON error envelope from `api-gateway` when upstream query
forwarding fails. Include a stable error code, a concise reliability hint, and
query-bridge posture metadata so clients and operators can distinguish upstream
query outages from generic gateway failures.

## Risks

- introducing a new gateway-specific error shape could drift from future shared
  API error contracts if expanded too broadly

## Rollback

- remove the structured gateway query-bridge error helper
- fall back to the previous plain `502 Bad Gateway` response

## Test Strategy

- add focused gateway forwarding-failure tests
- run focused Go tests for `pkg/plugins/api` and
  `cmd/microservices/api-gateway`
- run race tests for `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase412-api-gateway-query-bridge-error-surface.md`

## Decision

- Add a structured JSON failure surface for gateway query-bridge forwarding
  errors.

## Status

Status: Approved
