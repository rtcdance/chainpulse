#!/bin/bash
set -e

echo "🤖 Initializing AI Coding Session"
echo "=================================="

# 1. Auto-activate skills based on staged changes
if [ -f "scripts/auto-activate-skills.sh" ]; then
  ./scripts/auto-activate-skills.sh
else
  echo "⚠️  auto-activate-skills.sh not found, skipping"
fi

# 2. Check if active skills exist
if [ ! -f ".codex/active-skills.md" ]; then
  echo "⚠️  No active skills found. Run: git add <files> first"
  exit 1
fi

# 3. Build AI context
echo "" > .codex/ai-session-context.md
echo "# AI Coding Session Context" >> .codex/ai-session-context.md
echo "" >> .codex/ai-session-context.md
echo "## Active Skills" >> .codex/ai-session-context.md
cat .codex/active-skills.md >> .codex/ai-session-context.md
echo "" >> .codex/ai-session-context.md

echo "✅ AI context ready: .codex/ai-session-context.md"
echo ""
echo "Next: Copy context to AI or use steering file"
