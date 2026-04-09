Title: Grafana Monitoring Blueprint 8.1 Dashboard
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: monitoring, docs

## Status

Approved for implementation. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The archived architecture blueprint section `8.1` recommends a local debugging dashboard that covers:

1. performance
2. resource usage
3. business metrics
4. system health

The repository currently has a narrow `chainpulse-indexer.json` dashboard that focuses on indexing metrics only. It does not yet provide the broader single-screen operator view suggested by the blueprint.

## Scope

This slice will:

1. replace or extend the current Grafana dashboard provisioning with a blueprint-aligned dashboard
2. add panels for performance, resource usage, business metrics, and system health
3. keep the dashboard compatible with the existing Prometheus datasource setup
4. update the monitoring documentation to describe the new dashboard

## Non-Goals

This slice will not:

1. introduce new runtime metrics
2. redesign Prometheus scraping
3. add Alertmanager routing
4. modify service runtime behavior

## Selected Approach

Build a single local Grafana dashboard that groups the existing metrics into four operational sections:

1. performance panels for throughput and latency
2. resource panels for CPU and memory
3. business panels for ownership, reorg, and consistency signals
4. system-health panels for service availability and readiness posture

The first version will use existing metrics where possible and clearly label any panel that is only meaningful after the corresponding metric is available.

## Risks

1. Some metrics may not be emitted in every local mode, so panels could show `no data` during partial runs.
2. Overly broad panels can become noisy if metric names do not map cleanly across monolithic and microservice modes.
3. JSON dashboard provisioning is easy to break with malformed panel structure.

## Rollback

Rollback is straightforward:

1. restore the previous dashboard JSON
2. revert any documentation edits for the new dashboard
3. keep Prometheus and service runtime unchanged

## Test Strategy

1. validate the dashboard JSON structure with a lightweight parser
2. verify the provisioning path still exists under `monitoring/grafana`
3. run the repository's relevant shell or Go checks for any touched scripts or docs

## Quality Gate Plan

1. inspect the dashboard JSON for valid structure and panel coverage
2. run any script or documentation checks introduced by the edit
3. if available, confirm the compose config still references the dashboard provisioning directory

## Decision

Approved as the next blueprint-aligned observability slice.

## Implementation Notes

Implemented in:

- `monitoring/grafana/dashboards/chainpulse-indexer.json`
- `monitoring/grafana/dashboards/provider.yml`
- `monitoring/grafana/datasources/prometheus.yml`
- `scripts/verify-grafana-blueprint-81.sh`
- `docs/deployment/monitoring.md`
- `docs/IMPLEMENTATION_STATUS.md`
- `docs/INDEX.md`

The existing narrow Grafana dashboard was upgraded to a blueprint `8.1` local debug monitor with performance, resource, business, and system-health sections. Grafana provisioning now includes an explicit Prometheus datasource and dashboard provider so the compose-mounted Grafana container can load the dashboard automatically.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-grafana-blueprint-81.sh`
2. `bash scripts/verify-grafana-blueprint-81.sh --help`
3. `bash scripts/verify-grafana-blueprint-81.sh`
4. `docker compose -f docker/docker-compose.microservices.yml config --services`
