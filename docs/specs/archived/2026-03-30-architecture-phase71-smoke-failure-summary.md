# Phase 71 Smoke Failure Summary

## Title
Phase 71 - Add failure-first summary section to baseline governance smoke markdown report

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
- `docs/specs/2026-03-30-architecture-phase71-smoke-failure-summary.md`

## Context
Phase 70 added family aggregation, but when failures occur operators still need to scan the full case table to find them.

## Problem Statement
Without a failure-first section, smoke markdown remains slower to use during incident response and CI troubleshooting.

## Scope
- Add a compact `Failure Summary` section to smoke markdown report.
- Show the section only when there are failed smoke cases.
- Keep JSON and Prometheus artifacts unchanged.

## Non-Goals
- No runtime service/domain behavior changes.
- No changes to smoke execution semantics.

## Options Considered
- Option A: keep only full case table.
- Option B: add failure-first markdown summary.

## Selected Approach
Choose Option B for faster human triage with minimal reporting complexity.

## Data / Contract Impact
- Markdown report gains conditional `Failure Summary` section.
- JSON/Prometheus contracts remain unchanged.

## Risks
- Minimal; reporting-only logic.

## Rollback Plan
- Remove conditional failure summary block from smoke markdown output.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase71-smoke-failure-summary.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- inspect:
  - `build/migration-governance/baseline-scope-smoke.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase71-smoke-failure-summary.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to improve smoke markdown troubleshooting ergonomics.

## Implementation Summary
- Added conditional `Failure Summary` markdown section to
  `build/migration-governance/baseline-scope-smoke.md`.
- The section is rendered before `Case Results` and only appears when failed
  smoke scenarios exist.
- Kept smoke JSON/Prometheus outputs and scenario execution semantics unchanged.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase71-smoke-failure-summary.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- verify conditional section behavior in:
  - `build/migration-governance/baseline-scope-smoke.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
