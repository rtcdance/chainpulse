#!/usr/bin/env bash

set -euo pipefail

PROFILE="minimal"

API_GATEWAY_PORT="${API_GATEWAY_PORT:-8080}"
API_SERVICE_PORT="${API_SERVICE_PORT:-8081}"
EVENT_PROCESSOR_PORT="${EVENT_PROCESSOR_PORT:-8082}"
PULLER_PORT="${PULLER_PORT:-8083}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-local-runnable-app.sh [--profile minimal|full]

Profiles:
  minimal  Verifies api-service + api-gateway
  full     Verifies api-service + api-gateway + event-processor + puller

Environment overrides:
  API_GATEWAY_PORT
  API_SERVICE_PORT
  EVENT_PROCESSOR_PORT
  PULLER_PORT
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ "${PROFILE}" != "minimal" && "${PROFILE}" != "full" ]]; then
  echo "Unsupported profile: ${PROFILE}" >&2
  usage
  exit 1
fi

log() {
  printf '[verify-local-app] %s\n' "$*"
}

fetch() {
  local url="$1"
  curl -fsS "${url}"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

verify_api_service() {
  local base="http://localhost:${API_SERVICE_PORT}"
  log "Checking api-service health"
  fetch "${base}/health" >/dev/null

  log "Checking api-service runtime summary"
  local summary
  summary="$(fetch "${base}/runtime/summary")"
  assert_contains "${summary}" "\"service\":\"api-service\"" "api-service runtime summary"
  assert_contains "${summary}" "\"query\"" "api-service runtime summary"
  assert_contains "${summary}" "\"security\"" "api-service runtime summary"
  assert_contains "${summary}" "\"security_posture\":\"api-service-security-unconfigured\"" "api-service runtime summary"
  assert_contains "${summary}" "\"auth_enabled\":false" "api-service runtime summary"
  assert_contains "${summary}" "\"rate_limit_enabled\":false" "api-service runtime summary"
}

verify_api_gateway() {
  local base="http://localhost:${API_GATEWAY_PORT}"
  log "Checking api-gateway health"
  fetch "${base}/health" >/dev/null

  log "Checking api-gateway runtime summary"
  local summary
  summary="$(fetch "${base}/runtime/summary")"
  assert_contains "${summary}" "\"service\":\"api-gateway\"" "api-gateway runtime summary"
  assert_contains "${summary}" "\"query_bridge_posture\"" "api-gateway runtime summary"
  assert_contains "${summary}" "\"upstream_query_health_state\"" "api-gateway runtime summary"
  assert_contains "${summary}" "\"security_posture\":\"gateway-security-unconfigured\"" "api-gateway runtime summary"
  assert_contains "${summary}" "\"auth_enabled\":false" "api-gateway runtime summary"
  assert_contains "${summary}" "\"rate_limit_enabled\":false" "api-gateway runtime summary"

  log "Checking api-gateway query forwarding"
  local query
  query="$(fetch "${base}/events?limit=5")"
  assert_contains "${query}" "\"data\"" "api-gateway query forwarding"
}

verify_event_processor() {
  local base="http://localhost:${EVENT_PROCESSOR_PORT}"
  log "Checking event-processor health"
  fetch "${base}/health" >/dev/null

  log "Checking event-processor runtime summary"
  local summary
  summary="$(fetch "${base}/runtime/summary")"
  assert_contains "${summary}" "\"service\":\"event-processor\"" "event-processor runtime summary"
  assert_contains "${summary}" "\"processor\"" "event-processor runtime summary"
  assert_contains "${summary}" "\"security\"" "event-processor runtime summary"
  assert_contains "${summary}" "\"security_posture\":\"event-processor-security-unconfigured\"" "event-processor runtime summary"
  assert_contains "${summary}" "\"auth_enabled\":false" "event-processor runtime summary"
  assert_contains "${summary}" "\"rate_limit_enabled\":false" "event-processor runtime summary"

  log "Checking event-processor runtime control"
  local control
  control="$(fetch "${base}/runtime/control")"
  assert_contains "${control}" "\"control\"" "event-processor runtime control"
  assert_contains "${control}" "\"target\":\"consume-loop-intake\"" "event-processor runtime control"
}

verify_puller() {
  local base="http://localhost:${PULLER_PORT}"
  log "Checking puller health"
  fetch "${base}/health" >/dev/null

  log "Checking puller runtime summary"
  local summary
  summary="$(fetch "${base}/runtime/summary")"
  assert_contains "${summary}" "\"service\":\"puller\"" "puller runtime summary"
  assert_contains "${summary}" "\"metrics\"" "puller runtime summary"
  assert_contains "${summary}" "\"security\"" "puller runtime summary"
  assert_contains "${summary}" "\"security_posture\":\"puller-security-unconfigured\"" "puller runtime summary"
  assert_contains "${summary}" "\"auth_enabled\":false" "puller runtime summary"
  assert_contains "${summary}" "\"rate_limit_enabled\":false" "puller runtime summary"

  log "Checking puller runtime control"
  local control
  control="$(fetch "${base}/runtime/control")"
  assert_contains "${control}" "\"control\"" "puller runtime control"
  assert_contains "${control}" "\"target\":\"polling-loop\"" "puller runtime control"
}

verify_api_service
verify_api_gateway

if [[ "${PROFILE}" == "full" ]]; then
  verify_event_processor
  verify_puller
fi

log "Verification passed for profile=${PROFILE}"
