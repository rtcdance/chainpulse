# ChainPulse Development Guide

This file is automatically loaded by Claude Code at session start.

## Before Writing Any Code

**MANDATORY**: You MUST follow this sequence:

### 1. Detect Active Skills
```bash
# If files are staged
./scripts/auto-activate-skills.sh

# If no staged files, manually activate based on task
```

### 2. Read Active Skills
```bash
cat .codex/active-skills.md
```

### 3. Load Skill Definitions
For each active skill, read its definition:
```bash
cat .codex/skills/<skill-name>/SKILL.md
```

### 4. Read Constraints
```bash
cat .codex/BEHAVIORAL_CONSTRAINTS.md
cat .codex/PRE_CODING_CHECKLIST.md
```

## Coding Workflow

### Step 1: Declare Skills
Before coding, explicitly state:
```markdown
## Active Skills for This Task

1. design-review-gate
   - Why: [reason]
   - Exit Criteria: [list]

2. web3-reorg-idempotency
   - Why: [reason]
   - Exit Criteria: [list]
```

### Step 2: Code
Follow ALL constraints from:
- Active skill definitions
- BEHAVIORAL_CONSTRAINTS.md
- PRE_CODING_CHECKLIST.md

### Step 3: Verify
Before marking complete, verify:
```markdown
## Exit Criteria Verification

- [x] Skill 1 - Criterion A: [how verified]
- [x] Skill 1 - Criterion B: [how verified]
- [x] Skill 2 - Criterion A: [how verified]
```

## Key Constraints

### Minimal Implementation
- Write ONLY code needed for stated requirement
- No speculative features
- No premature abstractions
- No unsolicited "improvements"

### Test Discipline
- Add tests ONLY when explicitly requested
- Modify tests ONLY when behavior changes

### Dependency Hygiene
- Ask before adding dependencies
- Update DEPENDENCY_APPROVAL.md

## File Organization

```
pkg/
├── domain/          # Pure business logic
├── application/     # Use cases
├── adapters/        # External integrations
├── infrastructure/  # Cross-cutting
└── plugins/         # Swappable implementations
```

## Quick Reference

- **Skills Index**: `.codex/skills/INDEX.md`
- **Active Skills**: `.codex/active-skills.md`
- **Constraints**: `.codex/BEHAVIORAL_CONSTRAINTS.md`
- **Checklist**: `.codex/PRE_CODING_CHECKLIST.md`

## Enforcement

If you violate any skill exit criteria or constraints:
1. Code review will reject
2. CI checks will fail
3. You must fix before merge

**Golden Rule**: Write the MINIMUM code to solve the STATED problem.
