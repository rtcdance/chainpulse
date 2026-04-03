# Phase 404 - Execution Control Alignment Refresh

## Status
Status: Approved

## Summary

Refresh the execution-control line after target metadata alignment so the
repository records the stronger post-phase-403 boundary more accurately.

## Problem

Phase 402 correctly closed the current line as stage-complete for the
service-shaped execution-control baseline.

But phase 403 materially strengthened the aligned contract layer:

- target metadata is now aligned across both writable pilots
- shared validation now covers target-aware envelopes for both services

Without a refresh, the repository risks understating the current line by
describing it only as a completed service-shaped baseline without recording how
close the aligned layer now is to a future shared control contract.

## Decision

Record the current execution-control line as:

- **stage-complete for the aligned execution-control baseline**

Interpretation:

- the current line remains intentionally service-shaped at the action and route
  level
- the aligned contract layer is now stronger than before because it includes:
  - envelope fields
  - control-core fields
  - target metadata
  - shared validation
- default work on this line should still stop here unless a new, explicit
  reopen goal is selected

## Scope

In scope:

- refresh execution-control wording in the coverage summary
- record the stronger aligned-layer maturity after phase 403
- define the next deliberate reopen goals from this stronger point

Out of scope:

- renaming routes
- normalizing service-specific control actions
- introducing auth/policy/distributed coordination

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase404-execution-control-alignment-refresh.md`

## Exit Criteria

- The repository explicitly records that the execution-control line is now
  complete for the aligned control baseline, not only for the earlier
  service-shaped baseline wording.
- Future work is framed as a deliberate reopen beyond this aligned baseline.
