#!/bin/bash
set -e

# Check file organization rules
# Usage: scripts/check-file-organization.sh

ERRORS=0

# Whitelist for root directory
ROOT_WHITELIST=(
  "README.md"
  "LICENSE"
  "Makefile"
  "go.mod"
  "go.sum"
  ".gitignore"
  ".golangci.yml"
  "CODE_OF_CONDUCT.md"
  # Project docs and config
  "ARCHITECTURE_RULES.md"
  "CLAUDE.md"
  "RUNNABLE_APP.md"
  "SECURITY_BASELINE.md"
  "SECURITY_ROLLOUT.md"
  "DEPENDENCY_APPROVAL.md"
  "chainpulse"
)

echo "Checking file organization..."

# Check for files at root (excluding whitelist)
while IFS= read -r file; do
  filename=$(basename "$file")
  if [[ ! " ${ROOT_WHITELIST[@]} " =~ " ${filename} " ]]; then
    echo "❌ File at root not in whitelist: $file"
    ((ERRORS++))
  fi
done < <(find . -maxdepth 1 -type f ! -name ".*")

# Check for utils/ or helpers/ in pkg/
if [ -d "pkg/utils" ] || [ -d "pkg/helpers" ] || [ -d "pkg/common" ]; then
  echo "❌ Found dumping ground directory: pkg/utils, pkg/helpers, or pkg/common"
  ((ERRORS++))
fi

# Check for generated code outside pkg/generated/
while IFS= read -r file; do
  if [[ ! "$file" =~ ^pkg/generated/ ]] && [[ ! "$file" =~ _test\.go$ ]]; then
    echo "❌ Generated/mock file outside pkg/generated/: $file"
    ((ERRORS++))
  fi
done < <(find pkg -type f -name "mock_*.go" -o -name "*_generated.go" 2>/dev/null || true)

# Check for temp files
while IFS= read -r file; do
  echo "❌ Temporary file found: $file"
  ((ERRORS++))
done < <(find . -type f \( -name "*.tmp" -o -name "*.bak" -o -name "*_backup.*" \) 2>/dev/null || true)

if [ $ERRORS -eq 0 ]; then
  echo "✅ File organization check passed"
  exit 0
else
  echo "❌ Found $ERRORS organization issues"
  exit 1
fi
