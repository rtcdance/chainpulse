# AI Coding Session Initialization

**Purpose**: Automatically inject active skills into AI context before coding.

## Mechanism

When AI starts coding, inject this prompt:

```markdown
# Active Skills for This Session

You MUST follow these skills for the current task:

{AUTO_GENERATED_SKILLS_LIST}

## Mandatory Checks Before Coding

1. Read all active skill files from `.codex/skills/`
2. Declare which skills apply and why
3. List exit criteria that must be met
4. Only then start coding

## After Coding

Verify ALL exit criteria from active skills are met.
If any criterion fails, fix before marking task complete.
```

## Implementation Options

### Option 1: System Prompt Injection (Recommended)

Add to AI system prompt:
```
Before writing any code, you MUST:

1. Run: `./scripts/auto-activate-skills.sh` to detect active skills
2. Read: `.codex/active-skills.md` for the list
3. Read: Each skill file in `.codex/skills/<skill-name>/SKILL.md`
4. Declare: Which skills apply and their exit criteria
5. Code: Following all skill constraints
6. Verify: All exit criteria met before completion

Active skills are in: `.codex/active-skills.md`
Skill definitions are in: `.codex/skills/*/SKILL.md`
Behavioral constraints are in: `.codex/BEHAVIORAL_CONSTRAINTS.md`
```

### Option 2: Context File (For Claude/Cursor/Copilot)

Create `.claude/context.md`:
```markdown
# Coding Context

## Active Skills
{{file:.codex/active-skills.md}}

## Behavioral Constraints
{{file:.codex/BEHAVIORAL_CONSTRAINTS.md}}

## Pre-Coding Checklist
{{file:.codex/PRE_CODING_CHECKLIST.md}}
```

### Option 3: Prompt Template

Create `.codex/ai-session-prompt.md`:
```markdown
# AI Coding Session

## Task
[User describes task here]

## Auto-Detected Active Skills
[Run: ./scripts/auto-activate-skills.sh]
[Content of: .codex/active-skills.md]

## Skill Details
[For each active skill, include content of .codex/skills/<skill>/SKILL.md]

## Your Response Format

### 1. Skill Declaration
List which skills apply and why:
- skill-name: [reason]

### 2. Exit Criteria Checklist
- [ ] Criterion 1
- [ ] Criterion 2

### 3. Implementation
[Your code here]

### 4. Verification
Confirm all exit criteria met:
- [x] Criterion 1 - [how verified]
- [x] Criterion 2 - [how verified]
```

## Auto-Injection Script

```bash
#!/bin/bash
# scripts/ai-session-init.sh

echo "🤖 Initializing AI Coding Session"
echo "=================================="

# 1. Auto-activate skills
./scripts/auto-activate-skills.sh

# 2. Build AI prompt
cat > .codex/ai-session-context.md <<EOF
# AI Coding Session Context

## Active Skills
$(cat .codex/active-skills.md)

## Skill Definitions

EOF

# 3. Append each skill definition
SKILLS=$(grep -oP '(?<=`).*?(?=`)' .codex/active-skills.md)
for skill in $SKILLS; do
  if [ -f ".codex/skills/$skill/SKILL.md" ]; then
    echo "### $skill" >> .codex/ai-session-context.md
    echo "" >> .codex/ai-session-context.md
    cat ".codex/skills/$skill/SKILL.md" >> .codex/ai-session-context.md
    echo "" >> .codex/ai-session-context.md
  fi
done

# 4. Append constraints
cat >> .codex/ai-session-context.md <<EOF

## Behavioral Constraints
$(cat .codex/BEHAVIORAL_CONSTRAINTS.md)

## Pre-Coding Checklist
$(cat .codex/PRE_CODING_CHECKLIST.md)

---

**Instructions for AI:**

1. Read all active skills above
2. Declare which apply to current task
3. List exit criteria
4. Code following all constraints
5. Verify all criteria met
EOF

echo "✅ AI context ready: .codex/ai-session-context.md"
echo ""
echo "📋 Copy this file content to AI chat to initialize session"
```
