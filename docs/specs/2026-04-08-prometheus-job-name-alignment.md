# Prometheus Job Name Alignment

## Title
Prometheus job name alignment for event-processor readiness smoke

## Type
- bugfix

## Status
- Draft | In Review | Approved | Implemented
Status: Implemented

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-04-08

## Related Modules
- `monitoring/prometheus/prometheus.yml`
- `scripts/verify-prometheus-live-smoke.sh`
- `scripts/verify-prometheus-scrape-baseline.sh`

## Context
The Docker compose microservices readiness flow starts successfully and the full runnable verification passes, but the final Prometheus smoke fails.

## Problem Statement
Prometheus scrapes the event processor using the job name `chainpulse-event-processor`, while the repository smoke scripts still expect `chainpulse-processor`. This leaves the stack healthy but marks acceptance as failed.

## Scope
- Align Prometheus smoke expectations to the configured event-processor job name.
- Keep the current Prometheus scrape config unchanged.
- Re-run readiness and acceptance checks after the script update.
- Unblock repository quality gates when they expose adjacent acceptance blockers during verification.

## Non-Goals
- No changes to service ports, scrape targets, or metrics payloads.
- No changes to alert rules or dashboard behavior.

## Options Considered
- Option A: Rename the Prometheus job back to `chainpulse-processor`.
- Option B: Update verification scripts to expect `chainpulse-event-processor`.

## Selected Approach
Choose Option B because it matches the current Prometheus configuration and service naming already used by the compose stack.

## Data / Contract Impact
No external API contract impact. Internal verification expectations change to match the runtime observability contract already in place.

## Risks
- Other scripts may still reference the old job name.
- Mitigation: search for legacy references and align them in the same change.

## Rollback Plan
Revert the verification script changes if the repository standard is intentionally reverted to `chainpulse-processor`.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-04-08-prometheus-job-name-alignment.md`
- `bash scripts/verify-prometheus-scrape-baseline.sh`
- `bash scripts/verify-prometheus-live-smoke.sh` against the compose stack
- `bash scripts/verify-docker-compose-microservices-readiness.sh`
- `npm test`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `scripts/dev-micro-loop.sh --mode full --base HEAD`

## Quality Gates
- Spec approval check passes.
- Prometheus scrape baseline passes.
- Compose readiness passes end-to-end.
- Playwright acceptance remains green.

## Review Notes
- Approved as a bugfix to restore observability acceptance parity with the active compose configuration.

## Implementation Summary
- Updated Prometheus smoke scripts to expect `chainpulse-event-processor`, matching the active scrape configuration.
- Re-ran repository readiness and acceptance checks to confirm the observability baseline is green again.
- Fixed small cache lint issues surfaced by the repository quality gates.
- Updated migration manifest `spec_ref` entries that still pointed at pre-archive spec paths.

## Final Verification
- `bash scripts/spec-approval-check.sh docs/specs/2026-04-08-prometheus-job-name-alignment.md`
- `bash scripts/verify-prometheus-scrape-baseline.sh`
- `bash scripts/verify-docker-compose-microservices-readiness.sh`
- `npm test`
- `scripts/dev-micro-loop.sh --mode full --base HEAD`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
