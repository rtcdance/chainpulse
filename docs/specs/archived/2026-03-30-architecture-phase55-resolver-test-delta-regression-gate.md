# Phase 55 Resolver Test Delta Regression Gate

## Title
Phase 55 - Add baseline resolver test delta comparison and regression signaling

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
- `scripts/compare-baseline-resolver-test.sh`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/guides/OPERATIONS_GUIDE.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 54 added resolver test artifacts, but there is no baseline-vs-current delta script and no regression signal contract for resolver test trends.

## Problem Statement
Without a resolver test delta comparator, CI and local governance loops cannot detect trend regressions (for example failed case count increases) in a standardized way.

## Scope
- Add resolver test delta comparator script:
  - input: current/baseline Prometheus snapshots
  - output: markdown + tsv delta reports
  - regression signals and status field
- Wire comparator into full micro-loop, Makefile, and CI policy-contract workflow.
- Append resolver test delta markdown to CI step summary.
- Document baseline path, outputs, and failure-mode env control.

## Non-Goals
- No runtime service/domain behavior change.
- No baseline refresh workflow extension in this phase.

## Options Considered
- Option A: keep resolver trend checks manual.
- Option B: add governed compare script with standard warn/enforce mode.

## Selected Approach
Choose Option B to align resolver observability with existing KPI/smoke/registry delta governance pattern.

## Data / Contract Impact
- Adds resolver delta artifact contract:
  - `build/migration-governance/baseline-resolver-test-delta.tsv`
  - `build/migration-governance/baseline-resolver-test-delta.md`
- Uses baseline file:
  - `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom`

## Risks
- Minimal; additive check and reporting only.

## Rollback Plan
- Remove resolver delta compare script and CI/local wiring.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase55-resolver-test-delta-regression-gate.md`
- `bash -n scripts/compare-baseline-resolver-test.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`
- verify generated files:
  - `build/migration-governance/baseline-resolver-test-delta.tsv`
  - `build/migration-governance/baseline-resolver-test-delta.md`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase55-resolver-test-delta-regression-gate.md`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`

## Review Notes
- Approved to extend resolver governance into standardized trend regression checks.

## Implementation Summary
- Added `scripts/compare-baseline-resolver-test.sh` with:
  - baseline/current Prom metrics comparison
  - regression signal evaluation (`failed_cases_increased`, `status_regressed_to_fail`)
  - warn/enforce gate via `CHAINPULSE_BASELINE_RESOLVER_TEST_DELTA_FAILURE_MODE`
- Added resolver baseline file:
  - `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom`
- Wired compare step into:
  - `scripts/dev-micro-loop.sh` (full mode)
  - `Makefile` target `compare-baseline-resolver-test`
  - CI `policy-contract` workflow
- Added CI summary append for resolver delta markdown.
- Updated operations/index/architecture docs with resolver delta contract.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase55-resolver-test-delta-regression-gate.md`
- `bash -n scripts/compare-baseline-resolver-test.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`
- `make compare-baseline-resolver-test`
- `./scripts/check-migration-baseline-governance.sh`
- `./scripts/check-migration-changelog-quality.sh`
