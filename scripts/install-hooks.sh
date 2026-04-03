#!/bin/bash
set -e

echo "📦 Installing Git Hooks"
echo "======================"

HOOKS_DIR=".git/hooks"

if [ ! -d "$HOOKS_DIR" ]; then
  echo "❌ Not a git repository"
  exit 1
fi

# Create pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'HOOK'
#!/bin/bash
set -e

echo "🔍 Running Pre-Commit Checks"
echo "============================="

# 1. Auto-activate skills
if [ -f "scripts/auto-activate-skills.sh" ]; then
  ./scripts/auto-activate-skills.sh
fi

# 2. Run pre-coding checklist
if [ -f "scripts/pre-coding-checklist.sh" ]; then
  ./scripts/pre-coding-checklist.sh
fi

echo ""
echo "✅ Pre-commit checks passed"
HOOK

chmod +x "$HOOKS_DIR/pre-commit"

echo "✅ Installed: pre-commit hook"
echo ""
echo "Usage:"
echo "  git add <files>"
echo "  git commit -m 'message'"
echo "  → Auto-activates skills + runs checklist"
