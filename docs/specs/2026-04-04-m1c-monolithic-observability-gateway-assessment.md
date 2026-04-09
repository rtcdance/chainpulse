Title: M1c Monolithic Observability Gateway Assessment
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1c` was scoped to the minimum blueprint-aligned monolithic observability and
gateway baseline after `M1a` runtime closure and `M1b` resilience closure.
Three focused slices are now complete:

1. explicit `/metrics` runtime-route contract
2. gateway runtime-route inventory surfacing
3. gateway route method contract hardening

The remaining question is no longer which narrow operator/API slice to add next,
but whether the current monolithic observability + gateway surface has reached a
clean milestone boundary that should be closed before moving to `M2`.

## Scope

This assessment determines whether `M1c` now satisfies the minimum acceptable
milestone boundary for:

1. operator-facing monolithic runtime visibility
2. gateway runtime-route clarity
3. truthful gateway runtime/API hardening

## Non-Goals

This assessment will not:

1. introduce new routes or protocol surfaces
2. expand into deployment-mode switching
3. claim full `ARCHITECTURE_v1.md` parity for observability platforms
4. reopen `M1a` or `M1b`

## Assessment

`M1c` now has the following stage-complete baseline:

1. monolithic gateway exposes `/health*`, `/runtime/summary`, `/runtime/control`,
   and `/metrics` as explicit runtime routes
2. runtime summary now reports gateway route inventory and runtime-surface posture
3. gateway wrong-method requests are rejected with an explicit `405` + `Allow`
   contract instead of silently falling through path-only matching
4. monolithic runtime summary now combines indexing, puller, recovery, reorg,
   gateway, and metrics facts into a compact operator-facing surface

This is strong enough to classify `M1c` as:

`stage-complete for the monolithic observability + gateway baseline`

## Decision

Approved as the closing assessment for `M1c`.

`M1c` should now be treated as complete, and follow-on work should move to `M2`
instead of continuing to add small monolithic gateway/operator slices by default.

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-monolithic-observability-gateway-assessment.md`

## Implementation Notes

This record captures the completion boundary for the `M1c` milestone rather than
introducing additional runtime behavior.

## Verification Summary

The following check passed after implementation:

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-monolithic-observability-gateway-assessment.md`
