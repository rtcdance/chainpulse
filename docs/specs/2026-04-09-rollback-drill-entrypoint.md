## Status
Status: Approved

## Summary

The repository now has production verification and deployment-readiness
entrypoints, but the documented rollback requirement is still only a manual
checklist section. There is no repo-root rollback drill command that operators
can invoke as part of a release gate.

## Decision

- Add `scripts/rollback-drill.sh` as the repository rollback-drill entrypoint.
- Keep the script explicit about scope: it validates rollback prerequisites,
  records the current and previous release identifiers, optionally runs live
  pre/post verification, and prints the required manual execution sequence.
- Wire the new entrypoint into the production checklist and go-live blockers so
  rollback evidence is anchored to a concrete command.

## Acceptance

- `scripts/rollback-drill.sh` exists and supports `--help`.
- The script fails closed when `CURRENT_RELEASE` or `PREVIOUS_RELEASE` is
  missing.
- The script can optionally run `scripts/verify-production.sh` before and after
  the rollback drill when `RUN_PRECHECK=1` and/or `RUN_POSTCHECK=1` are set.
- Production deployment docs reference the rollback drill entrypoint instead of
  only free-form example commands.
