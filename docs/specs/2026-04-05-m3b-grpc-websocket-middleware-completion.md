Title: M3b gRPC WebSocket Middleware Completion
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/plugins/api/grpc, pkg/plugins/api/websocket

## Status

Approved for implementation.

## Problem Statement

The gRPC and WebSocket protocol plugins expose `RegisterRoute(...)` and
`Use(...)`, but their actual request-processing paths do not consistently use
the same routed API layer. That leaves middleware registration partially
ornamental: the plugins can report middleware presence while the protocol
request path does not reliably execute the registered middleware stack.

## Scope

This slice will:

1. make gRPC and WebSocket plugin route registration feed the active API layer
2. make gRPC and WebSocket plugin middleware registration feed the active API
   layer
3. add a focused request-processing seam for the gRPC plugin so middleware
   execution can be tested directly
4. switch WebSocket message processing onto the request processor path
5. add focused compatibility tests proving middleware is actually executed

## Non-Goals

This slice will not:

1. redesign gRPC transport bindings or generated protobuf services
2. add new auth or rate-limit policy semantics
3. change WebSocket message envelope formats
4. retrofit every other protocol plugin in the same slice

## Selected Approach

1. keep the current plugin APIs intact and make them wire through the shared
   `apiLayer` that the request processor already uses
2. add a small `ProcessRequest(...)` helper to the gRPC plugin for adapter-side
   request execution and tests
3. update the WebSocket plugin to process messages through
   `processor.ProcessRequest(...)` / `HandleError(...)` instead of bypassing the
   processor with direct `apiLayer.Handle(...)`
4. add focused tests that assert middleware-added response headers are present
   on both protocol paths

## Risks

1. middleware now executing on these protocol paths could expose previously
   hidden behavior differences
2. duplicate middleware registration would stack if callers wire the same
   middleware twice through multiple layers

## Rollback Plan

1. remove the gRPC request-processing helper
2. revert route/middleware registration to the previous plugin-local storage
3. switch WebSocket message handling back to direct API-layer invocation

## Test Strategy

1. unit test gRPC plugin request processing executes registered middleware
2. unit test WebSocket plugin request processing executes registered middleware
3. preserve existing lifecycle and middleware-count tests

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-grpc-websocket-middleware-completion.md`
2. `go test -short ./pkg/plugins/api/grpc/... ./pkg/plugins/api/websocket/...`
3. `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes

Approved as the smallest fix that turns existing gRPC/WebSocket middleware
registration from declarative bookkeeping into a real executed protocol path
without changing consumer-facing payload contracts.
