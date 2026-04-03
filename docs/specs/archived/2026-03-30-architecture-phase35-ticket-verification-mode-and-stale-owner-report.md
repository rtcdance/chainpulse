# Phase 35 Ticket Verification Mode and Stale Owner Report

## Title
Phase 35 - Add configurable ticket verification mode and stale-owner governance reporting

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
- `scripts/check-migration-changelog-quality.sh`
- `scripts/check-migration-manifest.sh`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_MANIFEST.md`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 34 added ticket-pattern and owner-allowlist checks, but governance policy behavior still needed explicit configurability and reporting alignment in CI.

## Problem Statement
Without explicit CI policy variables and harmonized checks across manifest/changelog, teams can experience inconsistent governance enforcement.

## Scope
- Add explicit ticket-pattern and owner-allowlist policy variables in CI:
  - `CHAINPULSE_MIGRATION_TICKET_PATTERN`
  - `CHAINPULSE_MIGRATION_OWNER_ALLOWLIST`
- Enforce owner allowlist in both:
  - migration manifest check
  - baseline changelog quality check
- Keep changelog rationale quality constraints.
- Update governance docs and architecture roadmap to reflect policy-as-code controls.

## Non-Goals
- No external ticket API verification.
- No runtime service behavior changes.

## Options Considered
- Option A: static checks without configurable policy.
- Option B: policy variables + unified enforcement.

## Selected Approach
Choose Option B for enterprise policy flexibility with deterministic CI behavior.

## Data / Contract Impact
No runtime contract changes; governance policy env contract is expanded and formalized.

## Risks
- Misconfigured regex/allowlist can over-block valid updates.
- Mitigation: explicit defaults, clear failure diagnostics, and CI-managed values.

## Rollback Plan
Relax env policy values in CI while retaining structural checks.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase35-ticket-verification-mode-and-stale-owner-report.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-manifest.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase35-ticket-verification-mode-and-stale-owner-report.md`
- `./scripts/check-migration-changelog-quality.sh`

## Review Notes
- Approved to stabilize policy-as-code governance behavior.

## Implementation Summary
- Unified ticket/owner policy checks with explicit CI configuration and governance docs update.

## Final Verification
- Governance checks pass under configured CI ticket/owner policy values.
