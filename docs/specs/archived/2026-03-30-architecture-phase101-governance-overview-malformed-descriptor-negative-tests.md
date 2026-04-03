# Phase 101 Governance Overview Malformed Descriptor Negative Tests

## Title
Phase 101 - Add negative tests for malformed overview descriptors

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
- `docs/specs/2026-03-30-architecture-phase101-governance-overview-malformed-descriptor-negative-tests.md`

## Context
Phase 100 added fail-fast validation for setup, aggregate, and row descriptors,
but those guards are not yet covered by explicit negative-path tests.

## Problem Statement
Without negative regression coverage, malformed-descriptor protections could
accidentally regress while the happy-path suite continues to pass.

## Scope
- Add negative tests for malformed setup descriptors.
- Add negative tests for malformed aggregate descriptors.
- Add negative tests for malformed row descriptors.
- Assert stable failure messages for each malformed input class.

## Non-Goals
- No changes to overview rendering logic.
- No expansion of overview happy-path behavior.

## Options Considered
- Option A: rely on manual inspection of descriptor validation helpers.
- Option B: add explicit negative-path regression coverage.

## Selected Approach
Choose Option B so the new fail-fast protections become part of the maintained
contract of the overview test harness.

## Data / Contract Impact
- No output contract changes.
- Test harness gains malformed-input regression coverage.

## Risks
- Minimal; failure-message wording becomes mildly more contractual.

## Rollback Plan
- Remove negative tests if descriptor validation is intentionally relaxed later.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase101-governance-overview-malformed-descriptor-negative-tests.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase101-governance-overview-malformed-descriptor-negative-tests.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to close the loop on descriptor validation hardening.

## Implementation Summary
- Added negative-path helpers and checks for malformed setup, aggregate, and row descriptors.
- Preserved happy-path coverage while verifying the new validation guards.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase101-governance-overview-malformed-descriptor-negative-tests.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
