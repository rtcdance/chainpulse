#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: scripts/verify-staging.sh

Runs the repository-local staging verification bundle:
  1. deployment readiness static checks
  2. smoke tests
  3. production verification suite baseline

This script validates the repo-local staging baseline. It does not perform
environment-specific cloud deployment checks.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-staging] %s\n' "$*"
}

log "Running deployment readiness checks"
(
  cd "${ROOT_DIR}"
  bash scripts/deployment-readiness-check.sh
)

log "Running staging smoke tests"
(
  cd "${ROOT_DIR}"
  bash scripts/smoke-tests.sh
)

log "Running staging verification suite"
(
  cd "${ROOT_DIR}"
  bash scripts/production-verification-suite.sh
)

log "Staging verification passed"
