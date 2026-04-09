Title: Live WebSocket Acceptance Alignment
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/plugins/api, frontend, docs/specs

## Status

Implemented on 2026-04-09 after live verification against the Docker stack and
the H5 demo. Header remains `Status: Approved` for repository checker
compatibility.

## Problem Statement

After the H5 demo routing alignment, live browser acceptance still fails on the
WebSocket surface.

Observed facts:

1. `api-gateway` does not expose `/ws` in the current runtime
2. `api-gateway` does route `/events/subscribe` into the subscription handler
3. direct WebSocket handshakes to `/events/subscribe` still fail with a non-101
   response

This leaves the H5 WebSocket acceptance view blocked even though the frontend
can now reach the backend through same-origin proxying.

## Scope

This bugfix will:

1. make the gateway-backed `/events/subscribe` WebSocket upgrade path work in
   the live runtime
2. add focused coverage for gateway subscription upgrade handling
3. preserve the existing HTTP and subscription route contracts

## Non-Goals

This bugfix will not:

1. add a new backend `/ws` contract if it is not already part of the runtime
2. redesign the WebSocket message envelope
3. change unrelated HTTP acceptance behavior

## Selected Approach

1. inspect the gateway subscription upgrade path and remove any response-writer
   wrapping that breaks `websocket.Upgrader`
2. keep H5 fallback behavior pointed at the gateway subscription route for live
   acceptance
3. verify with a focused live WebSocket handshake probe plus the H5 demo

## Risks

1. gateway middleware may rely on response wrapping semantics that do not apply
   cleanly to upgraded connections
2. live subscription behavior may reveal a second backend issue after the
   handshake starts working

## Rollback Plan

1. revert the gateway subscription upgrade path change
2. leave the H5 demo using HTTP-only acceptance until WebSocket is reopened

## Test Strategy

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-09-live-websocket-acceptance-alignment.md`
2. focused Go tests around gateway subscription handling
3. focused live WebSocket handshake probe against the Docker stack
4. rerun the H5 WebSocket acceptance path

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-09-live-websocket-acceptance-alignment.md`
2. focused Go tests for the touched package
3. focused live handshake verification

## Review Notes

Approved as the smallest useful backend fix needed to finish live H5
acceptance after the frontend proxy alignment.

## Implementation Summary

1. initialized the gateway runtime subscription handler so `/events/subscribe`
   no longer returns `Handler not initialized`
2. restored the gateway `/ws` route and mapped it to the existing
   subscription handler path used by the H5 demo
3. bypassed HTTP tracing response wrapping for WebSocket upgrade requests so
   `http.Hijacker` remains available to the upgrader

## Verification Summary

1. `bash scripts/spec-approval-check.sh docs/specs/2026-04-09-live-websocket-acceptance-alignment.md`
   passed before implementation
2. `go test ./cmd/microservices/api-gateway -run 'TestBuildAPIGatewayRuntimeRolloutComponents|TestAPIGatewaySubscriptionRouteSupportsWebSocketUpgrade'`
   passed
3. `go test ./pkg/plugins/api -run 'TestNewGatewayRouterIntegration|TestGatewayRouterIntegrationRuntimeRouteInventory'`
   passed
4. `go test ./pkg/plugins/api/http -run 'TestHTTPPluginPropagatesTraceContextToNativeHandler|TestHTTPPluginPreservesHijackerForWebSocketNativeHandler'`
   passed
5. live WebSocket probes to `ws://localhost:8080/ws` and
   `ws://127.0.0.1:3000/__ws/api-gateway/ws` both reached `open`
