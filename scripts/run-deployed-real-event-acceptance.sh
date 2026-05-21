#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"
API_URL="${API_URL:-http://127.0.0.1:8080}"
EXPECT_API="${EXPECT_API:-1}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-30}"

usage() {
  cat <<'EOF'
Usage: scripts/run-deployed-real-event-acceptance.sh

Inject a real on-chain event after deployment and verify it.

Environment variables:
  RPC_URL     default: http://127.0.0.1:8545
  API_URL     default: http://127.0.0.1:8080
  EXPECT_API  default: 1 (set 0 to validate chain-side only)
  WAIT_TIMEOUT_SECONDS default: 30
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

log() {
  printf '[real-event-acceptance] %s\n' "$*"
}

wait_for_rpc() {
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))
  while (( SECONDS < deadline )); do
    if curl -sS -X POST "${RPC_URL}" \
      -H 'Content-Type: application/json' \
      --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

if ! wait_for_rpc; then
  log "ERROR: rpc endpoint is unavailable: ${RPC_URL}"
  exit 1
fi

if [[ "${EXPECT_API}" == "1" ]]; then
  if ! curl -fsS "${API_URL}/runtime/summary" >/dev/null 2>&1; then
    log "ERROR: api runtime summary is unavailable: ${API_URL}/runtime/summary"
    exit 1
  fi
fi

log "running deployed real event acceptance via go tool"
cd "${ROOT_DIR}"
EXPECT_API="${EXPECT_API}" RPC_URL="${RPC_URL}" API_URL="${API_URL}" go run ./cmd/tools/deployed-real-event-acceptance
