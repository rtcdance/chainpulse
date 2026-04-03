# Phase 95 Governance Overview Aggregate Scenario Helper

## Title
Phase 95 - Convert aggregate overview assertions to scenario-driven helpers

## Type
- architecture
- operations

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
platform-team

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `scripts/test-append-governance-overview-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase95-governance-overview-aggregate-scenario-helper.md`

## Context
Phase 94 converted primary row checks to a compact scenario loop, but aggregate
state assertions still appear as repeated helper calls across `fail`, `warn`,
`info`, and `ok`.

## Problem Statement
Repeated aggregate assertion blocks make it harder to scan the full aggregate
state matrix and increase the cost of adjusting one top-level field.

## Scope
- Introduce a compact aggregate-scenario helper for overview tests.
- Preserve existing aggregate coverage and exact expected strings.
- Keep rendering logic unchanged.

## Non-Goals
- No changes to overview output format.
- No changes to fixture semantics or row coverage.

## Options Considered
- Option A: keep separate helper invocations per aggregate state.
- Option B: move aggregate expectations into a small scenario loop.

## Selected Approach
Choose Option B so the aggregate-state matrix reads as a concise test table.

## Data / Contract Impact
- No output contract changes.
- Test structure only.

## Risks
- Minimal; compact scenario encoding must stay readable.

## Rollback Plan
- Restore explicit aggregate helper calls for each scenario.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase95-governance-overview-aggregate-scenario-helper.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase95-governance-overview-aggregate-scenario-helper.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make the aggregate-state matrix easier to read and extend.

## Implementation Summary
- Added a scenario-driven aggregate assertion helper.
- Reduced repeated aggregate helper invocations while preserving exact checks.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase95-governance-overview-aggregate-scenario-helper.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
