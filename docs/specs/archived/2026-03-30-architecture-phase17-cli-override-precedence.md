# Phase 17 CLI Override Precedence

## Title
Phase 17 - Add CLI override ingestion with explicit precedence `CLI > env > mode defaults`

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
- `pkg/application/bootstrap/core_config_overrides_cli.go`
- `pkg/application/bootstrap/core_config_overrides_cli_test.go`
- `cmd/microservices/api-service/main.go`
- `cmd/monolithic/chainpulse/main.go`
- `docs/ARCHITECTURE.md`

## Context
Phase 16 introduced environment-based override ingestion, but command-line override ingestion and precedence policy were not yet implemented.

## Problem Statement
Operators need deterministic override precedence across deployment modes; without CLI support, emergency runtime overrides are limited.

## Scope
- Add CLI parsing for:
  - `--core-api-type`
  - `--core-api-port`
  - `--core-feature-flags`
- Add validation for parsed CLI values.
- Add merge helper with explicit precedence (`CLI > env`).
- Wire merged overrides into both startup entrypoints.
- Add tests for CLI parsing and precedence merge behavior.

## Non-Goals
- No generic command framework migration.
- No positional-argument behavior changes.

## Options Considered
- Option A: keep env-only overrides.
- Option B: add dedicated CLI parser and precedence merge.

## Selected Approach
Choose Option B with lightweight parser that ignores unrelated args and only consumes recognized override flags.

## Data / Contract Impact
No external API contract changes.

## Risks
- Misparsed CLI values can block startup.
- Mitigation: strict validation and explicit error messages.

## Rollback Plan
Remove CLI parser integration and revert to env-only override ingestion.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase17-cli-override-precedence.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase17-cli-override-precedence.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved for operational flexibility with deterministic precedence.

## Implementation Summary
- Added CLI parser + merge helper and integrated into both startup paths.

## Final Verification
- Fast gate and bootstrap tests pass with CLI precedence coverage.
