---
name: "skill-template"
description: "Template for creating new skills. Use this as a reference when creating custom skills for AI behavior constraints."
---

# Skill Template

This is a template for creating new skills that constrain AI behavior.

## Structure

```
.trae/skills/<skill-name>/
└── SKILL.md
```

## SKILL.md Format

```markdown
---
name: "<skill-name>"
description: "<what it does AND when to invoke it. Keep under 200 chars>"
---

# <Skill Title>

## Purpose
<What this skill accomplishes>

## When to Invoke
<Specific triggers or conditions>

## Instructions
<Step-by-step guidelines>

## Examples
<Concrete examples of expected behavior>

## Constraints
<Rules that must be followed>
```

## Required Fields

| Field | Description |
|-------|-------------|
| `name` | Unique identifier (lowercase, hyphens) |
| `description` | Must include: (1) functionality, (2) trigger conditions |
| body | Detailed instructions in markdown |

## Best Practices

1. **Be Specific**: Define exact trigger conditions
2. **Be Actionable**: Provide clear, executable instructions
3. **Be Concise**: Keep descriptions under 200 characters
4. **Include Examples**: Show expected input/output
5. **Define Constraints**: List rules that must not be violated

## Creating a New Skill

1. Create directory: `mkdir -p .trae/skills/<name>`
2. Create SKILL.md with proper frontmatter
3. Include purpose, triggers, instructions, examples, and constraints
4. Test the skill behavior
