# Phase 305 - WebSocket Connection Metrics

## Status
Status: Approved

## Summary

Add a compact runtime connection metrics surface to the websocket plugin so the
transport can expose client count, transport posture, connection posture, and a
reliability hint.

## Problem

The websocket plugin already exposes isolated facts such as client count and
TLS metrics, but it has no single compact surface that describes whether the
runtime is stopped, idle, or actively serving clients, nor whether it is
running on a TLS-capable transport.

## Decision

Add `GetConnectionMetrics()` to the websocket plugin and expose:

- `running`
- `client_count`
- `transport_posture`
- `connection_posture`
- `reliability_hint`

Keep the slice intentionally small:

- no websocket message envelope changes
- no new protocol frames
- only runtime metrics-level surfacing

## Scope

In scope:

- websocket connection metrics surface
- compact transport/connection posture
- focused websocket plugin tests

Out of scope:

- websocket event payload metadata
- subscription-level protocol redesign
- broader websocket control plane

## Validation

- `go test ./pkg/plugins/api/websocket -run 'TestWebSocketPluginGetClientCount|TestWebSocketPluginGetConnectionMetricsPlaintextIdle|TestWebSocketPluginGetConnectionMetricsTLSIdle|TestWebSocketPluginGetConnectionMetricsActiveHint'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase305-websocket-connection-metrics.md`

## Exit Criteria

- The websocket plugin exposes a compact connection metrics surface.
- Focused websocket tests confirm plaintext, TLS-capable, and active-client postures.
