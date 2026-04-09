Title: M3b Prometheus Metric Standardization
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/core, monitoring/prometheus, monitoring/grafana, docs/deployment

## Status

Approved for implementation.

## Problem Statement

The repository now exposes Prometheus text on `/metrics`, but the emitted
application metric names still use mixed legacy names without a consistent
`chainpulse_` prefix, and many series do not expose the blueprint-required
`chain_id` label. That leaves Grafana queries and alerts inconsistent and makes
cross-chain triage harder.

## Scope

This slice will:

1. standardize exported application metric names to `chainpulse_*`
2. normalize chain labels to `chain_id`
3. add a stable fallback `chain_id` label for application metrics that are not
   chain-scoped at emission time
4. update Grafana, Prometheus alert rules, and smoke scripts to query the
   standardized metric names
5. add focused tests for exporter normalization

## Non-Goals

This slice will not:

1. rename every in-process `RecordCounter` / `RecordGauge` call site
2. change Prometheus built-in runtime metric names such as `go_*`
3. redesign the metrics collector interface
4. add new chain-aware tags to every existing producer path in this same slice

## Selected Approach

1. keep in-process metric recording unchanged and standardize at Prometheus
   export time inside `pkg/core/metrics.go`
2. prefix non-runtime application metrics with `chainpulse_` unless they are
   already prefixed
3. normalize `chain`, `chain-id`, and `chain_id` tags into a single
   `chain_id` label
4. inject `chain_id="global"` when an exported application metric has no chain
   label
5. update Grafana queries, alert rules, docs, and Prometheus live smoke checks
   to consume the standardized metric names

## Risks

1. existing dashboards or ad hoc queries using legacy metric names will stop
   matching until updated
2. adding `chain_id="global"` may increase label cardinality by one extra
   series per metric family
3. external consumers that scrape `/metrics` and expect old names will need to
   migrate

## Rollback Plan

1. remove exporter-time prefixing and tag normalization
2. restore Grafana, alerts, and smoke queries to the previous names
3. keep the old `/metrics` exposition contract until downstream assets are
   realigned again

## Test Strategy

1. unit test exporter output for prefixed metric names
2. unit test exporter output for `chain_id` normalization and defaulting
3. focused verification for updated dashboard/alert/live-smoke queries

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-prometheus-metric-standardization.md`
2. `go test -short ./pkg/core/... ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./cmd/microservices/api-gateway/... ./cmd/microservices/api-service/... ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
3. `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes

Approved as the smallest P0 fix that makes Prometheus output and monitoring
assets conform to a stable naming contract without forcing a repository-wide
rewrite of metric producer call sites.
