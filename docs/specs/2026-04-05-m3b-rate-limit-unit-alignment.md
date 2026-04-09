Title: M3b Rate Limit Unit Alignment
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor, pkg/plugins/api, docs

## Status

Approved for implementation.

## Problem Statement

The architecture blueprint and API guidance expect operator-facing rate limit
configuration in requests per minute, but the current microservice entrypoints
still parse and present those limits as requests per second. That mismatch
means:

1. operators configure stricter or looser limits than intended
2. startup output and docs misstate the active unit
3. security-surface tests currently encode the wrong contract

## Scope

This slice will:

1. switch command-layer rate limit configuration semantics to requests per
   minute
2. convert requests per minute into requests per second only at the
   `RateLimiter` wiring boundary
3. update startup output, tests, and operator-facing docs to state `req/min`
4. preserve the existing environment variable names for compatibility

## Non-Goals

This slice will not:

1. redesign the underlying token bucket implementation
2. change rate-limit middleware enable/disable behavior
3. add new auth or security policies
4. version the environment variable names

## Selected Approach

Keep the internal rate limiter unchanged and fix the unit mismatch at the
command/config boundary:

1. rename command config fields from `PerSecond` to `PerMinute`
2. interpret existing `*_RATE_LIMIT` env vars as requests per minute
3. convert `req/min` to `req/s` when building `api.RateLimitConfig`
4. derive burst size from the per-minute budget using a small bounded window

## Risks

1. existing local setups may observe lower effective throughput if they were
   previously tuned assuming `req/s`
2. docs outside the actively maintained runnable path may still mention the old
   unit until separately cleaned up

## Rollback Plan

1. revert command config field renames
2. revert the per-minute to per-second conversion helper
3. restore previous tests and docs

## Test Strategy

1. update command config tests for all four microservice entrypoints
2. update security wiring tests to assert converted per-second values
3. run focused microservice and API rate-limiter package tests

## Quality Gates

1. `go test -short ./cmd/microservices/api-gateway/... ./cmd/microservices/api-service/... ./cmd/microservices/puller/... ./cmd/microservices/event-processor/... ./pkg/plugins/api/...`
2. `scripts/dev-micro-loop.sh --mode fast --base HEAD`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-rate-limit-unit-alignment.md`

## Review Notes

Approved as the smallest blueprint-aligned fix for the current rate-limit unit
mismatch.
