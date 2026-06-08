#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DASHBOARD_FILE="${ROOT_DIR}/monitoring/grafana/dashboards/chainpulse-indexer.json"
PROVIDER_FILE="${ROOT_DIR}/monitoring/grafana/dashboards/provider.yml"
DATASOURCE_FILE="${ROOT_DIR}/monitoring/grafana/datasources/prometheus.yml"

usage() {
  cat <<'EOF'
Usage: scripts/verify-grafana-blueprint-81.sh

Validates the blueprint 8.1 Grafana dashboard JSON and its provisioning files.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[verify-grafana-blueprint-81] %s\n' "$*"
}

assert_file_contains() {
  local file="$1"
  local needle="$2"
  local label="$3"
  if ! grep -Fq "${needle}" "${file}"; then
    printf 'grafana blueprint verification failed: %s missing "%s"\n' "${label}" "${needle}" >&2
    exit 1
  fi
}

require_file() {
  local file="$1"
  if [[ ! -f "${file}" ]]; then
    printf 'grafana blueprint verification failed: missing file %s\n' "${file}" >&2
    exit 1
  fi
}

require_file "${DASHBOARD_FILE}"
require_file "${PROVIDER_FILE}"
require_file "${DATASOURCE_FILE}"

log "Validating dashboard JSON structure"
python3 - "${DASHBOARD_FILE}" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as fh:
    payload = json.load(fh)

if payload.get("title") != "ChainPulse Blueprint 8.1 Local Debug Monitor":
    raise SystemExit("unexpected dashboard title")

panels = payload.get("panels", [])
if len(panels) < 12:
    raise SystemExit(f"expected at least 12 panels, got {len(panels)}")

titles = [panel.get("title", "") for panel in panels]
for expected in [
    "Performance | Event Throughput",
    "Performance | Query Cache Latency P95",
    "Resource | Memory Usage MB",
    "Business | Ownership Mode",
    "System Health | Service Availability",
]:
    if expected not in titles:
        raise SystemExit(f"missing dashboard panel {expected!r}")
PY

log "Checking Grafana provisioning files"
assert_file_contains "${PROVIDER_FILE}" "apiVersion: 1" "dashboard provider"
assert_file_contains "${PROVIDER_FILE}" "path: /etc/grafana/provisioning/dashboards" "dashboard provider"
assert_file_contains "${DATASOURCE_FILE}" "name: Prometheus" "datasource"
assert_file_contains "${DATASOURCE_FILE}" "uid: chainpulse-prometheus" "datasource"
assert_file_contains "${DATASOURCE_FILE}" "url: http://prometheus:9090" "datasource"

log "Grafana blueprint 8.1 dashboard verification passed"
