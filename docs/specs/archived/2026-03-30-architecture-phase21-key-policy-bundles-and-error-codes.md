# Phase 21 Key Policy Bundles and Error Codes

## Title
Phase 21 - Introduce per-key override policy bundles and structured validation error codes

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
Phase 20 introduced preset-based policy resolution, but override validation still used coarse denylist checks and non-typed errors.

## Problem Statement
Enterprise operations require deterministic policy behavior per override key (`API type`, `API port`, `feature flags`) and machine-readable error codes for incident triage/automation.

## Scope
- Add key-level policy bundle model:
  - allowed API types
  - API port range
  - denied feature flags when set to `true`
- Resolve key policy bundle by environment tier + preset.
- Keep existing allowlist gate (`CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES`) as profile admission control.
- Add structured policy error codes for validation failures.
- Extend startup metric tags with key-policy dimensions.
- Add table-driven tests for:
  - production policy enforcement
  - strict/balanced/open preset behavior
  - allowlist gate rejection code
  - per-key failure code assertions

## Non-Goals
- No remote policy engine.
- No externalized policy registry.

## Options Considered
- Option A: keep generic string errors.
- Option B: add typed policy errors and per-key bundles.

## Selected Approach
Choose Option B for operational clarity, safer automation, and enterprise-grade diagnosability.

## Data / Contract Impact
No external API contract changes. Internal validation now returns typed policy errors with stable codes.

## Risks
- Rule tightening can reject previously tolerated configurations.
- Mitigation: profile-aware presets (`strict/balanced/open`) and explicit allowlist override.

## Rollback Plan
Revert to prior coarse validation path by bypassing key bundle checks in startup bootstrap.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase21-key-policy-bundles-and-error-codes.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `go test -short ./pkg/application/bootstrap/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase21-key-policy-bundles-and-error-codes.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to strengthen policy governance and improve operational debugging.

## Implementation Summary
- Added key-policy bundle resolution, typed policy errors/codes, and expanded tests/telemetry tags.

## Final Verification
- Fast gate and bootstrap tests pass with error-code and key-policy matrix coverage.
