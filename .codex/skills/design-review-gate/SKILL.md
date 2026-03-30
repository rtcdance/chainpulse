# Skill: design-review-gate

## Trigger

Use this skill for any feature work, bugfix, refactor, architecture change, or behavior change.

## Hard Gate

No code changes are allowed until a design/spec document exists and is approved.

Required document location:

- `docs/specs/<yyyy-mm-dd>-<short-topic>.md`

Required header fields:

- `Title`
- `Type` (`feature` | `bugfix` | `refactor` | `architecture`)
- `Status` (`Draft` | `In Review` | `Approved` | `Implemented`)
- `Owner`
- `Reviewers`
- `Related Modules`

Coding may start only when `Status: Approved`.

## Must Do

1. Create/update spec document before implementation.
2. Include problem statement, scope, non-goals, options, selected approach, risks, rollback.
3. Include test strategy and quality gate plan.
4. Record review notes and decision.
5. After delivery, update status to `Implemented`.

## Must Not

- No direct coding from chat intent alone.
- No hidden implementation before approval.
- No "minor bugfix exemption" unless explicitly approved in the spec.

## Exit Criteria

- Approved spec exists before code changes.
- Implementation matches approved scope.
- Spec updated with final decision and verification summary.
