#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROM_CONFIG_FILE="${ROOT_DIR}/monitoring/prometheus/prometheus.yml"
ALERT_RULES_FILE="${ROOT_DIR}/monitoring/prometheus/alerts/chainpulse.yml"
DEV_COMPOSE_FILE="${ROOT_DIR}/docker/docker-compose.dev.yml"
MICROSERVICES_COMPOSE_FILE="${ROOT_DIR}/docker/docker-compose.microservices.yml"

usage() {
  cat <<'EOF'
Usage: scripts/verify-prometheus-scrape-baseline.sh

Validates the repository Prometheus scrape baseline by checking:
  1. Prometheus config and alert rules files exist
  2. expected scrape jobs are declared
  3. expected alert rules are declared
  4. docker compose files mount the real Prometheus config path
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-prometheus-scrape] %s\n' "$*"
}

require_file() {
  local path="$1"
  if [[ ! -f "${path}" ]]; then
    printf 'prometheus scrape verification failed: missing file %s\n' "${path}" >&2
    exit 1
  fi
}

assert_file_contains() {
  local file="$1"
  local needle="$2"
  local label="$3"
  if ! grep -Fq "${needle}" "${file}"; then
    printf 'prometheus scrape verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

require_file "${PROM_CONFIG_FILE}"
require_file "${ALERT_RULES_FILE}"
require_file "${DEV_COMPOSE_FILE}"
require_file "${MICROSERVICES_COMPOSE_FILE}"

log "Checking Prometheus scrape jobs"
for job in \
  "job_name: 'chainpulse-monolithic'" \
  "job_name: 'chainpulse-puller'" \
  "job_name: 'chainpulse-event-processor'" \
  "job_name: 'chainpulse-api-gateway'" \
  "job_name: 'chainpulse-api-service'"
do
  assert_file_contains "${PROM_CONFIG_FILE}" "${job}" "prometheus scrape config"
done

log "Checking Prometheus alert rules"
for rule in \
  "alert: HighEventFailureRate" \
  "alert: HighDLQDepth" \
  "alert: SlowQueryLatency" \
  "alert: ServiceDown"
do
  assert_file_contains "${ALERT_RULES_FILE}" "${rule}" "prometheus alert rules"
done

log "Checking compose mounts reference the real Prometheus config path"
for compose_file in "${DEV_COMPOSE_FILE}" "${MICROSERVICES_COMPOSE_FILE}"; do
  assert_file_contains "${compose_file}" "./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro" "${compose_file}"
done

log "Prometheus scrape baseline verification passed"
