Title: Phase 425 Execution Service Security Surface
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/puller, cmd/microservices/event-processor, pkg/plugins/api/auth_middleware.go, pkg/plugins/api/rate_limiter.go, cmd/microservices/puller/QUICKSTART.md, cmd/microservices/event-processor/QUICKSTART.md

## Status

Status: Implemented

## Problem Statement

The API gateway and API service now both expose optional, default-off security
surfaces for their public entrypoints. The execution services still expose
runtime, metrics, and in some cases control endpoints, but those entrypoints do
not yet advertise or wire a comparable optional security boundary. To keep the
blueprinted app coherent and avoid leaving the execution-control slices as the
only unconstrained public surfaces, the execution services should expose the
same opt-in security surface with the same default-off behavior.

## Scope

- wire optional authentication and rate limiting through the puller and
  event-processor bootstrap paths
- keep the current runnable app behavior unchanged by default
- surface the execution-service security posture in `/runtime/summary`
- document the local configuration knobs for enabling the security surface

## Non-Goals

- no mandatory auth rollout for the current runnable slice
- no new identity provider integration
- no protocol-wide auth redesign
- no deployment platform changes

## Selected Approach

Add optional execution-service security controls that remain disabled by default:

- authentication middleware
- rate limiting middleware

Wire them into each service request chain only when explicitly configured.
Expose the following in each runtime summary:

- `auth_posture`
- `rate_limit_posture`
- `security_posture`
- `security_hint`

## Risks

- overcomplicating the current runnable path if the defaults are not clearly off
- confusing local developers if the new env vars are not documented clearly

## Rollback

- disable the new execution-service security wiring
- remove the new runtime summary fields
- remove the root documentation references

## Test Strategy

- add focused puller and event-processor tests for the optional security wrappers
- keep the existing runnable smoke paths passing with the default disabled mode
- run spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase425-execution-service-security-surface.md`

## Decision

- Introduce an optional, default-off execution-service security surface so the
  repository can show the blueprint-aligned auth/rate-limit direction without
  regressing the current runnable app.
