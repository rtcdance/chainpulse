# Phase 16 Env Override Ingestion

## Title
Phase 16 - Map environment overrides into shared `CoreConfigOverrides` with validation and audit summary

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
- `pkg/application/bootstrap/core_config_overrides_env.go`
- `pkg/application/bootstrap/core_config_overrides_env_test.go`
- `cmd/microservices/api-service/main.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 15 introduced shared mode config builders and override policy, but there was no standardized ingestion path from environment/CLI sources into `CoreConfigOverrides`.

## Problem Statement
Without explicit ingestion and validation, operators cannot safely apply runtime mode overrides and teams cannot audit which overrides were activated.

## Scope
- Add environment-driven override parsing to `CoreConfigOverrides`.
- Validate values (`APIType`, `APIPort`, feature-flag format and bool values).
- Provide audit summary string for startup logs.
- Integrate in both api-service and monolithic entrypoints.

## Non-Goals
- No CLI flag parser in this phase.
- No change to existing base env keys.

## Options Considered
- Option A: Parse overrides separately in each command.
- Option B: Shared parsing utility in bootstrap package.

## Selected Approach
Choose Option B to keep override behavior deterministic across deployment modes.

## Data / Contract Impact
No external API contract change.

## Risks
- Misconfigured env values can fail startup.
- Mitigation: strict validation + explicit error messages.

## Rollback Plan
Remove shared env ingestion and revert entrypoints to empty override application.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase16-env-override-ingestion.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase16-env-override-ingestion.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved for operator safety and deployment observability.

## Implementation Summary
- Added shared env override parser/validator and wired audit logs in both entrypoints.

## Final Verification
- Fast gate and bootstrap package tests pass with env override parsing coverage.
