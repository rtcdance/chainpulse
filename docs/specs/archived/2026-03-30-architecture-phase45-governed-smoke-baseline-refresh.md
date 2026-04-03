# Phase 45 Governed Smoke Baseline Refresh

## Title
Phase 45 - Add governed refresh workflow for baseline-scope smoke baseline with changelog linkage

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
- `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 44 introduced smoke baseline delta checks, but refresh governance for the smoke baseline was not yet unified with existing KPI/health baseline refresh and changelog policies.

## Problem Statement
If smoke baseline updates are not governed in the same workflow, teams can bypass audit discipline and weaken regression detection reliability.

## Scope
- Extend baseline update workflow to optionally refresh smoke baseline:
  - `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- Add smoke baseline refresh controls:
  - `CHAINPULSE_MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE_FILE`
  - `CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=true|false`
- Extend baseline governance check to enforce changelog requirement when smoke baseline changes.
- Extend scope-alignment logic:
  - any health-related baseline change (`health` or `smoke`) maps to `health-only` unless KPI baseline also changes, then `dual`.

## Non-Goals
- No automatic smoke baseline refresh in CI.
- No runtime service behavior changes.

## Options Considered
- Option A: keep smoke baseline refresh outside governed update script.
- Option B: unify smoke baseline refresh with governed baseline lifecycle.

## Selected Approach
Choose Option B for consistent enterprise governance and auditable baseline management.

## Data / Contract Impact
- No runtime API/domain contract changes.
- Governance process contract expands to include smoke baseline refresh controls.

## Risks
- Over-refreshing smoke baseline can hide regressions.
- Mitigation: keep refresh behind explicit guarded update command and changelog linkage.

## Rollback Plan
- Set `CHAINPULSE_REFRESH_BASELINE_SCOPE_SMOKE_BASELINE=false` in refresh workflow.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase45-governed-smoke-baseline-refresh.md`
- `bash -n scripts/update-migration-governance-baseline.sh`
- `bash -n scripts/check-migration-baseline-governance.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase45-governed-smoke-baseline-refresh.md`
- `./scripts/check-migration-baseline-governance.sh`

## Review Notes
- Approved to align smoke baseline refresh with existing governed baseline policy.

## Implementation Summary
- Added smoke baseline refresh support and env controls in baseline update script.
- Extended baseline governance checks and scope-alignment logic for smoke baseline changes.
- Updated CI env and operations docs for explicit smoke baseline control variables.

## Final Verification
- Baseline governance checks pass with smoke baseline included in governance path.
