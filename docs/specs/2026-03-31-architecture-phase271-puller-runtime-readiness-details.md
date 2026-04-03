# Phase 271 - Puller Runtime Readiness Details

## Status
Status: Approved

## Summary

Strengthen the `puller` execution service plane by wiring rollout-aware
readiness details and runtime component details into the minimal HTTP health
surface.

## Problem

`puller` already exposes a minimal runtime HTTP health surface, but the surface
was still shallow in one important way:

- `/health/ready` did not clearly surface rollout-aware readiness details
- `/health/components` did not carry a runtime component view derived from the
  same rollout/runtime state

That left the service with HTTP exposure, but without a strong bridge between:

- execution rollout posture
- checkpoint/poll runtime state
- readiness/component details

## Decision

Add rollout-aware runtime readiness/component helpers in
`cmd/microservices/puller` and wire them into the existing health handler path.

The new runtime surface should:

1. derive readiness details from the current runtime rollout state
2. derive a runtime component status from the same runtime rollout state
3. expose rollout gate facts through:
   - `/health/ready`
   - `/health/components`

## Scope

In scope:

- puller runtime readiness detail builder
- puller runtime component status builder
- focused tests for helper logic
- focused HTTP route coverage for `/health/ready` and `/health/components`
- architecture summary refresh

Out of scope:

- deeper rollout contract changes
- broader service-plane expansion outside the existing health surface
- new checkpoint sources beyond the current runtime state model

## Result

`puller` now exposes:

- rollout-aware readiness details
- rollout-aware component details for the polling runtime
- a clearer execution-service-plane bridge between runtime rollout posture and
  minimal HTTP health routes

This keeps `puller` aligned with the stronger execution-service-plane baseline
already established for `event-processor`.
