# Phase 321 - Shared Error Handler Runtime Metrics

## Status
Status: Approved

## Summary

Extend the shared error handler from raw circuit-breaker state to a compact
runtime metrics surface with circuit posture, retry posture, and a reliability
hint.

## Problem

The shared error handler already exposes raw circuit-breaker state, failure
count, success count, and retry budget, but callers still have to interpret
those numbers manually to decide whether the handler is ready, probing, or
blocking behind an open circuit.

## Decision

Add `GetRuntimeMetrics()` to `ErrorHandler` and expose:

- `circuit_posture`
- `retry_posture`
- `reliability_hint`

Keep the change intentionally small:

- no circuit-breaker redesign
- no retry-policy contract change
- only shared runtime metrics surfacing

## Scope

In scope:

- shared error-handler runtime metrics
- compact circuit/retry posture
- focused error-handler tests

Out of scope:

- retry-policy redesign
- per-error diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase321-shared-error-handler-runtime-metrics.md`

## Exit Criteria

- `ErrorHandler` exposes a compact runtime metrics surface beyond raw circuit state.
- Focused error-handler tests confirm ready, open-circuit, and probing posture classification.
