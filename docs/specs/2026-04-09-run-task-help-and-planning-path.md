## Status
Status: Approved

## Summary

After moving milestone prompt files into `docs/archive/planning/`,
`scripts/run-task.sh --help` now falls through into a missing-prompt error
because the script never had explicit help handling.

## Decision

- Add `-h` / `--help` handling to `scripts/run-task.sh`.
- Update the usage text to mention that prompt packs now live under
  `docs/archive/planning/`.

## Acceptance

- `bash scripts/run-task.sh --help` exits successfully.
- The task runner still resolves milestone prompts from `docs/archive/planning/`.
