---
name: "micro-loop-delivery"
description: "Execute spec-first, test-first micro-cycles with mandatory quality gates. Invoke for all code changes."
---

# Skill: micro-loop-delivery

## Trigger

Use this skill for all code changes.

## Must Do

1. Start from a small scoped spec (context, scope, acceptance, risks, verification).
2. Implement in small increments.
3. Add/update unit tests in each increment.
4. Run fast gate every loop:
   - `scripts/dev-micro-loop.sh --mode fast`
5. Run full gate before merge:
   - `scripts/dev-micro-loop.sh --mode full`
6. Report exactly what checks ran and what results were obtained.

## ChainPulse Pointers

- Workflow contract: `docs/guides/ENGINEERING_CONSTRAINT_FRAMEWORK.md`
- Unit test standards: `docs/guides/UNIT_TEST_STANDARDS.md`
- Error handling patterns: `docs/guides/ERROR_HANDLING_GUIDE.md`
- Script: `scripts/dev-micro-loop.sh`

## Must Not

- No "big bang" change without intermediate checks.
- No skipping tests/static analysis silently.
- No completion claim when quality gates are not run.

## Exit Criteria

- Unit tests added or updated for changed behavior.
- Fast/full gates pass, or failures are explicitly documented with next action.
- Acceptance criteria are met and verified.
