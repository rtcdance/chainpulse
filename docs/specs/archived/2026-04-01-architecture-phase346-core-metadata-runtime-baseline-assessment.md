# Phase 346 - Core Metadata Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core metadata runtime surface after it reached compact
facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 345, request and response metadata now expose compact runtime
surfaces with:

- raw metadata metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core metadata slice as a stable baseline with a stop-line, rather
than continuing to add small runtime metrics by default.

## Decision

Classify the current core metadata work as:

- `stage-complete for the core metadata runtime baseline`

This means:

- the current metadata runtime surface is strong enough to pause by default
- the baseline already exposes compact runtime semantics for protocol
  attribution, request tracking, and response timing readiness

It does **not** mean:

- metadata schema semantics have been redesigned
- transport-specific diagnostics are complete
- broader control-plane orchestration has been finalized

## Scope

In scope:

- core metadata runtime baseline assessment
- explicit stop-line for the current core metadata runtime surface
- architecture/index documentation updates

Out of scope:

- metadata schema redesign
- protocol contract changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase346-core-metadata-runtime-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestRequestMetadataRuntimeMetricsUnobserved|TestRequestMetadataRuntimeMetricsReady|TestResponseMetadataRuntimeMetricsWatch|TestResponseMetadataRuntimeMetricsReady'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase346-core-metadata-runtime-baseline-assessment.md` should pass while the spec remains in `Approved` state.

## Exit Criteria

- The docs explicitly describe the core metadata runtime surface as a stable
  baseline with a stop-line.
- Future core metadata runtime expansion is treated as an explicit reopen
  rather than default continuation.
