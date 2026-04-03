# Phase 27 Repo Migration Manifest and CI Overdue Gate

## Title
Phase 27 - Add repo-level migration manifest with owner/deadline/status and CI overdue validation

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
- `docs/operations/MIGRATION_MANIFEST.md`
- `scripts/check-migration-manifest.sh`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/ARCHITECTURE.md`

## Context
Phase 26 added policy-metric-specific cutoff enforcement, but broader observability migration tracking still lacked a repository-wide governance artifact.

## Problem Statement
Without a shared migration manifest and automated overdue checks, ownership and deadlines for observability migrations can drift and miss enterprise commitments.

## Scope
- Add repo-level migration manifest CSV with required fields:
  - `id,domain,owner,status,deadline,severity,notes`
- Add manifest governance doc describing schema and process.
- Add manifest validation script:
  - format/status/date validation
  - overdue fail gate
  - warning-window notifications
- Integrate manifest check into:
  - CI `policy-contract` job
  - local full micro-loop
  - Makefile target
- Add docs index and operations guide links.

## Non-Goals
- No external issue tracker synchronization.
- No runtime service behavior changes.

## Options Considered
- Option A: maintain migrations in ad-hoc docs or tickets.
- Option B: enforce a machine-checkable in-repo manifest with CI gating.

## Selected Approach
Choose Option B for deterministic governance and onboarding clarity.

## Data / Contract Impact
No runtime API contract changes.

## Risks
- CSV manual editing errors may trigger CI failures.
- Mitigation: strict parser with explicit diagnostics and docs.

## Rollback Plan
Disable `check-migration-manifest.sh` in CI and keep manifest as informational only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase27-repo-migration-manifest-and-ci-overdue-gate.md`
- `./scripts/check-migration-manifest.sh`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase27-repo-migration-manifest-and-ci-overdue-gate.md`
- `./scripts/check-migration-manifest.sh`

## Review Notes
- Approved to extend migration governance beyond policy metrics.

## Implementation Summary
- Added manifest artifacts and CI/local overdue checks for repository-wide migration tracking.

## Final Verification
- Spec gate and manifest check pass; CI and local full loop include overdue validation.
