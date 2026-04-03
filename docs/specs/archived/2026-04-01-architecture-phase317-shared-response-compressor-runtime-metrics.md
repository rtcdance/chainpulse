# Phase 317 - Shared Response Compressor Runtime Metrics

## Status
Status: Approved

## Summary

Extend the shared response compressor from raw counters to a compact runtime
metrics surface with coverage posture, efficiency posture, and a reliability
hint.

## Problem

The shared response compressor already exposes raw response counts, compression
ratio, and duration metrics, but callers still have to interpret those numbers
manually to determine whether compression is active, bypassed, or inefficient.

## Decision

Add `GetRuntimeMetrics()` to `ResponseCompressor` and expose:

- `coverage_posture`
- `efficiency_posture`
- `reliability_hint`

Keep the change intentionally small:

- no compression policy redesign
- no transport contract change
- only shared runtime metrics surfacing

## Scope

In scope:

- shared response-compressor runtime metrics
- compact coverage/efficiency posture
- focused compressor tests

Out of scope:

- compression-threshold redesign
- per-response diagnostics
- broader transport-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestResponseCompressorRuntimeMetricsUnobserved|TestResponseCompressorRuntimeMetricsBypassed|TestResponseCompressorRuntimeMetricsEfficient'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase317-shared-response-compressor-runtime-metrics.md`

## Exit Criteria

- `ResponseCompressor` exposes a compact runtime metrics surface beyond raw counters.
- Focused compressor tests confirm unobserved, bypassed, and efficient posture classification.
