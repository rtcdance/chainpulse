# Phase 361 - Core Plugin Registry Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `PluginRegistry.GetRegistryMetrics()` with the compact posture-oriented
runtime surface while preserving existing registry counter compatibility.

## Problem

The core plugin registry already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetRegistryMetrics()` still see
only raw counters and active plugin counts. Those callers do not receive the
same compact posture and reliability signals.

## Decision

Extend `GetRegistryMetrics()` to include:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep existing registry fields unchanged:

- `total_loaded`
- `total_unloaded`
- `active_plugins`
- `total_errors`

Keep the change intentionally small:

- no plugin lifecycle redesign
- no plugin interface contract change
- only legacy registry metrics alignment

## Scope

In scope:

- core plugin-registry legacy metrics posture alignment
- compatibility preservation for existing counter consumers
- focused registry metrics tests

Out of scope:

- lifecycle orchestration redesign
- per-plugin diagnostics expansion
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestPluginRegistryMetricsIncludesPostureFields|TestPluginRegistryRuntimeMetricsUnobserved|TestPluginRegistryRuntimeMetricsReady|TestPluginRegistryRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase361-core-plugin-registry-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/core -run 'TestPluginRegistryMetrics|TestPluginRegistryMetricsIncludesPostureFields|TestPluginRegistryRuntimeMetricsUnobserved|TestPluginRegistryRuntimeMetricsReady|TestPluginRegistryRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase361-core-plugin-registry-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetRegistryMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing registry counter fields remain compatible.
- Focused tests confirm aligned posture values for unobserved, ready, and degraded states.
