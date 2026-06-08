#!/usr/bin/env bash

set -euo pipefail

API_GATEWAY_URL="${API_GATEWAY_URL:-http://localhost:8080}"
API_SERVICE_URL="${API_SERVICE_URL:-http://localhost:8081}"
EVENT_PROCESSOR_URL="${EVENT_PROCESSOR_URL:-http://localhost:8082}"
PULLER_URL="${PULLER_URL:-http://localhost:8083}"
API_GATEWAY_AUTH_HEADER="${API_GATEWAY_AUTH_HEADER:-}"
API_SERVICE_AUTH_HEADER="${API_SERVICE_AUTH_HEADER:-}"
EVENT_PROCESSOR_AUTH_HEADER="${EVENT_PROCESSOR_AUTH_HEADER:-}"
PULLER_AUTH_HEADER="${PULLER_AUTH_HEADER:-}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-production.sh

Performs live production-oriented HTTP contract checks against a deployed
ChainPulse environment.

Environment variables:
  API_GATEWAY_URL       default: http://localhost:8080
  API_SERVICE_URL       default: http://localhost:8081
  EVENT_PROCESSOR_URL   default: http://localhost:8082
  PULLER_URL            default: http://localhost:8083
  API_GATEWAY_AUTH_HEADER     optional curl header such as 'X-API-Key: key'
  API_SERVICE_AUTH_HEADER     optional curl header such as 'X-API-Key: key'
  EVENT_PROCESSOR_AUTH_HEADER optional curl header such as 'X-API-Key: key'
  PULLER_AUTH_HEADER          optional curl header such as 'X-API-Key: key'
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-production] %s\n' "$*"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'production verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
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

log "Checking api-gateway health"
gateway_health="$(fetch "${API_GATEWAY_URL}/health" "${API_GATEWAY_AUTH_HEADER}")"
assert_contains "${gateway_health}" '"status":"healthy"' "api-gateway /health"

log "Checking api-gateway runtime summary"
gateway_summary="$(fetch "${API_GATEWAY_URL}/runtime/summary" "${API_GATEWAY_AUTH_HEADER}")"
assert_contains "${gateway_summary}" '"service":"api-gateway"' "api-gateway /runtime/summary"
assert_contains "${gateway_summary}" '"deployment_mode":"microservice"' "api-gateway /runtime/summary"
assert_contains "${gateway_summary}" '"runtime_mode":"runtime-wired"' "api-gateway /runtime/summary"
assert_contains "${gateway_summary}" '"query_bridge_posture":"query-bridge-ready"' "api-gateway /runtime/summary"
assert_contains "${gateway_summary}" '"security_posture":"gateway-security-ready"' "api-gateway /runtime/summary"

log "Checking api-gateway rollout readiness"
gateway_rollout="$(fetch "${API_GATEWAY_URL}/health/rollout" "${API_GATEWAY_AUTH_HEADER}")"
assert_contains "${gateway_rollout}" '"advisory"' "api-gateway /health/rollout"
assert_contains "${gateway_rollout}" '"ready":true' "api-gateway /health/rollout"

log "Checking api-service health"
service_health="$(fetch "${API_SERVICE_URL}/health" "${API_SERVICE_AUTH_HEADER}")"
assert_contains "${service_health}" '"status":"healthy"' "api-service /health"

log "Checking api-service runtime summary"
service_summary="$(fetch "${API_SERVICE_URL}/runtime/summary" "${API_SERVICE_AUTH_HEADER}")"
assert_contains "${service_summary}" '"service":"api-service"' "api-service /runtime/summary"
assert_contains "${service_summary}" '"deployment_mode":"microservice"' "api-service /runtime/summary"
assert_contains "${service_summary}" '"security_posture":"api-service-security-ready"' "api-service /runtime/summary"

log "Checking event-processor health"
processor_health="$(fetch "${EVENT_PROCESSOR_URL}/health" "${EVENT_PROCESSOR_AUTH_HEADER}")"
assert_contains "${processor_health}" '"status":"healthy"' "event-processor /health"

log "Checking event-processor runtime summary"
processor_summary="$(fetch "${EVENT_PROCESSOR_URL}/runtime/summary" "${EVENT_PROCESSOR_AUTH_HEADER}")"
assert_contains "${processor_summary}" '"service":"event-processor"' "event-processor /runtime/summary"
assert_contains "${processor_summary}" '"deployment_mode":"microservice"' "event-processor /runtime/summary"
assert_contains "${processor_summary}" '"runtime_mode":"runtime-wired"' "event-processor /runtime/summary"

log "Checking puller health"
puller_health="$(fetch "${PULLER_URL}/health" "${PULLER_AUTH_HEADER}")"
assert_contains "${puller_health}" '"status":"healthy"' "puller /health"

log "Checking puller runtime summary"
puller_summary="$(fetch "${PULLER_URL}/runtime/summary" "${PULLER_AUTH_HEADER}")"
assert_contains "${puller_summary}" '"service":"puller"' "puller /runtime/summary"
assert_contains "${puller_summary}" '"deployment_mode":"microservice"' "puller /runtime/summary"
assert_contains "${puller_summary}" '"runtime_mode":"runtime-wired"' "puller /runtime/summary"

log "Production verification passed"
