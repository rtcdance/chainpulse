# Phase 19 Profile Allowlist Policy Matrix

## Title
Phase 19 - Add optional non-production profile allowlist for overrides and profile-matrix policy tests

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
- `pkg/application/bootstrap/core_config_override_policy.go`
- `pkg/application/bootstrap/core_config_override_policy_test.go`
- `cmd/microservices/api-service/main.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 18 introduced production denylist enforcement and override audit tags, but non-production profile gating remained open-ended.

## Problem Statement
Enterprise operations often require explicit control over which non-production profiles can apply runtime overrides (for example, `staging` or `canary`) while keeping production denylist protection.

## Scope
- Add optional profile allowlist ingestion from environment:
  - `CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES`
- Enforce allowlist only when configured and only for non-production profiles.
- Keep production denylist behavior unchanged.
- Extend audit tags with allowlist state indicators.
- Add table-driven profile matrix tests for:
  - production denylist paths
  - allowlist enabled/disabled paths
  - staging/canary allowlisted paths
  - non-allowlisted profile rejection when overrides are present

## Non-Goals
- No remote policy management service.
- No per-key policy matrix by profile in this phase.

## Options Considered
- Option A: keep non-production fully open.
- Option B: add optional allowlist controls with default-open behavior when unset.

## Selected Approach
Choose Option B to preserve backward compatibility while enabling stricter enterprise policy where needed.

## Data / Contract Impact
No external API contract changes. Adds one optional env variable for startup policy behavior.

## Risks
- Misconfigured allowlist may reject startup when overrides are present.
- Mitigation: clear error messages and explicit audit tags.

## Rollback Plan
Unset `CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES` to disable allowlist enforcement and retain production denylist only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase19-profile-allowlist-policy-matrix.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `go test -short ./pkg/application/bootstrap/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase19-profile-allowlist-policy-matrix.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to improve enterprise runtime governance with low migration risk.

## Implementation Summary
- Added optional allowlist ingestion, allowlist-aware validation, expanded audit tags, and profile matrix tests.

## Final Verification
- Fast gate and bootstrap package tests pass with allowlist policy coverage.
