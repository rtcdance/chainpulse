#!/usr/bin/env bash

set -euo pipefail

ALERTMANAGER_URL="${ALERTMANAGER_URL:-}"
EXPECTED_RECEIVER="${EXPECTED_RECEIVER:-}"
REQUIRE_ALERTS_API="${REQUIRE_ALERTS_API:-1}"
RUN_PRODUCTION_PRECHECK="${RUN_PRODUCTION_PRECHECK:-0}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'EOF'
Usage: ALERTMANAGER_URL=<url> EXPECTED_RECEIVER=<receiver> scripts/alert-delivery-check.sh

Runs the repository external alert-delivery gate for a deployed ChainPulse
environment.

Required environment variables:
  ALERTMANAGER_URL      base URL of the Alertmanager instance under test
  EXPECTED_RECEIVER     receiver name that must exist in Alertmanager routing

Optional environment variables:
  REQUIRE_ALERTS_API       set to 1 to require /api/v2/alerts to respond (default: 1)
  RUN_PRODUCTION_PRECHECK  set to 1 to run scripts/verify-production.sh first
  API_GATEWAY_URL          forwarded to scripts/verify-production.sh
  API_SERVICE_URL          forwarded to scripts/verify-production.sh
  EVENT_PROCESSOR_URL      forwarded to scripts/verify-production.sh
  PULLER_URL               forwarded to scripts/verify-production.sh

This script validates Alertmanager reachability and receiver-routing contract.
It does not claim to prove a full Slack/PagerDuty delivery unless the target
environment is already wired to those downstream systems.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[alert-delivery-check] %s\n' "$*"
}

fail() {
  printf 'alert delivery check failed: %s\n' "$*" >&2
  exit 1
}

fetch() {
  local url="$1"
  curl -fsS "${url}"
}

assert_toggle() {
  local name="$1"
  local value="$2"
  if [[ "${value}" != "0" && "${value}" != "1" ]]; then
    fail "${name} must be 0 or 1"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    fail "${label} missing \"${needle}\""
  fi
}

run_verify_production() {
  (
    cd "${ROOT_DIR}"
    bash scripts/verify-production.sh
  )
}

if [[ -z "${ALERTMANAGER_URL}" ]]; then
  fail "missing required environment variable ALERTMANAGER_URL"
fi

if [[ -z "${EXPECTED_RECEIVER}" ]]; then
  fail "missing required environment variable EXPECTED_RECEIVER"
fi

assert_toggle "REQUIRE_ALERTS_API" "${REQUIRE_ALERTS_API}"
assert_toggle "RUN_PRODUCTION_PRECHECK" "${RUN_PRODUCTION_PRECHECK}"

if [[ "${RUN_PRODUCTION_PRECHECK}" == "1" ]]; then
  log "Running production precheck"
  run_verify_production
fi

log "Checking Alertmanager readiness"
ready_payload="$(fetch "${ALERTMANAGER_URL}/-/ready")"
assert_contains "${ready_payload}" "OK" "alertmanager /-/ready"

log "Checking Alertmanager status/config surface"
status_payload="$(fetch "${ALERTMANAGER_URL}/api/v2/status")"
assert_contains "${status_payload}" "${EXPECTED_RECEIVER}" "alertmanager /api/v2/status"

log "Checking Alertmanager receiver routing"
receivers_payload="$(fetch "${ALERTMANAGER_URL}/api/v2/receivers")"
assert_contains "${receivers_payload}" "\"name\":\"${EXPECTED_RECEIVER}\"" "alertmanager /api/v2/receivers"

if [[ "${REQUIRE_ALERTS_API}" == "1" ]]; then
  log "Checking Alertmanager alerts API"
  alerts_payload="$(fetch "${ALERTMANAGER_URL}/api/v2/alerts")"
  assert_contains "${alerts_payload}" "[" "alertmanager /api/v2/alerts"
fi

log "Alert delivery gate passed"
