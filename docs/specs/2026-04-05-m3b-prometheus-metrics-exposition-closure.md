Title: M3b Prometheus Metrics Exposition Closure
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/core, pkg/plugins/api, cmd/monolithic/chainpulse, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor, monitoring

## Status

Approved for implementation.

## Problem Statement

The repository now contains Grafana dashboards, Prometheus scrape configs, and
alert rules, but the runtime metrics surfaces are not Prometheus-compatible in
practice:

1. several `/metrics` handlers still return JSON metric dumps instead of
   Prometheus exposition text
2. some services advertise `/metrics` but do not actually wire a runtime
   metrics provider
3. monitoring dashboards and alert rules reference metric names that are not
   emitted by the currently running services

This makes the monitoring stack effectively dead code even though the files
exist in the repository.

## Scope

This slice will:

1. add Prometheus exposition support for the default in-memory metrics
   collector
2. update runtime `/metrics` handlers to return Prometheus text instead of JSON
3. wire `/metrics` on services that already advertise the endpoint
4. align Grafana dashboard queries and alert rules to metric names that are
   actually emitted by the current runtime
5. add focused tests for metrics exposition and route availability

## Non-Goals

This slice will not:

1. introduce a third-party Prometheus client library
2. redesign all runtime metrics names repository-wide
3. add new durable monitoring backends
4. claim full production monitoring completeness

## Selected Approach

Keep the change additive and repository-local:

1. teach the current `DefaultMetricsCollector` to export Prometheus text for
   counters, gauges, and histogram buckets
2. switch the existing `/metrics` routes to use that exporter
3. wire the missing gateway/service runtime metrics providers where startup
   output already claims the route exists
4. update dashboard and alert expressions to target the metrics already emitted
   by the current codebase

## Risks

1. histogram bucket choices may not perfectly match every metric's unit
2. changing `/metrics` from JSON to Prometheus text could break any internal
   consumer that incorrectly relied on the old debug payload
3. alert expressions can still show `no data` if the underlying code path has
   never emitted that metric in the current runtime mode

## Rollback Plan

1. revert the Prometheus exposition helper
2. restore the prior JSON-based `/metrics` handlers
3. restore the previous dashboard and alert expressions

## Test Strategy

1. unit test Prometheus exposition output for counters, gauges, and histograms
2. unit test `/metrics` routes for monolithic and microservice runtimes
3. unit test that API gateway/service expose `/metrics` when advertised
4. run focused package tests plus the fast micro-loop gate

## Quality Gates

1. `go test -short ./pkg/core/... ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-gateway/... ./cmd/microservices/api-service/... ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
2. `scripts/dev-micro-loop.sh --mode fast --base HEAD`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-prometheus-metrics-exposition-closure.md`

## Review Notes

Approved as the smallest change that turns the existing Prometheus, Grafana,
and alerting assets into a runnable monitoring path instead of dead code.
