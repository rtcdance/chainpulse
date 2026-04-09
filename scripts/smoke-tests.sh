#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: scripts/smoke-tests.sh

Runs the repository-local staging smoke baseline by reusing the microservice
deployment smoke entrypoint.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[smoke-tests] %s\n' "$*"
}

log "Running repository-local deployment smoke"
(
  cd "${ROOT_DIR}"
  bash scripts/verify-microservice-deployment-smoke.sh
)

log "Smoke tests passed"
