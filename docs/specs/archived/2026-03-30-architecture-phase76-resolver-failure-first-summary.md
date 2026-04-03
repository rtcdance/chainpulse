# Phase 76 Resolver Failure-First Summary

## Title
Phase 76 - Add failure-first rendering to resolver test summaries

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
- `scripts/test-baseline-update-resolver.sh`
- `scripts/lib/governance_summary.sh`
- `scripts/append-baseline-resolver-test-summary.sh`
- `scripts/test-append-baseline-resolver-test-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase76-resolver-failure-first-summary.md`

## Context
Smoke summaries already surface failure-first markdown sections, while resolver
summaries only expose failed counts and full case tables.

## Problem Statement
Without failure-first resolver rendering, operators must scan the full resolver
case table to find failed cases during CI triage.

## Scope
- Add conditional `Failure Summary` section to resolver markdown artifact.
- Update resolver CI highlights to reuse that failure summary when failures
  exist.
- Preserve existing resolver artifact contracts and helper entrypoints.

## Non-Goals
- No changes to resolver test semantics.
- No changes to resolver delta logic or CI workflow structure.

## Options Considered
- Option A: keep resolver counts-only summary.
- Option B: add failure-first resolver report and summary rendering.

## Selected Approach
Choose Option B to align resolver troubleshooting ergonomics with smoke
highlights while preserving current contracts.

## Data / Contract Impact
- Resolver markdown artifact gains conditional `Failure Summary` section.
- CI resolver highlight section may now include failed-case preview when
  failures exist.
- JSON/Prom outputs remain unchanged.

## Risks
- Low; reporting-only changes.

## Rollback Plan
- Remove resolver failure-summary block from markdown artifact and highlight
  rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase76-resolver-failure-first-summary.md`
- `bash -n scripts/lib/governance_summary.sh`
- `bash -n scripts/test-baseline-update-resolver.sh`
- `bash -n scripts/append-baseline-resolver-test-summary.sh`
- `bash -n scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase76-resolver-failure-first-summary.md`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-baseline-update-resolver.sh`

## Review Notes
- Approved to align resolver summary ergonomics with the smoke failure-first
  pattern.

## Implementation Summary
- Added conditional `Failure Summary` rendering to
  `build/migration-governance/baseline-resolver-test.md`.
- Extended `scripts/lib/governance_summary.sh` with reusable failure-section
  rendering.
- Updated resolver CI highlights and their regression tests to surface failed
  resolver cases before delta details.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase76-resolver-failure-first-summary.md`
- `bash -n scripts/lib/governance_summary.sh`
- `bash -n scripts/test-baseline-update-resolver.sh`
- `bash -n scripts/append-baseline-resolver-test-summary.sh`
- `bash -n scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-resolver-test.sh`
