#!/usr/bin/env bash
# ChainPulse Event Simulator - Continuous blockchain event generation
# Deploys ERC-20 contracts on Anvil chains and generates Transfer/Approval events
# Usage: bash docker/simulate-events.sh [start|stop|status]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${GREEN}[SIM]${NC} $*"; }
warn()  { echo -e "${YELLOW}[SIM]${NC} $*"; }
error() { echo -e "${RED}[SIM]${NC} $*"; }
sim()   { echo -e "${CYAN}[SIM]${NC} $*"; }

# Minimal ERC-20 Solidity source (compiled inside Anvil container via forge)
ERC20_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract TestToken {
    string public name = "TestToken";
    string public symbol = "TTK";
    uint8 public decimals = 18;
    uint256 public totalSupply;
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    constructor() { totalSupply = 1000000 * 10 ** 18; balanceOf[msg.sender] = totalSupply; }
    function transfer(address to, uint256 value) public returns (bool) { require(balanceOf[msg.sender] >= value); balanceOf[msg.sender] -= value; balanceOf[to] += value; emit Transfer(msg.sender, to, value); return true; }
    function approve(address spender, uint256 value) public returns (bool) { allowance[msg.sender][spender] = value; emit Approval(msg.sender, spender, value); return true; }
}'

# Chain configurations (container_name:chain_name:host_port)
# Works for both monolithic and microservices stacks
CHAINS_MONO="chainpulse-anvil-ethereum:ethereum:8545 chainpulse-anvil-polygon:polygon:8546 chainpulse-anvil-bsc:bsc:8547"
CHAINS_MS="chainpulse-ms-anvil-ethereum:ethereum:18545 chainpulse-ms-anvil-polygon:polygon:18546 chainpulse-ms-anvil-bsc:bsc:18547 chainpulse-ms-anvil-arbitrum:arbitrum:18548 chainpulse-ms-anvil-optimism:optimism:18549 chainpulse-ms-anvil-base:base:18550 chainpulse-ms-anvil-avalanche:avalanche:18551"

# Anvil default accounts
ACCOUNTS=(
    "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
    "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
    "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
    "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
    "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
)
KEYS=(
    "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
    "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
    "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
    "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
)

# State file for tracking deployed contracts
STATE_DIR="/tmp/chainpulse-sim"
mkdir -p "$STATE_DIR"

# Detect which stack is running
detect_stack() {
    if docker ps --format "{{.Names}}" | grep -q "chainpulse-ms-puller"; then
        echo "microservices"
    elif docker ps --format "{{.Names}}" | grep -q "chainpulse-app"; then
        echo "monolithic"
    else
        echo "none"
    fi
}

get_chains() {
    local stack=$(detect_stack)
    if [ "$stack" = "microservices" ]; then
        echo "$CHAINS_MS"
    elif [ "$stack" = "monolithic" ]; then
        echo "$CHAINS_MONO"
    else
        echo ""
    fi
}

# ============================================
# Deploy ERC-20 on a chain using forge create
# ============================================
deploy_token() {
    local container="$1"
    local chain="$2"
    local state_file="$STATE_DIR/${chain}.token"

    if [ -f "$state_file" ]; then
        cat "$state_file"
        return
    fi

    info "Deploying TestToken on $chain (container: $container)..."
    local deployer_key="${KEYS[0]}"

    # Write Solidity source to a temp file inside the container
    local sol_tmp="/tmp/TestToken_${chain}.sol"
    docker exec "$container" sh -c "cat > '$sol_tmp' << 'SOLEOF'
$(echo "$ERC20_SOL")
SOLEOF"

    # Create writable directories for forge (cache + out)
    docker exec "$container" sh -c "mkdir -p /tmp/forge-out /tmp/forge-cache && chmod 777 /tmp/forge-out /tmp/forge-cache" 2>/dev/null || true

    # Deploy using forge create (compiles + deploys in one step)
    local result
    result=$(docker exec "$container" forge create \
        "$sol_tmp":TestToken \
        --rpc-url http://localhost:8545 \
        --private-key "$deployer_key" \
        --out /tmp/forge-out \
        --cache-path /tmp/forge-cache \
        --broadcast 2>&1)

    local contract_address
    contract_address=$(echo "$result" | grep "Deployed to:" | awk '{print $3}' || echo "")

    if [ -z "$contract_address" ]; then
        # Fallback: try parsing any 0x address from the last few lines
        contract_address=$(echo "$result" | grep -oE "0x[0-9a-fA-F]{40}" | tail -1 || echo "")
    fi

    if [ -n "$contract_address" ]; then
        echo "$contract_address" > "$state_file"
        info "  $chain: TestToken deployed at $contract_address"
        echo "$contract_address"
    else
        error "  $chain: Failed to deploy TestToken"
        echo "$result" | tail -10
        echo ""
    fi
}

# ============================================
# Generate random events on a chain
# ============================================
generate_events() {
    local container="$1"
    local chain="$2"
    local token_addr="$3"
    local count="${4:-3}"

    local from_idx=$((RANDOM % ${#KEYS[@]}))
    local to_idx=$(( (from_idx + 1 + RANDOM % ( (${#KEYS[@]} - 1) ) ) % ${#KEYS[@]}))

    local from_key="${KEYS[$from_idx]}"
    local from_addr="${ACCOUNTS[$from_idx]}"
    local to_addr="${ACCOUNTS[$to_idx]}"
    local amount=$(( (RANDOM % 1000 + 1) * 1000000000000000000 ))  # 1-1000 tokens

    # Transfer
    sim "$chain: Transfer ${ACCOUNTS[$from_idx]:0:10}... → ${to_addr:0:10}... ($(( amount / 1000000000000000000 )) TTK)"
    docker exec "$container" cast send \
        --rpc-url http://localhost:8545 \
        --private-key "$from_key" \
        "$token_addr" \
        "transfer(address,uint256)" "$to_addr" "$amount" 2>&1 | grep -E "blockNumber|transactionHash" || true

    # Randomly also do an Approval (30% chance)
    if [ $((RANDOM % 10)) -lt 3 ]; then
        local spender_idx=$(( (to_idx + 1) % ${#KEYS[@]} ))
        local spender_addr="${ACCOUNTS[$spender_idx]}"
        local approve_amount=$(( (RANDOM % 500 + 1) * 1000000000000000000 ))

        sim "$chain: Approval ${from_addr:0:10}... → ${spender_addr:0:10}... ($(( approve_amount / 1000000000000000000 )) TTK)"
        docker exec "$container" cast send \
            --rpc-url http://localhost:8545 \
            --private-key "$from_key" \
            "$token_addr" \
            "approve(address,uint256)" "$spender_addr" "$approve_amount" 2>&1 | grep -E "blockNumber|transactionHash" || true
    fi
}

# ============================================
# Start continuous simulation
# ============================================
sim_start() {
    local stack=$(detect_stack)
    if [ "$stack" = "none" ]; then
        error "No ChainPulse stack detected. Start one first."
        exit 1
    fi

    info "Detected stack: $stack"
    info "Deploying ERC-20 tokens and starting continuous event simulation..."

    # Clean old state
    rm -f "$STATE_DIR"/*.token

    # Deploy tokens
    local chains=$(get_chains)
    for chain_entry in $chains; do
        local container="${chain_entry%%:*}"
        local chain
        chain=$(echo "$chain_entry" | cut -d: -f2)
        deploy_token "$container" "$chain"
    done

    # Start background simulator
    local pid_file="$STATE_DIR/sim.pid"
    if [ -f "$pid_file" ]; then
        local old_pid
        old_pid=$(cat "$pid_file")
        if kill -0 "$old_pid" 2>/dev/null; then
            warn "Simulator already running (PID $old_pid). Stopping..."
            kill "$old_pid" 2>/dev/null || true
        fi
    fi

    (
        while true; do
            for chain_entry in $chains; do
                local container="${chain_entry%%:*}"
                local chain
                chain=$(echo "$chain_entry" | cut -d: -f2)
                local token_file="$STATE_DIR/${chain}.token"

                if [ -f "$token_file" ]; then
                    local token_addr
                    token_addr=$(cat "$token_file")
                    if [ -n "$token_addr" ]; then
                        generate_events "$container" "$chain" "$token_addr" 1
                    fi
                fi
            done

            # Random sleep between 3-15 seconds to simulate irregular event arrival
            local sleep_time=$(( RANDOM % 13 + 3 ))
            sleep "$sleep_time"
        done
    ) &

    local sim_pid=$!
    echo "$sim_pid" > "$pid_file"
    info "Simulator started (PID $sim_pid)"
    info "Events will appear in ChainPulse within seconds."
    info "Query: curl http://localhost:18080/events?limit=10  (or :8080 for monolithic)"
}

# ============================================
# Stop simulation
# ============================================
sim_stop() {
    local pid_file="$STATE_DIR/sim.pid"
    if [ -f "$pid_file" ]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null
            info "Simulator stopped (was PID $pid)"
        else
            warn "Simulator not running (stale PID file)"
        fi
        rm -f "$pid_file"
    else
        warn "No simulator PID file found"
    fi
}

# ============================================
# Show simulation status
# ============================================
sim_status() {
    local stack=$(detect_stack)
    info "Stack: $stack"

    local pid_file="$STATE_DIR/sim.pid"
    if [ -f "$pid_file" ]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            info "Simulator: running (PID $pid)"
        else
            warn "Simulator: not running (stale PID file)"
        fi
    else
        warn "Simulator: not started"
    fi

    # Show deployed tokens
    for f in "$STATE_DIR"/*.token; do
        [ -f "$f" ] || continue
        local chain
        chain=$(basename "$f" .token)
        local addr
        addr=$(cat "$f")
        info "  $chain: $addr"
    done

    # Show event count
    if [ "$stack" = "microservices" ]; then
        local event_count
        event_count=$(curl -sf "http://localhost:18080/events?limit=500" 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); t=p.get('total',0); print(t if t<500 else f'{t}+')" 2>/dev/null || echo "?")
        info "Events indexed: $event_count"
    elif [ "$stack" = "monolithic" ]; then
        local event_count
        event_count=$(curl -sf "http://localhost:8080/events?limit=500" 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('pagination',{}); t=p.get('total',0); print(t if t<500 else f'{t}+')" 2>/dev/null || echo "?")
        info "Events indexed: $event_count"
    fi
}

# ============================================
# Main
# ============================================
case "${1:-help}" in
    start)
        sim_start
        ;;
    stop)
        sim_stop
        ;;
    status)
        sim_status
        ;;
    help|*)
        echo "ChainPulse Event Simulator - Continuous blockchain event generation"
        echo ""
        echo "Usage: bash docker/simulate-events.sh <command>"
        echo ""
        echo "Commands:"
        echo "  start   Deploy ERC-20 tokens and start generating Transfer/Approval events"
        echo "  stop    Stop the background event generator"
        echo "  status  Show simulation status and event counts"
        echo ""
        echo "The simulator:"
        echo "  - Deploys an ERC-20 token contract on each Anvil chain"
        echo "  - Generates Transfer and Approval events at random intervals (3-15s)"
        echo "  - Varies sender/receiver from 5 Anvil pre-funded accounts"
        echo "  - Works with both monolithic and microservices stacks"
        echo ""
        echo "Quick start:"
        echo "  bash docker/acceptance-microservices.sh build && bash docker/acceptance-microservices.sh up"
        echo "  bash docker/simulate-events.sh start"
        echo "  bash docker/simulate-events.sh status"
        ;;
esac
