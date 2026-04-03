# Phase 343 - Core Base Request Runtime Metrics

## Status
Status: Implemented

## Summary

Extend the core base request from raw method/path fields to a compact runtime
metrics surface with coverage posture, runtime posture, and a reliability hint.

## Problem

The core base request already stores method, path, headers, body, and
parameters, but callers still have to inspect those fields manually to decide
whether the request is minimally staged, metadata-only, fully parameterized, or
degraded due to missing routing metadata.

## Decision

Add `GetRuntimeMetrics()` to `BaseRequest` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no request-construction redesign
- no routing contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core base-request runtime metrics
- compact method/path/parameter coverage posture
- focused base-request tests

Out of scope:

- request parsing redesign
- per-protocol transport diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestBaseRequestRuntimeMetricsStaged|TestBaseRequestRuntimeMetricsReady|TestBaseRequestRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase343-core-base-request-runtime-metrics.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestBaseRequestRuntimeMetricsStaged|TestBaseRequestRuntimeMetricsReady|TestBaseRequestRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase343-core-base-request-runtime-metrics.md` passed.

## Exit Criteria

- `BaseRequest` exposes a compact runtime metrics surface beyond raw method/path fields.
- Focused base-request tests confirm staged, ready, and degraded posture classification.
