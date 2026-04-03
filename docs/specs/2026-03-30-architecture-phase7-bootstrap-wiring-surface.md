# Phase 7 Bootstrap Wiring Surface

## Title
Phase 7 - Add runtime domain query bridge wiring surface in API bootstrap

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
- `pkg/plugins/api/gateway.go`
- `cmd/microservices/api-service/main.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 6 introduced optional `domainquery.Service` support in `EventQueryHandler`, but runtime bootstrap still had no explicit wiring surface in API plugin composition.

## Problem Statement
Without a runtime wiring surface, bootstrap cannot expose bridge readiness or support incremental domain-first routing activation in production paths.

## Scope
- Add optional domain query service injection capability to API gateway plugin.
- Wire the domain query facade created in API service bootstrap into API gateway plugin.
- Add bridge configuration observability fields to plugin metrics/status.

## Non-Goals
- No replacement of retrieval-service-first query flow.
- No broad API routing refactor.
- No behavioral switch to domain-first execution in this phase.

## Options Considered
- Option A: Wait for full handler runtime refactor, no interim wiring surface.
- Option B: Add low-risk bootstrap wiring surface now, keep behavior unchanged.

## Selected Approach
Choose Option B:
- Introduce `SetDomainQueryService(domainquery.Service)` and bridge-status accessors in `APIGatewayPlugin`.
- Call setter from `cmd/microservices/api-service/main.go` where domain query facade already exists.
- Log bridge readiness during startup.

## Data / Contract Impact
- API plugin gains additive methods only; no breaking change to existing constructor.

## Risks
- Bridge may be configured but unused by downstream routes initially.
- Mitigation: expose explicit status and track next phase for route-level activation.

## Rollback Plan
- Revert new setter/status methods and bootstrap call in API service.

## Test and Verification Plan
- Build and run fast micro-loop.
- Confirm startup logs indicate bridge configured in API service bootstrap.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase7-bootstrap-wiring-surface.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved for additive, low-risk runtime wiring progression.

## Implementation Summary
- Added API gateway bootstrap wiring surface for domain query bridge and connected it in API service main.

## Final Verification
- Fast gate passes and runtime bootstrap has explicit bridge configuration path.
