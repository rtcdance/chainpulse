# Phase 8 Runtime Handler Instantiation

## Title
Phase 8 - Instantiate EventQueryHandler in production bootstrap and wire domain bridge

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
- `cmd/microservices/api-service/main.go`
- `pkg/plugins/api/gateway.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 7 added a runtime wiring surface for domain query bridge in `APIGatewayPlugin`. However, production bootstrap still did not instantiate `EventQueryHandler`.

## Problem Statement
Without runtime instantiation, domain bridge wiring cannot be propagated into the event query handler lifecycle in production bootstrap paths.

## Scope
- Instantiate `EventStore`, `EventMetadataStore`, `EventRetrievalService`, and `EventQueryHandler` in API service bootstrap.
- Initialize retrieval service and event query handler with startup context.
- Inject event query handler into API gateway plugin through additive setter.
- Keep existing external API behavior unchanged.

## Non-Goals
- No route-path migration in this phase.
- No replacement of retrieval-first query path.
- No monolithic bootstrap refactor in this phase.

## Options Considered
- Option A: Delay instantiation until full route migration.
- Option B: Instantiate now with additive wiring and no behavior switch.

## Selected Approach
Choose Option B to de-risk future migration:
- Add `SetEventQueryHandler(...)` and status accessors to API gateway plugin.
- Build/initialize event retrieval stack in API service startup and inject into plugin.
- Log bridge and handler readiness.

## Data / Contract Impact
- Additive API plugin methods and status flags only; no breaking interface changes.

## Risks
- Additional startup initialization steps increase bootstrap complexity.
- Mitigation: reuse existing DB manager/timeouts and fail fast on initialization errors.

## Rollback Plan
- Revert handler instantiation and plugin injection changes.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase8-runtime-handler-instantiation.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase8-runtime-handler-instantiation.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as an additive wiring step to support next-phase route migration.

## Implementation Summary
- Runtime bootstrap now instantiates and wires event query handler with domain query bridge.

## Final Verification
- Fast gate passes and startup path has explicit handler + bridge wiring.
