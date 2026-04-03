# Phase 11 Runtime E2E Event Route Validation

## Title
Phase 11 - Add runtime-composition test proving `/events/{id}` reaches domain-first path

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
- `pkg/plugins/api/gateway_runtime_integration_test.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 10 enabled runtime route composition in API gateway, but we still need an automated proof that the composed request path can hit migrated domain-first event query logic.

## Problem Statement
Without an end-to-end runtime composition test, regressions can silently break the new wiring.

## Scope
- Add a focused runtime integration test in API plugin package.
- Validate that composed route handling for `/events/{id}` returns domain-first result.

## Non-Goals
- No external service/container-based E2E in this phase.
- No performance/load test in this phase.

## Options Considered
- Option A: Only rely on unit tests for handlers.
- Option B: Add composition-level test over gateway runtime integration.

## Selected Approach
Choose Option B:
- Build gateway plugin with wired handlers in test.
- Invoke composed router integration for `/events/{id}`.
- Assert domain-first event id is returned.

## Data / Contract Impact
No production contract change.

## Risks
- Test may become brittle if internal composition contract changes.
- Mitigation: keep assertions minimal and behavior-focused.

## Rollback Plan
Remove composition test if architecture changes invalidate this layer.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase11-runtime-e2e-event-route.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase11-runtime-e2e-event-route.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as guardrail test for runtime composition milestone.

## Implementation Summary
- Added runtime composition test covering event-by-id domain-first path.

## Final Verification
- Fast gate passes with new runtime integration test.
