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
  "CLAUDE.md"
  "chainpulse"
  "chainpulse-docker"
  # Node.js project files
  "package.json"
  "package-lock.json"
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
# Only check staged files, not the entire repo (pre-existing mocks are allowed)
STAGED_MOCKS=$(git diff --cached --name-only --diff-filter=A 2>/dev/null | grep "mock_.*\.go$" || true)
if [ -n "$STAGED_MOCKS" ]; then
  while IFS= read -r file; do
    if [[ ! "$file" =~ ^pkg/generated/ ]] && [[ ! "$file" =~ ^pkg/plugins/database/mock_ ]] && [[ ! "$file" =~ _test\.go$ ]]; then
      echo "❌ New generated/mock file outside pkg/generated/: $file"
      ((ERRORS++))
    fi
  done <<< "$STAGED_MOCKS"
fi

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
