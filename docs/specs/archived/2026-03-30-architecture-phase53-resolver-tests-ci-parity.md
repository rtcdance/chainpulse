# Phase 53 Resolver Tests CI Parity

## Title
Phase 53 - Integrate baseline resolver shell tests into CI policy-contract workflow

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
- `.github/workflows/ci.yml`
- `scripts/test-baseline-update-resolver.sh`
- `scripts/dev-micro-loop.sh`
- `docs/ARCHITECTURE.md`

## Context
Phase 52 added resolver shell tests and local full-loop integration, but CI policy-contract job did not execute the same resolver test suite.

## Problem Statement
Without CI execution, resolver regressions can still pass remote checks if local safeguards are skipped.

## Scope
- Add resolver shell test step to CI `policy-contract` workflow:
  - `./scripts/test-baseline-update-resolver.sh`
- Keep local full-loop behavior unchanged (already integrated).
- Document CI/local parity completion.

## Non-Goals
- No runtime service behavior changes.
- No change to lint/test/build job topology.

## Options Considered
- Option A: keep resolver tests local-only.
- Option B: run resolver tests in CI policy-contract for parity.

## Selected Approach
Choose Option B for governance consistency and earlier regression detection.

## Data / Contract Impact
- No runtime contract changes.
- CI governance check contract now includes resolver function tests.

## Risks
- Minimal; resolver tests are fast and deterministic.

## Rollback Plan
- Remove resolver test step from CI policy-contract job.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase53-resolver-tests-ci-parity.md`
- `bash -n .github/workflows/ci.yml`
- `./scripts/test-baseline-update-resolver.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase53-resolver-tests-ci-parity.md`
- CI policy-contract executes resolver shell tests.

## Review Notes
- Approved to align CI with local governance verification standards.

## Implementation Summary
- Added `Run baseline update resolver tests` step to policy-contract workflow.
- Completed CI/local parity for resolver governance checks.

## Final Verification
- Resolver shell tests pass locally and are now enforced in CI workflow.
