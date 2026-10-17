#!/usr/bin/env bash

set -euo pipefail

NAMESPACE="${NAMESPACE:-chainpulse}"
EXIT_CODE=0

log() {
  printf '[verify-staging] %s\n' "$*"
}

fail() {
  printf '[verify-staging] FAIL: %s\n' "$*" >&2
  EXIT_CODE=1
}

log "Verifying staging deployment in namespace ${NAMESPACE}..."

require_kubectl() {
  if ! command -v kubectl >/dev/null 2>&1; then
    log "kubectl not found, skipping cluster checks"
    exit 0
  fi
}

require_kubectl

log "Pods:"
kubectl get pods -n "${NAMESPACE}" -o wide || true

log "Services:"
kubectl get svc -n "${NAMESPACE}" || true

log "Deployments:"
kubectl get deployments -n "${NAMESPACE}" || true

# Check deployments are ready
for dep in postgres redis zookeeper kafka chainpulse-monolithic chainpulse-microservice; do
  if kubectl get deployment "${dep}" -n "${NAMESPACE}" >/dev/null 2>&1; then
    ready=$(kubectl get deployment "${dep}" -n "${NAMESPACE}" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    if [[ "${ready}" -gt 0 ]]; then
      log "✓ deployment/${dep} ready (${ready} replicas)"
    else
      fail "✗ deployment/${dep} not ready"
    fi
  fi
done

# Check PDBs exist
for pdb in chainpulse-monolithic-pdb chainpulse-microservice-pdb postgres-pdb redis-pdb kafka-pdb zookeeper-pdb mongodb-pdb; do
  if kubectl get pdb "${pdb}" -n "${NAMESPACE}" >/dev/null 2>&1; then
    log "✓ pdb/${pdb} exists"
  else
    log "⚠ pdb/${pdb} not found (may be in overlay only)"
  fi
done

if [[ ${EXIT_CODE} -ne 0 ]]; then
  fail "Staging verification completed with failures"
else
  log "Staging verification passed"
fi

exit ${EXIT_CODE}