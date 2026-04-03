# Phase 302 - gRPC Stream Delivery Posture

## Status
Status: Approved

## Summary

Extend the gRPC streaming metrics surface from raw source hints to compact
delivery posture for both server-stream and client-stream event flows.

## Problem

After phase 301, gRPC streaming metrics could expose source posture, but they
still required readers to interpret raw counts and error totals by hand to
decide whether delivery currently looked healthy, idle, or degraded.

## Decision

Add compact delivery posture to `StreamingService.GetMetrics()`:

- `server_delivery_posture`
- `client_delivery_posture`

Use existing metrics and source posture to classify high-level states such as:

- delivered
- error
- idle
- unobserved

Keep the change intentionally small:

- no protobuf change
- no stream payload rewrite
- only metrics-level posture classification

## Scope

In scope:

- compact delivery posture classification
- streaming metrics surface updates
- focused streaming posture tests

Out of scope:

- transport-level retry semantics
- detailed per-stream timeline analysis
- broader gRPC service-plane redesign

## Validation

- `go test ./pkg/plugins/api/grpc -run 'TestServerStreamEvents|TestClientStreamEvents|TestStreamingContextCancellation|TestStreamingMetrics|TestStreamingErrorHandling|TestStreamingTimeout|TestMultipleStreamsMetrics|TestClientStreamMetricsIncludeSourcePosture'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase302-grpc-stream-delivery-posture.md`

## Exit Criteria

- gRPC streaming metrics expose compact server/client delivery posture.
- Focused streaming tests confirm success and error postures remain stable.
