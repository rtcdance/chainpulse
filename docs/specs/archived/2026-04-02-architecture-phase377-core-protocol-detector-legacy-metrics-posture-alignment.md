# Phase 377 - Core Protocol Detector Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `ProtocolDetector.GetMetrics()` with the compact posture-oriented runtime
surface while preserving existing protocol metrics compatibility.

## Problem

Core protocol detector already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetMetrics()` still receive only
raw supported-protocol lists and counts. Those callers do not get the same
compact posture and reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep existing protocol metrics fields unchanged:

- `supported_protocols`
- `protocol_count`

Keep the change intentionally small:

- no protocol-detection redesign
- no routing contract change
- only legacy protocol metrics alignment

## Scope

In scope:

- core protocol-detector legacy metrics posture alignment
- compatibility preservation for existing protocol metrics consumers
- focused detector metrics tests

Out of scope:

- protocol-detection redesign
- per-request diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestProtocolDetectorMetricsIncludesPostureFields|TestProtocolDetectorRuntimeMetricsUnobserved|TestProtocolDetectorRuntimeMetricsWatch|TestProtocolDetectorRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase377-core-protocol-detector-legacy-metrics-posture-alignment.md`

## Verification Summary

- Validation commands should pass after legacy protocol metrics and focused tests are aligned.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing protocol metrics fields remain compatible.
- Focused tests confirm aligned posture values for unobserved/watch/ready states.
