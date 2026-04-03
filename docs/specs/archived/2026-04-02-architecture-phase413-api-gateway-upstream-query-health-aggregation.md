Title: Phase 413 - API Gateway Upstream Query Health Aggregation
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: pkg/plugins/api, cmd/microservices/api-gateway

## Problem Statement

`api-gateway` now exposes upstream query bridge posture and a structured bridge
failure surface, but the current runtime summary still relies on static handler
availability unless bridge health is refreshed explicitly. For the runnable app
goal, the external entrypoint should more directly express whether upstream
`api-service` instances are actually healthy enough to serve query traffic.

## Scope

- add active upstream query health refresh for gateway summary usage
- expose compact upstream query health state through `api-gateway`
  `/runtime/summary`
- keep the work limited to read-only query upstream health aggregation
- add focused tests and update docs

## Non-Goals

- no service discovery redesign
- no subscription upstream health aggregation
- no gateway-wide health contract redesign

## Selected Approach

Extend the gateway upstream query bridge with a refreshable health-check path
that probes configured upstream handlers via their existing `/health` endpoint.
Use the refreshed counts to classify an honest compact health state in the
gateway runtime summary.

## Risks

- health probing could overstate or understate real query readiness if upstream
  `/health` is much weaker than query path health

## Rollback

- remove active bridge-health refresh from the runtime summary path
- keep the existing static bridge posture and failure surface

## Test Strategy

- add focused tests for upstream query health refresh and runtime summary
- run focused Go tests for `pkg/plugins/api` and
  `cmd/microservices/api-gateway`
- run race tests for `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase413-api-gateway-upstream-query-health-aggregation.md`

## Decision

- Add active upstream query health aggregation for the gateway runtime summary.

## Status

Status: Approved
