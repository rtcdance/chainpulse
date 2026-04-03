# Phase 74 Resolver Job Summary Highlights

## Title
Phase 74 - Add baseline resolver test highlights to CI job summary

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
- `scripts/append-baseline-resolver-test-summary.sh`
- `scripts/test-append-baseline-resolver-test-summary.sh`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase74-resolver-job-summary-highlights.md`

## Context
Smoke highlights now surface compact smoke and drift signals in CI, but resolver
governance outputs are still appended only as full markdown artifacts.

## Problem Statement
Without compact resolver highlights, operators must scan the full resolver test
and delta artifacts to determine resolver health and regression status.

## Scope
- Add a resolver CI summary helper that reads:
  - `baseline-resolver-test.md`
  - `baseline-resolver-test-delta.md` when present
- Include top-level resolver status and total/pass/fail counters.
- Include resolver delta regression status and regression-signal preview.
- Wire the helper into the `policy-contract` CI job before the full resolver
  markdown artifacts are appended.

## Non-Goals
- No changes to resolver test/delta execution semantics or artifact contracts.
- No changes to baseline refresh or governance mutation flows.

## Options Considered
- Option A: keep appending only full resolver markdown artifacts.
- Option B: prepend compact resolver highlights and keep full artifacts.

## Selected Approach
Choose Option B to align resolver operator ergonomics with the smoke summary
pattern while preserving detailed audit artifacts.

## Data / Contract Impact
- CI job summary gains a compact resolver highlight section.
- Existing resolver test and delta artifacts remain unchanged.

## Risks
- Minimal; summary rendering only.

## Rollback Plan
- Remove resolver helper invocation from CI and delete the helper scripts.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase74-resolver-job-summary-highlights.md`
- `bash -n scripts/append-baseline-resolver-test-summary.sh`
- `bash -n scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`
- inspect helper output against generated resolver/delta markdown.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase74-resolver-job-summary-highlights.md`
- `./scripts/test-append-baseline-resolver-test-summary.sh`

## Review Notes
- Approved to align resolver summary visibility with smoke CI summary ergonomics.

## Implementation Summary
- Added `scripts/append-baseline-resolver-test-summary.sh` to append compact
  resolver highlights to a target CI summary file.
- Added `scripts/test-append-baseline-resolver-test-summary.sh` to validate
  resolver summary rendering in pass/fail delta-aware scenarios.
- Updated `.github/workflows/ci.yml` to prepend resolver highlights before the
  full resolver test markdown artifact.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase74-resolver-job-summary-highlights.md`
- `bash -n scripts/append-baseline-resolver-test-summary.sh`
- `bash -n scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`
- inspect helper output against generated resolver/delta markdown.
