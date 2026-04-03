# Phase 303 - gRPC Stream Reliability Hint

## Status
Status: Approved

## Summary

Extend the gRPC streaming metrics surface from source and delivery posture to a
compact reliability hint for server-stream and client-stream flows.

## Problem

After phase 302, gRPC streaming metrics could expose source posture and
delivery posture, but operators still had to interpret those fields manually to
decide how much confidence to place in the current stream behavior.

## Decision

Add compact reliability hints to `StreamingService.GetMetrics()`:

- `server_reliability_hint`
- `client_reliability_hint`

Derive the hint from existing source and delivery posture so the gRPC metrics
surface now carries:

- facts
- posture
- hint

Keep the change intentionally small:

- no protobuf change
- no stream payload metadata
- only metrics-level guidance

## Scope

In scope:

- compact reliability hint generation
- gRPC streaming metrics surface updates
- focused streaming reliability tests

Out of scope:

- retry orchestration
- advanced health scoring
- broader gRPC service-plane redesign

## Validation

- `go test ./pkg/plugins/api/grpc -run 'TestServerStreamEvents|TestClientStreamEvents|TestStreamingContextCancellation|TestStreamingMetrics|TestStreamingErrorHandling|TestStreamingTimeout|TestMultipleStreamsMetrics|TestClientStreamMetricsIncludeSourcePosture'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase303-grpc-stream-reliability-hint.md`

## Exit Criteria

- gRPC streaming metrics expose compact server/client reliability hints.
- Focused streaming tests confirm stable success and degraded-path guidance.
