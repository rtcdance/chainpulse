#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXIT_CODE=0

log() {
  printf '[verify-production-readiness] %s\n' "$*"
}

warn() {
  printf '[verify-production-readiness] WARN: %s\n' "$*" >&2
  EXIT_CODE=1
}

log "Running production readiness checks..."

# Check required scripts exist
for script in verify-production.sh verify-staging.sh deploy-staging.sh run-k8s-deploy.sh rollback-drill.sh soak-check.sh; do
  if [[ -x "${ROOT_DIR}/scripts/${script}" || -f "${ROOT_DIR}/scripts/${script}" ]]; then
    log "✓ scripts/${script} exists"
  else
    warn "✗ scripts/${script} missing"
  fi
done

# Check K8s manifests are valid
if command -v kubectl >/dev/null 2>&1; then
  for overlay in monolithic microservice; do
    if [[ -d "${ROOT_DIR}/k8s/overlays/${overlay}" ]]; then
      log "✓ k8s/overlays/${overlay} exists"
    else
      warn "✗ k8s/overlays/${overlay} missing"
    fi
  done
fi

# Check monitoring configs
for cfg in monitoring/prometheus/prometheus.yml monitoring/grafana/dashboards/chainpulse-indexer.json monitoring/alertmanager/alertmanager.yml; do
  if [[ -f "${ROOT_DIR}/${cfg}" ]]; then
    log "✓ ${cfg} exists"
  else
    warn "✗ ${cfg} missing"
  fi
done

# Check deployment documentation
for doc in docs/deployment/go-live-blockers.md docs/deployment/production-checklist.md docs/operations/SECRETS.md; do
  if [[ -f "${ROOT_DIR}/${doc}" ]]; then
    log "✓ ${doc} exists"
  else
    warn "✗ ${doc} missing"
  fi
done

# Run Go vet
if command -v go >/dev/null 2>&1; then
  if go vet ./... 2>/dev/null; then
    log "✓ go vet passed"
  else
    warn "✗ go vet failed"
  fi
else
  warn "✗ go not found, skipping vet check"
fi

# Verify Dockerfiles exist
for df in docker/Dockerfile docker/Dockerfile.microservices docker/Dockerfile.prebuilt frontend/Dockerfile; do
  if [[ -f "${ROOT_DIR}/${df}" ]]; then
    log "✓ ${df} exists"
  else
    warn "✗ ${df} missing"
  fi
done

if [[ ${EXIT_CODE} -ne 0 ]]; then
  warn "Production readiness check completed with ${EXIT_CODE} warnings"
else
  log "Production readiness check passed"
fi

exit ${EXIT_CODE}