# Phase 327 - Core Plugin Registry Runtime Metrics

## Status
Status: Approved

## Summary

Extend the core plugin registry from raw registry counters to a compact runtime
metrics surface with coverage posture, runtime posture, and a reliability hint.

## Problem

The core plugin registry already exposes raw loaded/unloaded/error counters and
active plugin counts, but callers still have to inspect individual plugin
statuses manually to decide whether the registry is empty, partially running,
or degraded.

## Decision

Add `GetRuntimeMetrics()` to `PluginRegistry` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no plugin lifecycle redesign
- no plugin interface contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core plugin-registry runtime metrics
- compact coverage/runtime posture
- focused registry tests

Out of scope:

- lifecycle orchestration redesign
- per-plugin diagnostics expansion
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestPluginRegistryRuntimeMetricsUnobserved|TestPluginRegistryRuntimeMetricsReady|TestPluginRegistryRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase327-core-plugin-registry-runtime-metrics.md`

## Exit Criteria

- `PluginRegistry` exposes a compact runtime metrics surface beyond raw registry counters.
- Focused registry tests confirm unobserved, ready, and degraded posture classification.
