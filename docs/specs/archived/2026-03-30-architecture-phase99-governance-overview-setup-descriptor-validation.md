# Phase 99 Governance Overview Setup Descriptor Validation

## Title
Phase 99 - Add validation for overview setup descriptors

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
- `docs/specs/2026-03-30-architecture-phase99-governance-overview-setup-descriptor-validation.md`

## Context
Phase 98 converted overview setup helpers to compact data blocks, but malformed
descriptors could still be written without an explicit failure mode.

## Problem Statement
Without validation, unknown descriptor kinds or missing fields can create silent
test drift or partial fixture setup that is harder to diagnose.

## Scope
- Validate setup descriptor kind values.
- Validate required field counts per descriptor kind.
- Fail fast with stable error messages when a setup descriptor is invalid.

## Non-Goals
- No changes to overview rendering logic.
- No expansion of scenario coverage beyond current overview states.

## Options Considered
- Option A: trust descriptor authors and keep setup parsing permissive.
- Option B: add small validation guards around descriptor parsing.

## Selected Approach
Choose Option B so the data-driven harness remains safe as it grows.

## Data / Contract Impact
- No output contract changes.
- Test harness now fails explicitly on malformed setup descriptors.

## Risks
- Minimal; descriptor validation may require future extension as new kinds are added.

## Rollback Plan
- Remove validation guards and revert to permissive descriptor parsing.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase99-governance-overview-setup-descriptor-validation.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase99-governance-overview-setup-descriptor-validation.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to strengthen the data-driven overview test harness.

## Implementation Summary
- Added validation helpers for setup descriptor kind and field counts.
- Preserved existing scenario behavior while making malformed descriptors fail fast.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase99-governance-overview-setup-descriptor-validation.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
