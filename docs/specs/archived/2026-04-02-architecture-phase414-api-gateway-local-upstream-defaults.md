Title: Phase 414 - API Gateway Local Upstream Defaults
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway

## Problem Statement

The gateway query bridge is now functional, but the default upstream list still
assumes clustered hostnames like `api-service-1:8081`. That works for a
microservice deployment story, but it is a poor default for the current goal of
shipping a runnable app quickly because local runs of `api-gateway` and
`api-service` do not align by default.

## Scope

- make `api-gateway` default upstream wiring local-runnable first
- support environment override for upstream services using a compact string
  format
- add focused configuration tests and update docs

## Non-Goals

- no service discovery redesign
- no deployment-manifest changes
- no gateway query bridge behavior changes beyond startup configuration

## Selected Approach

Use `http://localhost:8081` as the default upstream query endpoint for
`api-gateway` and add a `GATEWAY_UPSTREAM_SERVICES` environment variable for
comma-separated override values. Keep the change strictly at configuration
loading so runtime bridge logic stays unchanged.

## Risks

- cluster-oriented users may rely on the previous static defaults if they do
  not set environment overrides explicitly

## Rollback

- restore the previous static upstream list in `loadGatewayConfig`
- remove override parsing helper and tests

## Test Strategy

- add focused config parsing tests for `api-gateway`
- run focused Go tests for `cmd/microservices/api-gateway`
- run race tests for `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase414-api-gateway-local-upstream-defaults.md`

## Decision

- Make gateway upstream defaults local-runnable and env-overridable.

## Status

Status: Approved
