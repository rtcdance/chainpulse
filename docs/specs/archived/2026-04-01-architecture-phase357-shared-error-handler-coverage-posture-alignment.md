# Phase 357 - Shared Error Handler Coverage Posture Alignment

## Status
Status: Approved

## Summary

Align shared error-handler runtime metrics to expose explicit
`coverage_posture` while preserving existing `circuit_posture` compatibility.

## Problem

`ErrorHandler.GetRuntimeMetrics()` currently exposes `circuit_posture` as
its primary coverage-like signal, which diverges from the broader runtime
metrics contract that uses `coverage_posture` across components.

## Decision

Extend `GetRuntimeMetrics()` to include:

- `coverage_posture` (aligned with existing circuit posture semantics)
- existing `circuit_posture` retained for backward compatibility

Keep the change intentionally small:

- no circuit-breaker algorithm redesign
- no retry policy behavior change
- only runtime metrics field alignment

## Scope

In scope:

- error-handler runtime coverage field alignment
- compatibility preservation for existing circuit posture consumers
- focused runtime metrics tests

Out of scope:

- breaker state machine redesign
- retry budget tuning policy changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase357-shared-error-handler-coverage-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase357-shared-error-handler-coverage-posture-alignment.md` passed.

## Exit Criteria

- `ErrorHandler` runtime metrics expose `coverage_posture` and retain
  `circuit_posture` compatibility.
- Focused tests confirm aligned posture values for ready/open/probing states.
