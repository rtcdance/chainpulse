# Phase 345 - Core Metadata Runtime Metrics

## Status
Status: Implemented

## Summary

Extend core request/response metadata models from raw fields to a compact
runtime metrics surface with coverage posture, runtime posture, and a
reliability hint.

## Problem

`RequestMetadata` and `ResponseMetadata` already carry protocol, timing, and
attribution fields, but callers still have to inspect those fields manually to
decide whether metadata is unconfigured, partial, watch-level, or ready for
runtime observability.

## Decision

Add `GetRuntimeMetrics()` to both metadata models and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no metadata schema redesign
- no protocol contract change
- only core runtime metrics surfacing

## Scope

In scope:

- request/response metadata runtime metrics
- compact protocol/timing/attribution posture
- focused metadata tests

Out of scope:

- metadata persistence redesign
- per-transport diagnostics enrichment
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestRequestMetadataRuntimeMetricsUnobserved|TestRequestMetadataRuntimeMetricsReady|TestResponseMetadataRuntimeMetricsWatch|TestResponseMetadataRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase345-core-metadata-runtime-metrics.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestRequestMetadataRuntimeMetricsUnobserved|TestRequestMetadataRuntimeMetricsReady|TestResponseMetadataRuntimeMetricsWatch|TestResponseMetadataRuntimeMetricsReady'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase345-core-metadata-runtime-metrics.md` passed.

## Exit Criteria

- `RequestMetadata` and `ResponseMetadata` expose compact runtime metrics
  surfaces beyond raw fields.
- Focused metadata tests confirm unobserved/watch/ready posture coverage.
