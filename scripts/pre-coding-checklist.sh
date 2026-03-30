#!/bin/bash
set -e

echo "🔍 Pre-Coding Checklist Validation"
echo "===================================="

ERRORS=0

# 1. Check spec approval
echo "1. Checking spec approval..."
if ! find docs/specs -name "*.md" -type f -exec grep -l "Status: Approved" {} + 2>/dev/null | grep -q .; then
  echo "   ❌ No approved spec found in docs/specs/"
  ((ERRORS++))
else
  echo "   ✅ Approved spec found"
fi

# 2. Check skill declaration
echo "2. Checking skill declaration..."
if [ ! -f ".codex/active-skills.md" ]; then
  echo "   ⚠️  No active skills declared (create .codex/active-skills.md)"
  # Not blocking for now
else
  echo "   ✅ Active skills declared"
fi

# 3. Check dependency changes
echo "3. Checking dependency approval..."
if git diff --cached --name-only 2>/dev/null | grep -q "go.mod"; then
  if [ ! -f "DEPENDENCY_APPROVAL.md" ] || ! grep -q "Approved by:" DEPENDENCY_APPROVAL.md; then
    echo "   ❌ go.mod changed but no approval in DEPENDENCY_APPROVAL.md"
    ((ERRORS++))
  else
    echo "   ✅ Dependency approval found"
  fi
else
  echo "   ✅ No dependency changes"
fi

# 4. Check for test strategy (if new feature)
echo "4. Checking test strategy..."
NEW_GO_FILES=$(git diff --cached --name-only --diff-filter=A 2>/dev/null | grep "\.go$" | grep -v "_test\.go$" || true)
if [ -n "$NEW_GO_FILES" ]; then
  TEST_FILES=$(git diff --cached --name-only --diff-filter=A 2>/dev/null | grep "_test\.go$" || true)
  if [ -z "$TEST_FILES" ]; then
    echo "   ⚠️  New Go files added without tests (acceptable if trivial)"
  else
    echo "   ✅ Test files included"
  fi
else
  echo "   ✅ No new Go files"
fi

# 5. Check file organization
echo "5. Checking file organization..."
if ! scripts/check-file-organization.sh 2>/dev/null; then
  echo "   ❌ File organization check failed"
  ((ERRORS++))
else
  echo "   ✅ File organization valid"
fi

echo ""
if [ $ERRORS -eq 0 ]; then
  echo "✅ Pre-coding checklist passed"
  exit 0
else
  echo "❌ Pre-coding checklist failed with $ERRORS errors"
  echo ""
  echo "Required actions:"
  echo "  1. Create approved spec in docs/specs/"
  echo "  2. Update DEPENDENCY_APPROVAL.md if adding dependencies"
  echo "  3. Fix file organization issues"
  exit 1
fi
