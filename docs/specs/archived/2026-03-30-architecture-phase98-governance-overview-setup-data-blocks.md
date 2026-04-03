# Phase 98 Governance Overview Setup Data Blocks

## Title
Phase 98 - Convert overview setup helpers to compact data blocks

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
- `docs/specs/2026-03-30-architecture-phase98-governance-overview-setup-data-blocks.md`

## Context
Phase 97 added descriptor-based assertions for non-primary states, but setup
helpers still repeat direct fixture writer calls for each scenario.

## Problem Statement
Repeated setup calls make scenario evolution noisier than necessary and create
more places to touch when one fixture field changes.

## Scope
- Introduce compact setup data blocks for overview scenarios.
- Keep existing scenario meanings, coverage, and rendered expectations.
- Preserve the setup/run/assert structure added in Phase 96.

## Non-Goals
- No changes to overview rendering logic.
- No changes to aggregate or row assertion behavior.

## Options Considered
- Option A: keep direct fixture calls in each setup helper.
- Option B: route setup through scenario data descriptors.

## Selected Approach
Choose Option B so the setup layer matches the existing row and aggregate
descriptor style and stays easier to evolve.

## Data / Contract Impact
- No output contract changes.
- Test structure only.

## Risks
- Minimal; compact setup encoding must remain readable.

## Rollback Plan
- Restore direct fixture writer calls inside setup helpers.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase98-governance-overview-setup-data-blocks.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase98-governance-overview-setup-data-blocks.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to finish the setup layer refactor in the overview regression
  harness.

## Implementation Summary
- Added compact setup descriptor helpers for overview scenarios.
- Reduced repeated direct fixture writes while preserving scenario readability.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase98-governance-overview-setup-data-blocks.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
