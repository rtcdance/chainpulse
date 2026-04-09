## Status
Status: Approved

## Summary

The `docs/` root currently mixes active documentation, planning artifacts,
historical prompt packs, and stale navigation hints in one flat surface. That
creates three problems:

1. root-level doc discovery is noisy
2. `docs/README.md` contains outdated links and an incomplete directory map
3. planning and historical prompt material is not clearly separated from active
   operational documentation

## Decision

- Create `docs/planning/` for current planning/AI workflow documents.
- Create `docs/archive/planning/` for historical prompt-pack documents.
- Move prompt-pack files out of the `docs/` root into the archive planning
  folder.
- Move active planning references (`EXECUTION_PLAN.md`,
  `GPT_PROMPT_TEMPLATE.md`, `AI_LOG.md`) into `docs/planning/`.
- Refresh doc navigation files and architecture rules to reference the new
  locations and remove stale links.

## Acceptance

- `docs/` root is reduced to active top-level reference docs.
- `docs/README.md` reflects the real directory layout and only links to valid
  files.
- `docs/INDEX.md` references the new planning paths.
- `ARCHITECTURE_RULES.md` references the relocated planning docs.
