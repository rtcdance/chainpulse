# Phase 9 Domain-First Event Query by ID

## Title
Phase 9 - Migrate event-by-id handler path to domain-first with retrieval fallback

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
- `pkg/plugins/api/event_query_handler.go`
- `pkg/plugins/api/event_query_handler_test.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 8 completed runtime handler instantiation and bridge wiring. Event query by ID still used retrieval-first with domain bridge fallback.

## Problem Statement
We need at least one concrete production path migrated to domain-first while preserving safety fallback to legacy retrieval behavior.

## Scope
- For hash-like event IDs, execute `domainQuery.QueryByHash` first when bridge is configured.
- If domain query returns nil/error, fall back to retrieval service path.
- Add focused tests for domain-first hit and fallback behavior.

## Non-Goals
- No migration of list or filter query paths.
- No removal of retrieval service dependencies.
- No API schema changes.

## Options Considered
- Option A: Keep retrieval-first until all routes are ready.
- Option B: Migrate one route incrementally with guarded fallback.

## Selected Approach
Choose Option B:
- Keep hash-shape guard (`0x` + 66 chars) for domain-first path.
- Preserve retrieval behavior for non-hash IDs and domain misses/errors.
- Add metrics for domain-first success/error.

## Data / Contract Impact
No external API contract changes.

## Risks
- Divergence between domain query and retrieval data shape.
- Mitigation: reuse existing response conversion and fallback logic.

## Rollback Plan
Revert handler ordering to retrieval-first path.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase9-domain-first-event-by-id.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase9-domain-first-event-by-id.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as first concrete route migration toward domain-first execution.

## Implementation Summary
- Event-by-id handler now attempts domain-first query for hash IDs and falls back safely.

## Final Verification
- Fast gate passes with targeted tests for domain-first and fallback behavior.
