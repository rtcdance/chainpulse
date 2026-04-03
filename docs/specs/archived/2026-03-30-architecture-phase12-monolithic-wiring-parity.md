# Phase 12 Monolithic Wiring Parity

## Title
Phase 12 - Align monolithic bootstrap wiring with microservice runtime route composition

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
- `cmd/monolithic/chainpulse/main.go`
- `docs/ARCHITECTURE.md`

## Context
Microservice API path now includes domain bridge, event/query handler instantiation, and runtime route composition. Monolithic bootstrap still used a reduced wiring path.

## Problem Statement
Without monolithic parity, debug mode and production-like behavior diverge, reducing confidence in architecture migration outcomes.

## Scope
- Add database manager bootstrap in monolithic path.
- Instantiate and initialize query service and domain query bridge.
- Instantiate and wire event query/subscription/health handlers into API gateway plugin.
- Enable runtime route composition in monolithic startup path.
- Add graceful shutdown for newly introduced components.

## Non-Goals
- No deep indexing subsystem refactor.
- No schema or protocol changes.
- No broad deployment-mode unification in this phase.

## Options Considered
- Option A: Keep monolithic lightweight and accept behavior drift.
- Option B: Bring monolithic wiring to parity with microservice bootstrap.

## Selected Approach
Choose Option B:
- Reuse existing components and initialization order proven in API service.
- Keep additive startup/shutdown steps and explicit status logs.

## Data / Contract Impact
No external contract change.

## Risks
- Additional bootstrap dependencies can increase startup failure surface.
- Mitigation: fail-fast checks with explicit logs and bounded context timeouts.

## Rollback Plan
Revert monolithic bootstrap additions and return to prior lightweight gateway setup.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase12-monolithic-wiring-parity.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase12-monolithic-wiring-parity.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to support single-binary debug workflow with production-like wiring behavior.

## Implementation Summary
- Monolithic bootstrap now wires domain/query/event runtime components similarly to microservice path.

## Final Verification
- Fast gate passes after monolithic wiring parity update.
