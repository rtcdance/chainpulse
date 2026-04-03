# Phase 270 - Event-Processor Runtime Readiness Details

## Status
Status: Approved

## Summary

Strengthen the `event-processor` execution service plane by wiring rollout-aware
readiness details and runtime component details into the minimal HTTP health
surface.

## Problem

`event-processor` already exposes a minimal runtime HTTP health surface, but the
surface was still shallow in one important way:

- `/health/ready` did not clearly surface rollout-aware readiness details
- `/health/components` did not carry an execution-runtime component view derived
  from the same rollout/runtime state

That left the service with HTTP exposure, but without a strong bridge between:

- execution rollout posture
- readiness details
- component-level runtime status

## Decision

Add rollout-aware runtime readiness/component helpers in
`cmd/microservices/event-processor` and wire them into the existing health
handler path.

The new runtime surface should:

1. derive readiness details from the current runtime rollout state
2. derive a runtime component status from the same runtime rollout state
3. expose rollout gate facts through:
   - `/health/ready`
   - `/health/components`

## Scope

In scope:

- event-processor runtime readiness detail builder
- event-processor runtime component status builder
- focused tests for helper logic
- focused HTTP route coverage for `/health/ready` and `/health/components`
- architecture summary refresh

Out of scope:

- deeper rollout contract changes
- new execution-runtime signals beyond the current state model
- broader service-plane expansion outside the existing health surface

## Result

`event-processor` now exposes:

- rollout-aware readiness details
- rollout-aware component details for the indexing runtime
- a clearer execution-service-plane bridge between runtime rollout posture and
  minimal HTTP health routes

This moves the service plane forward without reopening the paused
rollout/control foreground line.
