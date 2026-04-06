#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: scripts/run-production-readiness-rehearsal.sh

Runs the current minimum production-readiness rehearsal by executing:
  1. microservice deployment smoke
  2. microservice observability baseline
  3. microservice alert-readiness baseline
  4. microservice chaos baseline
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[production-rehearsal] %s\n' "$*"
}

log "Running deployment smoke"
(
  cd "${ROOT_DIR}"
  bash scripts/verify-microservice-deployment-smoke.sh
)

log "Running observability baseline"
(
  cd "${ROOT_DIR}"
  bash scripts/verify-microservice-observability-baseline.sh
)

log "Running alert-readiness baseline"
(
  cd "${ROOT_DIR}"
  bash scripts/verify-microservice-alert-readiness.sh
)

log "Running chaos baseline"
(
  cd "${ROOT_DIR}"
  bash scripts/chaos-test.sh
)

log "Production-readiness rehearsal passed"
