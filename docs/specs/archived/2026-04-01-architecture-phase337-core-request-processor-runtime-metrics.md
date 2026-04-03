# Phase 337 - Core Request Processor Runtime Metrics

## Status
Status: Implemented

## Summary

Extend the core default request processor from raw API-layer wiring to a
compact runtime metrics surface with coverage posture, runtime posture, and a
reliability hint.

## Problem

The core default request processor already delegates requests into the API
layer, but callers still have to inspect API-layer presence, route coverage,
and error-mapper wiring manually to decide whether the processor is absent,
partially wired, or ready.

## Decision

Add `GetRuntimeMetrics()` to `DefaultRequestProcessor` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no request-processing redesign
- no request/response contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core request-processor runtime metrics
- compact API-layer-derived coverage/runtime posture
- focused request-processor tests

Out of scope:

- processing pipeline redesign
- per-request diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestDefaultRequestProcessorRuntimeMetricsUnobserved|TestDefaultRequestProcessorRuntimeMetricsWatch|TestDefaultRequestProcessorRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase337-core-request-processor-runtime-metrics.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestDefaultRequestProcessorRuntimeMetricsUnobserved|TestDefaultRequestProcessorRuntimeMetricsWatch|TestDefaultRequestProcessorRuntimeMetricsReady'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase337-core-request-processor-runtime-metrics.md` passed.

## Exit Criteria

- `DefaultRequestProcessor` exposes a compact runtime metrics surface beyond raw API-layer presence.
- Focused request-processor tests confirm unobserved, watch, and ready posture classification.
