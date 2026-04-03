# Phase 80 Governance Overview Legend

## Title
Phase 80 - Add legend for governance overview severity levels

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
- `scripts/append-governance-overview-summary.sh`
- `scripts/test-append-governance-overview-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase80-governance-overview-legend.md`

## Context
Phase 79 introduced normalized `Level` values in the governance overview, but
their meaning still relies on prior knowledge.

## Problem Statement
Without a legend, operators may misinterpret `ok|warn|fail|info` semantics when
reading the CI overview.

## Scope
- Add a short legend block below `Governance Overview`.
- Explain the meaning of:
  - `ok`
  - `warn`
  - `fail`
  - `info`

## Non-Goals
- No changes to severity mapping logic.
- No UI styling changes.

## Options Considered
- Option A: keep implicit semantics.
- Option B: add an explicit overview legend.

## Selected Approach
Choose Option B for clearer operator ergonomics and lower interpretation cost.

## Data / Contract Impact
- `Governance Overview` gains a small legend block after the table.
- Existing row content remains unchanged.

## Risks
- Minimal; summary rendering only.

## Rollback Plan
- Remove the legend block from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase80-governance-overview-legend.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase80-governance-overview-legend.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make overview severity semantics self-explanatory.

## Implementation Summary
- Added an inline legend block to `Governance Overview` explaining
  `ok|warn|fail|info` semantics.
- Extended overview regression test coverage and updated operations/architecture
  docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase80-governance-overview-legend.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
