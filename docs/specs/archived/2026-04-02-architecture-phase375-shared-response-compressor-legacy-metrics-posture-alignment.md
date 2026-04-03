# Phase 375 - Shared Response Compressor Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `ResponseCompressor.GetMetrics()` with the compact posture-oriented
runtime surface while preserving existing compression metric compatibility.

## Problem

Shared response compressor already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetMetrics()` still receive only
raw response counts, ratios, and duration metrics. Those callers do not get
the same compact posture and reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `efficiency_posture`
- `reliability_hint`

Keep existing compression metrics fields unchanged:

- `total_responses`
- `compressed_count`
- `original_size`
- `compressed_size`
- `compression_ratio`
- `avg_duration_ms`
- `total_duration`

Keep the change intentionally small:

- no compression policy redesign
- no transport contract change
- only legacy compression metrics alignment

## Scope

In scope:

- shared response-compressor legacy metrics posture alignment
- compatibility preservation for existing compression metric consumers
- focused compressor metrics tests

Out of scope:

- compression-threshold redesign
- per-response diagnostics
- broader transport-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestResponseCompressorMetricsIncludesPostureFields|TestResponseCompressorRuntimeMetricsUnobserved|TestResponseCompressorRuntimeMetricsBypassed|TestResponseCompressorRuntimeMetricsEfficient'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase375-shared-response-compressor-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestResponseCompressorMetricsIncludesPostureFields|TestResponseCompressorRuntimeMetricsUnobserved|TestResponseCompressorRuntimeMetricsBypassed|TestResponseCompressorRuntimeMetricsEfficient'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase375-shared-response-compressor-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing compression metric fields remain compatible.
- Focused tests confirm aligned posture values for unobserved/bypassed/efficient states.
