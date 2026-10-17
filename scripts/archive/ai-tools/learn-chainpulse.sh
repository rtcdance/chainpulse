#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────
# learn-chainpulse.sh — Full-chain debugging environment
# ──────────────────────────────────────────────────────────
# One script: start Anvil → deploy EventEmitter → emit events
# → launch debugger. Zero local deps beyond Docker + Go.
#
# Usage:
#   bash scripts/learn-chainpulse.sh up      # Start + deploy + emit
#   bash scripts/learn-chainpulse.sh debug   # up + dlv with breakpoints
#   bash scripts/learn-chainpulse.sh replay  # Re-emit (contract stays)
#   bash scripts/learn-chainpulse.sh status  # Show Anvil + contract
#   bash scripts/learn-chainpulse.sh down    # Stop everything
# ──────────────────────────────────────────────────────────
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/lib/docker_acceptance.sh"

# ── Configuration ──────────────────────────────────────────
ANVIL_RPC="${ANVIL_RPC:-http://localhost:8545}"
ANVIL_SERVICE="${ANVIL_SERVICE:-anvil-ethereum}"
COMPOSE_FILE="${COMPOSE_FILE:-docker/docker-compose.yml}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-90}"
DOCKER_ACCEPTANCE_LOG_FN="log"

PRIVATE_KEY="${PRIVATE_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"
CONTRACT_STATE="${ROOT_DIR}/.learn-contract-state"

# ── Helpers ────────────────────────────────────────────────
log()   { printf '\033[36m[learn]\033[0m %s\n' "$*"; }
ok()    { printf '\033[32m  ✓ %s\033[0m\n' "$*"; }
fail()  { printf '\033[31m  ✗ %s\033[0m\n' "$*"; exit 1; }

usage() {
  cat <<'EOF'
Usage: bash scripts/learn-chainpulse.sh <command>

Commands:
  up      Start Anvil → compile → deploy → emit 9 events
  debug   up + launch delve with full learning breakpoints
  replay  Re-emit events to existing contract (no re-deploy)
  status  Show Anvil health + saved contract info
  down    Stop Anvil + clean up state

Environment:
  ANVIL_RPC      default: http://localhost:8545
  COMPOSE_FILE   default: docker/docker-compose.yml
EOF
}

chain_id() {
  curl -sS -X POST "${ANVIL_RPC}" \
    -H 'Content-Type: application/json' \
    --data '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' 2>/dev/null \
    | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'],16))" 2>/dev/null || echo "0"
}

# ── cmd_up ─────────────────────────────────────────────────
cmd_up() {
  docker_acceptance_require_docker

  log "Starting ${ANVIL_SERVICE} via docker compose..."
  docker compose -f "${ROOT_DIR}/${COMPOSE_FILE}" up -d --wait "${ANVIL_SERVICE}" 2>&1 | sed 's/^/  /'
  ok "Anvil ready at ${ANVIL_RPC} (chain ID: $(chain_id))"

  log "Compiling EventEmitter.sol..."
  docker run --rm \
    -v "${ROOT_DIR}/contracts:/contracts" \
    ghcr.io/foundry-rs/foundry:latest \
    forge build --root /contracts --out /contracts/out --evm-version cancun 2>&1 | sed 's/^/  /'

  local bytecode
  bytecode=$(python3 -c "
import json
with open('${ROOT_DIR}/contracts/out/EventEmitter.sol/EventEmitter.json') as f:
    print(json.load(f)['bytecode']['object'])
" 2>/dev/null) || fail "bytecode extraction failed"

  log "Deploying EventEmitter..."
  local deploy_out tx_hash contract_addr
  deploy_out=$(docker run --rm \
    ghcr.io/foundry-rs/foundry:latest \
    cast send --rpc-url "${ANVIL_RPC}" --private-key "${PRIVATE_KEY}" \
      --legacy --create "${bytecode}" 2>&1)

  tx_hash=$(echo "${deploy_out}" | grep -E "^transactionHash" | awk '{print $2}')
  contract_addr=$(echo "${deploy_out}" | grep -E "^contractAddress|^deployedTo" | awk '{print $2}')

  if [ -z "${contract_addr}" ]; then
    contract_addr=$(docker run --rm \
      ghcr.io/foundry-rs/foundry:latest \
      cast receipt --rpc-url "${ANVIL_RPC}" "${tx_hash}" contractAddress 2>/dev/null || echo "")
  fi

  [ -z "${contract_addr}" ] && fail "Could not get contract address"

  # Save state
  cat > "${CONTRACT_STATE}" <<STATE
ANVIL_RPC=${ANVIL_RPC}
CONTRACT_ADDR=${contract_addr}
DEPLOY_TX=${tx_hash}
CHAIN_ID=$(chain_id)
PRIVATE_KEY=${PRIVATE_KEY}
STATE
  chmod 600 "${CONTRACT_STATE}"
  ok "EventEmitter deployed at ${contract_addr}"

  cmd_replay
}

# ── cmd_replay ─────────────────────────────────────────────
cmd_replay() {
  if [ ! -f "${CONTRACT_STATE}" ]; then
    fail "No contract state. Run 'up' first."
  fi
  source "${CONTRACT_STATE}"

  docker_acceptance_require_docker
  [ "$(chain_id)" = "0" ] && fail "Anvil not reachable at ${ANVIL_RPC}"

  log "Emitting events to ${CONTRACT_ADDR}..."

  for i in 1 2 3; do
    local tx
    tx=$(docker run --rm \
      ghcr.io/foundry-rs/foundry:latest \
      cast send --rpc-url "${ANVIL_RPC}" --private-key "${PRIVATE_KEY}" \
        --legacy "${CONTRACT_ADDR}" "emitTransfer(address,uint256)" \
        "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" "$((i * 100))" 2>&1 \
      | grep "^transactionHash" | awk '{print $2}')
    ok "Transfer #${i}: ${tx}"
  done

  local tx_custom
  tx_custom=$(docker run --rm \
    ghcr.io/foundry-rs/foundry:latest \
    cast send --rpc-url "${ANVIL_RPC}" --private-key "${PRIVATE_KEY}" \
      --legacy "${CONTRACT_ADDR}" 'emitCustom(string)' \
      '"hello from learn-chainpulse"' 2>&1 \
    | grep "^transactionHash" | awk '{print $2}')
  ok "CustomEvent: ${tx_custom}"

  local tx_batch
  tx_batch=$(docker run --rm \
    ghcr.io/foundry-rs/foundry:latest \
    cast send --rpc-url "${ANVIL_RPC}" --private-key "${PRIVATE_KEY}" \
      --legacy "${CONTRACT_ADDR}" "emitBatch(uint256)" 5 2>&1 \
    | grep "^transactionHash" | awk '{print $2}')
  ok "Batch(5): ${tx_batch}"

  echo ""
  printf '\033[36m  9 events emitted — ChainPulse can now index them\033[0m\n'
  echo ""
  echo "  Set breakpoints:"
  echo "    pkg/plugins/pullers/https_jsonrpc_puller.go  ← real RPC calls"
  echo "    pkg/core/event_decoder.go                    ← ABI decoding"
  echo "    pkg/adapters/indexing/monolithic_memory_storage.go  ← storage"
  echo ""
  echo "  Or launch the debugger:"
  echo "    bash scripts/learn-chainpulse.sh debug"
}

# ── cmd_debug ──────────────────────────────────────────────
cmd_debug() {
  cmd_up

  log "Starting delve with learning breakpoints..."
  echo ""
  echo "╔════════════════════════════════════════════════════╗"
  echo "║  ChainPulse Full-Chain Debug Session              ║"
  echo "║                                                   ║"
  echo "║  9 events waiting at ${CONTRACT_ADDR}     ║"
  echo "║  RPC: ${ANVIL_RPC}                          ║"
  echo "║                                                   ║"
  echo "║  Debugger commands:                               ║"
  echo "║    break https_jsonrpc_puller.go:301              ║"
  echo "║    break eventbus.go:87                           ║"
  echo "║    break reorg_handler.go:128                     ║"
  echo "║    continue                                       ║"
  echo "╚════════════════════════════════════════════════════╝"
  echo ""

  cd "${ROOT_DIR}"
  dlv debug ./cmd/monolithic/chainpulse --init "${ROOT_DIR}/.dlv/learn-init.dlv" 2>/dev/null \
    || dlv debug ./cmd/monolithic/chainpulse
}

# ── cmd_status ─────────────────────────────────────────────
cmd_status() {
  docker_acceptance_require_docker

  local cid
  cid=$(docker compose -f "${ROOT_DIR}/${COMPOSE_FILE}" ps -q "${ANVIL_SERVICE}" 2>/dev/null || echo "")
  if [ -z "${cid}" ]; then
    echo "Anvil: stopped"
  else
    local cid_chain
    cid_chain=$(chain_id)
    echo "Anvil: running (chain ID: ${cid_chain})"
    echo "  ${ANVIL_RPC}"
  fi
  echo ""

  if [ -f "${CONTRACT_STATE}" ]; then
    source "${CONTRACT_STATE}"
    echo "Contract: ${CONTRACT_ADDR:-}"
    echo "  deploy tx: ${DEPLOY_TX:-}"
    echo "  chain ID:  ${CHAIN_ID:-}"
  else
    echo "Contract: not deployed"
    echo "  Run 'bash scripts/learn-chainpulse.sh up' to deploy"
  fi
}

# ── cmd_down ────────────────────────────────────────────────
cmd_down() {
  docker_acceptance_require_docker
  log "Stopping ${ANVIL_SERVICE}..."
  docker compose -f "${ROOT_DIR}/${COMPOSE_FILE}" rm -sf "${ANVIL_SERVICE}" 2>&1 | sed 's/^/  /' || true
  rm -f "${CONTRACT_STATE}"
  ok "Done"
}

# ── Main ───────────────────────────────────────────────────
case "${1:-help}" in
  up)     cmd_up ;;
  debug)  cmd_debug ;;
  replay) cmd_replay ;;
  status) cmd_status ;;
  down)   cmd_down ;;
  help|--help|-h) usage ;;
  *) echo "Unknown command: $1"; usage; exit 1 ;;
esac
