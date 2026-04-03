Title: Phase 423 API Gateway Security Surface
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, pkg/plugins/api/auth_middleware.go, pkg/plugins/api/rate_limiter.go, README.md, RUNNABLE_APP.md

## Status

Status: Approved

## Problem Statement

The runnable app baseline is complete, but the API gateway still lacks a clear
security surface boundary in its runtime model. The architecture blueprint
expects gateway-level authentication and rate limiting, and the repository
already has reusable components. We need the smallest safe step that makes the
gateway's security posture explicit without breaking the current runnable app.

## Scope

- wire optional gateway authentication and rate limiting through the existing
  API gateway entrypoint
- keep the current runnable app behavior unchanged by default
- surface the gateway security posture in `/runtime/summary`
- document the local configuration knobs for enabling the security surface

## Non-Goals

- no mandatory auth rollout for the current runnable slice
- no new identity provider integration
- no protocol-wide auth redesign
- no deployment platform changes

## Selected Approach

Add optional gateway security controls that remain disabled by default:

- authentication middleware
- rate limiting middleware

Wire them into the gateway request chain only when explicitly configured. Expose
the following in the gateway runtime summary:

- `auth_posture`
- `rate_limit_posture`
- `security_posture`
- `security_hint`

## Risks

- overcomplicating the current runnable path if the defaults are not clearly off
- confusing local developers if the new env vars are not documented clearly

## Rollback

- disable the new gateway security wiring
- remove the new runtime summary fields
- remove the root documentation references

## Test Strategy

- add focused gateway tests for the optional security wrappers
- keep the existing runnable smoke path passing with the default disabled mode
- run spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase423-api-gateway-security-surface.md`

## Decision

- Introduce an optional, default-off gateway security surface so the repository
  can show the blueprint-aligned auth/rate-limit direction without regressing
  the current runnable app.
