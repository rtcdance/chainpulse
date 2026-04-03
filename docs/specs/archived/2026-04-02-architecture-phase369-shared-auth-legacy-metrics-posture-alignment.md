# Phase 369 - Shared Auth Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `Authentication.GetMetrics()` with the compact posture-oriented runtime
surface while preserving existing token count compatibility.

## Problem

Shared authentication already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetMetrics()` still receive only
raw token counts. Those callers do not get the same compact posture and
reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep existing authentication metrics fields unchanged:

- `total_tokens`
- `active_tokens`
- `expired_tokens`

Keep the change intentionally small:

- no token-validation redesign
- no permission contract change
- only legacy authentication metrics alignment

## Scope

In scope:

- shared auth legacy metrics posture alignment
- compatibility preservation for existing token count consumers
- focused authentication metrics tests

Out of scope:

- token rotation redesign
- per-token diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestAuthenticationMetricsIncludesPostureFields|TestAuthenticationRuntimeMetricsUnconfigured|TestAuthenticationRuntimeMetricsReady|TestAuthenticationRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase369-shared-auth-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestAuthenticationMetricsIncludesPostureFields|TestAuthenticationRuntimeMetricsUnconfigured|TestAuthenticationRuntimeMetricsReady|TestAuthenticationRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase369-shared-auth-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing token count fields remain compatible.
- Focused tests confirm aligned posture values for unconfigured/ready/degraded states.
