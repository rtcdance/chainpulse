Title: M3b WebSocket Rate Limit Hardening
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/plugins/api, cmd/microservices/api-gateway, docs/api

## Status

Implemented.

## Problem Statement

The gateway currently applies rate limiting at the outer HTTP handler layer, but
the WebSocket subscription handler itself has no explicit handshake rate-limit
guard. That leaves the subscription upgrade path too easy to reuse without
limit enforcement and makes the WebSocket security boundary weaker than the
rest of the entrypoint.

## Scope

This slice will:

1. add explicit rate-limit support to `EventSubscriptionHandler`
2. reuse the configured gateway rate limiter for WebSocket handshake requests
3. avoid double consumption when the outer middleware has already applied a
   rate-limit check
4. add focused tests for WebSocket handshake throttling

## Non-Goals

This slice will not:

1. redesign the token bucket implementation
2. add long-lived message-rate throttling after upgrade
3. change connection-pool size semantics
4. add auth requirements beyond the current security surface

## Selected Approach

1. update `RateLimitMiddleware` to store `RateLimitInfo` in request context for
   downstream handlers
2. let `EventSubscriptionHandler` check that context first
3. if no prior rate-limit check exists, apply a direct handshake rate-limit
   check before `websocket.Upgrader.Upgrade(...)`
4. inject the configured gateway limiter into the subscription handler during
   gateway initialization

## Risks

1. handshake requests may be rejected earlier than before when security surfaces
   are enabled
2. future alternate entrypoints must still wire the limiter into the
   subscription handler to inherit the same protection

## Rollback Plan

1. remove the subscription-handler limiter hook
2. remove the context propagation change in `RateLimitMiddleware`
3. restore previous tests and docs

## Test Strategy

1. unit test that middleware writes rate-limit info into context
2. unit test that repeated WebSocket handshake attempts are throttled when the
   subscription handler owns the limiter
3. unit test that gateway initialization injects the limiter into the
   subscription handler

## Quality Gates

1. `go test -short ./pkg/plugins/api/... ./cmd/microservices/api-gateway/...`
2. `scripts/dev-micro-loop.sh --mode fast --base HEAD`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-websocket-rate-limit-hardening.md`

## Review Notes

Approved as the smallest fix that hardens the WebSocket upgrade path without
redesigning the broader rate-limiter model.

## Implementation Notes

1. `RateLimitMiddleware` now writes `RateLimitInfo` into the request context so
   downstream handlers can detect an already-consumed rate-limit check.
2. `EventSubscriptionHandler` now supports an optional handshake limiter and
   returns `429 Too Many Requests` before upgrade when the WebSocket entrypoint
   exceeds budget.
3. `APIGatewayPlugin` now injects the configured gateway limiter into the
   subscription handler during initialization so the shared limiter covers the
   WebSocket upgrade path.

## Verification Summary

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-websocket-rate-limit-hardening.md`
2. `go test -short ./pkg/plugins/api/... ./cmd/microservices/api-gateway/...`
3. `scripts/dev-micro-loop.sh --mode fast --base HEAD`
