## Status
Status: Approved

## Summary

The production deployment checklist references `scripts/production-verification-suite.sh`
and `scripts/deployment-readiness-check.sh`, but those entrypoints do not yet exist.
That leaves the checklist partially non-executable.

## Decision

- Add `scripts/production-verification-suite.sh` as the repo-root production baseline verification entrypoint.
- Add `scripts/deployment-readiness-check.sh` as a static release-readiness gate for required docs and scripts.
- Keep both scripts lightweight and compositional by reusing existing verification commands.

## Acceptance

- Both scripts exist and provide `--help`.
- `production-verification-suite.sh` reuses the current production rehearsal baseline.
- `deployment-readiness-check.sh` fails when required go-live artifacts are missing.
