#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────
# dev.sh — ChainPulse Local Development Environment
# ──────────────────────────────────────────────────────────
# One script: infra up → build → chain up → debug / stop
#
# Usage:
#   bash scripts/dev/dev.sh start    # Minimal (Anvil + infra + chainpulse)
#   bash scripts/dev/dev.sh start:full    # All chains + infra + chainpulse
#   bash scripts/dev/dev.sh start:real    # Sepolia real chain + infra + chainpulse
#   bash scripts/dev/dev.sh stop          # Stop everything
#   bash scripts/dev/dev.sh status        # Show running state
#   bash scripts/dev/dev.sh restart       # Full restart
# ──────────────────────────────────────────────────────────
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${ROOT_DIR}/scripts/lib/docker_acceptance.sh" 2>/dev/null || true

# ── Configuration ──────────────────────────────────────────
CHAINPULSE_PORT="${CHAINPULSE_PORT:-8081}"
PG_PASSWORD="${POSTGRES_PASSWORD:-chainpulse_dev}"
COMPOSE_FILE="${ROOT_DIR}/docker/docker-compose.yml"
LOG_DIR="${LOG_DIR:-/tmp/chainpulse-dev}"
CHAINPULSE_BIN="${CHAINPULSE_BIN:-/tmp/chainpulse-monolithic}"

# Real Sepolia RPC (default)
SEPOLIA_RPC="${SEPOLIA_RPC:-https://1rpc.io/sepolia}"

# Known Sepolia contract with event activity
SEPOLIA_CONTRACT="${SEPOLIA_CONTRACT:-0xa95267db6d3e14b6ea5a06a091c1b3aedf4ba346}"

# ── Colors ─────────────────────────────────────────────────
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { printf "${CYAN}[dev]${NC} %s\n" "$*"; }
ok()    { printf "${GREEN}  ✓ %s${NC}\n" "$*"; }
warn()  { printf "${YELLOW}  ⚠ %s${NC}\n" "$*"; }
fail()  { printf "${RED}  ✗ %s${NC}\n" "$*"; exit 1; }

# ── Helper: wait for port ──────────────────────────────────
wait_for_port() {
  local name=$1 port=$2 timeout=${3:-30}
  info "Waiting for $name on port $port..."
  for i in $(seq 1 $timeout); do
    if lsof -i :$port -P -n 2>/dev/null | grep -q LISTEN; then
      ok "$name ready (port $port)"
      return 0
    fi
    sleep 1
  done
  warn "$name not ready after ${timeout}s"
  return 1
}

# ── Cmd: build ─────────────────────────────────────────────
cmd_build() {
  info "Building ChainPulse monolithic binary..."
  cd "${ROOT_DIR}"
  go build -o "${CHAINPULSE_BIN}" ./cmd/monolithic/chainpulse 2>&1
  ok "Built: ${CHAINPULSE_BIN} ($(stat -f%z "${CHAINPULSE_BIN}" 2>/dev/null | numfmt --to=iec 2>/dev/null || echo "?"))"
}

# ── Cmd: infra ─────────────────────────────────────────────
cmd_infra() {
  local mode="${1:-minimal}"

  info "Starting infrastructure..."

  if [ "$mode" = "full" ]; then
    docker compose -f "${COMPOSE_FILE}" up -d postgres redis mongodb \
      anvil-ethereum anvil-polygon anvil-bsc 2>&1
  else
    docker compose -f "${COMPOSE_FILE}" up -d postgres redis mongodb anvil-ethereum 2>&1
  fi

  # Wait for services
  wait_for_port "PostgreSQL" 5432 30
  wait_for_port "Redis" 6379 15
  wait_for_port "MongoDB" 27017 15

  if [ "$mode" != "real" ]; then
    wait_for_port "Anvil (eth)" 8545 30
  fi

  ok "Infrastructure ready"
}

# ── Cmd: start (Anvil mode) ────────────────────────────────
cmd_start() {
  local mode="${1:-minimal}"
  local anvil_port=8545
  local chain="ethereum"
  local rpc="http://localhost:${anvil_port}"
  local contract_addrs=""

  mkdir -p "${LOG_DIR}"

  # 1. Build
  cmd_build

  # 2. Start infra
  cmd_infra "$mode"

  # 3. Deploy EventEmitter.sol + emit events (Anvil mode)
    if [ "$mode" != "real" ]; then
    info "Compiling and deploying EventEmitter.sol..."
    cd "${ROOT_DIR}"
    npx --yes solc@0.8.26 --bin --abi \
      contracts/EventEmitter.sol -o /tmp/compiled 2>/dev/null

    local bytecode
    bytecode=$(cat /tmp/compiled/contracts_EventEmitter_sol_EventEmitter.bin 2>/dev/null || echo "")
    if [ -n "$bytecode" ]; then
      local deploy_out
      deploy_out=$(docker run --rm --entrypoint cast \
        ghcr.io/foundry-rs/foundry:latest send \
        --rpc-url "http://host.docker.internal:${anvil_port}" \
        --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
        --create "${bytecode}" 2>&1)
      local contract_addr
      contract_addr=$(echo "${deploy_out}" | grep "contractAddress" | awk '{print $2}')
      if [ -n "$contract_addr" ]; then
        ok "EventEmitter deployed: ${contract_addr}"
        contract_addrs="${contract_addr}"

        # Emit test events
        for val in 100 200 300; do
          docker run --rm --entrypoint cast ghcr.io/foundry-rs/foundry:latest send \
            --rpc-url "http://host.docker.internal:${anvil_port}" \
            --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
            "${contract_addr}" "emitTransfer(address,uint256)" \
            "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" "${val}" >/dev/null 2>&1
        done
        ok "3 Transfer events emitted"
      fi
    else
      warn "Contract compilation failed, indexing without events"
    fi
  fi

  # 4. Start ChainPulse
  cmd_start_chainpulse "$mode" "$rpc" "$chain" "$contract_addrs"

  ok "Development environment ready!"
  echo ""
  echo "  API:        http://localhost:${CHAINPULSE_PORT}"
  echo "  Health:     http://localhost:${CHAINPULSE_PORT}/health"
  echo "  GraphQL:    http://localhost:${CHAINPULSE_PORT}/graphql"
  echo "  Metrics:    http://localhost:${CHAINPULSE_PORT}/metrics"
  echo ""
  echo "  Debug with: bash scripts/dev/dev.sh debug"
  echo "  Stop with:  bash scripts/dev/dev.sh stop"
}

# ── Cmd: start:real (Sepolia mode) ─────────────────────────
cmd_start_real() {
  mkdir -p "${LOG_DIR}"

  # Get latest Sepolia block for START_BLOCK
  local latest
  latest=$(curl -s -m 10 -X POST "${SEPOLIA_RPC}" \
    -H 'Content-Type: application/json' \
    --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    2>/dev/null | python3 -c "import sys,json;print(int(json.load(sys.stdin)['result'],16))" 2>/dev/null || echo "0")
  local start=$((latest > 2000 ? latest - 2000 : 0))

  info "Sepolia latest block: ${latest} (starting from ${start})"

  cmd_build
  cmd_infra "minimal"

  cmd_start_chainpulse "real" "${SEPOLIA_RPC}" "sepolia" "${SEPOLIA_CONTRACT}" "${start}"

  ok "Real-chain development environment ready!"
  echo ""
  echo "  API:        http://localhost:${CHAINPULSE_PORT}"
  echo "  Health:     http://localhost:${CHAINPULSE_PORT}/health"
  echo "  Chain:      Sepolia (real)"
  echo "  Contract:   ${SEPOLIA_CONTRACT}"
}

# ── Shared: start chainpulse ───────────────────────────────
cmd_start_chainpulse() {
  local mode=$1 rpc=$2 chain=$3 contract=$4 start_block=${5:-0}

  info "Starting ChainPulse monothic (mode: ${mode})..."

  export CHAINS="${chain}"
  export BLOCKCHAIN_NODE_URLS="${rpc}"
  export DATA_PULLER_TYPE=https-jsonrpc
  export DATABASE_TYPE=postgres
  export DATABASE_URL="postgres://chainpulse:${PG_PASSWORD}@localhost:5432/chainpulse?sslmode=disable"
  export CACHE_TYPE=redis
  export CACHE_CONNECTION_URL=localhost:6379
  export MQ_TYPE=redis
  export MQ_CONNECTION_URL=localhost:6379
  export DEPLOYMENT_MODE=monolithic
  export LOG_LEVEL=info
  export API_PORT="${CHAINPULSE_PORT}"

  if [ -n "$contract" ]; then
    export CONTRACT_ADDRESSES="${contract}"
  fi

  if [ "$start_block" -gt 0 ]; then
    export START_BLOCK="${start_block}"
  fi

  # Kill any existing instance
  pkill -f "chainpulse-monolithic" 2>/dev/null || true
  sleep 1

  "${CHAINPULSE_BIN}" > "${LOG_DIR}/chainpulse.log" 2>&1 &
  local pid=$!

  # Wait for health
  for i in $(seq 1 15); do
    if curl -s -m 2 "http://localhost:${CHAINPULSE_PORT}/health" 2>/dev/null | python3 -c "import sys,json;d=json.load(sys.stdin);sys.exit(0 if d.get('status')=='healthy' else 1)" 2>/dev/null; then
      ok "ChainPulse healthy (PID: ${pid})"
      return 0
    fi
    sleep 2
  done

  warn "ChainPulse health check timed out — check ${LOG_DIR}/chainpulse.log"
  return 1
}

# ── Cmd: stop ──────────────────────────────────────────────
cmd_stop() {
  info "Stopping development environment..."

  pkill -f "chainpulse-monolithic" 2>/dev/null && ok "ChainPulse stopped" || true

  docker compose -f "${COMPOSE_FILE}" rm -sf anvil-ethereum anvil-polygon anvil-bsc 2>/dev/null || true
  ok "Anvil chains stopped"

  warn "PostgreSQL, Redis, MongoDB left running (they keep data)"
  warn "  To also stop infra: docker compose -f ${COMPOSE_FILE} rm -sf postgres redis mongodb"
  warn "  To also remove data: docker compose -f ${COMPOSE_FILE} down -v postgres redis mongodb"
}

# ── Cmd: status ────────────────────────────────────────────
cmd_status() {
  echo "=== ChainPulse Dev Environment ==="
  echo ""

  if pgrep -f "chainpulse-monolithic" >/dev/null 2>&1; then
    local health
    health=$(curl -s -m 3 "http://localhost:${CHAINPULSE_PORT}/health" 2>/dev/null | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('status','?'))" 2>/dev/null || echo "unreachable")
    echo "ChainPulse: ${GREEN}running${NC} (health: ${health})"
    echo "  PID:       $(pgrep -f 'chainpulse-monolithic')"
    echo "  API:       http://localhost:${CHAINPULSE_PORT}"
    echo "  Log:       ${LOG_DIR}/chainpulse.log"
  else
    echo "ChainPulse: ${RED}stopped${NC}"
  fi
  echo ""

  echo "Docker:"
  for svc in postgres redis mongodb; do
    local status
    status=$(docker compose -f "${COMPOSE_FILE}" ps "${svc}" 2>/dev/null | tail -1 | awk '{print $5}')
    echo "  ${svc}: ${status:-not found}"
  done

  echo ""
  echo "Environment:"
  echo "  CHAINPULSE_PORT:  ${CHAINPULSE_PORT}"
  echo "  LOG_DIR:          ${LOG_DIR}"
  echo "  CHAINPULSE_BIN:   ${CHAINPULSE_BIN}"
}

# ── Cmd: debug ─────────────────────────────────────────────
cmd_debug() {
  if ! pgrep -f "chainpulse-monolithic" >/dev/null 2>&1; then
    warn "ChainPulse is not running. Start it first:"
    warn "  bash scripts/dev/dev.sh start"
    exit 1
  fi

  local pid
  pid=$(pgrep -f "chainpulse-monolithic")
  info "Attaching delve to PID ${pid}..."

  echo ""
  echo "╔════════════════════════════════════════════════════╗"
  echo "║  Delve Debugger Attached                           ║"
  echo "║                                                    ║"
  echo "║  Useful breakpoints:                               ║"
  echo "║                                                    ║"
  echo "║  pkg/plugins/pullers/https_jsonrpc_puller.go:497   ║"
  echo "║    → getLogs() - see real eth_getLogs calls        ║"
  echo "║                                                    ║"
  echo "║  pkg/core/event_decoder.go:30                      ║"
  echo "║    → DecodeEventData() - ABI decoding              ║"
  echo "║                                                    ║"
  echo "║  pkg/services/indexing/chain_indexer.go:134        ║"
  echo "║    → IndexBlocks() - block indexing                ║"
  echo "║                                                    ║"
  echo "║  pkg/services/reorg/reorg_handler.go:190           ║"
  echo "║    → HandleReorg() - reorg detection               ║"
  echo "╚════════════════════════════════════════════════════╝"
  echo ""

  dlv attach "${pid}" --headless --listen=:2345 --api-version=2 --accept-multiclient
}

# ── Cmd: logs ──────────────────────────────────────────────
cmd_logs() {
  if [ -f "${LOG_DIR}/chainpulse.log" ]; then
    tail -f "${LOG_DIR}/chainpulse.log"
  else
    fail "No log file found at ${LOG_DIR}/chainpulse.log"
  fi
}

# ── Cmd: replay ────────────────────────────────────────────
cmd_replay() {
  # Re-emit events to existing contract (if any)
  local state_file="${ROOT_DIR}/.learn-contract-state"
  if [ ! -f "${state_file}" ]; then
    fail "No contract state found. Run 'start' or 'learn' first."
  fi

  source "${state_file}"
  bash "${ROOT_DIR}/scripts/learn-chainpulse.sh" replay
}

# ── Main ───────────────────────────────────────────────────
case "${1:-help}" in
  start)
    cmd_start "${2:-minimal}"
    ;;
  start:full)
    cmd_start "full"
    ;;
  start:real)
    cmd_start_real
    ;;
  stop)
    cmd_stop
    ;;
  restart)
    cmd_stop
    sleep 2
    cmd_start "${2:-minimal}"
    ;;
  restart:full)
    cmd_stop
    sleep 2
    cmd_start "full"
    ;;
  restart:real)
    cmd_stop
    sleep 2
    cmd_start_real
    ;;
  status)
    cmd_status
    ;;
  debug)
    cmd_debug
    ;;
  logs)
    cmd_logs
    ;;
  build)
    cmd_build
    ;;
  replay)
    cmd_replay
    ;;
  help|--help|-h)
    cat <<'EOF'
Usage: bash scripts/dev/dev.sh <command>

Environment:
  CHAINPULSE_PORT   default: 8081 (nginx uses 8080)
  SEPOLIA_RPC       default: https://1rpc.io/sepolia
  LOG_DIR           default: /tmp/chainpulse-dev
  CHAINPULSE_BIN    default: /tmp/chainpulse-monolithic

Commands:
  start             Start minimal environment (Anvil + infra + chainpulse)
  start:full        Start multi-chain Anvil environment
  start:real        Start with real Sepolia testnet
  stop              Stop chainpulse (keeps infra running)
  restart           Full restart
  status            Show running state
  debug             Attach delve debugger to running chainpulse
  logs              Follow chainpulse logs
  build             Build chainpulse binary only
  replay            Re-emit events to existing contract
EOF
    ;;
  *)
    echo "Unknown command: $1"
    echo "Usage: bash scripts/dev/dev.sh <start|stop|status|debug|logs|build>"
    exit 1
    ;;
esac
