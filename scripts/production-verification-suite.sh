#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: scripts/production-verification-suite.sh

Runs the current repository production verification baseline:
  1. deployment smoke
  2. observability baseline
  3. alert-readiness baseline
  4. chaos baseline

This is a repo-local verification suite entrypoint. It does not certify full
production readiness by itself; pair it with docs/deployment/go-live-blockers.md.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[production-verification-suite] %s\n' "$*"
}

log "Running current production baseline verification suite"
(
  cd "${ROOT_DIR}"
  bash scripts/run-production-readiness-rehearsal.sh
)

log "Production verification suite passed"
