## Status
Status: Approved

## Summary

The go-live blockers require a sustained soak test on real chain RPCs, but the
repository does not provide a standard entrypoint for that verification. As a
result, soak evidence is left as an ad hoc manual activity.

## Decision

- Add `scripts/soak-check.sh` as the repository soak-check entrypoint.
- Keep the script explicit about scope: it does not generate synthetic load, but
  it repeatedly samples the live deployment over a configured duration and
  fails if health or runtime contract checks regress during the soak window.
- Wire the soak entrypoint into the production checklist, blocker list, and
  deployment-readiness static gate.

## Acceptance

- `scripts/soak-check.sh` exists and supports `--help`.
- The script fails closed when `DURATION_SECONDS` or `INTERVAL_SECONDS` is
  invalid.
- The script can optionally run `scripts/verify-production.sh` before and after
  the soak window when `RUN_PRECHECK=1` and/or `RUN_POSTCHECK=1` are set.
- Deployment docs reference the soak entrypoint as the standard evidence path
  for the sustained soak blocker.
