## Status
Status: Approved

## Summary

The new Docker one-click acceptance flow surfaces two small but recurring
maintenance issues:

1. Docker Compose emits an obsolete `version` warning on every run.
2. Local cache and generated binary ignore rules are too loose:
   - `frontend/.vite/` shows up as noise in `git status`
   - the unanchored `chainpulse` ignore pattern can accidentally match nested
     source paths such as `cmd/monolithic/chainpulse`

## Decision

- Remove the obsolete top-level `version` field from repository compose files.
- Add `frontend/.vite/` to `.gitignore`.
- Anchor the generated binary ignore entries to the repository root so they only
  match the intended top-level artifacts.

## Acceptance

- Compose commands no longer emit the obsolete `version` warning.
- `frontend/.vite/` is ignored by Git.
- Root generated artifacts remain ignored.
- Nested source paths under `cmd/monolithic/chainpulse` are no longer
  unintentionally matched by the top-level `chainpulse` ignore rule.
