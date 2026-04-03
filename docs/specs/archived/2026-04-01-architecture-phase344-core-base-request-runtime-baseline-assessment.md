# Phase 344 - Core Base Request Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core base-request runtime surface after it reached compact
facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 343, the core base request now exposes a compact runtime surface
with:

- raw request metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core base-request slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core base-request work as:

- `stage-complete for the core base-request runtime baseline`

This means:

- the current core base-request runtime surface is strong enough to pause by
  default
- the baseline already exposes compact runtime semantics for method/path
  validity and parameter/payload coverage

It does **not** mean:

- request lifecycle semantics have been redesigned
- transport-specific parsing diagnostics are complete
- broader control-plane orchestration has been finalized

## Scope

In scope:

- core base-request runtime baseline assessment
- explicit stop-line for the current core base-request runtime surface
- architecture/index documentation updates

Out of scope:

- request parsing redesign
- routing contract changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase344-core-base-request-runtime-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestBaseRequestRuntimeMetricsStaged|TestBaseRequestRuntimeMetricsReady|TestBaseRequestRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase344-core-base-request-runtime-baseline-assessment.md` should pass while the spec remains in `Approved` state.

## Exit Criteria

- The docs explicitly describe the core base-request runtime surface as a stable
  baseline with a stop-line.
- Future core base-request runtime expansion is treated as an explicit reopen
  rather than default continuation.
