Title: Phase 424 API Service Security Surface
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-service, pkg/plugins/api/auth_middleware.go, pkg/plugins/api/rate_limiter.go, cmd/microservices/api-service/QUICKSTART.md, cmd/microservices/api-service/README.md

## Status

Status: Implemented

## Problem Statement

The API gateway now has an optional, default-off security surface that makes its
auth and rate limiting posture explicit without disturbing the runnable
baseline. The API service still exposes the same public runtime and query
surface, but it does not yet advertise or wire a comparable optional security
boundary. To keep the blueprinted app coherent and avoid growing a one-sided
entrypoint story, the API service should expose the same kind of opt-in security
surface with the same default-off behavior.

## Scope

- wire optional API service authentication and rate limiting through the
  existing microservice bootstrap
- keep the current runnable app behavior unchanged by default
- surface the API service security posture in `/runtime/summary`
- document the local configuration knobs for enabling the security surface

## Non-Goals

- no mandatory auth rollout for the current runnable slice
- no new identity provider integration
- no protocol-wide auth redesign
- no deployment platform changes

## Selected Approach

Add optional API service security controls that remain disabled by default:

- authentication middleware
- rate limiting middleware

Wire them into the API service request chain only when explicitly configured.
Expose the following in the API service runtime summary:

- `auth_posture`
- `rate_limit_posture`
- `security_posture`
- `security_hint`

## Risks

- overcomplicating the current runnable path if the defaults are not clearly off
- confusing local developers if the new env vars are not documented clearly

## Rollback

- disable the new API service security wiring
- remove the new runtime summary fields
- remove the root documentation references

## Test Strategy

- add focused API service tests for the optional security wrappers
- keep the existing runnable smoke path passing with the default disabled mode
- run spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase424-api-service-security-surface.md`

## Decision

- Introduce an optional, default-off API service security surface so the
  repository can show the blueprint-aligned auth/rate-limit direction without
  regressing the current runnable app.
