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
CONFIRMED="${CONFIRMED:-0}"
ROLLBACK_DEPLOYMENT="${ROLLBACK_DEPLOYMENT:-chainpulse-monolithic}"
ROLLBACK_NAMESPACE="${ROLLBACK_NAMESPACE:-chainpulse}"

usage() {
  cat <<'EOF'
Usage: CURRENT_RELEASE=<release> PREVIOUS_RELEASE=<release> scripts/rollback-drill.sh

Runs the repository rollback-drill gate for a ChainPulse release.

Required environment variables:
  CURRENT_RELEASE    release currently under evaluation
  PREVIOUS_RELEASE   rollback target release

Optional environment variables:
  ROLLBACK_TARGET     environment label for the drill (default: production)
  RUN_PRECHECK        set to 1 to run scripts/verify-production.sh before drill
  RUN_POSTCHECK       set to 1 to run scripts/verify-production.sh after drill
  ROLLBACK_NOTES      free-form operator notes printed in the drill output
  CONFIRMED           set to 1 to skip interactive confirmation
  ROLLBACK_DEPLOYMENT target deployment name (default: chainpulse-monolithic)
  ROLLBACK_NAMESPACE  target namespace (default: chainpulse)
  API_GATEWAY_URL       forwarded to scripts/verify-production.sh
  API_SERVICE_URL       forwarded to scripts/verify-production.sh
  EVENT_PROCESSOR_URL   forwarded to scripts/verify-production.sh
  PULLER_URL            forwarded to scripts/verify-production.sh
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

require_kubectl() {
  if ! command -v kubectl >/dev/null 2>&1; then
    fail "kubectl not found"
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
log "Target deployment: ${ROLLBACK_DEPLOYMENT}/${ROLLBACK_NAMESPACE}"

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
=================================
1. Confirm release freeze for ${ROLLBACK_TARGET} and notify oncall / release owner.
2. Snapshot the current deployment state for ${CURRENT_RELEASE}:
   - image digests / release artifact identifiers
   - active configuration / secrets revision
   - latest database backup or checkpoint reference
3. Execute kubectl rollout undo:
     kubectl rollout undo deployment/${ROLLBACK_DEPLOYMENT} \\
       -n ${ROLLBACK_NAMESPACE} \\
       --to-revision=PREVIOUS_REVISION
   or use the 'kubectl' rollback path below.
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

require_kubectl

show_revision_history() {
  kubectl rollout history "deployment/${ROLLBACK_DEPLOYMENT}" -n "${ROLLBACK_NAMESPACE}" || true
}

log "Current deployment revision history:"
show_revision_history
echo ""

if [[ "${CONFIRMED}" != "1" ]]; then
  printf 'Execute rollout undo for deployment/%s in namespace %s? [y/N]: ' "${ROLLBACK_DEPLOYMENT}" "${ROLLBACK_NAMESPACE}"
  read -r confirm
  if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
    log "Rollback aborted by operator"
    exit 0
  fi
fi

execute_rollback() {
  kubectl rollout undo "deployment/${ROLLBACK_DEPLOYMENT}" -n "${ROLLBACK_NAMESPACE}"
  kubectl rollout status "deployment/${ROLLBACK_DEPLOYMENT}" -n "${ROLLBACK_NAMESPACE}" --timeout=300s
}

log "Executing rollback..."
if execute_rollback; then
  log "Rollback completed successfully"
else
  fail "Rollback failed — manual intervention required"
fi

log "Post-rollback revision history:"
show_revision_history

if [[ "${RUN_POSTCHECK}" == "1" ]]; then
  log "Running post-rollback production verification"
  run_verify_production
else
  log "Skipping post-rollback production verification (set RUN_POSTCHECK=1 to enable)"
fi

log "Rollback drill gate completed"