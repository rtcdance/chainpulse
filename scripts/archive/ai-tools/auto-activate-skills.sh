#!/bin/bash
set -e

echo "🎯 Auto-Activating Skills Based on Changes"
echo "==========================================="

CHANGED_FILES=$(git diff --cached --name-only --diff-filter=AM 2>/dev/null || echo "")
if [ -z "$CHANGED_FILES" ]; then
  echo "⚠️  No staged changes found"
  exit 0
fi

ACTIVE_SKILLS=()

# Always active
ACTIVE_SKILLS+=("design-review-gate")

# File path triggers
while IFS= read -r file; do
  case "$file" in
    pkg/domain/*.go)
      ACTIVE_SKILLS+=("web3-go-architecture-guardrails" "deterministic-testing")
      ;;
    pkg/adapters/*.go)
      ACTIVE_SKILLS+=("adapter-contract-testing" "web3-go-architecture-guardrails")
      ;;
    pkg/services/indexing/*.go)
      ACTIVE_SKILLS+=("web3-reorg-idempotency" "indexer-state-consistency")
      ;;
    pkg/plugins/rpc/*.go)
      ACTIVE_SKILLS+=("chaos-resilience-testing" "rate-limiting-backpressure")
      ;;
    pkg/adapters/contracts/*.go)
      ACTIVE_SKILLS+=("smart-contract-integration-safety" "web3-security-patterns")
      ;;
    *_test.go)
      ACTIVE_SKILLS+=("deterministic-testing")
      ;;
    go.mod)
      ACTIVE_SKILLS+=("dependency-upgrade-governance")
      ;;
  esac
done <<< "$CHANGED_FILES"

# Pattern triggers
DIFF_CONTENT=$(git diff --cached 2>/dev/null || echo "")

if echo "$DIFF_CONTENT" | grep -qE "go func|make\(chan|sync\."; then
  ACTIVE_SKILLS+=("go-concurrency-patterns" "concurrency-safety")
fi

if echo "$DIFF_CONTENT" | grep -qE "crypto\.|keystore|private.*key"; then
  ACTIVE_SKILLS+=("web3-security-patterns" "security-compliance-baseline")
fi

# Remove duplicates
UNIQUE_SKILLS=($(printf '%s\n' "${ACTIVE_SKILLS[@]}" | sort -u))

# Generate active-skills.md
mkdir -p .codex
cat > .codex/active-skills.md <<EOFMD
# Auto-Activated Skills

**Generated**: $(date +%Y-%m-%d\ %H:%M:%S)
**Changed Files**: $(echo "$CHANGED_FILES" | wc -l)

## Active Skills (${#UNIQUE_SKILLS[@]})

EOFMD

for skill in "${UNIQUE_SKILLS[@]}"; do
  echo "- \`$skill\`" >> .codex/active-skills.md
done

echo ""
echo "✅ Activated ${#UNIQUE_SKILLS[@]} skills:"
printf '   - %s\n' "${UNIQUE_SKILLS[@]}"
echo ""
echo "📝 Details: .codex/active-skills.md"
