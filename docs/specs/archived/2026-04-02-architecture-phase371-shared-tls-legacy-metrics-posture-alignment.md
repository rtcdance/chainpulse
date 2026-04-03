# Phase 371 - Shared TLS Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `TLSManager.GetMetrics()` with the compact posture-oriented runtime
surface while preserving existing TLS metrics compatibility.

## Problem

Shared TLS already exposes posture fields through `GetRuntimeMetrics()`, but
legacy callers of `GetMetrics()` still receive only raw reload/error counters.
Those callers do not get the same compact posture and reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `certificate_posture` (compatibility)
- `reload_posture`
- `reliability_hint`

Keep existing TLS metrics fields unchanged:

- `reloads`
- `errors`
- `last_reload_at`

Keep the change intentionally small:

- no TLS lifecycle redesign
- no certificate reload algorithm change
- only legacy TLS metrics alignment

## Scope

In scope:

- shared TLS legacy metrics posture alignment
- compatibility preservation for existing TLS counter consumers
- focused TLS metrics tests

Out of scope:

- certificate provisioning redesign
- TLS config policy changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerMetricsIncludesPostureFields|TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue|TestTLSManagerRuntimeMetricsUnobserved|TestTLSManagerRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase371-shared-tls-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerMetricsIncludesPostureFields|TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue|TestTLSManagerRuntimeMetricsUnobserved|TestTLSManagerRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase371-shared-tls-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing TLS metric fields remain compatible.
- Focused tests confirm aligned posture values for ready/reload-due/unobserved/degraded states.
