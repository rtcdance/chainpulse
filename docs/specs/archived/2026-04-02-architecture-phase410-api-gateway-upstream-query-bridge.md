Title: Phase 410 - API Gateway Upstream Query Bridge
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, pkg/plugins/api

## Problem Statement

The current `api-gateway` runtime/operator surface is stronger after phase 409,
but the external entrypoint still does not honestly satisfy the `ARCHITECTURE_v1.md`
microservice blueprint for the runnable app because query read routes are not
actually bridged to `api-service`. The gateway currently wires local event query
route shapes, but those handlers are not backed by the real query microservice.

## Scope

- add a minimal upstream query bridge from `api-gateway` to configured
  `api-service` endpoints
- limit the bridge to read-only event query routes:
  - `/events`
  - `/events/{id}`
  - `/events/chain/{chainId}`
  - `/events/contract/{address}`
  - `/events/name/{eventName}`
- reuse the existing request-router/load-balancer primitives instead of adding
  a new proxy stack
- add focused tests and update docs

## Non-Goals

- no subscription proxying
- no auth/policy/rate-limit redesign
- no service discovery redesign
- no gateway control-plane changes

## Options Considered

### Option 1 - Only record the runnable-app gap

Pros:
- lowest short-term risk

Cons:
- leaves the outer entrypoint unable to perform the core query role expected by
  the architecture blueprint

### Option 2 - Add a small upstream query bridge now

Pros:
- directly closes the most important runnable-app gap
- reuses existing router/load-balancer code
- keeps the change focused on read-only query routes

Cons:
- adds transport-level coupling at the gateway edge that may later be replaced
  by stronger service discovery or policy layers

## Selected Approach

Choose Option 2.

Bridge the `api-gateway` event query routes to configured upstream
`api-service` endpoints using the existing `RequestRouter` forwarding path.
Keep the bridge narrowly scoped to query reads and preserve local route shape
and metrics collection.

## Risks

- request-forwarding bugs could make query routes fail with 5xx instead of the
  current local-handler failures
- gateway route behavior may look healthier than the upstream service actually
  is if error handling is too optimistic

## Rollback

- remove upstream query endpoint wiring from `api-gateway`
- remove router handler attachment and forwarding logic
- fall back to the previous local route shape

## Test Strategy

- add focused gateway integration tests that forward query requests to a test
  upstream server
- run focused Go tests for `pkg/plugins/api` and
  `cmd/microservices/api-gateway`
- run race tests for `cmd/microservices/api-gateway`
- run spec approval check

## Quality Gates

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/api-gateway/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/api-gateway/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase410-api-gateway-upstream-query-bridge.md`

## Review Notes

- This phase intentionally prefers a narrow runnable-app bridge over broader
  gateway redesign work.

## Decision

- Add a minimal upstream query bridge from `api-gateway` to `api-service`.

## Status

Status: Approved
