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
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
tool_bin="$(go env GOBIN)"
if [[ -z "$tool_bin" ]]; then tool_bin="$(go env GOPATH)/bin"; fi
export PATH="$tool_bin:$PATH"

echo "[micro-loop] mode=$MODE base_ref=$BASE_REF"

changed_go_files="$(git diff --name-only "$BASE_REF"...HEAD -- '*.go' || true)"
if [[ -z "$changed_go_files" ]]; then
  changed_go_files="$(git diff --name-only -- '*.go' || true)"
fi
changed_go_files="$(printf '%s\n%s\n' "$changed_go_files" "$(git ls-files --others --exclude-standard -- '*.go')" | sed '/^$/d' | sort -u)"

if [[ -z "$changed_go_files" ]]; then
  echo "[micro-loop] no changed .go files found; running default package checks"
  changed_pkgs="./pkg/..."
else
  changed_pkgs="$(echo "$changed_go_files" | xargs -n1 dirname | sort -u | sed 's|^|./|')"
fi

# Integration-only directories have no ordinary Go package. Their tests belong
# to the provisioned integration lane; passing them to lint aborts its analysis.
selected_pkgs=""
for pkg in $changed_pkgs; do
  if list_output="$(go list -f '{{join .GoFiles " "}}' "$pkg" 2>&1)"; then
    selected_pkgs+="$pkg"$'\n'
  elif [[ "$list_output" == *"build constraints exclude all Go files"* ]]; then
    echo "[micro-loop] integration-only package excluded from unit lane: $pkg"
  else
    echo "$list_output"
    exit 1
  fi
done
changed_pkgs="$(printf '%s' "$selected_pkgs" | sed '/^$/d')"

echo "[micro-loop] changed packages:"
echo "$changed_pkgs"

echo "[micro-loop] step 0: compile"
for pkg in $changed_pkgs; do go build "$pkg"; done

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
  if ! lint_output="$(golangci-lint run --tests=false --new-from-rev=HEAD~1 ${changed_pkgs} 2>&1)"; then
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
    staticcheck_bin="${STATICCHECK_BIN:-staticcheck}"
    need_cmd "$staticcheck_bin"
    "$staticcheck_bin" $changed_pkgs
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
  echo "[micro-loop] step 7: bounded metric labels and lag contracts"
  evidence_dir="${CHAINPULSE_MICRO_EVIDENCE_DIR:-build/micro-loop}"
  mkdir -p "$evidence_dir"
  go test -json -short ./pkg/observability \
    -run '^(TestLabelGuard|TestLag|TestGuarded)' -count=1 > "$evidence_dir/metrics.jsonl"
  python3 scripts/acceptance/verify-tests.py --jsonl "$evidence_dir/metrics.jsonl" --min-tests 1 --fail-on-unknown-skip

  # The old migration-governance scripts were archived and referenced deleted
  # policy tests. Full delivery verifies the actual SQL schema and adapters.
  echo "[micro-loop] step 8: real adapters, fault recovery and deployed reorg"
  : "${CHAINPULSE_TEST_DATABASE_URL:?full gate requires an isolated migrated PostgreSQL database}"
  : "${CHAINPULSE_TEST_REDIS_URL:?full gate requires isolated Redis}"
  : "${CHAINPULSE_TEST_KAFKA_BROKERS:?full gate requires reachable Kafka}"
  : "${CHAINPULSE_PG_FORWARDER:?full gate requires an explicit isolated DB fault container}"
  : "${CHAINPULSE_REDIS_CONTAINER:?full gate requires an explicit isolated Redis fault container}"
  if [[ "${CHAINPULSE_RUN_DEPLOYED_REORG:-}" != "1" ]]; then
    echo "full gate requires CHAINPULSE_RUN_DEPLOYED_REORG=1 with an isolated Anvil, emitter and current monolithic binary"
    exit 1
  fi
  if [[ "${CHAINPULSE_RUN_DEPLOYED_BLOCKS:-}" != "1" ]]; then
    echo "full gate requires CHAINPULSE_RUN_DEPLOYED_BLOCKS=1 with current native puller/processor binaries and an isolated microservice DB"
    exit 1
  fi
  : "${CHAINPULSE_DEPLOYED_MICRO_DATABASE_URL:?full gate requires an isolated native pipeline database}"
  : "${CHAINPULSE_DEPLOYED_PROCESSOR_BINARY:?full gate requires the current event processor binary}"
  : "${CHAINPULSE_DEPLOYED_PULLER_BINARY:?full gate requires the current puller binary}"
  : "${CHAINPULSE_DEPLOYED_API_SERVICE_BINARY:?full gate requires the current API service binary}"
  : "${CHAINPULSE_DEPLOYED_GATEWAY_BINARY:?full gate requires the current API gateway binary}"
  if [[ "${CHAINPULSE_RUN_DEPLOYED_DUAL:-}" != "1" ]]; then
    echo "full gate requires CHAINPULSE_RUN_DEPLOYED_DUAL=1 with two isolated chains and mode databases"
    exit 1
  fi
  if [[ "${CHAINPULSE_RUN_DEPLOYED_DUAL_FAULT:-}" != "1" ]]; then
    echo "full gate requires CHAINPULSE_RUN_DEPLOYED_DUAL_FAULT=1 for mandatory dual-chain fault isolation"
    exit 1
  fi
  : "${CHAINPULSE_DUAL_MONO_DATABASE_URL:?full gate requires isolated dual-chain monolithic DB}"
  : "${CHAINPULSE_DUAL_MICRO_DATABASE_URL:?full gate requires isolated dual-chain microservices DB}"
  export CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 CHAINPULSE_RUN_REDIS_INTEGRATION=1 CHAINPULSE_RUN_KAFKA_INTEGRATION=1
  go test -json -race -p 1 -parallel 1 -tags=integration ./test/contracts ./test/acceptance \
    -count=1 -timeout=10m > "$evidence_dir/contracts.jsonl"
  python3 scripts/acceptance/verify-tests.py --jsonl "$evidence_dir/contracts.jsonl" \
    --min-tests 1 --fail-on-unknown-skip --required-cases test/acceptance/required-cases.json \
    --required-lane contracts --required-lane e2e --required-lane evm-e2e --report "$evidence_dir/contracts-report.json"

  echo "[micro-loop] step 9: schema fresh, upgrade and rollback"
  MIGRATION_TEST_DSN="$CHAINPULSE_TEST_DATABASE_URL" go test -json -tags=migration ./test/migration \
    -count=1 -timeout=10m > "$evidence_dir/migrations.jsonl"
  python3 scripts/acceptance/verify-tests.py --jsonl "$evidence_dir/migrations.jsonl" --min-tests 1 --fail-on-unknown-skip

  echo "[micro-loop] step 10: acceptance tooling and negative controls"
  for tool in verify-tests skip-report collect-evidence openapi-drift bench-gate dep-approval-gate acceptance-gate; do
    python3 "scripts/acceptance/$tool.py" --self-test
  done
  python3 scripts/acceptance/openapi-drift.py --report "$evidence_dir/openapi-report.json"
  python3 scripts/acceptance/negative-cases.py --json "$evidence_dir/negative-controls.json"

fi

echo "[micro-loop] done"
