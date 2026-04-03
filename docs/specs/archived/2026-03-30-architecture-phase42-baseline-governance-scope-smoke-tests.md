# Phase 42 Baseline Governance Scope Smoke Tests

## Title
Phase 42 - Add baseline governance scope smoke tests for kpi-only/health-only/dual paths

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
- `scripts/check-migration-baseline-governance.sh`
- `scripts/check-migration-changelog-quality.sh`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 41 introduced scope tagging and scope-alignment enforcement, but there was no dedicated smoke suite that continuously proves behavior across expected and failure paths.

## Problem Statement
Without scenario-level smoke tests, regressions in scope-alignment policy may not be detected quickly when shell scripts evolve.

## Scope
- Add isolated smoke script that provisions temporary git repositories and validates:
  - `dual` scope success when both baselines change
  - `kpi-only` scope success when only KPI baseline changes
  - `health-only` scope success when only health baseline changes
  - expected failure when scope does not match baseline diff
- Integrate smoke script into:
  - CI `policy-contract` job
  - local full micro-loop
  - Makefile target

## Non-Goals
- No runtime service behavior changes.
- No e2e network-level integration tests.

## Options Considered
- Option A: rely on ad-hoc manual script checks.
- Option B: codify baseline-scope behavior as repeatable smoke tests.

## Selected Approach
Choose Option B to provide deterministic regression detection for governance policy scripts.

## Data / Contract Impact
- No runtime API or domain contract changes.
- Governance verification workflow gains scenario-level smoke coverage.

## Risks
- Smoke script can become stale if baseline file formats change.
- Mitigation: keep fixtures minimal and aligned with script defaults.

## Rollback Plan
- Remove smoke script from CI and local loop while retaining core governance checks.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase42-baseline-governance-scope-smoke-tests.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase42-baseline-governance-scope-smoke-tests.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to strengthen governance-policy confidence with explicit scenario smoke tests.

## Implementation Summary
- Added isolated smoke script with three success scenarios and one expected-failure scenario.
- Wired smoke tests into CI, local full loop, and Makefile.
- Updated operations governance workflow documentation.

## Final Verification
- Smoke script passes and reports all scenarios successful.
