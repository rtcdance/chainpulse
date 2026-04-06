Title: OTel Distributed Tracing Alignment
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/observability, pkg/plugins/api/http, pkg/plugins/api/grpc, cmd/microservices

## Status

Approved for implementation. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The repository already exposes a custom tracer abstraction in `pkg/observability`,
but the runtime does not yet provide a real OpenTelemetry-backed tracer or a
clear HTTP entrypoint instrumentation path that matches blueprint §5.3
Distributed Tracing.

That leaves three gaps:

1. request spans are not exported through the standard OTel pipeline
2. trace context propagation is limited to the custom `traceparent` helpers
3. HTTP request handling does not consistently create server spans with useful
   request attributes

## Scope

This slice will:

1. add an OpenTelemetry-backed tracer implementation behind the existing
   `pkg/observability` API
2. wire W3C trace context extraction/injection through the custom tracer
   bridge
3. add HTTP server-span instrumentation around the API transport entrypoint
4. propagate trace context across outbound HTTP forwarding paths where the
   gateway already constructs client requests
5. keep current custom tracer tests passing while expanding coverage for OTel
   behavior

## Non-Goals

This slice will not:

1. redesign the full observability stack
2. add a production collector deployment
3. replace Prometheus metrics or logging behavior
4. instrument every internal business method in one pass
5. change public API request/response contracts

## Selected Approach

Use the existing `pkg/observability` tracer as the compatibility surface, but
back it with a real OTel tracer provider and W3C propagator in a small, local
runtime helper.

The implementation will:

1. initialize a tracer provider with a resource that identifies the service
2. expose a tracer that can start/end spans while preserving the existing
   `Span` snapshot model
3. add an HTTP middleware/helper that starts a server span for each inbound
   request, annotates the span with method/path/status, and preserves the span
   context on the request
4. inject trace context into outbound HTTP requests that already use the
   gateway forwarding client paths

The default behavior remains safe in local tests: if no OTel provider is
configured, the wrapper falls back to the compatibility tracer behavior instead
of failing request handling.

## Risks

1. OTel dependencies may increase module churn if versions drift.
2. HTTP path cardinality can explode if raw URLs are used as span names without
   normalization.
3. Double instrumentation is possible if both the transport wrapper and a child
   handler create spans for the same operation.

## Rollback

Rollback is straightforward:

1. restore the previous `pkg/observability` implementation
2. remove the HTTP tracing wrapper wiring
3. keep the existing request handling and metrics paths intact

## Test Strategy

1. unit test span creation, context propagation, and span attribute capture
2. unit test HTTP server middleware/span wrapper behavior
3. verify outbound client propagation injects `traceparent`
4. run the observability package tests and the API package tests that exercise
   the HTTP transport path

## Quality Gate Plan

1. keep the change small and local to observability/transport helpers
2. validate `pkg/observability` tests first
3. validate API HTTP/gateway tests next
4. run the repository fast gate before finishing

## Decision

Approved as the blueprint-aligned OTel tracing slice.

## Implementation Notes

The first implementation pass will prefer a compatibility bridge over a broad
rewrite so the existing `Tracer` interface remains usable across the codebase.

