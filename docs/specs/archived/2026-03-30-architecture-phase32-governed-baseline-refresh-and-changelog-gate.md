# Phase 32 Governed Baseline Refresh and Changelog Gate

## Title
Phase 32 - Add guarded baseline refresh workflow and changelog gate for migration KPI baseline updates

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
- `scripts/update-migration-governance-baseline.sh`
- `scripts/check-migration-baseline-governance.sh`
- `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- `docs/operations/MIGRATION_GOVERNANCE_BASELINE.prom`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/ARCHITECTURE.md`

## Context
Phase 31 introduced KPI delta reports, but baseline updates lacked governance controls and explicit change history requirements.

## Problem Statement
Uncontrolled baseline refresh can hide meaningful KPI regressions and reduce review trust.

## Scope
- Add guarded baseline update script requiring explicit env opt-in:
  - `CHAINPULSE_ALLOW_BASELINE_UPDATE=true`
- Add baseline governance check script that enforces:
  - baseline changes must include changelog updates
  - configurable diff ref for CI comparison
- Add baseline changelog document as first-class artifact.
- Integrate baseline governance check into CI `policy-contract` job.
- Integrate baseline governance check into local full micro-loop.
- Add Make targets for baseline check/update workflow.

## Non-Goals
- No automatic baseline update in CI.
- No PR auto-comment posting in this phase.

## Options Considered
- Option A: allow direct baseline edits without controls.
- Option B: guarded update + changelog gate + CI enforcement.

## Selected Approach
Choose Option B for auditable, reviewable baseline maintenance.

## Data / Contract Impact
No runtime contract changes; governance workflow and artifacts expanded.

## Risks
- Extra governance step may slow emergency updates.
- Mitigation: explicit env override and clear scripts for manual operation.

## Rollback Plan
Disable baseline governance step in CI and use export/compare-only workflow.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase32-governed-baseline-refresh-and-changelog-gate.md`
- `./scripts/export-migration-governance-kpi.sh`
- `./scripts/compare-migration-governance-kpi.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase32-governed-baseline-refresh-and-changelog-gate.md`
- `./scripts/check-migration-baseline-governance.sh`

## Review Notes
- Approved for auditable baseline governance.

## Implementation Summary
- Added guarded baseline refresh and CI changelog gate for baseline diffs.

## Final Verification
- Spec gate and baseline governance checks pass with no unintended baseline drift.
