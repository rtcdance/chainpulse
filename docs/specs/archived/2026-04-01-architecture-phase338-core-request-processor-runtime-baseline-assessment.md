# Phase 338 - Core Request Processor Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core request-processor runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 337, the core request processor now exposes a compact runtime
surface with:

- raw request-processor metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core request-processor slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core request-processor work as:

- `stage-complete for the core request-processor runtime baseline`

This means:

- the current core request-processor runtime surface is strong enough to pause
  by default
- the baseline already exposes compact runtime semantics for API-layer
  presence, route coverage, error-mapper wiring, and readiness

It does **not** mean:

- processing lifecycle semantics have been redesigned
- per-request diagnostics are complete
- broader control-plane orchestration has been finalized

## Scope

In scope:

- core request-processor runtime baseline assessment
- explicit stop-line for the current core request-processor runtime surface
- architecture/index documentation updates

Out of scope:

- processing-pipeline redesign
- request/response contract changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase338-core-request-processor-runtime-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestDefaultRequestProcessorRuntimeMetricsUnobserved|TestDefaultRequestProcessorRuntimeMetricsWatch|TestDefaultRequestProcessorRuntimeMetricsReady'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase338-core-request-processor-runtime-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `go test ./pkg/plugins/api/core -run 'TestDefaultRequestProcessorRuntimeMetricsUnobserved|TestDefaultRequestProcessorRuntimeMetricsWatch|TestDefaultRequestProcessorRuntimeMetricsReady'` already passes with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.

## Exit Criteria

- The docs explicitly describe the core request-processor runtime surface as a
  stable baseline with a stop-line.
- Future core request-processor runtime expansion is treated as an explicit
  reopen rather than default continuation.
