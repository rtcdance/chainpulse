# Phase 10 Runtime Route Composition

## Title
Phase 10 - Compose event query runtime handlers into API gateway request path

## Type
- architecture

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `pkg/plugins/api/http/plugin.go`
- `pkg/plugins/api/gateway.go`
- `cmd/microservices/api-service/main.go`
- `docs/ARCHITECTURE.md`

## Context
Phases 8-9 wired runtime handler dependencies and moved one handler path to domain-first logic, but API gateway runtime request composition still did not route incoming HTTP requests through `GatewayRouterIntegration`.

## Problem Statement
Without runtime route composition, the migrated handler logic is not exercised by live gateway requests.

## Scope
- Add optional native HTTP handler override to HTTP plugin.
- In API gateway plugin, compose `GatewayRouterIntegration` when event query/subscription/health handlers are all configured.
- Wire subscription and health handlers in API service bootstrap and initialize them.
- Expose runtime route composition status for observability.

## Non-Goals
- No GraphQL schema changes.
- No broad router redesign.
- No full monolithic bootstrap migration in this phase.

## Options Considered
- Option A: Defer request-path composition until full gateway rewrite.
- Option B: Add additive runtime composition hook and keep existing fallback behavior.

## Selected Approach
Choose Option B:
- Keep existing HTTP plugin processing as default path.
- Use native handler override only when gateway integration is composed successfully.
- Log route-composition readiness during startup.

## Data / Contract Impact
No external API contract changes; additive internal plugin methods only.

## Risks
- Misconfigured handlers could fail request path initialization.
- Mitigation: composition only enabled when all required handlers are set; initialization failure returns explicit error.

## Rollback Plan
Revert native handler override and plugin composition wiring.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase10-runtime-route-composition.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase10-runtime-route-composition.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as first end-to-end runtime activation step for migrated event query path.

## Implementation Summary
- API gateway now composes and exposes runtime route integration when handlers are wired.

## Final Verification
- Fast gate passes and startup path reports runtime route composition enabled.
