# Phase 2 Query Vertical Slice

## Title
Phase 2 - Introduce Query Domain/Application/Adapter Compatibility Slice

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
- `pkg/domain/query`
- `pkg/application/query`
- `pkg/adapters/query`
- `pkg/services/query`

## Context
Phase 1 introduced layer foundations. Next step is a minimal vertical slice to prove migration direction without runtime disruption.

## Problem Statement
`pkg/services/query` contains mixed concerns and currently has no layer-compatible compatibility facade for incremental migration.

## Scope
- Add domain-level query contract and DTOs.
- Add application-level facade that wraps legacy `pkg/services/query`.
- Add adapter compatibility constructor for wiring use in future bootstrap.
- Keep existing runtime behavior untouched.

## Non-Goals
- No full migration of query internals.
- No API or schema behavior changes.
- No replacement of existing `pkg/services/query` entrypoints.

## Options Considered
- Option A: Migrate query internals immediately.
- Option B: Create compatibility facade first and migrate internals later.

## Selected Approach
Option B for low risk and incremental delivery.

## Data / Contract Impact
No external contract changes. New internal contracts added for migration path.

## Risks
- Risk: duplicated DTO definitions can drift.
- Mitigation: keep one-way mapping wrappers small and explicit.

## Rollback Plan
- Remove new `pkg/domain/query`, `pkg/application/query`, `pkg/adapters/query` files.
- Existing legacy query service remains intact.

## Test and Verification Plan
- Compile-level checks for new packages and imports.
- Verify existing `pkg/services/query` remains unchanged.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase2-query-slice.md`
- `scripts/dev-micro-loop.sh --mode fast` when Go toolchain is available.

## Review Notes
- Approved for incremental architecture rollout.

## Implementation Summary
- Added a compatibility migration slice around query service.

## Final Verification
- Legacy behavior preserved.
- New layer boundaries are explicit for next migration phase.
