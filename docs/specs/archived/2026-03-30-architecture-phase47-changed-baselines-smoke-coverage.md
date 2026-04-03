# Phase 47 Changed-Baselines Smoke Coverage

## Title
Phase 47 - Add `changed_baselines` mismatch scenario to baseline governance smoke suite

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
- `scripts/check-migration-changelog-quality.sh`
- `scripts/check-migration-baseline-governance.sh`
- `docs/ARCHITECTURE.md`

## Context
Phase 46 added `changed_baselines` contract and alignment enforcement, but smoke coverage did not yet include a dedicated negative scenario for changed-set mismatch.

## Problem Statement
Without a built-in mismatch smoke case, regressions in changed-set enforcement could slip in unnoticed and rely on ad-hoc manual checks.

## Scope
- Extend baseline governance smoke suite with explicit negative case:
  - `changed_baselines_mismatch_should_fail`
- Keep existing scenario set:
  - `dual` pass
  - `kpi-only` pass
  - `health-only` pass
  - `scope` mismatch fail
- Validate total-case artifacts reflect new scenario count.

## Non-Goals
- No runtime service code changes.
- No CI job topology changes in this phase.

## Options Considered
- Option A: keep mismatch testing as manual command-only verification.
- Option B: formalize mismatch case in smoke script.

## Selected Approach
Choose Option B for deterministic, repeatable policy regression coverage.

## Data / Contract Impact
- No API/domain contract changes.
- Smoke artifact totals increase due to added scenario.

## Risks
- None significant beyond routine script maintenance.

## Rollback Plan
- Remove the mismatch scenario from smoke script and revert to previous four-case suite.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase47-changed-baselines-smoke-coverage.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- verify smoke artifacts report 5 total cases.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase47-changed-baselines-smoke-coverage.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to convert changed-baselines mismatch checks into permanent smoke coverage.

## Implementation Summary
- Added a new negative smoke scenario for `changed_baselines` mismatch.
- Smoke artifacts now include updated totals and per-case result row.

## Final Verification
- Smoke suite passes with the new mismatch case included.
