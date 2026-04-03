# Phase 52 Resolver Shell Tests

## Title
Phase 52 - Add lightweight shell tests for shared baseline update resolver

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
- `scripts/lib/baseline_update_resolver.sh`
- `scripts/test-baseline-update-resolver.sh`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/ARCHITECTURE.md`

## Context
Phase 51 extracted shared resolver logic, but lacked direct function-level checks that can quickly detect regression in normalization/resolve behavior.

## Problem Statement
Without targeted tests for the shared resolver, regressions may only surface through higher-level workflows, increasing debugging time.

## Scope
- Add lightweight shell test script for resolver functions:
  - normalization
  - scope validation
  - scope resolution
  - changed-set resolution
- Include positive and negative cases.
- Integrate resolver tests into:
  - local full micro-loop
  - Makefile target

## Non-Goals
- No Go unit tests in this phase.
- No runtime service behavior changes.

## Options Considered
- Option A: rely on integration scripts only.
- Option B: add dedicated shell test coverage for resolver functions.

## Selected Approach
Choose Option B for faster feedback and lower regression triage cost.

## Data / Contract Impact
- No API/domain/runtime contract changes.

## Risks
- Minimal; shell tests are read-only and isolated.

## Rollback Plan
- Remove resolver test script and integration hooks from Makefile/dev-loop.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase52-resolver-shell-tests.md`
- `bash -n scripts/test-baseline-update-resolver.sh`
- `./scripts/test-baseline-update-resolver.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase52-resolver-shell-tests.md`
- `./scripts/test-baseline-update-resolver.sh`

## Review Notes
- Approved to strengthen confidence in shared resolver behavior.

## Implementation Summary
- Added dedicated resolver shell test suite with assert helpers.
- Wired test suite into local full micro-loop and Makefile.

## Final Verification
- Resolver shell tests pass across all defined scenarios.
