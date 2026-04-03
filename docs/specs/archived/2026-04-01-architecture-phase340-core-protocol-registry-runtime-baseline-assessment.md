# Phase 340 - Core Protocol Registry Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core protocol-registry runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 339, the core protocol registry now exposes a compact runtime
surface with:

- raw protocol-registry metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core protocol-registry slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core protocol-registry work as:

- `stage-complete for the core protocol-registry runtime baseline`

This means:

- the current core protocol-registry runtime surface is strong enough to pause
  by default
- the baseline already exposes compact runtime semantics for handler
  registration, running coverage, and registry readiness

It does **not** mean:

- registry lifecycle semantics have been redesigned
- per-handler diagnostics are complete
- broader control-plane orchestration has been finalized

## Scope

In scope:

- core protocol-registry runtime baseline assessment
- explicit stop-line for the current core protocol-registry runtime surface
- architecture/index documentation updates

Out of scope:

- registry lifecycle redesign
- protocol-handler contract changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase340-core-protocol-registry-runtime-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestProtocolRegistryRuntimeMetricsUnobserved|TestProtocolRegistryRuntimeMetricsWatch|TestProtocolRegistryRuntimeMetricsReady'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase340-core-protocol-registry-runtime-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `go test ./pkg/plugins/api/core -run 'TestProtocolRegistryRuntimeMetricsUnobserved|TestProtocolRegistryRuntimeMetricsWatch|TestProtocolRegistryRuntimeMetricsReady'` already passes with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.

## Exit Criteria

- The docs explicitly describe the core protocol-registry runtime surface as a
  stable baseline with a stop-line.
- Future core protocol-registry runtime expansion is treated as an explicit
  reopen rather than default continuation.
