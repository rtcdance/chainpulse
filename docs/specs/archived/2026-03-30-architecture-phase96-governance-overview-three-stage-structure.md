# Phase 96 Governance Overview Three-Stage Structure

## Title
Phase 96 - Restructure overview regression script into setup/run/assert stages

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
- `scripts/test-append-governance-overview-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase96-governance-overview-three-stage-structure.md`

## Context
Phase 95 made the overview test table-driven, but the script body still mixes
fixture mutation, execution, and assertion blocks inline across scenarios.

## Problem Statement
Even with scenario helpers, the main flow is harder to scan when setup, run, and
assert steps are interleaved for each aggregate state.

## Scope
- Group scenario logic into explicit setup/run/assert helper functions.
- Preserve existing fixture semantics, aggregate coverage, and row coverage.
- Keep overview rendering logic unchanged.

## Non-Goals
- No changes to overview output contracts.
- No new scenario coverage beyond the existing aggregate matrix.

## Options Considered
- Option A: keep inline scenario sequencing.
- Option B: restructure into setup/run/assert stages.

## Selected Approach
Choose Option B so the regression script reads more like a small test harness
and less like an execution transcript.

## Data / Contract Impact
- No output contract changes.
- Test structure only.

## Risks
- Minimal; helper indirection must remain small enough to stay readable.

## Rollback Plan
- Revert to the previous inline scenario sequencing.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase96-governance-overview-three-stage-structure.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase96-governance-overview-three-stage-structure.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve readability of the overview regression harness.

## Implementation Summary
- Added setup/run/assert helpers for primary, warn, info, and ok overview states.
- Kept existing checks intact while making scenario flow easier to scan.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase96-governance-overview-three-stage-structure.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
