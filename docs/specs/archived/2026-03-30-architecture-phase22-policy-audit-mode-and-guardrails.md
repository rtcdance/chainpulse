# Phase 22 Policy Audit Mode and Rollout Guardrails

## Title
Phase 22 - Add policy audit-only mode and rollout guardrail telemetry

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
Phase 21 introduced key-level policy bundles and structured error codes, but policy rollout remained hard-enforce only.

## Problem Statement
Enterprise rollout often requires progressive hardening where teams first observe violations (audit-only) before enforcing startup blocks.

## Scope
- Add policy enforcement mode env:
  - `CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT`
  - values: `enforce` (default), `audit`
- Add unified policy evaluation output:
  - enforcement mode
  - violation code
  - violation and block decision flags
- In `audit` mode:
  - do not block startup on policy violations
  - emit warning logs and policy evaluation telemetry
- Add policy evaluation metric in both startup paths:
  - `core_config_overrides_policy_evaluation_total`
- Extend override audit tags with enforcement dimensions.
- Add tests for enforce/audit behavior and enforcement mode parsing.

## Non-Goals
- No dynamic runtime mode flips after startup.
- No remote policy rollout controller.

## Options Considered
- Option A: keep enforce-only startup.
- Option B: add audit-only rollout guardrail mode.

## Selected Approach
Choose Option B for safer enterprise migrations and policy hardening.

## Data / Contract Impact
No external API contract changes. Adds one optional env variable for startup behavior.

## Risks
- `audit` mode can permit risky startup configs if left enabled unintentionally.
- Mitigation: explicit telemetry, warning logs, and default `enforce` fallback.

## Rollback Plan
Set `CHAINPULSE_OVERRIDE_POLICY_ENFORCEMENT=enforce` (or unset) to restore strict blocking behavior.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase22-policy-audit-mode-and-guardrails.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `go test -short ./pkg/application/bootstrap/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase22-policy-audit-mode-and-guardrails.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to support phased policy rollout without service disruption.

## Implementation Summary
- Added audit-only policy evaluation mode, startup warning path, and rollout telemetry metric.

## Final Verification
- Fast gate and bootstrap package tests pass with enforce/audit mode coverage.
