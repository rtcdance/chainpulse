# Phase 13 Shared Bootstrap Constructor

## Title
Phase 13 - Introduce shared runtime bootstrap constructor for api-service and monolithic

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
- `pkg/application/bootstrap/runtime_wiring.go`
- `cmd/microservices/api-service/main.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/ARCHITECTURE.md`

## Context
After Phase 12, both startup paths had near-identical runtime component initialization logic, increasing maintenance cost and drift risk.

## Problem Statement
Duplicated bootstrap logic across deployment modes makes future architecture iterations error-prone and slows migration.

## Scope
- Add shared runtime wiring constructor in application layer.
- Migrate api-service and monolithic mains to use shared constructor.
- Keep behavior and startup ordering consistent.

## Non-Goals
- No full command bootstrap rewrite.
- No deployment workflow changes.
- No external API contract changes.

## Options Considered
- Option A: Keep duplicated bootstrap logic.
- Option B: Extract shared constructor and lifecycle helper.

## Selected Approach
Choose Option B:
- Introduce `bootstrap.BuildRuntimeWiring(...)` and `RuntimeWiring.Close(...)`.
- Replace duplicated init/close sequences in both commands.

## Data / Contract Impact
No external contract impact; internal additive package.

## Risks
- Shared constructor bug can affect both startup modes.
- Mitigation: keep constructor small, deterministic, and gate with fast checks.

## Rollback Plan
Revert constructor usage in both commands to prior local initialization.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase13-shared-bootstrap-constructor.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase13-shared-bootstrap-constructor.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to reduce startup drift and improve maintainability.

## Implementation Summary
- Added shared runtime constructor and migrated two command entrypoints to use it.

## Final Verification
- Fast gate passes with shared bootstrap constructor integrated.
