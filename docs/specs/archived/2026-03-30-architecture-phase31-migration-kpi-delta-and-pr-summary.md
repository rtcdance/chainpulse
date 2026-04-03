# Phase 31 Migration KPI Delta and PR Summary

## Title
Phase 31 - Add migration governance KPI delta comparison and PR summary draft generation

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
- `scripts/compare-migration-governance-kpi.sh`
- `scripts/export-migration-governance-kpi.sh`
- `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 30 added migration governance KPI export and query templates, but lacked commit-level delta tracking and PR-ready summary output.

## Problem Statement
Without a standard delta comparison, reviewers cannot quickly assess governance KPI movement per change set.

## Scope
- Add KPI comparison script:
  - compares current `.prom` snapshot with baseline `.prom`
  - outputs delta `.tsv` and markdown summary
- Add initial baseline file under operations docs.
- Integrate CI to:
  - run comparison
  - append delta report into job summary
  - include delta artifacts in uploaded bundle
- Add full micro-loop step for local delta generation.
- Add operation docs section for PR delta workflow.

## Non-Goals
- No GitHub API comment posting automation in this phase.
- No historical time-series storage backend changes.

## Options Considered
- Option A: raw KPI export only.
- Option B: export + baseline comparison + PR draft summary.

## Selected Approach
Choose Option B to improve reviewer signal and migration governance transparency.

## Data / Contract Impact
Adds baseline and delta artifact files:
- `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
- `build/migration-governance/migration-governance-delta.tsv`
- `build/migration-governance/migration-governance-delta.md`

## Risks
- Baseline staleness can mask expected changes.
- Mitigation: explicit baseline file in repo with review-driven updates.

## Rollback Plan
Remove compare step from CI and retain KPI export-only flow.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase31-migration-kpi-delta-and-pr-summary.md`
- `./scripts/export-migration-governance-kpi.sh`
- `./scripts/compare-migration-governance-kpi.sh`
- `./scripts/check-migration-manifest.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase31-migration-kpi-delta-and-pr-summary.md`
- `./scripts/compare-migration-governance-kpi.sh`

## Review Notes
- Approved to improve governance KPI change visibility in CI and reviews.

## Implementation Summary
- Added baseline compare script and CI/job summary integration for migration KPI delta.

## Final Verification
- Spec gate, manifest check, KPI export, and KPI compare all pass.
