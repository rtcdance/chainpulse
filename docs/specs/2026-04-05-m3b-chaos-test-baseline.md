Title: M3b Chaos Test Baseline
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, docker, cmd/microservices/puller, cmd/microservices/event-processor, cmd/microservices/api-service, docs/deployment

## Status

Approved for implementation.

## Problem Statement

The blueprint requires Phase 4 chaos validation for:

1. RPC node failure
2. Kafka disconnect
3. database unavailability

The repository currently has deployment, observability, alert, and Prometheus
verification entries, but it still lacks a single executable chaos-test script
that exercises those three failure classes against the runnable microservice
stack.

## Scope

This slice will:

1. add a repo-root `scripts/chaos-test.sh`
2. reuse the existing docker-compose microservice readiness smoke as the setup
   step
3. simulate:
   - RPC failure by stopping `anvil`
   - Kafka failure by stopping `kafka`
   - PostgreSQL failure by stopping `postgres`
4. validate recovery/degradation using existing runtime summary and metrics
   surfaces

## Non-Goals

This slice will not:

1. introduce Kubernetes chaos tooling
2. simulate network partitions at the packet layer
3. claim full production incident rehearsal completion
4. add new runtime recovery code paths

## Selected Approach

Keep the test runnable and repository-local:

1. start the compose microservice stack through the existing readiness smoke
   with `KEEP_STACK_UP=1`
2. perform targeted container stop/start actions with Docker Compose
3. assert existing runtime signals:
   - puller poll errors after RPC failure
   - event-processor degraded/unhealthy runtime posture after Kafka failure
   - api-service degraded/unhealthy query status after PostgreSQL failure
4. assert services recover after dependencies return

## Risks

1. local machines without Docker cannot execute the live chaos test
2. timing-sensitive recovery checks may need generous polling windows
3. dependency restart order may transiently affect unrelated services

## Rollback Plan

1. remove `scripts/chaos-test.sh`
2. remove docs that reference the script
3. keep existing readiness and rehearsal scripts unchanged

## Test Strategy

1. `bash -n scripts/chaos-test.sh`
2. `bash scripts/chaos-test.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-chaos-test-baseline.md`

## Quality Gates

1. `bash -n scripts/chaos-test.sh`
2. `bash scripts/chaos-test.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-chaos-test-baseline.md`

## Review Notes

Approved as the smallest executable chaos-test baseline that aligns with the
current repository-local microservice runtime surfaces.
