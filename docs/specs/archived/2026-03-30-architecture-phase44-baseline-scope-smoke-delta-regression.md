# Phase 44 Baseline Scope Smoke Delta Regression

## Title
Phase 44 - Add baseline-scope smoke delta regression check and CI trend summary

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
- `scripts/compare-baseline-scope-smoke.sh`
- `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 43 exported smoke artifacts and CI summary, but there was no baseline-delta regression gate to highlight cross-run trend degradation.

## Problem Statement
Without baseline-delta comparison, smoke results are point-in-time only and cannot automatically flag trend regression.

## Scope
- Add baseline-scope smoke delta script:
  - `scripts/compare-baseline-scope-smoke.sh`
  - outputs:
    - `build/migration-governance/baseline-scope-smoke-delta.tsv`
    - `build/migration-governance/baseline-scope-smoke-delta.md`
- Add baseline snapshot file:
  - `docs/operations/MIGRATION_BASELINE_SCOPE_SMOKE_BASELINE.prom`
- Add regression policy mode:
  - `CHAINPULSE_BASELINE_SCOPE_SMOKE_DELTA_FAILURE_MODE=warn|enforce`
- Integrate delta generation into:
  - CI `policy-contract` job
  - local full micro-loop
  - Makefile target
- Append smoke-delta markdown to CI summary.

## Non-Goals
- No runtime service behavior changes.
- No automatic baseline refresh for smoke baseline in this phase.

## Options Considered
- Option A: keep smoke metrics without baseline trend compare.
- Option B: add baseline trend compare and optional enforce gate.

## Selected Approach
Choose Option B for enterprise-grade regression visibility and controlled enforcement.

## Data / Contract Impact
- Governance artifact contract expanded with smoke delta outputs.
- No API/domain model contract impact.

## Risks
- Baseline staleness may create noisy deltas.
- Mitigation: explicit baseline file governance and warn-by-default failure mode.

## Rollback Plan
- Stop running smoke-delta compare in CI/local loop and keep smoke snapshot only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase44-baseline-scope-smoke-delta-regression.md`
- `bash -n scripts/compare-baseline-scope-smoke.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/compare-baseline-scope-smoke.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase44-baseline-scope-smoke-delta-regression.md`
- `./scripts/compare-baseline-scope-smoke.sh`

## Review Notes
- Approved to strengthen governance trend detection and CI summary signal quality.

## Implementation Summary
- Added smoke baseline delta script and baseline file.
- Added CI/local loop/Makefile integration for smoke delta compare.
- Updated governance docs for smoke delta artifacts and failure mode.

## Final Verification
- Smoke delta artifacts are generated and regression status is reported.
