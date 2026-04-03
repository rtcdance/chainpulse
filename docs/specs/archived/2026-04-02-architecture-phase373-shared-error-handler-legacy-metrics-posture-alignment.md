# Phase 373 - Shared Error Handler Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `ErrorHandler.GetMetrics()` with the compact posture-oriented runtime
surface while preserving existing error-handler metric compatibility.

## Problem

Shared error handler already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetMetrics()` still receive only
raw circuit-breaker state and retry counters. Those callers do not get the
same compact posture and reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `circuit_posture` (compatibility)
- `retry_posture`
- `reliability_hint`

Keep existing error-handler metrics fields unchanged:

- `circuit_breaker_state`
- `failure_count`
- `success_count`
- `last_failure_time`
- `max_retries`

Keep the change intentionally small:

- no circuit-breaker algorithm redesign
- no retry policy behavior change
- only legacy error-handler metrics alignment

## Scope

In scope:

- shared error-handler legacy metrics posture alignment
- compatibility preservation for existing metrics consumers
- focused error-handler metrics tests

Out of scope:

- breaker state machine redesign
- retry budget policy changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerMetricsIncludesPostureFields|TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase373-shared-error-handler-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerMetricsIncludesPostureFields|TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase373-shared-error-handler-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing error-handler metric fields remain compatible.
- Focused tests confirm aligned posture values for ready/open/probing states.
