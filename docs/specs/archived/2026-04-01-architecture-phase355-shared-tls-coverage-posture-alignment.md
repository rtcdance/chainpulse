# Phase 355 - Shared TLS Coverage Posture Alignment

## Status
Status: Approved

## Summary

Align shared TLS runtime metrics to expose explicit `coverage_posture` while
preserving existing `certificate_posture` compatibility.

## Problem

`TLSManager.GetRuntimeMetrics()` currently exposes `certificate_posture` as
its primary coverage-like signal, which diverges from the broader runtime
metrics contract that uses `coverage_posture` across components.

## Decision

Extend `GetRuntimeMetrics()` to include:

- `coverage_posture` (aligned with existing TLS certificate posture semantics)
- existing `certificate_posture` retained for backward compatibility

Keep the change intentionally small:

- no TLS lifecycle redesign
- no certificate reload algorithm change
- only runtime metrics field alignment

## Scope

In scope:

- TLS runtime coverage field alignment
- compatibility preservation for existing certificate posture consumers
- focused runtime metrics tests

Out of scope:

- certificate provisioning flow redesign
- TLS config policy expansion
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue|TestTLSManagerRuntimeMetricsUnobserved|TestTLSManagerRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase355-shared-tls-coverage-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue|TestTLSManagerRuntimeMetricsUnobserved|TestTLSManagerRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase355-shared-tls-coverage-posture-alignment.md` passed.

## Exit Criteria

- `TLSManager` runtime metrics expose `coverage_posture` and retain
  `certificate_posture` compatibility.
- Focused tests confirm posture values for ready/reload-due/unobserved/degraded
  states.
