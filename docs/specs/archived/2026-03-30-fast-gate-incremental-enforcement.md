# Fast Gate Incremental Enforcement

## Title
Align fast micro-loop quality gates with incremental development under legacy debt

## Type
- architecture

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Owner
ChainPulse Engineering

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `scripts/dev-micro-loop.sh`

## Context
Current `fast` micro-loop runs full-repository format/lint/vet/staticcheck, while unit tests are scoped to changed packages. This causes every incremental architecture step to be blocked by historical formatting and static-analysis debt unrelated to current changes.

## Problem Statement
Architecture migration cannot proceed in small safe cycles if fast gates fail on unrelated legacy issues.

## Scope
- Keep strict quality gates in `fast` mode, but scope them to changed files/packages.
- Preserve `full` mode as repository-wide validation.

## Non-Goals
- No removal of format/lint/vet/staticcheck checks.
- No broad formatting or refactor sweep in this change.

## Options Considered
- Option A: Keep current full-repo checks in fast mode.
- Option B: Scope fast checks incrementally; keep full mode unchanged.

## Selected Approach
Adopt Option B:
- `fast` mode:
  - format check on changed `.go` files only
  - lint on new issues only (`--new`), and gracefully skip when tool cannot derive analyzable change set
  - vet on changed packages
  - staticcheck deferred to full mode
  - unit tests on changed packages (`-short`, no race)
- `full` mode:
  - continue strict repository verification steps (including race checks)

## Data / Contract Impact
None.

## Risks
- Some legacy issues outside changed scope may remain undetected in fast loop.
- Mitigation: retain `full` mode for pre-merge or milestone validation.

## Rollback Plan
Revert `scripts/dev-micro-loop.sh` to previous full-repo fast behavior.

## Test and Verification Plan
- Run `scripts/dev-micro-loop.sh --mode fast --base HEAD` on local changes.
- Validate that unrelated legacy formatting debt no longer blocks fast loop.
- Run `scripts/dev-micro-loop.sh --mode full --base HEAD` before integration milestones.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-fast-gate-incremental-enforcement.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved for iterative architecture migration under enterprise legacy constraints.

## Implementation Summary
- Added incremental gate behavior for fast mode while retaining full-mode strictness.

## Final Verification
- Fast loop executes against changed scope and remains strict for modified code paths.
