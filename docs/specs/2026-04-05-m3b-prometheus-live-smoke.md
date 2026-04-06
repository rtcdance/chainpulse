Title: M3b Prometheus Live Smoke
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, docs/deployment, RUNNABLE_APP.md, README.md

## Status

Approved for implementation.

## Problem Statement

The repository now has:

1. Prometheus-compatible `/metrics` endpoints
2. Prometheus scrape config and alert rule baseline verification

But there is still no focused smoke that talks to a live Prometheus server and
verifies:

1. scrape targets are visible through the Prometheus API
2. key queries can be executed successfully

Without that, the monitoring path is still only partially verified.

## Scope

This slice will:

1. add a live Prometheus smoke script against an operator-supplied Prometheus URL
2. verify `targets` API reachability
3. verify configured ChainPulse scrape jobs are present
4. verify a small set of live instant queries execute successfully
5. document the script in runnable/operations docs

## Non-Goals

This slice will not:

1. auto-start Prometheus or compose services
2. require Docker during script execution
3. verify every dashboard panel expression
4. guarantee production-grade monitoring readiness by itself

## Selected Approach

Add a lightweight shell script that uses:

1. `curl` to query Prometheus HTTP APIs
2. `python3` to parse JSON responses

The script will fail fast if Prometheus is unreachable, if expected jobs are
absent from `/api/v1/targets`, or if key instant queries return an API error.

## Risks

1. a live smoke depends on an operator-provided running Prometheus endpoint
2. some metrics may exist but return empty result vectors in low-traffic runs

## Rollback Plan

1. remove the live smoke script
2. revert docs that reference it
3. keep static scrape baseline verification in place

## Test Strategy

1. `bash -n scripts/verify-prometheus-live-smoke.sh`
2. `bash scripts/verify-prometheus-live-smoke.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-prometheus-live-smoke.md`

## Quality Gates

1. `bash -n scripts/verify-prometheus-live-smoke.sh`
2. `bash scripts/verify-prometheus-live-smoke.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-prometheus-live-smoke.md`

## Review Notes

Approved as the smallest live validation step above static Prometheus config
checks.
