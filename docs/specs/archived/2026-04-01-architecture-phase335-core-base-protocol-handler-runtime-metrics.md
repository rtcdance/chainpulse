# Phase 335 - Core Base Protocol Handler Runtime Metrics

## Status
Status: Implemented

## Summary

Extend the core base protocol handler from raw running/router wiring to a
compact runtime metrics surface with coverage posture, runtime posture, and a
reliability hint.

## Problem

The core base protocol handler already exposes running state, processor wiring,
and router access, but callers still have to inspect those pieces manually to
decide whether the handler is unconfigured, merely idle, or fully ready.

## Decision

Add `GetRuntimeMetrics()` to `BaseProtocolHandler` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no protocol-handler lifecycle redesign
- no processor contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core base-protocol-handler runtime metrics
- compact handler coverage/runtime posture
- focused protocol-handler tests

Out of scope:

- lifecycle redesign
- per-request diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestBaseProtocolHandlerRuntimeMetricsUnobserved|TestBaseProtocolHandlerRuntimeMetricsIdle|TestBaseProtocolHandlerRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase335-core-base-protocol-handler-runtime-metrics.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestBaseProtocolHandlerRuntimeMetricsUnobserved|TestBaseProtocolHandlerRuntimeMetricsIdle|TestBaseProtocolHandlerRuntimeMetricsReady'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain because the default sandbox Go cache path was not writable.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase335-core-base-protocol-handler-runtime-metrics.md` passed.

## Exit Criteria

- `BaseProtocolHandler` exposes a compact runtime metrics surface beyond raw wiring state.
- Focused protocol-handler tests confirm unobserved, idle, and ready posture classification.
