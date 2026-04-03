# Phase 15 Deployment Mode Overrides

## Title
Phase 15 - Add deployment-mode override policy for shared bootstrap core config

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
- `pkg/application/bootstrap/core_config.go`
- `pkg/application/bootstrap/core_config_test.go`
- `cmd/microservices/api-service/main.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/ARCHITECTURE.md`

## Context
After Phase 13 shared startup constructor, deployment-mode-specific core config values were still assembled separately in command entrypoints.

## Problem Statement
Without a unified override policy, configuration drift can reappear between monolithic and microservice modes.

## Scope
- Add shared core config builders for API service and monolithic modes.
- Add explicit override struct for `APIType`, `APIPort`, and `FeatureFlags`.
- Apply override helper in both command entrypoints.
- Add table-driven tests for override precedence and feature flag merge behavior.

## Non-Goals
- No environment variable schema changes.
- No runtime behavior change outside config assembly.

## Options Considered
- Option A: keep per-command config assembly.
- Option B: shared builders + override policy with tests.

## Selected Approach
Choose Option B for consistency and testability.

## Data / Contract Impact
No external API contract changes.

## Risks
- Incorrect merge precedence for feature flags.
- Mitigation: table-driven tests for edge cases.

## Rollback Plan
Revert command entrypoints to local config assembly and remove shared override helper.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase15-deployment-mode-overrides.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase15-deployment-mode-overrides.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved to keep deployment-mode configuration deterministic under shared bootstrap.

## Implementation Summary
- Added shared mode config builders and override policy with tests.

## Final Verification
- Fast gate passes and mode config assembly uses shared helper in both entrypoints.
