# Phase 341 - Core Base Response Runtime Metrics

## Status
Status: Implemented

## Summary

Extend the core base response from raw status/body state to a compact runtime
metrics surface with coverage posture, runtime posture, and a reliability hint.

## Problem

The core base response already exposes status, headers, body, and send state,
but callers still have to inspect those fields manually to decide whether the
response is empty, staged, ready to send, or already sent.

## Decision

Add `GetRuntimeMetrics()` to `BaseResponse` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no response lifecycle redesign
- no send contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core base-response runtime metrics
- compact payload/send-state posture
- focused base-response tests

Out of scope:

- response serialization redesign
- per-protocol transport diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestBaseResponseRuntimeMetricsStaged|TestBaseResponseRuntimeMetricsReady|TestBaseResponseRuntimeMetricsSent'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase341-core-base-response-runtime-metrics.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestBaseResponseRuntimeMetricsStaged|TestBaseResponseRuntimeMetricsReady|TestBaseResponseRuntimeMetricsSent'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase341-core-base-response-runtime-metrics.md` passed.

## Exit Criteria

- `BaseResponse` exposes a compact runtime metrics surface beyond raw status/body state.
- Focused base-response tests confirm staged, ready, and sent posture classification.
