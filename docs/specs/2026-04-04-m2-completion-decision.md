Title: M2 Completion Decision
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/application/bootstrap, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2` already established a minimum truthful dual-mode baseline and then added a
transport-boundary posture layer driven by real gateway bridge facts.

The remaining question is whether `M2` still needs another code slice, or
whether the current repository state is already sufficient to mark the dual-mode
baseline complete for the scope of the v1 milestone plan.

## Scope

This decision will:

1. reassess the current `M2` boundary after transport-boundary posture landed
2. decide whether `M2` should remain in progress or be marked complete
3. define the handoff boundary from `M2` to `M3a`

## Assessment

`M2` should now be marked **completed**.

Reasoning:

1. `DEPLOYMENT_MODE` is a real cmd-layer contract rather than a documentation
   label.
2. The selected mode now drives real monolithic wiring choices for:
   - indexing storage
   - query surface
   - gateway surface
3. The selected transport boundary is no longer just a static string. It is
   now surfaced with truthful posture and hint values derived from real gateway
   bridge facts.
4. The remaining gaps are no longer about whether dual-mode switching exists as
   a truthful baseline. They are about broader microservice deployment
   verification and production hardening, which belong to `M3`.

So the more accurate statement is:

- `M2 = completed`
- current posture: `minimum truthful dual-mode baseline completed`

## Decision

Mark `M2` complete.

Do not keep `M2` open for additional summary-only or small wiring slices.

Shift the active milestone to:

- `M3a = in_progress`

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-completion-decision.md`

## Verification Summary

The following check passed after implementation:

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-completion-decision.md`
