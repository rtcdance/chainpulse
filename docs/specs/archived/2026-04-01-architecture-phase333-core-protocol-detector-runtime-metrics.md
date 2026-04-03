# Phase 333 - Core Protocol Detector Runtime Metrics

## Status
Status: Approved

## Summary

Extend the core protocol detector from raw supported-protocol metrics to a
compact runtime metrics surface with coverage posture, runtime posture, and a
reliability hint.

## Problem

The core protocol detector already exposes supported protocol names and counts,
but callers still have to inspect individual protocol registrations manually to
decide whether the detector is empty, HTTP-only, or broadly ready.

## Decision

Add `GetRuntimeMetrics()` to `ProtocolDetector` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no protocol-detection redesign
- no routing contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core protocol-detector runtime metrics
- compact protocol-coverage/runtime posture
- focused detector tests

Out of scope:

- protocol-detection redesign
- per-request diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestProtocolDetectorRuntimeMetricsUnobserved|TestProtocolDetectorRuntimeMetricsWatch|TestProtocolDetectorRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase333-core-protocol-detector-runtime-metrics.md`

## Exit Criteria

- `ProtocolDetector` exposes a compact runtime metrics surface beyond raw supported-protocol counts.
- Focused detector tests confirm unobserved, watch, and ready posture classification.
