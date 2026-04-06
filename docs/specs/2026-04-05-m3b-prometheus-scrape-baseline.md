Title: M3b Prometheus Scrape Baseline
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: docker, monitoring, scripts, docs/deployment

## Status

Approved for implementation.

## Problem Statement

The repository now exposes Prometheus-compatible `/metrics` endpoints, but the
monitoring path is still fragile because:

1. docker compose files mount the wrong Prometheus config path
2. there is no repository-level verifier for Prometheus scrape wiring
3. monitoring docs still describe stale config locations

That leaves Prometheus scrape readiness partially implicit even though the
monitoring assets exist.

## Scope

This slice will:

1. fix compose mounts to use the real Prometheus config path
2. add a repo-root verification script for Prometheus scrape baseline assets
3. validate scrape jobs, alert rules, and compose mount consistency
4. update monitoring docs to reference the real config path

## Non-Goals

This slice will not:

1. start containers automatically
2. require a live Prometheus process
3. add new alert rules beyond the current baseline
4. claim full monitoring-stack production readiness

## Selected Approach

Keep the verification lightweight and repository-local:

1. correct compose volumes to mount `monitoring/prometheus/prometheus.yml`
2. add a shell verifier that checks:
   - Prometheus config file exists
   - alert rules file exists
   - expected jobs are present
   - compose files mount the same config path
3. keep the script compatible with local CI and non-Docker environments

## Risks

1. static validation cannot guarantee a live scrape succeeds
2. future compose refactors could require script updates if path conventions change

## Rollback Plan

1. revert compose path changes
2. remove the Prometheus scrape verification script
3. restore prior monitoring doc references

## Test Strategy

1. `bash -n scripts/verify-prometheus-scrape-baseline.sh`
2. `bash scripts/verify-prometheus-scrape-baseline.sh --help`
3. `bash scripts/verify-prometheus-scrape-baseline.sh`
4. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-prometheus-scrape-baseline.md`

## Quality Gates

1. `bash -n scripts/verify-prometheus-scrape-baseline.sh`
2. `bash scripts/verify-prometheus-scrape-baseline.sh --help`
3. `bash scripts/verify-prometheus-scrape-baseline.sh`
4. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-prometheus-scrape-baseline.md`

## Review Notes

Approved as the smallest follow-up slice that turns the Prometheus monitoring
path from config drift into an explicit, verifiable repository baseline.
