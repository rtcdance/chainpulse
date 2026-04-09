Title: Runtime Summary Deployment Mode Alignment
Type: bugfix
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/event-processor, cmd/microservices/puller, frontend

## Status

Approved for implementation.

## Problem Statement

The H5 acceptance console reads `deployment_mode` from `/runtime/summary` to
render the Deployment Mode card.

Observed fact:

1. the frontend already reads `deployment_mode` and falls back to `unknown`
2. the current microservice `/runtime/summary` payloads expose `runtime_mode`
   but omit `deployment_mode`
3. the H5 therefore shows `Deployment Mode = unknown` even when the live stack
   is clearly running in microservice mode

## Scope

This bugfix will:

1. add `deployment_mode` to microservice runtime summary payloads
2. keep the value stable as `microservice` for these runtime endpoints
3. add focused test coverage for the updated response contract

## Non-Goals

This bugfix will not:

1. redesign runtime summary payload structure
2. change health or rollout report payloads
3. alter monolithic runtime behavior

## Selected Approach

1. add a `DeploymentMode` field to each microservice runtime summary response
2. populate the field with the literal `microservice`
3. update existing runtime summary tests to assert the field is present

## Risks

1. low risk because the change is additive and aligned with existing rollout
   report semantics

## Rollback Plan

1. revert the added runtime summary field and the focused assertions

## Test Strategy

1. `bash scripts/spec-approval-check.sh docs/specs/2026-04-09-runtime-summary-deployment-mode-alignment.md`
2. focused Go tests for the touched runtime summary handlers/builders
3. live probe against `http://localhost:8080/runtime/summary`

