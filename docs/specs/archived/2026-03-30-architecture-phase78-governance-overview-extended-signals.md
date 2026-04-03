# Phase 78 Governance Overview Extended Signals

## Title
Phase 78 - Extend governance overview with registry health and owner drift

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
- `build/migration-governance/ticket-registry-health.md`
- `build/migration-governance/ticket-registry-health-delta.md`
- `build/migration-governance/migration-owner-drift-report.md`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase78-governance-overview-extended-signals.md`

## Context
Phase 77 added a compact smoke/resolver overview block, but ticket-registry
health and owner-drift still require reading separate summary sections.

## Problem Statement
Without registry-health and owner-drift rows, the governance overview is still
incomplete as a first-pass operational snapshot.

## Scope
- Extend `Governance Overview` to include:
  - ticket registry health
  - owner drift
- Keep the overview compact by reusing a single table format.
- Preserve existing detailed summary sections below the overview.

## Non-Goals
- No changes to ticket-registry health export/delta contracts.
- No changes to owner-drift report generation semantics.

## Options Considered
- Option A: keep overview limited to smoke/resolver.
- Option B: extend overview with registry-health and owner-drift rows.

## Selected Approach
Choose Option B so the top of the CI summary provides a more complete
governance operating snapshot.

## Data / Contract Impact
- `Governance Overview` table gains additional rows for registry health and
  owner drift.
- Existing detailed artifacts remain unchanged.

## Risks
- Low; summary rendering only.

## Rollback Plan
- Remove registry-health and owner-drift parsing from the overview helper.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase78-governance-overview-extended-signals.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- `./scripts/compare-baseline-resolver-test.sh`
- `./scripts/compare-ticket-registry-health.sh`
- `./scripts/export-migration-owner-drift-report.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase78-governance-overview-extended-signals.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve overview completeness without changing detailed reports.

## Implementation Summary
- Extended `scripts/append-governance-overview-summary.sh` to include
  ticket-registry health and owner-drift rows.
- Extended `scripts/test-append-governance-overview-summary.sh` to validate the
  broader overview table contract.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase78-governance-overview-extended-signals.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- `./scripts/compare-baseline-resolver-test.sh`
- `./scripts/compare-ticket-registry-health.sh`
- `./scripts/export-migration-owner-drift-report.sh`
