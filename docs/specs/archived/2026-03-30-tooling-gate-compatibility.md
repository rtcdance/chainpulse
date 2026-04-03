# Tooling Gate Compatibility

## Title
Align local quality gate tooling with repository lint configuration

## Type
- architecture

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `Makefile`
- `scripts/dev-micro-loop.sh`

## Context
After installing Go 1.24 and running fast gate, checks failed because:
- `gofumpt` binary path was not guaranteed in current shell path.
- `golangci-lint` installed as v2, incompatible with existing `.golangci.yml`.

## Problem Statement
Quality gates cannot run reliably without toolchain path and linter compatibility fixes.

## Scope
- Pin `golangci-lint` installation to v1 branch compatible with existing config.
- Keep current `.golangci.yml` semantics unchanged.

## Non-Goals
- No migration of `.golangci.yml` to v2 format.
- No behavior changes in application runtime code.

## Selected Approach
Pin lint installer commands in Makefile to `github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8`.

## Data / Contract Impact
None.

## Risks
- Future teams may still install v2 manually.
- Mitigation: Makefile acts as source of truth for project gates.

## Rollback Plan
- Revert Makefile linter install line to previous value.

## Test and Verification Plan
- Run fast gate after patch and confirm lint startup no longer fails due config version mismatch.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-tooling-gate-compatibility.md`
- `scripts/dev-micro-loop.sh --mode fast` when toolchain is available.

## Review Notes
- Approved to unblock continuous architecture migration.

## Implementation Summary
- Makefile linter install path/version aligned with existing config format.

## Final Verification
- Fast gate tooling setup proceeds without v2 config parsing failure.
