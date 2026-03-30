#!/usr/bin/env bash

set -euo pipefail

MODE="fast"
BASE_REF="HEAD"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="${2:-fast}"
      shift 2
      ;;
    --base)
      BASE_REF="${2:-HEAD}"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: scripts/dev-micro-loop.sh [--mode fast|full] [--base <git-ref>]"
      exit 1
      ;;
  esac
done

if [[ "$MODE" != "fast" && "$MODE" != "full" ]]; then
  echo "Invalid --mode: $MODE (expected fast or full)"
  exit 1
fi

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1"
    exit 1
  fi
}

need_cmd go
need_cmd git
need_cmd make

echo "[micro-loop] mode=$MODE base_ref=$BASE_REF"

changed_go_files="$(git diff --name-only "$BASE_REF"...HEAD -- '*.go' || true)"
if [[ -z "$changed_go_files" ]]; then
  changed_go_files="$(git diff --name-only -- '*.go' || true)"
fi

if [[ -z "$changed_go_files" ]]; then
  echo "[micro-loop] no changed .go files found; running default package checks"
  changed_pkgs="./pkg/..."
else
  changed_pkgs="$(echo "$changed_go_files" | xargs -n1 dirname | sort -u | sed 's|^|./|')"
fi

echo "[micro-loop] changed packages:"
echo "$changed_pkgs"

echo "[micro-loop] step 1: format check"
make fmt-check

echo "[micro-loop] step 2: lint"
make lint

echo "[micro-loop] step 3: go vet"
make vet

echo "[micro-loop] step 4: staticcheck"
make staticcheck

echo "[micro-loop] step 5: unit tests (changed packages)"
for pkg in $changed_pkgs; do
  go test -race -short "$pkg"
done

if [[ "$MODE" == "full" ]]; then
  echo "[micro-loop] step 6: full unit scope"
  make test-unit
fi

echo "[micro-loop] done"
