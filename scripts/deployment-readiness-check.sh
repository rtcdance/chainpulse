#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: scripts/deployment-readiness-check.sh

Performs static deployment-readiness checks for repository-tracked go-live
artifacts. This script validates that the documented release gate entrypoints
and blocker documents exist before a production release is attempted.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[deployment-readiness-check] %s\n' "$*"
}

require_file() {
  local path="$1"
  if [[ ! -f "${ROOT_DIR}/${path}" ]]; then
    printf 'deployment readiness failed: missing file %s\n' "${path}" >&2
    exit 1
  fi
}

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${ROOT_DIR}/${path}"; then
    printf 'deployment readiness failed: %s missing "%s"\n' "${path}" "${needle}" >&2
    exit 1
  fi
}

log "Checking required production documents"
require_file "docs/deployment/production-checklist.md"
require_file "docs/deployment/go-live-blockers.md"

log "Checking required production verification entrypoints"
require_file "scripts/production-verification-suite.sh"
require_file "scripts/deployment-readiness-check.sh"
require_file "scripts/run-production-readiness-rehearsal.sh"
require_file "scripts/rollback-drill.sh"
require_file "scripts/soak-check.sh"
require_file "scripts/alert-delivery-check.sh"
require_file "scripts/data-consistency-drill.sh"

log "Checking production checklist references"
assert_contains "docs/deployment/production-checklist.md" "Go-Live Blockers"
assert_contains "docs/deployment/production-checklist.md" "./scripts/rollback-drill.sh"
assert_contains "docs/deployment/production-checklist.md" "./scripts/soak-check.sh"
assert_contains "docs/deployment/production-checklist.md" "./scripts/alert-delivery-check.sh"
assert_contains "docs/deployment/production-checklist.md" "./scripts/data-consistency-drill.sh"
assert_contains "docs/deployment/go-live-blockers.md" "P0 Blockers"
assert_contains "docs/deployment/go-live-blockers.md" "scripts/rollback-drill.sh"
assert_contains "docs/deployment/go-live-blockers.md" "scripts/soak-check.sh"
assert_contains "docs/deployment/go-live-blockers.md" "scripts/alert-delivery-check.sh"
assert_contains "docs/deployment/go-live-blockers.md" "scripts/data-consistency-drill.sh"
assert_contains "README.md" "staging-ready / rehearsal-ready"

log "Deployment readiness static checks passed"
