# Phase 29 Migration Manifest Spec Metadata Sync

## Title
Phase 29 - Add migration manifest and spec metadata consistency checks (owner/status/deadline)

## Type
- architecture

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
- `scripts/check-migration-manifest.sh`
- `docs/operations/MIGRATION_MANIFEST.csv`
- `docs/operations/MIGRATION_MANIFEST.md`
- `.github/workflows/ci.yml`
- `docs/ARCHITECTURE.md`

## Context
Phase 28 enforced `spec_ref` existence and approval for active migration items, but did not validate metadata consistency between manifest and spec content.

## Problem Statement
Even with linked specs, owner/status/deadline drift between manifest and spec can cause governance ambiguity and stale accountability.

## Scope
- Extend migration manifest validation with metadata sync checks for active items:
  - manifest `owner` appears in spec `## Owner`
  - manifest `status` maps to expected spec `## Delivery Status`
  - manifest `deadline` date exists in spec content
- Add env toggle:
  - `CHAINPULSE_MIGRATION_REQUIRE_SPEC_METADATA_SYNC=true|false` (default `true`)
- Update manifest-related specs to include aligned owner/deadline metadata.
- Wire CI env to explicitly enforce metadata sync.

## Non-Goals
- No runtime service behavior changes.
- No auto-editing of mismatched specs.

## Options Considered
- Option A: check only `spec_ref` existence/approval.
- Option B: enforce metadata consistency as part of manifest gate.

## Selected Approach
Choose Option B to guarantee traceable ownership and schedule integrity.

## Data / Contract Impact
No runtime API contract changes. Governance validation is stricter.

## Risks
- Existing specs may require one-time metadata backfill.
- Mitigation: clear error diagnostics and opt-out env toggle for emergency bypass.

## Rollback Plan
Set `CHAINPULSE_MIGRATION_REQUIRE_SPEC_METADATA_SYNC=false` to disable metadata sync temporarily.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase29-migration-manifest-spec-metadata-sync.md`
- `./scripts/check-migration-manifest.sh`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase29-migration-manifest-spec-metadata-sync.md`
- `./scripts/check-migration-manifest.sh`

## Review Notes
- Approved to harden manifest governance consistency.

## Implementation Summary
- Added metadata sync checks and aligned referenced specs for owner/deadline consistency.

## Final Verification
- Spec gate and migration manifest check pass with metadata sync enforcement enabled.
