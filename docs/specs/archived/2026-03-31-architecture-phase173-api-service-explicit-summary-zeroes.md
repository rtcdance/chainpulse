# Phase 173 - API Service Explicit Summary Zeroes

## Status
Status: Approved

## Why
- `api-service` rollout report now exposes truthful runtime-derived posture, but
  its ownership summary fields were still relying on zero-value defaults.
- Those summary fields have ownership-specific names, so we should not invent
  microservice-local substitute meanings for them.

## Scope
- Keep the `/health/rollout` schema unchanged.
- Make `api-service` explicitly populate ownership summary fields with zero
  values until real ownership counters are wired.

## Implementation
- Add a tiny summary helper that returns explicit zero ownership summary values.
- Use it in both the skeleton and runtime-derived producer paths.
- Add route and producer assertions so the zero-summary contract is intentional,
  not accidental.

## Validation
- Run api-service producer tests and route integration coverage.
- Run shared package tests and fast micro-loop gate.

## Exit Criteria
- `api-service` rollout report always emits explicit zero ownership summary
  counts while ownership runtime counters remain unwired.
- No field names or rollout states change.
