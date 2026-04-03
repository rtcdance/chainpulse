# Phase 73 Smoke Delta Job Summary Highlights

## Title
Phase 73 - Add baseline scope smoke delta highlights to CI job summary

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
- `docs/specs/2026-03-30-architecture-phase73-smoke-delta-job-summary-highlights.md`

## Context
Phase 72 surfaced smoke status highlights in the CI summary, but baseline drift
signals still require separately scanning the appended delta markdown.

## Problem Statement
Without delta-aware smoke highlights, operators cannot quickly correlate smoke
failures and baseline drift signals from one compact CI summary section.

## Scope
- Extend smoke CI highlight helper to optionally read
  `baseline-scope-smoke-delta.md`.
- Include delta status and key regression signal rows in the compact summary.
- Keep full smoke and delta markdown artifacts appended after the highlights.

## Non-Goals
- No changes to smoke/delta execution semantics or artifact contracts.
- No changes to baseline refresh logic.

## Options Considered
- Option A: keep smoke and delta summaries separate in the job summary.
- Option B: merge top smoke and delta signals into one compact highlight block.

## Selected Approach
Choose Option B for faster CI triage while preserving existing detailed
artifacts and append order.

## Data / Contract Impact
- CI job summary highlight section gains delta status and regression-signal
  preview.
- Existing smoke and delta artifacts remain unchanged.

## Risks
- Minimal; summary rendering only.

## Rollback Plan
- Remove delta parsing logic from the smoke summary helper.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase73-smoke-delta-job-summary-highlights.md`
- `bash -n scripts/append-baseline-scope-smoke-summary.sh`
- `bash -n scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- inspect helper output against generated smoke/delta markdown.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase73-smoke-delta-job-summary-highlights.md`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`

## Review Notes
- Approved to co-locate smoke status and drift signals in CI summary.

## Implementation Summary
- Extended `scripts/append-baseline-scope-smoke-summary.sh` to consume
  `baseline-scope-smoke-delta.md` when present.
- Added delta regression status and regression-signal preview rendering to the
  compact smoke CI summary block.
- Extended `scripts/test-append-baseline-scope-smoke-summary.sh` to validate
  delta-aware rendering in both pass and fail scenarios.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase73-smoke-delta-job-summary-highlights.md`
- `bash -n scripts/append-baseline-scope-smoke-summary.sh`
- `bash -n scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- inspect helper output against generated smoke/delta markdown.
