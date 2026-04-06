Title: M1c Gateway Method Contract Hardening
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/plugins/api, cmd/monolithic/chainpulse

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The monolithic gateway runtime surface now exposes `/health*`, `/runtime/summary`,
`/runtime/control`, and `/metrics`, but gateway request handling still matches only
by path. A request such as `POST /runtime/summary` can currently fall through to a
read-only route instead of being rejected with a truthful method contract boundary.
That weakens `M1c` API hardening and makes the gateway surface less explicit than
the minimum blueprint-aligned operator/API baseline.

## Scope

This slice will:

1. Enforce route method matching in gateway request handling.
2. Return `405 Method Not Allowed` for path matches with the wrong HTTP method.
3. Emit an `Allow` header with the registered route method.
4. Surface compact method-contract facts through monolithic runtime summary.
5. Add focused tests for the hardened method boundary.

## Non-Goals

This slice will not:

1. Add new routes.
2. Introduce versioning or auth changes.
3. Change handler business logic for successful requests.
4. Expand beyond the current gateway runtime/operator baseline.

## Selected Approach

Keep route matching path-based, then enforce the route's declared method in
`GatewayRouterIntegration.HandleRequest(...)`. This keeps the change narrow,
avoids broader router refactors, and still hardens every existing gateway route.
Expose a compact method-contract posture in monolithic runtime summary so operators
can tell that the gateway is now enforcing the intended read-only boundary.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-gateway-method-contract-hardening.md`

## Decision

Approved for implementation as the third `M1c` observability/API-gateway slice.

## Implementation Notes

Implemented in:

- `pkg/plugins/api/gateway_router_integration.go`
- `pkg/plugins/api/gateway_router_integration_test.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

Gateway request handling now rejects wrong-method requests with `405 Method Not Allowed`
and an `Allow` header. Monolithic runtime summary gateway facts now include a compact
method-contract posture and hint for the current `M1c` hardening baseline.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-gateway-method-contract-hardening.md`
