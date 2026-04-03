# Phase 301 - gRPC Stream Source Posture Metrics

## Status
Status: Approved

## Summary

Extend the gRPC streaming service metrics surface to expose a compact source
posture for server-stream and client-stream event flows.

## Problem

The gRPC API already has event streaming primitives, but it has no way to
externally express where the stream is being served from. That leaves the gRPC
data plane behind the HTTP and GraphQL query surfaces, which already expose
compact source semantics.

## Decision

Add an optional `StreamingSourcePostureBackend` interface that lets a streaming
backend describe compact source posture for:

- server-stream events
- client-stream batches

Expose the resulting posture through `StreamingService.GetMetrics()` as:

- `server_source_posture`
- `client_source_posture`

Keep the slice intentionally small:

- no protobuf/schema change
- no event envelope rewrite
- only metrics-surface source hints

## Scope

In scope:

- optional streaming source posture interface
- gRPC streaming metrics surface updates
- focused streaming metrics tests

Out of scope:

- protobuf contract changes
- stream payload metadata
- broader gRPC event-query redesign

## Validation

- `go test ./pkg/plugins/api/grpc/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase301-grpc-stream-source-posture-metrics.md`

## Exit Criteria

- gRPC streaming metrics can expose compact source posture when the backend
  provides it.
- Server-stream and client-stream paths both preserve existing behavior while
  surfacing the new metrics-level hint.
