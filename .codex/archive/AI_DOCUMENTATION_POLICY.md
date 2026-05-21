# AI-Generated Documentation Policy

**Last Updated**: 2026-03-30

## Active AI Documentation

Only `.codex/` directory contains active AI guidance:
- `.codex/skills/` - Engineering skills (22 skills)
- `.codex/skills/INDEX.md` - Skills catalog

## Archived AI Documentation

Other AI-generated docs moved to `.ai-archive/`:
- `.kiro/` → `.ai-archive/kiro/` (1.1MB, 88 files)
- `.trae/` → `.ai-archive/trae/` (104KB)

## Rationale

**Problem**: Multiple AI systems' historical docs create conflicting guidance and context pollution.

**Solution**: Single source of truth (`.codex/`) for AI-assisted development.

## .gitignore Update

```gitignore
# AI archives (historical reference only)
.ai-archive/
.kiro/
.trae/
```

## For Future AI Sessions

- **Use**: `.codex/skills/` for engineering patterns
- **Ignore**: `.ai-archive/`, `.kiro/`, `.trae/`
- **Update**: Only `.codex/` when adding new patterns
