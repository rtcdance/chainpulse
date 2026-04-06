Title: M2 Completion Record
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/application/bootstrap, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Summary

`M2` is complete.

This milestone delivered the minimum truthful dual-mode baseline required by the
current v1 milestone plan.

## Completed Scope

`M2` now includes:

1. deployment-mode parsing
2. adapter profile selection
3. deployment-mode-aware indexing storage selection
4. deployment-mode-aware query surface selection
5. deployment-mode-aware gateway surface selection
6. truthful transport-boundary posture and hint surfacing

## Resulting Boundary

After `M2`, the repository now has:

- a real `DEPLOYMENT_MODE` contract
- real mode-aware monolithic wiring choices
- a truthful transport-boundary summary instead of a mode label alone

What `M2` does **not** claim:

- full microservice deployment verification
- full production orchestration
- complete observability and alerting

Those move to `M3`.

## Handoff

The active milestone now becomes:

- `M3a`

`M3a` should focus on:

1. microservice deployment verification
2. cross-entrypoint runnable validation
3. proving the microservice-side baseline is runnable in practice

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-completion-record.md`

## Verification Summary

The following check passed after implementation:

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-completion-record.md`
