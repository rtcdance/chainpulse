## Status
Status: Approved

## Summary

The go-live blockers require data consistency, replay, and reorg recovery
validation, but the repository has no standard release-gate entrypoint for that
drill. Existing runtime surfaces expose checkpoint and reorg posture from the
puller, yet operators must currently interpret those signals ad hoc.

## Decision

- Add `scripts/data-consistency-drill.sh` as the repository consistency drill
  entrypoint.
- Keep the script explicit about scope: it does not fabricate blockchain reorgs
  or perform replay automatically, but it validates the live puller runtime
  checkpoint and reorg recovery contract before and after an operator-driven
  drill window.
- Wire the entrypoint into the production checklist, blocker list, and
  deployment-readiness static checks.

## Acceptance

- `scripts/data-consistency-drill.sh` exists and supports `--help`.
- The script fails closed when `TARGET_CHAIN` is missing.
- The script validates puller runtime summary fields for deployment mode,
  runtime mode, checkpoint coverage, and chain summary visibility.
- The script can optionally run `scripts/verify-production.sh` before and after
  the drill window when `RUN_PRECHECK=1` and/or `RUN_POSTCHECK=1` are set.
- Deployment docs reference the consistency drill entrypoint as the standard
  evidence path for replay / reorg recovery validation.
