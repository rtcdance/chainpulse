Title: M3b Microservice Observability Baseline
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/event-processor, cmd/microservices/puller

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M3a` closed the minimum deployment-verification baseline, but `M3b` needs a
repeatable way to verify the current observability posture across the four
foreground microservices.

The repository already exposes:

1. `/metrics`
2. `/runtime/summary`
3. `/health/rollout`

What is missing is a single repo-root verification entry that checks those
surfaces together as an observability baseline.

## Scope

This slice will:

1. add a repo-root observability verification script
2. start the full local runnable profile
3. verify metrics, runtime summary, and rollout advisory reachability across
   the four foreground microservices

## Non-Goals

This slice will not:

1. introduce Prometheus or Grafana provisioning
2. claim full `M3b` completion
3. add a real alert manager
4. redesign service metrics payloads

## Selected Approach

Add a narrow script that reuses the current full runnable profile and full local
verification, then asserts the most important observability facts:

1. `/metrics` is reachable
2. `runtime summary` reports collector availability
3. `health/rollout` exposes advisory output

## Quality Gates

1. `bash -n scripts/verify-microservice-observability-baseline.sh`
2. `bash scripts/verify-microservice-observability-baseline.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3b-microservice-observability-baseline.md`

## Decision

Approved for implementation as the first `M3b` slice.

## Implementation Notes

Implemented in:

- `scripts/verify-microservice-observability-baseline.sh`
- `RUNNABLE_APP.md`
- `README.md`
- `docs/MILESTONE_STATUS.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

The script keeps the implementation narrow by reusing the current full runnable
profile instead of introducing a second observability orchestration path.

## Verification Summary

The following checks passed after implementation:

1. `bash -n scripts/verify-microservice-observability-baseline.sh`
2. `bash scripts/verify-microservice-observability-baseline.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3b-microservice-observability-baseline.md`
