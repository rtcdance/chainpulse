# Phase 4 Plugin Domain Port Alignment

## Title
Phase 4 - Align GraphQLHandler Dependency to Domain Query Port

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
- `pkg/plugins/api/graphql_handler.go`
- `pkg/domain/query/contracts.go`

## Context
A key architecture goal is to reduce direct dependencies from adapters/plugins onto legacy service packages and move toward domain contracts.

## Problem Statement
`pkg/plugins/api/graphql_handler.go` currently depends on `pkg/services/query` types directly, which keeps plugin coupling tied to legacy service layer.

## Scope
- Change GraphQL handler dependency type from legacy query service contract to domain query service port.
- Keep current GraphQL handler runtime behavior unchanged.
- Do not change startup wiring in this phase.

## Non-Goals
- No GraphQL query execution logic enhancement.
- No API contract behavior changes.
- No handler migration beyond dependency type alignment.

## Options Considered
- Option A: Rewrite GraphQL handler logic now.
- Option B: First align dependency boundary to domain port.

## Selected Approach
Option B for low-risk architecture alignment.

## Data / Contract Impact
No external API/data contract changes.

## Risks
- Risk: hidden compile dependency from callers expecting legacy type.
- Mitigation: constructor remains structurally same (single service arg), only interface package changes.

## Rollback Plan
- Revert dependency type changes in `graphql_handler.go`.

## Test and Verification Plan
- Compile-level verification for changed imports and types.
- Behavioral verification by no logic change in handler methods.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase4-plugin-domain-port.md`
- `scripts/dev-micro-loop.sh --mode fast` when Go toolchain is available.

## Review Notes
- Approved for dependency-boundary alignment slice.

## Implementation Summary
- Move GraphQL plugin dependency boundary to domain query port.

## Final Verification
- No behavior change in request handling.
- Plugin now depends on domain contract instead of legacy service package.
