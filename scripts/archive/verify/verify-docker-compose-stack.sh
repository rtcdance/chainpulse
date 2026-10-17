#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker/docker-compose.dev.yml}"

usage() {
  cat <<'EOF'
Usage: scripts/verify-docker-compose-stack.sh

Performs a lightweight docker-compose stack verification by checking:
  1. the compose file exists
  2. required infrastructure services are declared
  3. observability services are declared when expected
  4. docker compose config can resolve the declared service set
  5. repo-level runnable foreground services are still described in runbooks

This script is intentionally lightweight and does not start containers.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-compose-stack] %s\n' "$*"
}

assert_file_contains() {
  local file="$1"
  local needle="$2"
  local label="$3"
  if ! grep -Fq "${needle}" "${file}"; then
    printf 'compose verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

assert_output_contains() {
  local haystack="$1"
  local needle="$2"
  local label="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    printf 'compose verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

COMPOSE_PATH="${ROOT_DIR}/${COMPOSE_FILE}"

if [[ ! -f "${COMPOSE_PATH}" ]]; then
  printf 'compose verification failed: missing compose file %s\n' "${COMPOSE_PATH}" >&2
  exit 1
fi

log "Checking infrastructure services in ${COMPOSE_FILE}"
assert_file_contains "${COMPOSE_PATH}" "postgres:" "compose infrastructure"
assert_file_contains "${COMPOSE_PATH}" "redis:" "compose infrastructure"
assert_file_contains "${COMPOSE_PATH}" "kafka:" "compose infrastructure"

if [[ "${COMPOSE_FILE}" == "docker/docker-compose.dev.yml" || "${COMPOSE_FILE}" == "docker/docker-compose.microservices.yml" ]]; then
  log "Checking observability services in ${COMPOSE_FILE}"
  assert_file_contains "${COMPOSE_PATH}" "prometheus:" "compose observability"
  assert_file_contains "${COMPOSE_PATH}" "grafana:" "compose observability"
  assert_file_contains "${COMPOSE_PATH}" "jaeger:" "compose observability"
  assert_file_contains "${COMPOSE_PATH}" "./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro" "compose prometheus mount"
fi

log "Checking resolved compose service set"
services_output="$(
  cd "${ROOT_DIR}"
  docker compose -f "${COMPOSE_FILE}" config --services 2>/dev/null
)"
assert_output_contains "${services_output}" "postgres" "resolved compose services"
assert_output_contains "${services_output}" "redis" "resolved compose services"
assert_output_contains "${services_output}" "kafka" "resolved compose services"

if [[ "${COMPOSE_FILE}" == "docker/docker-compose.dev.yml" || "${COMPOSE_FILE}" == "docker/docker-compose.microservices.yml" ]]; then
  assert_output_contains "${services_output}" "prometheus" "resolved compose services"
  assert_output_contains "${services_output}" "grafana" "resolved compose services"
  assert_output_contains "${services_output}" "jaeger" "resolved compose services"
fi

if [[ "${COMPOSE_FILE}" == "docker/docker-compose.yml" ]]; then
  assert_output_contains "${services_output}" "chainpulse" "resolved compose services"
fi

if [[ "${COMPOSE_FILE}" == "docker/docker-compose.microservices.yml" ]]; then
  assert_output_contains "${services_output}" "api-gateway" "resolved compose services"
  assert_output_contains "${services_output}" "api-service" "resolved compose services"
  assert_output_contains "${services_output}" "event-processor" "resolved compose services"
  assert_output_contains "${services_output}" "puller" "resolved compose services"
fi

log "Checking repo-level runnable documentation mentions foreground services"
assert_file_contains "${ROOT_DIR}/docs/project/RUNNABLE_APP.md" "api-gateway" "runnable app doc"
assert_file_contains "${ROOT_DIR}/docs/project/RUNNABLE_APP.md" "api-service" "runnable app doc"
assert_file_contains "${ROOT_DIR}/docs/project/RUNNABLE_APP.md" "event-processor" "runnable app doc"
assert_file_contains "${ROOT_DIR}/docs/project/RUNNABLE_APP.md" "puller" "runnable app doc"

log "Docker-compose stack verification passed for ${COMPOSE_FILE}"
