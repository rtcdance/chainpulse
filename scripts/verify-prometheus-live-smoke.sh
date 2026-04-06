#!/usr/bin/env bash

set -euo pipefail

PROM_URL="${PROM_URL:-http://localhost:9090}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-prometheus-live-smoke.sh [--prom-url URL]

Runs a live Prometheus smoke by checking:
  1. /api/v1/targets is reachable
  2. expected ChainPulse jobs are present
  3. a small set of instant queries executes successfully

Environment:
  PROM_URL   Base URL of the Prometheus server (default: http://localhost:9090)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prom-url)
      PROM_URL="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

log() {
  printf '[verify-prometheus-live] %s\n' "$*"
}

require_command() {
  local command_name="$1"
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    printf 'prometheus live smoke failed: missing command %s\n' "${command_name}" >&2
    exit 1
  fi
}

fetch_json() {
  local url="$1"
  curl -fsSL "${url}"
}

assert_targets_present() {
  local payload="$1"
  python3 - "$payload" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
if payload.get("status") != "success":
    raise SystemExit("targets API did not return success")

active = payload.get("data", {}).get("activeTargets", [])
jobs = {target.get("labels", {}).get("job") for target in active}
expected = {
    "chainpulse-monolithic",
    "chainpulse-puller",
    "chainpulse-processor",
    "chainpulse-api-gateway",
    "chainpulse-api-service",
}
missing = sorted(expected - jobs)
if missing:
    raise SystemExit(f"missing Prometheus jobs: {', '.join(missing)}")
PY
}

assert_query_success() {
  local payload="$1"
  local query="$2"
  python3 - "$payload" "$query" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
query = sys.argv[2]
if payload.get("status") != "success":
    raise SystemExit(f"query failed for {query!r}")

result_type = payload.get("data", {}).get("resultType")
if result_type not in {"vector", "matrix", "scalar"}:
    raise SystemExit(f"unexpected result type for {query!r}: {result_type!r}")
PY
}

require_command curl
require_command python3

log "Checking Prometheus targets API at ${PROM_URL}"
targets_payload="$(fetch_json "${PROM_URL}/api/v1/targets")"
assert_targets_present "${targets_payload}"

for query in \
  'up{job=~"chainpulse-.*"}' \
  'sum(chainpulse_gateway_request_success)' \
  'sum(chainpulse_event_processor_event_processed)' \
  'sum(chainpulse_mq_dead_letter_queue_size)'
do
  log "Running query: ${query}"
  query_payload="$(fetch_json "${PROM_URL}/api/v1/query?query=$(python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "${query}")")"
  assert_query_success "${query_payload}" "${query}"
done

log "Prometheus live smoke passed"
