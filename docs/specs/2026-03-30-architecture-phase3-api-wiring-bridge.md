# Phase 3 API Wiring Bridge

## Title
Phase 3 - Wire Query Domain Facade in API Service Bootstrap

## Type
- architecture

## Status
- Draft | In Review | Approved | Implemented
Status: Approved
Delivery Status: Implemented

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `cmd/microservices/api-service/main.go`
- `pkg/adapters/query`
- `pkg/application/query`

## Context
Phase 2 introduced query domain/application/adapter compatibility layer, but bootstrap wiring still only instantiates the legacy query service directly.

## Problem Statement
Without bootstrap bridge wiring, the new layered contract path is not exercised in runtime composition.

## Scope
- Update `cmd/microservices/api-service/main.go` to instantiate a domain-facing query facade from legacy query service.
- Keep existing legacy query service usage unchanged.
- Add logging note indicating compatibility bridge initialization.

## Non-Goals
- No API plugin signature changes.
- No runtime behavior replacement.
- No migration of GraphQL/query handlers in this phase.

## Options Considered
- Option A: Replace API handlers to consume domain service now.
- Option B: Add bootstrap bridge while keeping legacy path active.

## Selected Approach
Option B to minimize risk and enable incremental migration.

## Data / Contract Impact
No external contract changes.

## Risks
- Risk: confusion around dual query service references.
- Mitigation: keep bridge clearly labeled and non-authoritative for this phase.

## Rollback Plan
- Revert bootstrap bridge lines in `api-service/main.go`.
- Legacy runtime path remains unaffected.

## Test and Verification Plan
- Compile-level validation of new imports and wiring.
- Confirm legacy query service start/stop lifecycle remains unchanged.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase3-api-wiring-bridge.md`
- `scripts/dev-micro-loop.sh --mode fast` when Go toolchain is available.

## Review Notes
- Approved for no-behavior-change bootstrap bridge.

## Implementation Summary
- Add domain query facade initialization in API service bootstrap.

## Final Verification
- Existing query service path preserved.
- Bridge ready for next phase handler migration.
