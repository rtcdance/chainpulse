## Status
Status: Approved

## Summary

The repository root still contains several project guidance documents that are
not source code, build config, or true top-level entrypoints:

- `ARCHITECTURE_RULES.md`
- `DEPENDENCY_APPROVAL.md`
- `RUNNABLE_APP.md`
- `SECURITY_BASELINE.md`
- `SECURITY_ROLLOUT.md`

They make the root noisier than necessary and blur the distinction between
code/config entrypoints and project documentation.

## Decision

- Create `docs/project/` for project-level operational and governance docs that
  are still active references.
- Move the five root guidance docs into that folder.
- Update scripts, README files, and doc links to reference the relocated paths.
- Update file-organization checks so these docs are no longer expected at the
  repository root.

## Acceptance

- The five project guidance docs live under `docs/project/`.
- Root-level tooling and docs reference the new locations.
- Root file-organization checks no longer whitelist the moved files.
- The Docker verification and task-launch helper still work with the new paths.
