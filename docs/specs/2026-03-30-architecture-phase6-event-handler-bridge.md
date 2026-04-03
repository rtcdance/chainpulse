# Phase 6 Event Query Handler Domain Bridge

## Title
Phase 6 - Add Optional Domain Query Bridge to EventQueryHandler

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
- `pkg/plugins/api/event_query_handler.go`
- `pkg/domain/query/contracts.go`

## Context
EventQueryHandler currently depends on retrieval service only. We need a safe incremental bridge to domain query contract without replacing existing flow.

## Problem Statement
No bridge exists for handler-level adoption of domain query service in compatibility mode.

## Scope
- Add optional domain query service field and setter on EventQueryHandler.
- In `HandleGetEventByID`, add low-risk fallback to domain query only when:
  - retrieval service returns not found
  - id input matches hash-like pattern
  - domain query service is configured
- Keep existing behavior unchanged in default path.

## Non-Goals
- No replacement of retrieval service flow.
- No route or API contract changes.
- No broad handler migration in this phase.

## Options Considered
- Option A: Fully replace retrieval service with domain service.
- Option B: Add optional bridge fallback for compatibility.

## Selected Approach
Option B for minimal risk and reversible rollout.

## Data / Contract Impact
No external contract changes.

## Risks
- Risk: fallback may return event without metadata.
- Mitigation: fallback only triggers after not-found and response conversion already tolerates nil metadata.

## Rollback Plan
- Revert optional bridge additions in `event_query_handler.go`.

## Test and Verification Plan
- Compile-level validation.
- Manual logic review for fallback guard conditions.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase6-event-handler-bridge.md`
- `scripts/dev-micro-loop.sh --mode fast` when Go toolchain is available.

## Review Notes
- Approved for optional compatibility bridge only.

## Implementation Summary
- Add optional domain query fallback path in event handler.

## Final Verification
- Default retrieval path unchanged.
- Optional domain bridge available for phased adoption.
