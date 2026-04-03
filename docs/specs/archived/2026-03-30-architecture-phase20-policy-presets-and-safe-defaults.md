# Phase 20 Policy Presets and Safe Defaults

## Title
Phase 20 - Add layered override policy presets (strict/balanced/open) with startup-safe environment-tier defaults

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
Phase 19 introduced optional profile allowlist gating for non-production overrides, but policy operation still required explicit allowlist management.

## Problem Statement
Enterprise deployments need predictable policy behavior by environment tier without requiring manual allowlist edits for every service startup.

## Scope
- Add policy preset env var:
  - `CHAINPULSE_OVERRIDE_POLICY_PRESET`
  - allowed values: `strict`, `balanced`, `open`
- Add startup-safe preset defaults when env var is unset/invalid:
  - production => `strict`
  - staging/canary/preprod/qa/test => `balanced`
  - development and other local profiles => `open`
- Keep explicit allowlist env override with highest precedence:
  - `CHAINPULSE_OVERRIDE_POLICY_ALLOW_PROFILES`
- Add policy resolution helper returning:
  - effective preset
  - source (`default`, `env_preset`, `env_allow_profiles`)
  - resolved allowlist
- Extend override audit tags with policy preset/source dimensions.
- Add table-driven tests for preset parsing, defaulting, and resolution precedence.

## Non-Goals
- No remote policy control-plane.
- No per-feature-flag profile matrix beyond current production denylist.

## Options Considered
- Option A: require explicit allowlist everywhere.
- Option B: add layered presets + safe defaults + explicit override precedence.

## Selected Approach
Choose Option B for enterprise operability and safer defaults with backward-compatible explicit controls.

## Data / Contract Impact
No external API contract changes. Adds one optional env variable for policy preset behavior.

## Risks
- Unexpected behavior if operators assume invalid preset values are accepted.
- Mitigation: deterministic fallback to profile-safe defaults and auditable policy tags.

## Rollback Plan
Unset `CHAINPULSE_OVERRIDE_POLICY_PRESET` and rely on explicit allowlist only, or keep default-open development behavior.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase20-policy-presets-and-safe-defaults.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `go test -short ./pkg/application/bootstrap/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase20-policy-presets-and-safe-defaults.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to improve enterprise policy ergonomics and startup safety across environment tiers.

## Implementation Summary
- Added preset resolver, environment-tier defaults, explicit allowlist precedence, and policy audit tag dimensions.

## Final Verification
- Fast gate and bootstrap package tests pass with preset/default/precedence coverage.
