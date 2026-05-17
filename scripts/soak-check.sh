#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

API_GATEWAY_URL="${API_GATEWAY_URL:-http://localhost:8080}"
API_SERVICE_URL="${API_SERVICE_URL:-http://localhost:8081}"
EVENT_PROCESSOR_URL="${EVENT_PROCESSOR_URL:-http://localhost:8082}"
PULLER_URL="${PULLER_URL:-http://localhost:8083}"
API_GATEWAY_AUTH_HEADER="${API_GATEWAY_AUTH_HEADER:-}"
API_SERVICE_AUTH_HEADER="${API_SERVICE_AUTH_HEADER:-}"
EVENT_PROCESSOR_AUTH_HEADER="${EVENT_PROCESSOR_AUTH_HEADER:-}"
PULLER_AUTH_HEADER="${PULLER_AUTH_HEADER:-}"

DURATION_SECONDS="${DURATION_SECONDS:-1800}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-60}"
RUN_PRECHECK="${RUN_PRECHECK:-0}"
RUN_POSTCHECK="${RUN_POSTCHECK:-0}"
SOAK_LABEL="${SOAK_LABEL:-production}"

PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
SAMPLE_PROMETHEUS="${SAMPLE_PROMETHEUS:-1}"
PROMETHEUS_MIN_EPS="${PROMETHEUS_MIN_EPS:-0}"
PROMETHEUS_MAX_ERROR_RATE="${PROMETHEUS_MAX_ERROR_RATE:-}"

usage() {
  cat <<'EOF'
Usage: scripts/soak-check.sh

Runs the repository soak-check gate for a deployed ChainPulse environment.

Environment variables:
  DURATION_SECONDS      total soak window in seconds (default: 1800)
  INTERVAL_SECONDS      sample interval in seconds (default: 60)
  RUN_PRECHECK          set to 1 to run scripts/verify-production.sh before soak
  RUN_POSTCHECK         set to 1 to run scripts/verify-production.sh after soak
  SOAK_LABEL            environment label for reporting (default: production)
  PROMETHEUS_URL        Prometheus API base URL (default: http://localhost:9090)
  SAMPLE_PROMETHEUS     set to 1 to sample Prometheus metrics each cycle (default: 1)
  PROMETHEUS_MIN_EPS    minimum acceptable events per second (default: 0)
  PROMETHEUS_MAX_ERROR_RATE  maximum acceptable error rate (default: unset)
  API_GATEWAY_URL       default: http://localhost:8080
  API_SERVICE_URL       default: http://localhost:8081
  EVENT_PROCESSOR_URL   default: http://localhost:8082
  PULLER_URL            default: http://localhost:8083
  API_GATEWAY_AUTH_HEADER     optional curl header such as 'X-API-Key: key'
  API_SERVICE_AUTH_HEADER     optional curl header such as 'X-API-Key: key'
  EVENT_PROCESSOR_AUTH_HEADER optional curl header such as 'X-API-Key: key'
  PULLER_AUTH_HEADER          optional curl header such as 'X-API-Key: key'

This script does not generate synthetic load. It repeatedly samples health and
runtime contract endpoints across the soak window and fails if stability
regresses.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[soak-check] %s\n' "$*"
}

fail() {
  printf 'soak check failed: %s\n' "$*" >&2
  exit 1
}

assert_positive_integer() {
  local name="$1"
  local value="$2"
  if ! [[ "${value}" =~ ^[0-9]+$ ]] || [[ "${value}" -le 0 ]]; then
    fail "${name} must be a positive integer"
  fi
}

assert_toggle() {
  local name="$1"
  local value="$2"
  if [[ "${value}" != "0" && "${value}" != "1" ]]; then
    fail "${name} must be 0 or 1"
  fi
}

fetch() {
  local url="$1"
  local auth_header="${2:-}"
  if [[ -n "${auth_header}" ]]; then
    curl -fsS -H "${auth_header}" "${url}"
    return
  fi
  curl -fsS "${url}"
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

sample_prometheus_metrics() {
  local label="${1:-sample}"

  local query_url="${PROMETHEUS_URL}/api/v1/query"

  local eps
  eps="$(curl -fsS --data-urlencode 'query=rate(chainpulse_events_processed_total[1m])' "${query_url}" 2>/dev/null || true)"

  local errors
  errors="$(curl -fsS --data-urlencode 'query=rate(chainpulse_errors_total[1m])' "${query_url}" 2>/dev/null || true)"

  local latency_p95
  latency_p95="$(curl -fsS --data-urlencode 'query=histogram_quantile(0.95, rate(chainpulse_api_request_duration_seconds_bucket[1m]))' "${query_url}" 2>/dev/null || true)"

  log "Prometheus metrics [${label}]"
  log "  events/sec: ${eps}"
  log "  errors/sec: ${errors}"
  log "  api p95 latency: ${latency_p95}"

  if [[ -n "${PROMETHEUS_MIN_EPS}" && "${PROMETHEUS_MIN_EPS}" != "0" ]]; then
    local eps_val
    eps_val="$(echo "${eps}" | grep -o '"value":\[[^,]*,"[^"]*"\]' | grep -o '"[^"]*"\]' | tr -d '"]' || true)"
    if [[ -n "${eps_val}" ]] && (( $(echo "${eps_val} < ${PROMETHEUS_MIN_EPS}" | bc -l 2>/dev/null || echo 0) )); then
      fail "events/sec ${eps_val} below minimum ${PROMETHEUS_MIN_EPS}"
    fi
  fi

  if [[ -n "${PROMETHEUS_MAX_ERROR_RATE}" ]]; then
    local error_val
    error_val="$(echo "${errors}" | grep -o '"value":\[[^,]*,"[^"]*"\]' | grep -o '"[^"]*"\]' | tr -d '"]' || true)"
    if [[ -n "${error_val}" ]] && (( $(echo "${error_val} > ${PROMETHEUS_MAX_ERROR_RATE}" | bc -l 2>/dev/null || echo 0) )); then
      fail "error rate ${error_val} exceeds maximum ${PROMETHEUS_MAX_ERROR_RATE}"
    fi
  fi
}

sample_service() {
  local base_url="$1"
  local health_path="$2"
  local runtime_path="$3"
  local service="$4"
  local auth_header="$5"

  local health
  local runtime
  health="$(fetch "${base_url}${health_path}" "${auth_header}")"
  runtime="$(fetch "${base_url}${runtime_path}" "${auth_header}")"

  assert_contains "${health}" '"status":"healthy"' "${service} ${health_path}"
  assert_contains "${runtime}" "\"service\":\"${service}\"" "${service} ${runtime_path}"
  assert_contains "${runtime}" '"deployment_mode":"microservice"' "${service} ${runtime_path}"
}

sample_window() {
  sample_service "${API_GATEWAY_URL}" "/health" "/runtime/summary" "api-gateway" "${API_GATEWAY_AUTH_HEADER}"
  gateway_runtime="$(fetch "${API_GATEWAY_URL}/runtime/summary" "${API_GATEWAY_AUTH_HEADER}")"
  assert_contains "${gateway_runtime}" '"runtime_mode":"runtime-wired"' "api-gateway /runtime/summary"
  assert_contains "${gateway_runtime}" '"security_posture":"gateway-security-ready"' "api-gateway /runtime/summary"

  sample_service "${API_SERVICE_URL}" "/health" "/runtime/summary" "api-service" "${API_SERVICE_AUTH_HEADER}"
  api_runtime="$(fetch "${API_SERVICE_URL}/runtime/summary" "${API_SERVICE_AUTH_HEADER}")"
  assert_contains "${api_runtime}" '"security_posture":"api-service-security-ready"' "api-service /runtime/summary"

  sample_service "${EVENT_PROCESSOR_URL}" "/health" "/runtime/summary" "event-processor" "${EVENT_PROCESSOR_AUTH_HEADER}"
  processor_runtime="$(fetch "${EVENT_PROCESSOR_URL}/runtime/summary" "${EVENT_PROCESSOR_AUTH_HEADER}")"
  assert_contains "${processor_runtime}" '"runtime_mode":"runtime-wired"' "event-processor /runtime/summary"

  sample_service "${PULLER_URL}" "/health" "/runtime/summary" "puller" "${PULLER_AUTH_HEADER}"
  puller_runtime="$(fetch "${PULLER_URL}/runtime/summary" "${PULLER_AUTH_HEADER}")"
  assert_contains "${puller_runtime}" '"runtime_mode":"runtime-wired"' "puller /runtime/summary"
}

assert_positive_integer "DURATION_SECONDS" "${DURATION_SECONDS}"
assert_positive_integer "INTERVAL_SECONDS" "${INTERVAL_SECONDS}"
assert_toggle "RUN_PRECHECK" "${RUN_PRECHECK}"
assert_toggle "RUN_POSTCHECK" "${RUN_POSTCHECK}"
assert_toggle "SAMPLE_PROMETHEUS" "${SAMPLE_PROMETHEUS}"

if [[ "${INTERVAL_SECONDS}" -gt "${DURATION_SECONDS}" ]]; then
  fail "INTERVAL_SECONDS must be less than or equal to DURATION_SECONDS"
fi

samples=$(( (DURATION_SECONDS + INTERVAL_SECONDS - 1) / INTERVAL_SECONDS ))

log "Starting soak check"
log "Environment: ${SOAK_LABEL}"
log "Duration seconds: ${DURATION_SECONDS}"
log "Interval seconds: ${INTERVAL_SECONDS}"
log "Planned samples: ${samples}"

if [[ "${RUN_PRECHECK}" == "1" ]]; then
  log "Running pre-soak production verification"
  run_verify_production
fi

for ((sample=1; sample<=samples; sample++)); do
  log "Running sample ${sample}/${samples}"
  sample_window
  if [[ "${SAMPLE_PROMETHEUS}" == "1" ]]; then
    sample_prometheus_metrics "${sample}/${samples}"
  fi
  if [[ "${sample}" -lt "${samples}" ]]; then
    sleep "${INTERVAL_SECONDS}"
  fi
done

if [[ "${RUN_POSTCHECK}" == "1" ]]; then
  log "Running post-soak production verification"
  run_verify_production
fi

log "Soak check passed"
