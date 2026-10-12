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
  "go.work"
  "go.work.sum"
  "CHANGELOG.md"
  ".gitignore"
  ".golangci.yml"
  "CODE_OF_CONDUCT.md"
  # Project docs and config
  "CLAUDE.md"
  "AGENTS.md"
  # Node.js project files
  "package.json"
  "package-lock.json"
  # Playwright config (root-level, read by default)
  "playwright.config.ts"
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

# Enforce deprecated top-level directories are not reintroduced.
if [ -d "services" ]; then
  echo "❌ Deprecated top-level directory exists: services/"
  ((ERRORS++))
fi
if [ -d "deployment" ]; then
  echo "❌ Deprecated top-level directory exists: deployment/"
  ((ERRORS++))
fi

# Check tracked files for generated/artifact paths and binary payloads.

# Artifact-path check: a pure shell case, no process spawns.
while IFS= read -r tracked; do
  [ -e "$tracked" ] || continue
  case "$tracked" in
    build/*|log/*|node_modules/*|frontend/node_modules/*|frontend/dist/*)
      echo "❌ Artifact path is tracked by git: $tracked"
      ((ERRORS++))
      ;;
  esac
done < <(git ls-files)

# Binary detection: call file(1) once over the whole tracked set, not once per
# path. One spawn per tracked file makes this gate O(files) in process spawns
# and takes minutes on a large repo; a single batched call is O(1) spawns.
# grep does the filtering, so the loop only reports what already matched.
while IFS= read -r desc; do
  echo "❌ Binary file is tracked by git: ${desc%%: *}"
  ((ERRORS++))
done < <(git ls-files -z | xargs -0 file 2>/dev/null | grep -E 'Mach-O|ELF|PE32' || true)

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
