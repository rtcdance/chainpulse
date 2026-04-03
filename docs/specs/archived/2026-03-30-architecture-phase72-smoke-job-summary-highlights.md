# Phase 72 Smoke Job Summary Highlights

## Title
Phase 72 - Surface baseline scope smoke highlights in CI job summary

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
- `scripts/append-baseline-scope-smoke-summary.sh`
- `scripts/test-append-baseline-scope-smoke-summary.sh`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase72-smoke-job-summary-highlights.md`

## Context
Phase 71 added a failure-first block to the smoke markdown artifact, but CI still
appends the full artifact without a compact highlight layer.

## Problem Statement
Without a dedicated job-summary highlight section, operators must scan the full
markdown artifact in the CI summary to find smoke status and top failing cases.

## Scope
- Add a helper script that appends baseline scope smoke highlights to a target
  summary file.
- Include top-level smoke status and total/pass/fail counters.
- Reuse the smoke report's `Failure Summary` block when failures exist.
- Wire the helper into the `policy-contract` CI job before the full smoke report
  is appended.

## Non-Goals
- No changes to smoke execution semantics or artifact contracts.
- No changes to baseline delta logic.

## Options Considered
- Option A: keep appending only the full smoke markdown artifact.
- Option B: prepend a compact CI highlight block and retain the full artifact.

## Selected Approach
Choose Option B for faster CI triage while preserving the detailed artifact
already used for auditability.

## Data / Contract Impact
- CI job summary gains a compact smoke highlight section.
- Existing smoke JSON/Prom/Markdown artifact contracts remain unchanged.

## Risks
- Minimal; summary rendering only.

## Rollback Plan
- Remove the helper invocation from CI and delete the helper script.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase72-smoke-job-summary-highlights.md`
- `bash -n scripts/append-baseline-scope-smoke-summary.sh`
- `bash -n scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- verify generated files:
  - `build/migration-governance/baseline-scope-smoke.md`
- inspect CI workflow wiring in:
  - `.github/workflows/ci.yml`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase72-smoke-job-summary-highlights.md`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`

## Review Notes
- Approved to improve CI summary ergonomics without changing governance checks.

## Implementation Summary
- Added `scripts/append-baseline-scope-smoke-summary.sh` to append compact
  baseline scope smoke highlights to a target CI summary file.
- Added `scripts/test-append-baseline-scope-smoke-summary.sh` to cover both
  pass and fail report rendering paths.
- Updated `.github/workflows/ci.yml` to append smoke highlights before the full
  smoke markdown artifact.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase72-smoke-job-summary-highlights.md`
- `bash -n scripts/append-baseline-scope-smoke-summary.sh`
- `bash -n scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- inspect CI wiring in:
  - `.github/workflows/ci.yml`
