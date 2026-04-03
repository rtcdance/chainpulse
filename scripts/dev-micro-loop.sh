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
if [[ -z "$changed_go_files" ]]; then
  make fmt-check
else
  need_cmd gofumpt
  unformatted_files=""
  while IFS= read -r file; do
    [[ -f "$file" ]] || continue
    result="$(gofumpt -l "$file")"
    if [[ -n "$result" ]]; then
      unformatted_files+="${result}"$'\n'
    fi
  done <<< "$changed_go_files"

  if [[ -n "$unformatted_files" ]]; then
    echo "Code is not formatted. Run 'gofumpt -w <files>' to fix."
    echo "$unformatted_files"
    exit 1
  fi
fi

echo "[micro-loop] step 2: lint"
if [[ "$MODE" == "full" || -z "$changed_go_files" ]]; then
  need_cmd golangci-lint
  patch_file="$(mktemp)"
  git diff -- '*.go' > "$patch_file"
  lint_output=""
  if ! lint_output="$(GOCACHE=${GOCACHE:-/tmp/chainpulse-go-build-cache} golangci-lint run --tests=false --new-from-patch "$patch_file" ./... 2>&1)"; then
    if [[ "$lint_output" == *"no go files to analyze"* ]]; then
      echo "[micro-loop] lint patch returned no analyzable files; skipping and relying on the remaining gates"
    else
      echo "$lint_output"
      rm -f "$patch_file"
      exit 1
    fi
  fi
  rm -f "$patch_file"
else
  need_cmd golangci-lint
  lint_output=""
  if ! lint_output="$(golangci-lint run --new ${changed_pkgs} 2>&1)"; then
    if [[ "$lint_output" == *"no go files to analyze"* ]]; then
      echo "[micro-loop] lint --new returned no analyzable files; skipping fast lint and relying on full gate"
    else
      echo "$lint_output"
      exit 1
    fi
  fi
fi

echo "[micro-loop] step 3: go vet"
if [[ "$MODE" == "full" || -z "$changed_go_files" ]]; then
  if [[ "$MODE" == "full" ]]; then
    go vet $changed_pkgs
  else
    make vet
  fi
else
  go vet $changed_pkgs
fi

echo "[micro-loop] step 4: staticcheck"
if [[ "$MODE" == "full" || -z "$changed_go_files" ]]; then
  if [[ "$MODE" == "full" ]]; then
    need_cmd staticcheck
    staticcheck $changed_pkgs
  else
    make staticcheck
  fi
else
  echo "[micro-loop] staticcheck skipped in fast mode (covered by full gate)"
fi

echo "[micro-loop] step 5: unit tests (changed packages)"
for pkg in $changed_pkgs; do
  if [[ "$MODE" == "full" ]]; then
    go test -race -short "$pkg"
  else
    go test -short "$pkg"
  fi
done

if [[ "$MODE" == "full" ]]; then
  echo "[micro-loop] step 6: full unit scope"
  make test-unit
  echo "[micro-loop] step 7: policy metric/tag contract"
  ./scripts/check-policy-metric-contract.sh
  echo "[micro-loop] step 8: migration manifest deadline check"
  ./scripts/check-migration-manifest.sh
  echo "[micro-loop] step 9: export migration governance KPI"
  ./scripts/export-migration-governance-kpi.sh
  echo "[micro-loop] step 10: compare migration governance KPI"
  ./scripts/compare-migration-governance-kpi.sh
  echo "[micro-loop] step 11: baseline update preflight"
  ./scripts/preflight-migration-baseline-update.sh
  echo "[micro-loop] step 12: baseline update resolver tests"
  ./scripts/test-baseline-update-resolver.sh
  echo "[micro-loop] step 13: compare baseline resolver test delta"
  ./scripts/compare-baseline-resolver-test.sh
  echo "[micro-loop] step 14: check migration baseline governance"
  ./scripts/check-migration-baseline-governance.sh
  echo "[micro-loop] step 15: baseline governance scope smoke tests"
  ./scripts/smoke-baseline-governance-scope.sh
  echo "[micro-loop] step 16: compare baseline scope smoke delta"
  ./scripts/compare-baseline-scope-smoke.sh
  echo "[micro-loop] step 17: check migration changelog quality"
  ./scripts/check-migration-changelog-quality.sh
  echo "[micro-loop] step 18: compare ticket registry health delta"
  ./scripts/compare-ticket-registry-health.sh
  echo "[micro-loop] step 19: export migration owner drift report"
  ./scripts/export-migration-owner-drift-report.sh
fi

echo "[micro-loop] done"
