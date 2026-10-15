#!/usr/bin/env bash
# verify-production-readiness.sh - fail-closed production readiness gate.
#
# Decision D19(a) (docs/project/2026-10-13-decision-requests.md): the script is
# WRITTEN, not skipped. `.github/workflows/deploy.yml` calls it twice - once in
# the `verify` job and again as the production pre-deployment check - and both
# call sites already fail closed when it is missing. Every check below must
# pass or this script exits non-zero. The only tolerated non-pass is an
# EXPLICITLY declared skip (SKIP_K8S=1); there is no implicit skip anywhere.
#
# Checks
#   1. build inputs    the exact files deploy.yml's build job consumes
#                      (docker/Dockerfile, docker/Dockerfile.microservices),
#                      present and non-empty. deploy.yml's build job pushes
#                      images directly and writes no digest file today, so
#                      there is no digest artifact to assert - but when
#                      IMAGE_DIGEST_FILE is set, that file must exist and be
#                      non-empty (the hook a future digest step wires into).
#   2. F17 approval    docs/project/DEPENDENCY_APPROVAL.md exists and is
#                      non-empty. Plan T15/F17: a dependency or tool version
#                      change without it fails the §10 DoD.
#   3. migrations      migrations/ holds at least one .sql file and the
#                      NNNNNN_ numbering scheme has no gaps (a gap means a
#                      fresh apply would skip a migration).
#   4. k8s manifests   `kubectl kustomize` builds k8s/base + both overlays,
#                      each producing >=1 resource (same pattern as ci.yml).
#                      Requires kubectl; SKIP_K8S=1 skips it loudly instead.
#   5. image tag       the image reference being deployed must be non-empty
#                      and must not be `latest`.
#
# Environment
#   DEPLOY_IMAGE       full reference (registry/repo:tag); its tag takes
#                      precedence over IMAGE_TAG
#   IMAGE_TAG          tag only; falls back to $GITHUB_REF_NAME (what a
#                      tag-push deploy is named; deploy.yml sets neither today)
#   IMAGE_DIGEST_FILE  optional path (absolute or repo-relative); when set it
#                      must exist and be non-empty
#   SKIP_K8S=1         skip check 4 explicitly (recorded as SKIP, not PASS)
#   READINESS_ROOT     repo-root override; used by --self-test
#
# Usage
#   bash scripts/verify-production-readiness.sh               run all checks
#   bash scripts/verify-production-readiness.sh --self-test   verify the gate
#   bash scripts/verify-production-readiness.sh --help
#
# Exit codes
#   0  every check passed (a declared SKIP counts as not-failed)
#   1  at least one check failed
#   2  usage / environment error

set -euo pipefail

ROOT_DIR="${READINESS_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"

usage() {
  sed -n '2,49p' "$SELF" | sed -e 's/^#//' -e 's/^ //'
}

# Per-check outcome lines collected for the final summary.
declare -a SUMMARY=()
FAILED=0

pass() { SUMMARY+=("PASS  $1"); printf 'PASS  %s\n' "$1"; }
fail() { SUMMARY+=("FAIL  $1"); printf 'FAIL  %s\n' "$1"; FAILED=1; }
skip() { SUMMARY+=("SKIP  $1"); printf 'SKIP  %s\n' "$1"; }

# --- check 1: build inputs ---------------------------------------------------
check_build_inputs() {
  local ok=0 f p
  for f in docker/Dockerfile docker/Dockerfile.microservices; do
    if [ ! -s "$ROOT_DIR/$f" ]; then
      printf '  missing or empty: %s (deploy.yml build job file: input)\n' "$f"
      ok=1
    fi
  done
  if [ -n "${IMAGE_DIGEST_FILE:-}" ]; then
    case "$IMAGE_DIGEST_FILE" in
      /*) p="$IMAGE_DIGEST_FILE" ;;
      *) p="$ROOT_DIR/$IMAGE_DIGEST_FILE" ;;
    esac
    if [ ! -s "$p" ]; then
      printf '  missing or empty: %s (IMAGE_DIGEST_FILE)\n' "$IMAGE_DIGEST_FILE"
      ok=1
    fi
  fi
  return "$ok"
}

# --- check 2: F17 dependency approval ---------------------------------------
check_dependency_approval() {
  local f="docs/project/DEPENDENCY_APPROVAL.md"
  if [ ! -s "$ROOT_DIR/$f" ]; then
    printf '  missing or empty: %s (plan F17: dependency changes without it fail the DoD)\n' "$f"
    return 1
  fi
  return 0
}

# --- check 3: migrations present and ordered --------------------------------
check_migrations() {
  local dir="$ROOT_DIR/migrations" ok=0
  if [ ! -d "$dir" ]; then
    printf '  missing directory: migrations/\n'
    return 1
  fi
  local count
  count=$(find "$dir" -maxdepth 1 -type f -name '*.sql' | wc -l | tr -d '[:space:]')
  if [ "$count" -lt 1 ]; then
    printf '  no .sql files in migrations/\n'
    return 1
  fi
  # Numbering scheme: NNNNNN_name.up.sql / NNNNNN_name.down.sql. Collect the
  # distinct prefixes; up/down pairs share one prefix by design. sed rather
  # than a case-loop: the case has to live inside the command substitution,
  # which older bash (macOS ships 3.2) refuses to parse.
  local numbers base n prev=""
  numbers=$(
    printf '%s\n' "$dir"/*.sql \
      | sed -n 's|.*/\([0-9]\{6\}\)_.*|\1|p' \
      | sort -u
  )
  if [ -z "$numbers" ]; then
    printf '  no NNNNNN_-prefixed migrations under migrations/; numbering scheme not visible\n'
    return 1
  fi
  for n in $numbers; do
    if [ -n "$prev" ] && [ $((10#$n)) -ne $((10#$prev + 1)) ]; then
      printf '  numbering gap between %s and %s\n' "$prev" "$n"
      ok=1
    fi
    prev=$n
  done
  printf '  %s .sql files, numbering %s..%s contiguous\n' "$count" "$(printf '%s\n' "$numbers" | head -1)" "$prev"
  return "$ok"
}

# --- check 4: k8s manifests build (ci.yml pattern) --------------------------
kustomize_builds() {
  local overlay out count ok=0
  for overlay in k8s/base k8s/overlays/monolithic k8s/overlays/microservice; do
    if ! out=$(kubectl kustomize "$ROOT_DIR/$overlay" 2>&1); then
      printf '  kubectl kustomize failed for %s: %s\n' "$overlay" "$out"
      ok=1
      continue
    fi
    count=$(printf '%s\n' "$out" | grep -c '^kind:' || true)
    printf '  %s: %s resources\n' "$overlay" "$count"
    if [ "${count:-0}" -lt 1 ]; then
      printf '  %s built no resources\n' "$overlay"
      ok=1
    fi
  done
  return "$ok"
}

# --- check 5: image tag is present and not latest ---------------------------
check_image_tag() {
  local tag=""
  if [ -n "${DEPLOY_IMAGE:-}" ]; then
    case "$DEPLOY_IMAGE" in
      *:*) tag="${DEPLOY_IMAGE##*:}" ;;
      *)
        printf '  DEPLOY_IMAGE=%s carries no tag\n' "$DEPLOY_IMAGE"
        return 1
        ;;
    esac
  elif [ -n "${IMAGE_TAG:-}" ]; then
    tag="$IMAGE_TAG"
  elif [ -n "${GITHUB_REF_NAME:-}" ]; then
    tag="$GITHUB_REF_NAME"
  fi
  if [ -z "$tag" ]; then
    printf '  image tag to deploy is empty (set DEPLOY_IMAGE or IMAGE_TAG; in CI GITHUB_REF_NAME supplies it)\n'
    return 1
  fi
  case "$(printf '%s' "$tag" | tr '[:upper:]' '[:lower:]')" in
    latest)
      printf '  image tag to deploy is "latest"; deploys must use an immutable tag or digest\n'
      return 1
      ;;
  esac
  printf '  tag under deploy: %s\n' "$tag"
  return 0
}

run_all() {
  echo "=== production readiness gate (root: $ROOT_DIR) ==="

  if check_build_inputs; then pass "1 build inputs (Dockerfiles${IMAGE_DIGEST_FILE:+, IMAGE_DIGEST_FILE})"; else fail "1 build inputs (Dockerfiles${IMAGE_DIGEST_FILE:+, IMAGE_DIGEST_FILE})"; fi

  if check_dependency_approval; then pass "2 dependency approval (docs/project/DEPENDENCY_APPROVAL.md)"; else fail "2 dependency approval (docs/project/DEPENDENCY_APPROVAL.md)"; fi

  if check_migrations; then pass "3 migrations present and ordered"; else fail "3 migrations present and ordered"; fi

  if [ "${SKIP_K8S:-0}" = "1" ]; then
    skip "4 k8s manifests build - SKIP_K8S=1 (explicitly declared, not a pass)"
  elif ! command -v kubectl >/dev/null 2>&1; then
    echo "  kubectl not available; install it or set SKIP_K8S=1 to skip this check explicitly"
    fail "4 k8s manifests build (kubectl unavailable)"
  elif kustomize_builds; then
    pass "4 k8s manifests build"
  else
    fail "4 k8s manifests build"
  fi

  if check_image_tag; then pass "5 image tag not latest/empty"; else fail "5 image tag not latest/empty"; fi

  echo "----------------------------------------"
  echo "summary"
  local line
  for line in "${SUMMARY[@]}"; do
    printf '  %s\n' "$line"
  done
  echo "----------------------------------------"
  if [ "$FAILED" -ne 0 ]; then
    echo "production readiness: FAIL"
    return 1
  fi
  echo "production readiness: PASS"
  return 0
}

# --- self-test ---------------------------------------------------------------
# Two assertions, both about the gate itself:
#   a) an empty repository root must be REJECTED (the gate fails when its
#      required files are missing - the failure mode deploy.yml guards for);
#   b) this repository's real root must be ACCEPTED (guards against a gate
#      that is always red, which would be as useless as one always green).
self_test() {
  local failures=0 tmp
  tmp=$(mktemp -d)

  if env -u DEPLOY_IMAGE -u IMAGE_TAG -u GITHUB_REF_NAME -u IMAGE_DIGEST_FILE \
      READINESS_ROOT="$tmp" SKIP_K8S=1 IMAGE_TAG=v0.0.0-selftest \
      bash "$SELF" >/dev/null 2>&1; then
    echo "self-test FAILED: an empty root passed the gate" >&2
    failures=1
  else
    echo "self-test ok: empty root rejected"
  fi
  rm -rf "$tmp"

  if env -u DEPLOY_IMAGE -u GITHUB_REF_NAME -u IMAGE_DIGEST_FILE \
      READINESS_ROOT="$ROOT_DIR" SKIP_K8S=1 IMAGE_TAG=v0.0.0-selftest \
      bash "$SELF" >/dev/null 2>&1; then
    echo "self-test ok: repository root passes"
  else
    echo "self-test FAILED: repository root did not pass; re-run without -v suppression:" >&2
    env -u DEPLOY_IMAGE -u GITHUB_REF_NAME -u IMAGE_DIGEST_FILE \
      READINESS_ROOT="$ROOT_DIR" SKIP_K8S=1 IMAGE_TAG=v0.0.0-selftest \
      bash "$SELF" >&2 || true
    failures=1
  fi

  if [ "$failures" -ne 0 ]; then
    echo "verify-production-readiness self-test FAILED" >&2
    return 1
  fi
  echo "verify-production-readiness self-test passed (empty root rejected, repository root accepted)"
  return 0
}

case "${1:-}" in
  '') run_all ;;
  --self-test) self_test ;;
  -h|--help) usage ;;
  *)
    usage >&2
    exit 2
    ;;
esac
