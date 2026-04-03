# Phase 339 - Core Protocol Registry Runtime Metrics

## Status
Status: Implemented

## Summary

Extend the core protocol registry from raw handler storage to a compact runtime
metrics surface with coverage posture, runtime posture, and a reliability hint.

## Problem

The core protocol registry already stores protocol handlers and can start or
stop them in bulk, but callers still have to inspect registration counts and
handler running states manually to decide whether the registry is empty,
registered-but-idle, or ready for protocol dispatch.

## Decision

Add `GetRuntimeMetrics()` to `ProtocolRegistry` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no registry lifecycle redesign
- no protocol-handler contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core protocol-registry runtime metrics
- compact handler-registration/running-state posture
- focused protocol-registry tests

Out of scope:

- registry lifecycle redesign
- per-handler diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestProtocolRegistryRuntimeMetricsUnobserved|TestProtocolRegistryRuntimeMetricsWatch|TestProtocolRegistryRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase339-core-protocol-registry-runtime-metrics.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestProtocolRegistryRuntimeMetricsUnobserved|TestProtocolRegistryRuntimeMetricsWatch|TestProtocolRegistryRuntimeMetricsReady'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase339-core-protocol-registry-runtime-metrics.md` passed.

## Exit Criteria

- `ProtocolRegistry` exposes a compact runtime metrics surface beyond raw handler registration.
- Focused protocol-registry tests confirm unobserved, watch, and ready posture classification.
