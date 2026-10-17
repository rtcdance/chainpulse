#!/usr/bin/env bash
# check-workspace-modules.sh - build, vet and test EVERY module in the Go workspace.
#
# Plan section 7.1, which names this requirement directly:
#
#   "根目录 `./...` 不等于整个 Go workspace。T01 从 `go work edit -json` 动态枚举每个
#    `.Use[].DiskPath`，对每个 module 执行 build/vet/test；根模块另外执行 cmd 测试。
#    CI 检查 go.work 的 go 指令不低于任一模块，版本/config 变更单独提交，不添加
#    toolchain 指令。"
#
#   "全量 lint 要覆盖实际 workspace 模块（先动态枚举再在模块执行），同时留存 root
#    标准口径结果；不能靠根命令未进入子模块来完成。"
#
# The last clause is the reason this script exists. `go build ./...`, `go vet ./...`
# and `go test ./...` at the repository root all stop at the current module, so the
# six nested modules under pkg/ were never built, vetted or tested by CI - while
# looking fully covered, because the root commands are the ones a reader checks.
#
# Usage
# -----
#     bash scripts/check-workspace-modules.sh            # build + vet + test
#     bash scripts/check-workspace-modules.sh --no-test  # build + vet only
#
# Exit codes
# ----------
#   0  every module built, vetted and tested; go.work's directive is not behind
#   1  a module failed, or go.work's go directive is lower than a module's
#   2  usage / environment error

set -euo pipefail

run_tests=1
if [ "${1:-}" = "--no-test" ]; then
  run_tests=0
elif [ -n "${1:-}" ]; then
  echo "usage: $0 [--no-test]" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [ ! -f go.work ]; then
  echo "error: no go.work at $repo_root; this check is about workspace completeness" >&2
  exit 2
fi

# Enumerate dynamically. A hard-coded list would silently stop covering a module
# the moment someone adds one, which is the failure this check exists to prevent.
modules=()
work_go=""
while IFS=$'\t' read -r kind value; do
  case "$kind" in
    GO) work_go="$value" ;;
    MODULE) modules+=("$value") ;;
  esac
done < <(python3 - <<'PY'
import json, subprocess
data = json.loads(subprocess.run(["go", "work", "edit", "-json"],
                                 capture_output=True, text=True, check=True).stdout)
print("GO\t" + str(data.get("Go", "")))
for use in data.get("Use", []):
    print("MODULE\t" + use["DiskPath"])
PY
)

if [ "${#modules[@]}" -eq 0 ]; then
  echo "error: go work edit -json reported no modules" >&2
  exit 2
fi

echo "workspace go directive: ${work_go:-<none>}"
echo "modules discovered: ${#modules[@]}"
for m in "${modules[@]}"; do echo "  - $m"; done
echo

failures=()

# --- go.work's go directive must not be behind any module's ------------------
# A workspace whose directive is lower than a member's cannot build that member.
for m in "${modules[@]}"; do
  if [ ! -f "$m/go.mod" ]; then
    failures+=("$m: no go.mod (go.work lists it as a module)")
    continue
  fi
  mod_go="$(awk '$1 == "go" && NF >= 2 { print $2; exit }' "$m/go.mod")"
  if [ -z "$mod_go" ]; then
    failures+=("$m: go.mod has no go directive")
    continue
  fi
  # The workspace directive must be >= the module's. Sorted ascending, the last
  # entry is the greater one, so it must be the workspace's - comparing against
  # the first entry instead reports 1.26.9 as "lower than" 1.26.0.
  if [ -n "$work_go" ] && [ "$(printf '%s\n%s\n' "$work_go" "$mod_go" | sort -V | tail -1)" != "$work_go" ]; then
    failures+=("go.work go $work_go is lower than $m's go $mod_go")
  fi
done

# --- build / vet / test, per module -----------------------------------------
for m in "${modules[@]}"; do
  [ -f "$m/go.mod" ] || continue
  echo "=== $m ==="

  if ! (cd "$m" && go build ./...); then
    failures+=("$m: go build ./... failed")
  fi
  if ! (cd "$m" && go vet ./...); then
    failures+=("$m: go vet ./... failed")
  fi

  if [ "$run_tests" -eq 1 ]; then
    if ! (cd "$m" && go test -short -count=1 ./...); then
      failures+=("$m: go test -short ./... failed")
    fi
  fi

  # Section 7.1: the root module additionally runs the cmd tests. `./...` already
  # covers ./cmd/..., so this is recorded rather than run twice.
  if [ "$m" = "." ]; then
    echo "  (root module also owns ./cmd/...; covered by ./... above)"
  fi
  echo
done

if [ "${#failures[@]}" -ne 0 ]; then
  echo "workspace check FAILED (${#failures[@]} issue(s)):" >&2
  for f in "${failures[@]}"; do echo "  - $f" >&2; done
  exit 1
fi

if [ "$run_tests" -eq 1 ]; then
  echo "workspace check passed: ${#modules[@]} modules built, vetted and tested"
else
  echo "workspace check passed: ${#modules[@]} modules built and vetted (tests not requested)"
fi
