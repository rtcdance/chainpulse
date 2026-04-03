# Phase 26 Policy v1 Deprecation Deadline Gate

## Title
Phase 26 - Add automated v1 metric deprecation deadline checks and CI blocking policy

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
- `scripts/check-policy-metric-contract.sh`
- `.github/workflows/ci.yml`
- `docs/operations/POLICY_METRIC_VERSIONING.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 25 introduced schema mode versioning and dual-write migration, but lacked an automated deadline gate to force deprecation completion.

## Problem Statement
Without a hard cutoff guard, legacy `v1` schema can persist indefinitely and delay observability migration.

## Scope
- Extend contract script with optional cutoff validation:
  - `CHAINPULSE_POLICY_V1_DEPRECATION_DATE` (`YYYY-MM-DD`)
  - `CHAINPULSE_POLICY_V1_DEPRECATION_WARN_DAYS` (default `14`)
- Before cutoff:
  - warning for non-`v2` mode when entering warning window
- On/after cutoff:
  - fail when `CHAINPULSE_POLICY_METRIC_SCHEMA_MODE != v2`
- Configure CI policy-contract job with rollout values.
- Document deadline gate behavior in operations versioning guide.

## Non-Goals
- No runtime deprecation auto-switch in service startup.
- No changes to metric payload content.

## Options Considered
- Option A: manual migration tracking in runbook.
- Option B: script-enforced cutoff in CI gate.

## Selected Approach
Choose Option B for deterministic enforcement.

## Data / Contract Impact
No runtime telemetry schema changes; CI gate behavior is strengthened.

## Risks
- Misconfigured cutoff date can block CI unexpectedly.
- Mitigation: explicit date format validation and warning window support.

## Rollback Plan
Unset `CHAINPULSE_POLICY_V1_DEPRECATION_DATE` in CI env to disable blocking.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase26-policy-v1-deprecation-deadline-gate.md`
- `./scripts/check-policy-metric-contract.sh`
- verify CI workflow env wiring in `.github/workflows/ci.yml`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase26-policy-v1-deprecation-deadline-gate.md`
- `./scripts/check-policy-metric-contract.sh`

## Review Notes
- Approved to guarantee v1 deprecation completion by policy date.

## Implementation Summary
- Added cutoff-aware checks in policy contract script and wired CI env policy.

## Final Verification
- Spec gate and local contract script pass; CI now contains deadline policy controls.
