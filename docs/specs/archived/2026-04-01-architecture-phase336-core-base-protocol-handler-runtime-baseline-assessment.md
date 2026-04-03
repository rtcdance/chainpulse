# Phase 336 - Core Base Protocol Handler Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core base protocol-handler runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 335, the core base protocol handler now exposes a compact runtime
surface with:

- raw handler metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core base protocol-handler slice as a stable baseline with a
stop-line, rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core base protocol-handler work as:

- `stage-complete for the core base protocol-handler runtime baseline`

This means:

- the current core base protocol-handler runtime surface is strong enough to
  pause by default
- the baseline already exposes compact runtime semantics for handler wiring,
  processor presence, middleware coverage, and running readiness

It does **not** mean:

- protocol-handler lifecycle semantics have been redesigned
- per-request diagnostics are complete
- broader control-plane orchestration has been finalized

## Scope

In scope:

- core base protocol-handler runtime baseline assessment
- explicit stop-line for the current core base protocol-handler runtime surface
- architecture/index documentation updates

Out of scope:

- protocol-handler lifecycle redesign
- request processor contract changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase336-core-base-protocol-handler-runtime-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestBaseProtocolHandlerRuntimeMetricsUnobserved|TestBaseProtocolHandlerRuntimeMetricsIdle|TestBaseProtocolHandlerRuntimeMetricsReady'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase336-core-base-protocol-handler-runtime-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `go test ./pkg/plugins/api/core -run 'TestBaseProtocolHandlerRuntimeMetricsUnobserved|TestBaseProtocolHandlerRuntimeMetricsIdle|TestBaseProtocolHandlerRuntimeMetricsReady'` already passes with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.

## Exit Criteria

- The docs explicitly describe the core base protocol-handler runtime surface
  as a stable baseline with a stop-line.
- Future core base protocol-handler runtime expansion is treated as an explicit
  reopen rather than default continuation.
