# Phase 28 Migration Manifest Spec Sync Gate

## Title
Phase 28 - Add automated migration-manifest to specs sync validation and CI gate

## Type
- architecture

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `docs/operations/MIGRATION_MANIFEST.csv`
- `scripts/check-migration-manifest.sh`
- `scripts/spec-approval-check.sh`
- `docs/operations/MIGRATION_MANIFEST.md`
- `docs/specs/2026-03-30-observability-indexer-health-sli-harmonization-plan.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 27 introduced repo-level migration manifest governance, but did not yet enforce explicit spec linkage for each active migration item.

## Problem Statement
Migration items without approved specs create execution ambiguity and weaken architecture governance guarantees.

## Scope
- Extend migration manifest schema with `spec_ref`.
- Require active migration items (`planned`, `in_progress`, `blocked`) to:
  - reference an existing spec file
  - pass `spec-approval-check`
- Keep overdue deadline checks from Phase 27.
- Add one missing observability planning spec to satisfy manifest coverage.
- Update migration governance docs with `spec_ref` contract.

## Non-Goals
- No runtime service behavior changes.
- No automatic generation of spec files from manifest.

## Options Considered
- Option A: keep spec linkage as manual convention.
- Option B: enforce spec linkage and approval in validation script.

## Selected Approach
Choose Option B for deterministic architecture traceability.

## Data / Contract Impact
Manifest CSV schema changed from 7 to 8 fields (`spec_ref` added).

## Risks
- Legacy rows without `spec_ref` will fail CI.
- Mitigation: clear error messages and one-time manifest migration.

## Rollback Plan
Set `CHAINPULSE_MIGRATION_REQUIRE_SPEC_SYNC=false` to temporarily bypass spec linkage enforcement.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase28-migration-manifest-spec-sync-gate.md`
- `./scripts/check-migration-manifest.sh`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase28-migration-manifest-spec-sync-gate.md`
- `./scripts/check-migration-manifest.sh`

## Review Notes
- Approved to harden migration-to-spec governance and ownership traceability.

## Implementation Summary
- Added `spec_ref` contract, automated spec approval checks, and manifest/schema docs update.

## Final Verification
- Spec gate and migration manifest check pass with spec sync enabled.
