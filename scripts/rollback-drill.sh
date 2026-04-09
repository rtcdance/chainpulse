#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERIFY_PRODUCTION_SCRIPT="${ROOT_DIR}/scripts/verify-production.sh"

CURRENT_RELEASE="${CURRENT_RELEASE:-}"
PREVIOUS_RELEASE="${PREVIOUS_RELEASE:-}"
RUN_PRECHECK="${RUN_PRECHECK:-0}"
RUN_POSTCHECK="${RUN_POSTCHECK:-0}"
ROLLBACK_TARGET="${ROLLBACK_TARGET:-production}"
ROLLBACK_NOTES="${ROLLBACK_NOTES:-}"

usage() {
  cat <<'EOF'
Usage: CURRENT_RELEASE=<release> PREVIOUS_RELEASE=<release> scripts/rollback-drill.sh

Runs the repository rollback-drill gate for a ChainPulse release.

Required environment variables:
  CURRENT_RELEASE    release currently under evaluation
  PREVIOUS_RELEASE   rollback target release

Optional environment variables:
  ROLLBACK_TARGET    environment label for the drill (default: production)
  RUN_PRECHECK       set to 1 to run scripts/verify-production.sh before drill
  RUN_POSTCHECK      set to 1 to run scripts/verify-production.sh after drill
  ROLLBACK_NOTES     free-form operator notes printed in the drill output
  API_GATEWAY_URL       forwarded to scripts/verify-production.sh
  API_SERVICE_URL       forwarded to scripts/verify-production.sh
  EVENT_PROCESSOR_URL   forwarded to scripts/verify-production.sh
  PULLER_URL            forwarded to scripts/verify-production.sh

This script does not perform the environment-specific rollback for you. It
validates prerequisites, optionally runs live verification, and prints the
required execution sequence so the release team can capture rollback evidence.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[rollback-drill] %s\n' "$*"
}

fail() {
  printf 'rollback drill failed: %s\n' "$*" >&2
  exit 1
}

require_env() {
  local name="$1"
  local value="$2"
  if [[ -z "${value}" ]]; then
    fail "missing required environment variable ${name}"
  fi
}

run_verify_production() {
  if [[ ! -x "${VERIFY_PRODUCTION_SCRIPT}" && ! -f "${VERIFY_PRODUCTION_SCRIPT}" ]]; then
    fail "missing scripts/verify-production.sh"
  fi
  (
    cd "${ROOT_DIR}"
    bash scripts/verify-production.sh
  )
}

require_env "CURRENT_RELEASE" "${CURRENT_RELEASE}"
require_env "PREVIOUS_RELEASE" "${PREVIOUS_RELEASE}"

if [[ "${CURRENT_RELEASE}" == "${PREVIOUS_RELEASE}" ]]; then
  fail "CURRENT_RELEASE and PREVIOUS_RELEASE must differ"
fi

if [[ "${RUN_PRECHECK}" != "0" && "${RUN_PRECHECK}" != "1" ]]; then
  fail "RUN_PRECHECK must be 0 or 1"
fi

if [[ "${RUN_POSTCHECK}" != "0" && "${RUN_POSTCHECK}" != "1" ]]; then
  fail "RUN_POSTCHECK must be 0 or 1"
fi

log "Preparing rollback drill"
log "Target environment: ${ROLLBACK_TARGET}"
log "Current release: ${CURRENT_RELEASE}"
log "Rollback release: ${PREVIOUS_RELEASE}"

if [[ -n "${ROLLBACK_NOTES}" ]]; then
  log "Operator notes: ${ROLLBACK_NOTES}"
fi

if [[ "${RUN_PRECHECK}" == "1" ]]; then
  log "Running pre-rollback production verification"
  run_verify_production
else
  log "Skipping pre-rollback production verification (set RUN_PRECHECK=1 to enable)"
fi

cat <<EOF

Rollback drill execution sequence
1. Confirm release freeze for ${ROLLBACK_TARGET} and notify oncall / release owner.
2. Snapshot the current deployment state for ${CURRENT_RELEASE}:
   - image digests / release artifact identifiers
   - active configuration / secrets revision
   - latest database backup or checkpoint reference
3. Execute the environment rollback from ${CURRENT_RELEASE} to ${PREVIOUS_RELEASE}
   using the deployment platform for ${ROLLBACK_TARGET}.
4. If schema or data rollback is required, restore only from the approved
   backup/checkpoint associated with ${PREVIOUS_RELEASE}.
5. Capture evidence:
   - rollback start/end timestamps
   - operator / approver identity
   - commands or pipeline run identifiers used
   - any data repair or replay actions taken

Expected post-rollback validation:
  - scripts/verify-production.sh
  - service logs show stable startup on ${PREVIOUS_RELEASE}
  - alerting stays below rollback watch thresholds

EOF

if [[ "${RUN_POSTCHECK}" == "1" ]]; then
  log "Running post-rollback production verification"
  run_verify_production
else
  log "Skipping post-rollback production verification (set RUN_POSTCHECK=1 to enable)"
fi

log "Rollback drill gate completed"
