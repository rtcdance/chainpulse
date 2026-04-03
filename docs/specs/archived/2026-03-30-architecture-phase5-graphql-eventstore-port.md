# Phase 5 GraphQL EventStore Port Migration

## Title
Phase 5 - Move GraphQL Module EventStore Dependency to Domain Query Port

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
- `pkg/plugins/api/graphql/resolvers.go`
- `pkg/plugins/api/graphql/schema.go`
- `pkg/plugins/api/graphql/mutations.go`

## Context
GraphQL module still depends directly on legacy `pkg/services/query.EventStore`, which keeps adapter/plugin layer coupled to legacy service package.

## Problem Statement
The dependency boundary for GraphQL EventStore should target domain contract, not legacy service package.

## Scope
- Add `EventStore` contract to `pkg/domain/query`.
- Update GraphQL resolver/schema/mutation types to depend on domain contract.
- Preserve existing runtime behavior.

## Non-Goals
- No GraphQL feature behavior changes.
- No API schema changes.
- No startup wiring replacement in this phase.

## Options Considered
- Option A: Keep legacy dependency until full migration.
- Option B: Move module dependency boundary now with contract parity.

## Selected Approach
Option B for incremental architecture alignment.

## Data / Contract Impact
No external API/data contract changes.

## Risks
- Risk: mismatch between legacy interface and new domain interface.
- Mitigation: mirror method signatures exactly in this phase.

## Rollback Plan
- Revert GraphQL imports/types and remove new domain `EventStore` contract file.

## Test and Verification Plan
- Compile-level verification of import/type updates.
- Behavior unchanged by construction (dependency type only).

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase5-graphql-eventstore-port.md`
- `scripts/dev-micro-loop.sh --mode fast` when Go toolchain is available.

## Review Notes
- Approved for dependency boundary migration slice.

## Implementation Summary
- GraphQL module now consumes domain query EventStore contract.

## Final Verification
- GraphQL behavior unchanged.
- Legacy coupling reduced at plugin boundary.
