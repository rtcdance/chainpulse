Title: M2 Dual-Mode Baseline Assessment
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/application/bootstrap, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2` has now accumulated five real slices:

1. deployment-mode parsing
2. adapter profile selection
3. indexing storage selection
4. query surface selection
5. gateway surface selection

At this point the main question is no longer whether `DEPLOYMENT_MODE` exists as
a label. The real question is whether the repository has already crossed into a
minimum truthful dual-mode baseline, or whether one more deeper cutover slice is
still required before `M2` can be closed.

## Scope

This assessment will:

1. summarize what is now truly mode-aware in the monolithic cmd path
2. identify the remaining highest-value gap in `M2`
3. decide whether `M2` should be marked complete or remain in progress

## Assessment

The repository now has a **truthful dual-mode baseline seam**, because
`DEPLOYMENT_MODE` affects real runtime choices in the monolithic cmd path:

- deployment posture
- adapter profile
- indexing storage adapter
- query surface
- gateway exposure surface

This is materially stronger than the earlier state where mode selection existed
only as startup metadata.

However, `M2` should **remain in progress**.

The main reason is that the current mode split is still concentrated inside the
monolithic cmd path. The repository does not yet have a stronger cross-entrypoint
shared wiring story that proves:

- the microservice mode is selected through the same canonical seam
- transport/runtime boundaries are aligned beyond operator/runtime-only posture
- dual-mode execution can be treated as stage-complete rather than intentionally
  partial

## Decision

Current state:

- `M2 = in_progress`
- posture: `minimum truthful dual-mode baseline established`

Do **not** mark `M2` complete yet.

The highest-value next slice is to keep pushing on **shared wiring / transport
boundary alignment**, not to add more posture-only fields.

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-dual-mode-baseline-assessment.md`

## Verification Summary

The following check passed after implementation:

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-dual-mode-baseline-assessment.md`
