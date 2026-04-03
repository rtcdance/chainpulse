# Phase 36 Ticket Registry Adapter and Failure Degrade Mode

## Title
Phase 36 - Add ticket registry adapter (file/http) and verification failure degrade mode

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
- `docs/operations/MIGRATION_TICKET_REGISTRY.txt`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 35 introduced ticket pattern and owner whitelist checks, but ticket verification lacked registry-based validation and failure degrade controls.

## Problem Statement
Pattern-only checks cannot ensure ticket IDs are actually approved/registered; strict external checks also need a controlled fallback mode for operational resilience.

## Scope
- Add ticket verification mode:
  - `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern|registry|both`
- Add ticket registry adapter source:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE=file|http`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_FILE`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_URL`
- Add verification failure behavior:
  - `CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE=enforce|warn`
- Add local registry file for default CI policy:
  - `docs/operations/MIGRATION_TICKET_REGISTRY.txt`
- Wire CI to `both + file + enforce` defaults.

## Non-Goals
- No direct integration with external issue tracker APIs beyond registry endpoint ingestion.
- No runtime service behavior changes.

## Options Considered
- Option A: keep pattern-only checks.
- Option B: add registry-aware verification with configurable strictness.

## Selected Approach
Choose Option B for stronger governance with operationally safe fallback.

## Data / Contract Impact
No runtime contract changes; governance env policy contract expanded.

## Risks
- Misconfigured registry source may block CI in enforce mode.
- Mitigation: explicit `warn` degrade mode and clear diagnostics.

## Rollback Plan
Set `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=pattern` to bypass registry checks.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase36-ticket-registry-adapter-and-failure-degrade-mode.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase36-ticket-registry-adapter-and-failure-degrade-mode.md`
- `./scripts/check-migration-changelog-quality.sh`

## Review Notes
- Approved to support stronger ticket governance with configurable resilience behavior.

## Implementation Summary
- Added registry-aware ticket verification and warn/enforce failure mode.

## Final Verification
- Changelog quality checks pass under CI `both+file+enforce` configuration.
