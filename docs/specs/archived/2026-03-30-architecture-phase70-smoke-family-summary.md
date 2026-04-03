# Phase 70 Smoke Family Summary

## Title
Phase 70 - Add scenario-family summary section to baseline governance smoke markdown report

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
- `scripts/smoke-baseline-governance-scope.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase70-smoke-family-summary.md`

## Context
Smoke report currently lists per-case results, but lacks grouped summaries that accelerate troubleshooting.

## Problem Statement
Without family-level aggregation, engineers must scan long case tables to locate failure clusters.

## Scope
- Add scenario-family aggregation in smoke markdown report.
- Families:
  - `scope`
  - `preflight`
  - `update`
  - `custom-path`
  - `template`
- Include per-family totals/pass/fail in markdown output.

## Non-Goals
- No runtime service/domain behavior changes.
- No changes to smoke JSON/Prometheus contracts.

## Options Considered
- Option A: keep per-case-only report.
- Option B: add family summary table.

## Selected Approach
Choose Option B for faster operator diagnosis and incident triage.

## Data / Contract Impact
- Markdown report gains a new `Family Summary` section.
- JSON/Prom outputs remain unchanged.

## Risks
- Minimal; reporting-only logic.

## Rollback Plan
- Remove family summary section generation from smoke markdown output.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase70-smoke-family-summary.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- inspect:
  - `build/migration-governance/baseline-scope-smoke.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase70-smoke-family-summary.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to improve smoke-report observability without changing governance contracts.

## Implementation Summary
- Added smoke markdown family summary aggregation logic grouped by:
  - `scope`
  - `preflight`
  - `update`
  - `custom-path`
  - `template`
- Added `Family Summary` section to
  `build/migration-governance/baseline-scope-smoke.md` with per-family
  total/pass/fail counters.
- Kept smoke JSON/Prometheus artifact contracts unchanged.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase70-smoke-family-summary.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- verify report section in:
  - `build/migration-governance/baseline-scope-smoke.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
