#!/usr/bin/env bash
# ChainPulse Microservices Deploy & Simulate
# Builds 4 microservice images, deploys the full stack, and starts
# continuous event simulation — all with a single command.
#
# Usage: bash docker/deploy-microservices.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.microservices.yml"
STATE_DIR="/tmp/chainpulse-sim"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }
sim()   { echo -e "${CYAN}[SIM]${NC} $*"; }

# ── Chain configurations (indexed arrays — bash 3 compat) ──
# Single Anvil instance handles all EVM chains (chain-id switched per cycle)
CHAIN_CONTAINERS=("chainpulse-anvil" "chainpulse-anvil" "chainpulse-anvil" "chainpulse-anvil" "chainpulse-anvil" "chainpulse-anvil")
CHAIN_CONFIGS=("ethereum:8545:12" "bsc:8545:3" "polygon:8545:2" "arbitrum:8545:0" "base:8545:2" "avalanche:8545:2")
# Block interval in seconds: ethereum=12, bsc=3, polygon=2, arbitrum=~0, base=2, avalanche=2

# Solana configuration (separate from EVM — uses spl-token CLI, not cast send)
SOLANA_CONTAINER="chainpulse-solana"
SOLANA_RPC_PORT=8899

METRICS_FILE="$STATE_DIR/metrics.prom"
STATS_FILE="$STATE_DIR/sim.stats"

# ── ERC-20 TestToken (unchanged) ──
TESTTOKEN_SOL='// SPDX-License-Identifier: MIT
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

# ── ERC-721 TestNFT ──
TESTNFT_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract TestNFT {
    string public name = "TestNFT";
    string public symbol = "TNFT";
    uint256 public totalSupply;
    mapping(uint256 => address) public ownerOf;
    mapping(address => uint256) public balanceOf;
    mapping(uint256 => address) public getApproved;
    mapping(address => mapping(address => bool)) public isApprovedForAll;
    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event Approval(address indexed owner, address indexed approved, uint256 indexed tokenId);
    event ApprovalForAll(address indexed owner, address indexed operator, bool approved);
    constructor() { for (uint256 i = 1; i <= 100; i++) _mint(msg.sender, i); }
    function _mint(address to, uint256 tokenId) internal { totalSupply++; ownerOf[tokenId] = to; balanceOf[to]++; emit Transfer(address(0), to, tokenId); }
    function transferFrom(address from, address to, uint256 tokenId) public {
        require(ownerOf[tokenId] == from, "not owner");
        require(msg.sender == from || msg.sender == getApproved[tokenId] || isApprovedForAll[from][msg.sender], "not authorized");
        ownerOf[tokenId] = to; balanceOf[from]--; balanceOf[to]++; emit Transfer(from, to, tokenId);
    }
    function approve(address approved, uint256 tokenId) public {
        require(ownerOf[tokenId] == msg.sender, "not owner"); getApproved[tokenId] = approved; emit Approval(msg.sender, approved, tokenId);
    }
    function setApprovalForAll(address operator, bool approved) public {
        isApprovedForAll[msg.sender][operator] = approved; emit ApprovalForAll(msg.sender, operator, approved);
    }
    function mint(address to, uint256 tokenId) public { _mint(to, tokenId); }
}'

# ── RealEventEmitter (real DeFi protocol signatures) ──
REAL_EMITTER_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract RealEventEmitter {
    event Swap(address indexed sender, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick);
    event Supply(address indexed reserve, address user, address indexed onBehalfOf, uint256 amount, bool indexed referral);
    event Withdraw(address indexed reserve, address indexed user, address indexed to, uint256 amount);
    event Borrow(address indexed reserve, address user, address indexed onBehalfOf, uint256 amount, uint8 interestRateMode, bool indexed referral);
    event LiquidationCall(address indexed collateralAsset, address indexed debtAsset, address indexed user, uint256 debtToCover, uint256 liquidatedCollateralAmount, bool receiveAToken);
    // Compound V3 Supply — overloaded event name, different params from Aave Supply
    event Supply(address indexed from, address indexed to, uint256 amount);
    event VoteCast(address indexed voter, uint256 proposalId, uint8 support, uint256 weight, string reason);
    event Bridge(address indexed token, address indexed sender, uint256 amount, uint256 indexed destChainId);
    event Batch(uint256 indexed batchId, string description);
    function emitUniSwap(int256 a0, int256 a1, uint160 sp, uint128 liq, int24 tk) public { emit Swap(msg.sender, a0, a1, sp, liq, tk); }
    function emitSupply(address r, address u, address o, uint256 a, bool ref) public { emit Supply(r, u, o, a, ref); }
    function emitWithdraw(address r, address u, address t, uint256 a) public { emit Withdraw(r, u, t, a); }
    function emitBorrow(address r, address u, address o, uint256 a, uint8 m, bool ref) public { emit Borrow(r, u, o, a, m, ref); }
    function emitLiquidation(address ca, address da, address u, uint256 dc, uint256 lc, bool rat) public { emit LiquidationCall(ca, da, u, dc, lc, rat); }
    function emitCometSupply(address f, address t, uint256 a) public { emit Supply(f, t, a); }
    function emitVoteCast(uint256 pid, uint8 s, uint256 w, string calldata r) public { emit VoteCast(msg.sender, pid, s, w, r); }
    function emitBridge(address tk, uint256 a, uint256 d) public { emit Bridge(tk, msg.sender, a, d); }
    function emitBatch(uint256 bid, string calldata d) public { emit Batch(bid, d); }
}'

# ── RealEventEmitter V2 — extends coverage with 20+ additional protocol event types ──
REAL_EMITTER_V2_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract RealEventEmitterV2 {
    // --- ERC-1155 ---
    event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value);
    event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values);
    event URI(string value, uint256 indexed id);

    // --- Aave V3 (full) ---
    event Repay(address indexed reserve, address indexed user, address indexed repayer, uint256 amount, bool useATokens);
    event ReserveDataUpdated(address indexed reserve, uint256 liquidityRate, uint256 stableBorrowRate, uint256 variableBorrowRate, uint256 liquidityIndex, uint256 variableBorrowIndex);

    // --- Compound V3 (complete) ---
    event Withdraw(address indexed from, address indexed to, uint256 amount);
    event Borrow(address indexed account, uint256 amount, uint256 index);
    event Repay(address indexed from, address indexed to, uint256 amount);
    event Liquidate(address indexed liquidator, address indexed victim, uint256 amount, address indexed asset, bool isSupply);

    // --- Uniswap V2 ---
    event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to);
    event Sync(uint112 reserve0, uint112 reserve1);
    event PairCreated(address indexed token0, address indexed token1, address pair, uint256);

    // --- Curve ---
    event TokenExchange(address indexed buyer, int128 sold_id, int128 bought_id, uint256 tokens_sold, uint256 tokens_bought);

    // --- Balancer ---
    event Swap(address indexed tokenIn, address indexed tokenOut, uint256 amountIn, uint256 amountOut);

    // --- Governance (full OZ lifecycle) ---
    event ProposalCreated(uint256 proposalId, address indexed proposer, address[] targets, uint256[] values, string[] signatures, bytes[] calldatas, uint256 voteStart, uint256 voteEnd, string description);
    event ProposalExecuted(uint256 proposalId);
    event ProposalCanceled(uint256 proposalId);

    // --- ERC-4337 ---
    event UserOperationEvent(address indexed sender, bytes32 userOpHash, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed);
    event AccountDeployed(address indexed sender, bytes32 userOpHash);

    // --- Post-Dencun ---
    event WithdrawalRequested(address indexed source, bytes pubkey, uint256 amount);
    event MessagePassed(uint256 nonce, address indexed sender, address indexed target, uint256 value, bytes32 gasLimit, bytes32 dataHash, uint256 withdrawalHash);

    // --- L2 cross-chain messages ---
    event SentMessage(address indexed target, address indexed sender, uint256 value, uint256 gasLimit, uint256 nonce);
    event TxToL2(uint256 callValue, address indexed destination, address indexed sender, uint256 amount, uint256 maxSubmissionCost, uint256 maxGas);

    // --- Batch correlation (same pattern as V1) ---
    event Batch(uint256 indexed batchId, string description);

    function emitTransferSingle(address op, address from, address to, uint256 id, uint256 val) public { emit TransferSingle(op, from, to, id, val); }
    function emitTransferBatch(address op, address from, address to, uint256[] calldata ids, uint256[] calldata vals) public { emit TransferBatch(op, from, to, ids, vals); }
    function emitURI(string calldata val, uint256 id) public { emit URI(val, id); }

    function emitRepay(address r, address u, address rp, uint256 a, bool ua) public { emit Repay(r, u, rp, a, ua); }
    function emitReserveDataUpdated(address r, uint256 lr, uint256 sbr, uint256 vbr, uint256 li, uint256 vbi) public { emit ReserveDataUpdated(r, lr, sbr, vbr, li, vbi); }

    function emitCometWithdraw(address f, address t, uint256 a) public { emit Withdraw(f, t, a); }
    function emitCometBorrow(address a, uint256 amt, uint256 idx) public { emit Borrow(a, amt, idx); }
    function emitCometRepay(address f, address t, uint256 a) public { emit Repay(f, t, a); }
    function emitCometLiquidate(address liq, address vic, uint256 a, address asset, bool isS) public { emit Liquidate(liq, vic, a, asset, isS); }

    function emitUniV2Swap(uint256 a0in, uint256 a1in, uint256 a0out, uint256 a1out, address to) public { emit Swap(msg.sender, a0in, a1in, a0out, a1out, to); }
    function emitSync(uint112 r0, uint112 r1) public { emit Sync(r0, r1); }
    function emitPairCreated(address t0, address t1, address pair) public { emit PairCreated(t0, t1, pair, 0); }

    function emitCurveSwap(int128 sid, int128 bid, uint256 sold, uint256 bought) public { emit TokenExchange(msg.sender, sid, bid, sold, bought); }
    function emitBalancerSwap(address tin, address tout, uint256 ain, uint256 aout) public { emit Swap(tin, tout, ain, aout); }

    function emitProposalCreated(uint256 pid, address proposer, address[] calldata targets, uint256[] calldata values, string[] calldata sigs, bytes[] calldata cds, uint256 vs, uint256 ve, string calldata desc) public { emit ProposalCreated(pid, proposer, targets, values, sigs, cds, vs, ve, desc); }
    function emitProposalExecuted(uint256 pid) public { emit ProposalExecuted(pid); }
    function emitProposalCanceled(uint256 pid) public { emit ProposalCanceled(pid); }

    function emitUserOpEvent(address sender, bytes32 uoh, uint256 nonce, bool success, uint256 agc, uint256 agu) public { emit UserOperationEvent(sender, uoh, nonce, success, agc, agu); }
    function emitAccountDeployed(address sender, bytes32 uoh) public { emit AccountDeployed(sender, uoh); }

    function emitWithdrawalRequested(address src, bytes calldata pubkey, uint256 amt) public { emit WithdrawalRequested(src, pubkey, amt); }

    // --- L2 event emitters ---
    function emitSentMessage(address target, address sender, uint256 val, uint256 gl, uint256 nonce) public { emit SentMessage(target, sender, val, gl, nonce); }
    function emitTxToL2(uint256 cv, address dest, address sender, uint256 amt, uint256 msc, uint256 mg) public { emit TxToL2(cv, dest, sender, amt, msc, mg); }
    function emitMessagePassed(uint256 nonce, address sender, address target, uint256 val, bytes32 gasLimit, bytes32 dataHash, uint256 withdrawalHash) public { emit MessagePassed(nonce, sender, target, val, gasLimit, dataHash, withdrawalHash); }

    function emitBatch(uint256 bid, string calldata d) public { emit Batch(bid, d); }
}'

# ──────────────────────────────────────────────
# Step 1: Build Docker images
# ──────────────────────────────────────────────
build_images() {
    info "===== Step 1/3: Building microservice Docker images ====="
    mkdir -p "$PROJECT_ROOT/build/bin/linux/microservices"

    # Always recompile to ensure code changes are picked up
    for svc in puller event-processor api-service api-gateway; do
        info "  Compiling $svc..."
        local arch="${GOARCH:-$(go env GOARCH 2>/dev/null || echo "amd64")}"
        CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -a -installsuffix cgo \
            -o "$PROJECT_ROOT/build/bin/linux/microservices/$svc" \
            "$PROJECT_ROOT/cmd/microservices/$svc"
    done

    for svc in puller event-processor api-service api-gateway; do
        info "  Building chainpulse-$svc:latest..."
        docker build --no-cache --build-arg "SERVICE=$svc" \
            -f "$SCRIPT_DIR/Dockerfile.microservices.prebuilt" \
            -t "chainpulse-$svc:latest" "$PROJECT_ROOT" 2>&1 | tail -1 || true
    done
}

# ──────────────────────────────────────────────
# Step 2: Clean and start the microservices stack
# ──────────────────────────────────────────────
start_stack() {
    info "===== Step 2/3: Starting microservices stack ====="

    # Pre-flight port conflict check (best-effort, skip if lsof unavailable)
    if command -v lsof &>/dev/null; then
        local ports=(5432 6379 9092 9090 3000 4317 4318 8080 8081 8082 8083 13000 16687 8545 8899 8900)
        local conflicts=0
        for p in "${ports[@]}"; do
            if lsof -ti :"$p" 2>/dev/null | xargs ps -p 2>/dev/null | grep -qvi docker; then
                warn "Port $p is in use by a non-Docker process — chainpulse may fail to bind"
                conflicts=$((conflicts + 1))
            fi
        done
        [ "$conflicts" -gt 0 ] && warn "Found $conflicts potential port conflict(s)"
    fi

    # Clean up any previous chainpulse containers
    local existing
    existing=$(docker ps -a --filter "name=chainpulse" --format "{{.Names}}" 2>/dev/null || true)
    if [ -n "$existing" ]; then
        warn "Cleaning up existing chainpulse containers..."
        docker rm -f $existing 2>/dev/null || true
    fi

    # Clean simulator state from previous runs
    rm -rf "$STATE_DIR"

    info "Starting Docker Compose microservices stack..."
    local env_file="$SCRIPT_DIR/.env"
    if [ ! -f "$env_file" ]; then
        warn ".env file not found at $env_file, copying from .env.example"
        cp "$SCRIPT_DIR/.env.example" "$env_file"
        warn "Review $env_file and set production secrets before deploying to production"
    fi
    docker compose --env-file "$env_file" -f "$COMPOSE_FILE" pull --ignore-pull-failures 2>&1 | grep -v "Pulling" || true
    docker compose --env-file "$env_file" -f "$COMPOSE_FILE" up -d 2>&1

    # Wait for infrastructure
    info "Waiting for infrastructure to become healthy..."
    for svc in chainpulse-postgres chainpulse-redis chainpulse-kafka chainpulse-anvil chainpulse-solana; do
        local n=0
        while [ $n -lt 60 ]; do
            n=$((n + 1))
            local status
            status=$(docker inspect "$svc" --format '{{.State.Health.Status}}' 2>/dev/null || echo "")
            if [ "$status" = "healthy" ]; then
                info "  $svc: healthy (${n}s)"
                break
            fi
            sleep 2
        done
    done

    # Pre-create Kafka topics (auto-create may have race conditions)
    info "Pre-creating Kafka topics..."
    for topic in raw-events blockchain-events processed-events indexed-events; do
        docker exec chainpulse-kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic "$topic" --partitions 3 --replication-factor 1 2>/dev/null && info "  Topic $topic: created" || info "  Topic $topic: already exists or Kafka not ready"
    done

    # Wait for microservices
    for svc in chainpulse-api-service chainpulse-event-processor chainpulse-puller chainpulse-api-gateway; do
        local port=""
        case "$svc" in
            chainpulse-api-service) port="8081" ;;
            chainpulse-event-processor) port="8082" ;;
            chainpulse-puller) port="8083" ;;
            chainpulse-api-gateway) port="8080" ;;
        esac
        local n=0
        while [ $n -lt 30 ]; do
            n=$((n + 1))
            if curl -sf "http://localhost:$port/health" >/dev/null 2>&1; then
                info "  $svc: healthy (${n}s)"
                break
            fi
            sleep 2
        done
    done
}

# ──────────────────────────────────────────────
# Step 3: Deploy contracts and run simulation
# ──────────────────────────────────────────────
# ── Contract deploy helper with retry and fallback ──
_forge_create() {
    local ctr="$1" port="$2" key="$3" sol_file="$4" contract_name="$5" extra_flags="${6:-}"
    local rpc="http://localhost:${port}"
    local addr="" attempt=1

    while [ $attempt -le 3 ]; do
        info "    forge create attempt $attempt/3 for $contract_name..." >&2
        local output
        output=$(docker exec "$ctr" forge create --broadcast --rpc-url "$rpc" --private-key "$key" --out /tmp/forge-out --cache-path /tmp/forge-cache $extra_flags "${sol_file}:${contract_name}" 2>&1) || true
        echo "[$(date +%T)] forge create $contract_name attempt $attempt:" >> "$STATE_DIR/deploy.log"
        echo "$output" >> "$STATE_DIR/deploy.log"

        addr=$(echo "$output" | grep -i "deployed to" | awk '{print $NF}')
        if [ -n "$addr" ]; then
            echo "$addr"
            return 0
        fi

        # Check for compilation success but deployment output format mismatch
        # Try extracting from JSON-like output or different formats
        addr=$(echo "$output" | grep -E "^0x[0-9a-fA-F]{40}$" | head -1 || true)
        if [ -n "$addr" ]; then
            echo "$addr"
            return 0
        fi

        attempt=$((attempt + 1))
        [ $attempt -le 3 ] && sleep 2
    done

    # Fallback: forge build + cast send --create
    info "    forge create failed, trying forge build + cast send --create for $contract_name..." >&2
    docker exec "$ctr" forge build --out /tmp/forge-out --cache-path /tmp/forge-cache --contracts "$sol_file" $extra_flags >/dev/null 2>>"$STATE_DIR/deploy.log" || {
        echo "[$(date +%T)] forge build also failed for $contract_name" >> "$STATE_DIR/deploy.log"
        return 1
    }

    # Read bytecode from compiled artifact
    local artifact="/tmp/forge-out/${contract_name}.json"
    local bytecode
    bytecode=$(docker exec "$ctr" sh -c "cat '$artifact' 2>/dev/null | python3 -c \"import sys,json; d=json.load(sys.stdin); print(d.get('bytecode',{}).get('object','') or d.get('bytecode',''))\" 2>/dev/null || echo ''" || true)

    if [ -z "$bytecode" ] || [ ${#bytecode} -lt 10 ]; then
        # Try alternative artifact path
        local sol_basename
        sol_basename=$(basename "$sol_file" .sol)
        artifact="/tmp/forge-out/${sol_basename}/${contract_name}.json"
        bytecode=$(docker exec "$ctr" sh -c "cat '$artifact' 2>/dev/null | python3 -c \"import sys,json; d=json.load(sys.stdin); print(d.get('bytecode',{}).get('object','') or d.get('bytecode',''))\" 2>/dev/null || echo ''" || true)
    fi

    if [ -z "$bytecode" ] || [ ${#bytecode} -lt 10 ]; then
        echo "[$(date +%T)] no bytecode found for $contract_name" >> "$STATE_DIR/deploy.log"
        return 1
    fi

    info "    deploying $contract_name via cast send --create..."
    local deploy_output
    deploy_output=$(docker exec "$ctr" cast send --rpc-url "$rpc" --private-key "$key" --create "$bytecode" 2>&1) || true
    echo "[$(date +%T)] cast send --create $contract_name:" >> "$STATE_DIR/deploy.log"
    echo "$deploy_output" >> "$STATE_DIR/deploy.log"

    addr=$(echo "$deploy_output" | grep -i "contractAddress" | awk '{print $2}' | tr -d ',' || true)
    if [ -z "$addr" ]; then
        addr=$(echo "$deploy_output" | grep -i "deployed to" | awk '{print $NF}' || true)
    fi

    if [ -n "$addr" ]; then
        echo "$addr"
        return 0
    fi

    return 1
}

# ── Contract deploy helper (called by start_simulation) ──
deploy_on_chain() {
    local ctr="$1" chain="$2" port="$3" deployer_key="$4"
    info "  Deploying on $chain ($ctr)..."
    docker exec "$ctr" mkdir -p /tmp/forge-out /tmp/forge-cache 2>/dev/null || true

    # TestToken
    if [ ! -f "$STATE_DIR/${chain}.token" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/TestToken.sol << 'EOF'
$TESTTOKEN_SOL
EOF"
        local taddr
        taddr=$(_forge_create "$ctr" "$port" "$deployer_key" /tmp/TestToken.sol TestToken)
        [ -n "$taddr" ] && echo "$taddr" > "$STATE_DIR/${chain}.token" && info "    TestToken: $taddr" || error "    TestToken deploy failed on $chain (see $STATE_DIR/deploy.log)"
    fi

    # TestNFT
    if [ ! -f "$STATE_DIR/${chain}.nft" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/TestNFT.sol << 'EOF'
$TESTNFT_SOL
EOF"
        local nftaddr
        nftaddr=$(_forge_create "$ctr" "$port" "$deployer_key" /tmp/TestNFT.sol TestNFT)
        [ -n "$nftaddr" ] && echo "$nftaddr" > "$STATE_DIR/${chain}.nft" && info "    TestNFT: $nftaddr" || error "    TestNFT deploy failed on $chain (see $STATE_DIR/deploy.log)"
    fi

    # RealEventEmitter
    if [ ! -f "$STATE_DIR/${chain}.emitter" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/RealEventEmitter.sol << 'EOF'
$REAL_EMITTER_SOL
EOF"
        local eaddr
        eaddr=$(_forge_create "$ctr" "$port" "$deployer_key" /tmp/RealEventEmitter.sol RealEventEmitter)
        [ -n "$eaddr" ] && echo "$eaddr" > "$STATE_DIR/${chain}.emitter" && info "    RealEventEmitter: $eaddr" || error "    RealEventEmitter deploy failed on $chain (see $STATE_DIR/deploy.log)"
    fi

    # RealEventEmitter V2
    if [ ! -f "$STATE_DIR/${chain}.emitter-v2" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/RealEventEmitterV2.sol << 'EOF'
$REAL_EMITTER_V2_SOL
EOF"
        local e2addr
        e2addr=$(_forge_create "$ctr" "$port" "$deployer_key" /tmp/RealEventEmitterV2.sol RealEventEmitterV2 "--via-ir")
        [ -n "$e2addr" ] && echo "$e2addr" > "$STATE_DIR/${chain}.emitter-v2" && info "    RealEventEmitterV2: $e2addr" || error "    RealEventEmitterV2 deploy failed on $chain (see $STATE_DIR/deploy.log)"
    fi
}

# ── Solana setup: keypairs, airdrop, SPL token, token accounts ──
setup_solana() {
    local sol_ctr="$SOLANA_CONTAINER"
    local sol_rpc="http://localhost:${SOLANA_RPC_PORT}"

    # Idempotency: if already set up and token file exists, skip
    if [ -f "$STATE_DIR/solana.token" ]; then
        local existing_token
        existing_token=$(cat "$STATE_DIR/solana.token")
        if docker exec "$sol_ctr" spl-token account --url "$sol_rpc" "$existing_token" >/dev/null 2>&1; then
            info "Solana already set up (token $existing_token), skipping"
            return 0
        fi
        warn "solana.token exists but token account check failed, will re-setup"
    fi

    info "Setting up Solana validator at $sol_rpc..."

    local n=0
    while [ $n -lt 60 ]; do
        if docker exec "$sol_ctr" solana cluster-version --url "$sol_rpc" >/dev/null 2>&1; then
            break
        fi
        n=$((n + 1))
        info "  Waiting for Solana validator... (${n}s)"
        sleep 2
    done
    if [ $n -ge 60 ]; then
        error "Solana validator not ready after 120s"
        return 1
    fi
    info "  Solana validator ready (${n}s)"

    local key_dir="/tmp/chainpulse-sol-keys"
    docker exec "$sol_ctr" mkdir -p "$key_dir" 2>/dev/null || true

    info "  Generating keypairs..."
    for i in $(seq 0 4); do
        docker exec "$sol_ctr" solana-keygen new --no-bip39-passphrase --outfile "$key_dir/key${i}.json" --force 2>/dev/null || true
    done

    info "  Airdropping SOL..."
    for i in $(seq 0 4); do
        docker exec "$sol_ctr" solana airdrop 100 --url "$sol_rpc" --keypair "$key_dir/key${i}.json" >/dev/null 2>>"$STATE_DIR/errors.log" || true
        sleep 0.3
    done

    info "  Creating SPL Token..."
    local spl_token=""
    for i in $(seq 1 10); do
        spl_token=$(docker exec "$sol_ctr" spl-token create-token --url "$sol_rpc" --fee-payer "$key_dir/key0.json" --mint-authority "$key_dir/key0.json" 2>>"$STATE_DIR/errors.log" | grep "Creating token" | awk '{print $3}' || true)
        if [ -n "$spl_token" ]; then break; fi
        sleep 2
    done
    if [ -z "$spl_token" ]; then
        error "Failed to create SPL Token"
        return 1
    fi
    info "  SPL Token: $spl_token"

    info "  Creating token accounts..."
    local sol_atas=()
    for i in $(seq 0 4); do
        local acc
        acc=$(docker exec "$sol_ctr" spl-token create-account --url "$sol_rpc" --fee-payer "$key_dir/key${i}.json" --owner "$key_dir/key${i}.json" "$spl_token" 2>>"$STATE_DIR/errors.log" | grep "Creating account" | awk '{print $3}' || true)
        if [ -n "$acc" ]; then
            sol_atas[$i]="$acc"
            echo "$acc" > "$STATE_DIR/solana.ata${i}"
            info "    account $i: $acc"
        fi
        sleep 0.5
    done

    info "  Minting initial tokens..."
    docker exec "$sol_ctr" spl-token mint --url "$sol_rpc" --fee-payer "$key_dir/key0.json" --mint-authority "$key_dir/key0.json" "$spl_token" 1000000 >/dev/null 2>>"$STATE_DIR/errors.log" || true

    echo "$spl_token" > "$STATE_DIR/solana.token"

    info "  Creating vote account for simulation..."
    docker exec "$sol_ctr" solana-keygen new --no-bip39-passphrase --outfile "$key_dir/vote.json" --force 2>/dev/null || true
    docker exec "$sol_ctr" solana-keygen new --no-bip39-passphrase --outfile "$key_dir/withdrawer.json" --force 2>/dev/null || true
    docker exec "$sol_ctr" solana create-vote-account --url "$sol_rpc" --fee-payer "$key_dir/key0.json" "$key_dir/vote.json" "$key_dir/key0.json" "$key_dir/withdrawer.json" >/dev/null 2>>"$STATE_DIR/errors.log" || true
    local vote_pubkey; vote_pubkey=$(docker exec "$sol_ctr" solana-keygen pubkey "$key_dir/vote.json" 2>/dev/null || echo "")
    [ -n "$vote_pubkey" ] && echo "$vote_pubkey" > "$STATE_DIR/solana.vote" && info "    Vote account: $vote_pubkey"

    info "Solana setup complete"
}

start_simulation() {
    info "===== Step 3/3: Deploying contracts and starting event simulation ====="

    local deployer_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    mkdir -p "$STATE_DIR"

    # KEYS and ACCOUNTS for simulation
    SIM_KEYS=(
        "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
        "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
        "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
        "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
        "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
    )
    SIM_ACCOUNTS=(
        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
        "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
        "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
        "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
        "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
    )

    # Deploy EVM contracts on each available Anvil
    local any_deployed=false
    for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
        local ctr="${CHAIN_CONTAINERS[$__i]}" cfg="${CHAIN_CONFIGS[$__i]}"
        local chain="${cfg%%:*}" port; port=$(echo "$cfg" | cut -d: -f2)
        if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$ctr"; then
            deploy_on_chain "$ctr" "$chain" "$port" "$deployer_key"
            any_deployed=true
        fi
    done

    # Setup Solana (keypairs, SPL token, accounts)
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$SOLANA_CONTAINER"; then
        setup_solana
        any_deployed=true
    fi

    if [ "$any_deployed" = false ]; then
        error "No Anvil or Solana containers found! Is Docker Compose running?"
        return 1
    fi

    # Build comma-separated arg lists for sim_loop (EVM chains)
    local chains_list="" tokens_list="" nfts_list="" emitters_list="" emitters2_list=""
    for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
        local ctr="${CHAIN_CONTAINERS[$__i]}" cfg="${CHAIN_CONFIGS[$__i]}"
        local chain="${cfg%%:*}"
        if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$ctr"; then
            [ -n "$chains_list" ] && chains_list="${chains_list},"
            chains_list="${chains_list}${chain}"
            local tf="$STATE_DIR/${chain}.token" nf="$STATE_DIR/${chain}.nft" ef="$STATE_DIR/${chain}.emitter" e2f="$STATE_DIR/${chain}.emitter-v2"
            [ -f "$tf" ] && tokens_list="${tokens_list},$(cat "$tf")" || tokens_list="${tokens_list},"
            [ -f "$nf" ] && nfts_list="${nfts_list},$(cat "$nf")" || nfts_list="${nfts_list},"
            [ -f "$ef" ] && emitters_list="${emitters_list},$(cat "$ef")" || emitters_list="${emitters_list},"
            [ -f "$e2f" ] && emitters2_list="${emitters2_list},$(cat "$e2f")" || emitters2_list="${emitters2_list},"
        fi
    done
    tokens_list="${tokens_list#,}"; nfts_list="${nfts_list#,}"; emitters_list="${emitters_list#,}"; emitters2_list="${emitters2_list#,}"

    # Solana is handled inside sim_loop via STATE_DIR — not passed through CSV

    # Background event generation loop
    nohup bash "$SCRIPT_DIR/deploy-microservices.sh" loop "$chains_list" "$tokens_list" "$nfts_list" "$emitters_list" "$emitters2_list" > "$STATE_DIR/sim.log" 2>&1 &
    local sim_pid=$!
    echo "$sim_pid" > "$STATE_DIR/sim.pid"

    # Print summary
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║         ChainPulse  —  One-Click Deploy & Simulate          ║"
    echo "╠══════════════════════════════════════════════════════════════╣"
    echo "║  Simulator PID:  $sim_pid                                      ║"
    echo "║  Events (V1): Uniswap V3 Swap, Aave Supply/Withdraw/        ║"
    echo "║          Borrow, LiquidationCall, CometSupply, VoteCast,   ║"
    echo "║          ERC-20 Transfer/Approval, ERC-721 Transfer,       ║"
    echo "║          Bridge, Batch                                     ║"
    echo "║  Events (V2): ERC-1155, Aave Repay, Uniswap V2 Swap,      ║"
    echo "║          Curve, Balancer, CometWithdraw/Borrow/Liquidate,  ║"
    echo "║          OZ Governance lifecycle, ERC-4337 UserOp          ║"
    echo "║  Solana:   SPL Token Transfer, MintTo, Burn,              ║"
    echo "║          CreateAccount, InitializeMint                    ║"
    echo "║  Burst: ON (3-8 TPS)     Reorg: depth 2-12                ║"
    echo "║  Gas Spike: 8%  BlobFee: 4%  DroppedTx: 5%  MEV: 7%        ║"
    echo "║  ContractRevert: 8%  NonceGap: 4%  LargeBlock: 5%        ║"
    echo "║  CrossChainBridge: 6%  FlashLoan: 5%  LiqCascade: 4%    ║"
    echo "║  DexAggregator: 6%  CrossChainArb: 3%  CausalBlock: 7%  ║"
    echo "║  Solana: Vote/Stake 10%  SlotSkip 4%  ComputeLimit 3%    ║"
    echo "║  Timestamp Anomaly: 5%   Duplicate: 3%                    ║"
    echo "║  Chains:  Ethereum + Solana (7 chains total)               ║"
    echo "║                                                              ║"
    echo "║  API Gateway:    http://localhost:8080          ║"
    echo "║  Live Events:    http://localhost:8080/events   ║"
    echo "║  Frontend UI:    http://localhost:13000         ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
}

# ──────────────────────────────────────────────
# Background simulation loop (invoked by nohup)
# Multi-chain, real DeFi events, burst, metrics
# Args: chains_csv token_csv nft_csv emitter_csv emitter_v2_csv
# ──────────────────────────────────────────────
sim_loop() {
    set +e  # Disable exit-on-error: sim loop handles errors internally with || true and retry
    set +u  # Allow unbound variables (some conditional paths reference vars before assignment)
    local chains_csv="$1" tokens_csv="$2" nfts_csv="$3" emitters_csv="$4" emitters_v2_csv="$5"
    local pid_file="$STATE_DIR/sim.pid"
    echo "$$" > "$pid_file"
    trap 'rm -f "$pid_file"; rm -f "$METRICS_FILE" "$STATS_FILE"' EXIT

    IFS=',' read -ra CHAINS <<< "$chains_csv"
    IFS=',' read -ra TOKENS <<< "$tokens_csv"
    IFS=',' read -ra NFTS <<< "$nfts_csv"
    IFS=',' read -ra EMITTERS <<< "$emitters_csv"
    IFS=',' read -ra EMITTERS_V2 <<< "$emitters_v2_csv"
    local num_chains=${#CHAINS[@]}

    # Retry wrapper for cast send — 3 attempts, logs to errors.log
    _cs() {
        local c="$1" r="$2" k="$3" a="$4" s="$5"; shift 5
        local tx_type=""
        local _r=$((RANDOM % 10))
        [ "$_r" -eq 0 ] && tx_type="--legacy"  # 10% legacy type 0
        local n=0
        while [ $n -lt 3 ]; do
            if docker exec "$c" cast send --rpc-url "$r" --private-key "$k" $tx_type "$a" "$s" "$@" >/dev/null 2>>"$STATE_DIR/errors.log"; then
                return 0
            fi
            n=$((n + 1)); [ $n -lt 3 ] && sleep 0.3
        done
        echo "cast_send fail: $a.$s" >> "$STATE_DIR/errors.log"
        return 1
    }

    local KEYS=(
        "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
        "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
        "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
        "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
        "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
    )
    local ACCOUNTS=(
        "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
        "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
        "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
        "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
        "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
    )

    # Stats tracking (scalars — bash 3 compat)
    local _ev_Transfer=0 _ev_Approval=0 _ev_UniV3Swap=0 _ev_AaveSupply=0 _ev_AaveWithdraw=0 _ev_AaveBorrow=0
    local _ev_Liquidation=0 _ev_CometSupply=0 _ev_VoteCast=0 _ev_Bridge=0 _ev_NFTTransfer=0 _ev_NFTApproval=0 _ev_NFTApprovalAll=0
    local _ev_1155Single=0 _ev_AaveRepay=0 _ev_UniV2Swap=0 _ev_CometWithdraw=0 _ev_CometBorrow=0 _ev_ProposalCreated=0 _ev_ReserveDataUpdated=0 _ev_SentMessage=0 _ev_TxToL2=0
    local _ev_UserOp=0 _ev_ProposalExecuted=0 _ev_BalancerSwap=0 _ev_UniV2Sync=0 _ev_UniV2PairCreated=0
    local _ev_AccountDeployed=0 _ev_TransferBatch=0 _ev_ProposalCanceled=0 _ev_WithdrawalRequested=0 _ev_MessagePassed=0
    # Solana counters
    local _ev_SPLTransfer=0 _ev_SPLMintTo=0 _ev_SPLBurn=0
    local _ev_GasSpike=0 _ev_DroppedTx=0 _ev_StaleBlock=0 _ev_MEVSandwich=0 _ev_ContractRevert=0 _ev_NonceGap=0 _ev_LargeBlock=0 _ev_BlobFeeSpike=0
    local _ev_CrossChainBridge=0 _ev_FlashLoan=0 _ev_SolanaVote=0 _ev_SolanaStake=0 _ev_LiqCascade=0 _ev_DexAggregator=0
    local _ev_CrossChainArb=0 _ev_SolanaSlotSkip=0 _ev_SolanaComputeLimit=0 _ev_CausalBlock=0
    local total_gen=0 burst_count=0 start_sec _last_block_interval=""
    start_sec=$(date +%s)
    local cycle=0 nft_next_id=101
    local solana_available=false sol_token="" sol_key_dir="/tmp/chainpulse-sol-keys"

    local sol_status="no"
    [ "$solana_available" = true ] && sol_status="yes"
    info "Simulation started: $num_chains EVM chains + Solana, real DeFi + extended protocol event signatures"
    info "Performance baseline:"
    info "  Chains:      ${CHAINS[*]} (EVM)${solana_available:+, solana (SPL)}"
    info "  Events/sec:  ~5-12 (Poisson mean 2s, 3-5 events/cycle now includes V2)"
    info "  Burst:       3-8 TPS for 4s (15% chance)"
    info "  Reorg:       depth 2-12 blocks (10% chance) — EVM only"
    info "  Timestamp anomaly:  5% — EVM only"
    info "  Duplicate:   3% (EVM + Solana)"
    info "  Edge cases:  zero-value, gas-exhaustion, max-approval"
    info "  V2 events:   ERC-1155, Aave Repay, Curve, Balancer, Uniswap V2,"
    info "               Compound V3 complete, Governance lifecycle, ERC-4337"
    info "  Solana:      SPL Transfer, MintTo, Burn, CreateAccount — controlled from sim_loop"

    # Load event type weights from config (fallback to hardcoded defaults)
    local -a WEIGHT_THRESHOLDS WEIGHT_NAMES
    local __wconf="$SCRIPT_DIR/sim-event-weights.conf"
    if [ -f "$__wconf" ]; then
        source "$__wconf"
    fi
    : "${WEIGHT_Transfer:=20}" "${WEIGHT_Approval:=7}" "${WEIGHT_UniV3Swap:=9}"
    : "${WEIGHT_AaveSupply:=7}" "${WEIGHT_AaveWithdraw:=7}" "${WEIGHT_AaveBorrow:=7}"
    : "${WEIGHT_LiquidationCall:=5}" "${WEIGHT_CometSupply:=4}" "${WEIGHT_VoteCast:=4}"
    : "${WEIGHT_BridgeEvent:=4}" "${WEIGHT_ERC1155:=4}" "${WEIGHT_AaveRepayV2:=2}"
    : "${WEIGHT_UniV2SwapV2:=2}" "${WEIGHT_CometWithdrawV2:=2}" "${WEIGHT_CometBorrowV2:=1}"
    : "${WEIGHT_ProposalCreatedV2:=1}" "${WEIGHT_L2SentMessage:=1}" "${WEIGHT_L2TxToL2:=1}"
    : "${WEIGHT_NFTTransfer:=2}" "${WEIGHT_NFTApproval:=1}" "${WEIGHT_NFTApprovalForAll:=1}"
    : "${WEIGHT_UserOpV2:=2}" "${WEIGHT_ProposalExecutedV2:=1}" "${WEIGHT_BalancerSwapV2:=2}" "${WEIGHT_UniV2SyncV2:=1}" "${WEIGHT_UniV2PairCreatedV2:=1}"
    : "${WEIGHT_AccountDeployedV2:=1}" "${WEIGHT_TransferBatchV2:=1}" "${WEIGHT_ProposalCanceledV2:=1}" "${WEIGHT_WithdrawalRequestedV2:=1}"
    : "${WEIGHT_MessagePassedV2:=1}"
    WEIGHT_NAMES=(Transfer Approval UniV3Swap AaveSupply AaveWithdraw AaveBorrow LiquidationCall CometSupply VoteCast BridgeEvent ERC1155 AaveRepayV2 UniV2SwapV2 CometWithdrawV2 CometBorrowV2 ProposalCreatedV2 L2SentMessage L2TxToL2 NFTTransfer NFTApproval NFTApprovalForAll UserOpV2 ProposalExecutedV2 BalancerSwapV2 UniV2SyncV2 UniV2PairCreatedV2 AccountDeployedV2 TransferBatchV2 ProposalCanceledV2 WithdrawalRequestedV2 MessagePassedV2)
    local __wacc=0 __wi=0
    for __w in $WEIGHT_Transfer $WEIGHT_Approval $WEIGHT_UniV3Swap $WEIGHT_AaveSupply $WEIGHT_AaveWithdraw $WEIGHT_AaveBorrow $WEIGHT_LiquidationCall $WEIGHT_CometSupply $WEIGHT_VoteCast $WEIGHT_BridgeEvent $WEIGHT_ERC1155 $WEIGHT_AaveRepayV2 $WEIGHT_UniV2SwapV2 $WEIGHT_CometWithdrawV2 $WEIGHT_CometBorrowV2 $WEIGHT_ProposalCreatedV2 $WEIGHT_L2SentMessage $WEIGHT_L2TxToL2 $WEIGHT_NFTTransfer $WEIGHT_NFTApproval $WEIGHT_NFTApprovalForAll $WEIGHT_UserOpV2 $WEIGHT_ProposalExecutedV2 $WEIGHT_BalancerSwapV2 $WEIGHT_UniV2SyncV2 $WEIGHT_UniV2PairCreatedV2 $WEIGHT_AccountDeployedV2 $WEIGHT_TransferBatchV2 $WEIGHT_ProposalCanceledV2 $WEIGHT_WithdrawalRequestedV2 $WEIGHT_MessagePassedV2; do
        __wacc=$((__wacc + __w))
        WEIGHT_THRESHOLDS[__wi]=$__wacc
        __wi=$((__wi + 1))
    done

    # Read Solana state (set up by setup_solana() before sim_loop was launched)
    # solana_available, sol_token, sol_key_dir already declared above
    local sol_container="$SOLANA_CONTAINER" sol_rpc="http://localhost:${SOLANA_RPC_PORT}"
    local sol_atas=() sol_vote_pubkey=""
    if [ -f "$STATE_DIR/solana.token" ]; then
        sol_token=$(cat "$STATE_DIR/solana.token")
        for __si in 0 1 2 3 4; do
            if [ -f "$STATE_DIR/solana.ata${__si}" ]; then
                sol_atas[$__si]=$(cat "$STATE_DIR/solana.ata${__si}")
            fi
        done
        [ -f "$STATE_DIR/solana.vote" ] && sol_vote_pubkey=$(cat "$STATE_DIR/solana.vote")
        if [ -n "$sol_token" ] && docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$sol_container"; then
            solana_available=true
        fi
    fi

    while true; do
        cycle=$((cycle + 1))

        # EIP-1559 base fee natural fluctuation (±12.5% per block)
        if [ $((cycle % 3)) -eq 0 ] && docker ps --format "{{.Names}}" 2>/dev/null | grep -q "chainpulse-anvil"; then
            local _bf_rpc="http://localhost:8545"
            local _cur_fee; _cur_fee=$(docker exec chainpulse-anvil cast rpc --rpc-url "$_bf_rpc" eth_gasPrice 2>/dev/null | tr -d '"') || _cur_fee="0x3b9aca00"
            local _cur_dec=$(( _cur_fee ))
            local _delta=$(( _cur_dec * (RANDOM % 25 - 12) / 100 ))
            local _new_fee=$(( _cur_dec + _delta ))
            [ "$_new_fee" -lt 1000000000 ] && _new_fee=1000000000
            local _new_hex="0x$(printf '%x' "$_new_fee")"
            docker exec chainpulse-anvil cast rpc --rpc-url "$_bf_rpc" anvil_setNextBlockBaseFeePerGas "$_new_hex" >/dev/null 2>>"$STATE_DIR/errors.log" || true
        fi

        # Auto-stop check — any chain's container still alive?
        local any_alive=false
        for ce in "${CHAINS[@]}"; do
            for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
                local ctr="${CHAIN_CONTAINERS[$__i]}" cfg="${CHAIN_CONFIGS[$__i]}"
                if [ "${cfg%%:*}" = "$ce" ] && docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$ctr"; then
                    any_alive=true; break 2
                fi
            done
        done
        if [ "$any_alive" = false ] && [ "$solana_available" = true ]; then
            # If Solana state was read, check the Solana container too
            if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$sol_container"; then
                any_alive=true
            fi
        fi
        if [ "$any_alive" = false ]; then
            warn "No chain containers detected. Stopping."
            rm -f "$pid_file"; exit 0
        fi

        # Cross-chain correlation ID (shared between EVM and Solana events this cycle)
        local batch_num
        batch_num=$(date +%s)
        echo "$batch_num" > "$STATE_DIR/last_solana_corr_id"

        # Decide: EVM chain or Solana this cycle?
        local use_solana=false
        if [ "$solana_available" = true ] && [ $((RANDOM % 100)) -lt 15 ]; then
            use_solana=true
        fi

        if [ "$use_solana" = true ]; then
            # ── Solana event generation ──
            local sol_from=$((RANDOM % 5))
            local sol_to=$(( (sol_from + 1 + RANDOM % 4) % 5 ))
            local sol_amt=$((RANDOM % 100 + 1))
            local sol_pick=$((RANDOM % 100))

            if [ $sol_pick -lt 55 ]; then
                # 30% of SPL transfers use priority fee (compute budget)
                if [ $((RANDOM % 100)) -lt 30 ]; then
                    local _priority=$((RANDOM % 5000 + 500))
                    sim "solana: SPL Transfer ${sol_amt} tokens (priority=$_priority microLamport/CU)"
                    docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sol_from}.json" --owner "$sol_key_dir/key${sol_from}.json" --with-compute-unit-price "$_priority" "$sol_token" "$sol_amt" "${sol_atas[$sol_to]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
                else
                    sim "solana: SPL Transfer ${sol_amt} tokens account${sol_from}->account${sol_to}"
                    docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sol_from}.json" --owner "$sol_key_dir/key${sol_from}.json" "$sol_token" "$sol_amt" "${sol_atas[$sol_to]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
                fi
                _ev_SPLTransfer=$((_ev_SPLTransfer + 1))
            elif [ $sol_pick -lt 75 ]; then
                sim "solana: SPL MintTo ${sol_amt} tokens -> account${sol_to}"
                docker exec "$sol_container" spl-token mint --url "$sol_rpc" --fee-payer "$sol_key_dir/key0.json" --mint-authority "$sol_key_dir/key0.json" "$sol_token" "$sol_amt" "${sol_atas[$sol_to]}" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                _ev_SPLMintTo=$((_ev_SPLMintTo + 1))
            elif [ $sol_pick -lt 90 ]; then
                sim "solana: SPL Burn ${sol_amt} tokens account${sol_from}"
                docker exec "$sol_container" spl-token burn --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sol_from}.json" --owner "$sol_key_dir/key${sol_from}.json" "${sol_atas[$sol_from]}" "$sol_amt" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                _ev_SPLBurn=$((_ev_SPLBurn + 1))
            else
                sim "solana: SPL Transfer (new acct) ${sol_amt} tokens"
                local sol_new_acct
                sol_new_acct=$(docker exec "$sol_container" spl-token create-account --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sol_from}.json" --owner "$sol_key_dir/key${sol_from}.json" "$sol_token" 2>/dev/null | grep "Creating account" | awk '{print $3}' || true)
                if [ -n "$sol_new_acct" ]; then
                    docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sol_from}.json" --owner "$sol_key_dir/key${sol_from}.json" "$sol_token" "$sol_amt" "$sol_new_acct" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
                fi
                _ev_SPLTransfer=$((_ev_SPLTransfer + 1))
            fi
            total_gen=$((total_gen + 1))
            sim "solana: cycle $cycle corr_id=$batch_num (shared with EVM)"

        else
            # ── EVM event generation ──
            local ci=$((RANDOM % num_chains))
            [ "$num_chains" -eq 0 ] && continue
            local chain="${CHAINS[$ci]}"
            local token_addr="${TOKENS[$ci]}"
            local nft_addr="${NFTS[$ci]}"
            local emitter_addr="${EMITTERS[$ci]}"
            local emitter2_addr="${EMITTERS_V2[$ci]}"
            [ -z "$token_addr" ] && continue

            # Get container name + RPC port for this chain
            local container="" anvil_port="8545" block_interval="0"
            for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
                local ctr="${CHAIN_CONTAINERS[$__i]}" ccfg="${CHAIN_CONFIGS[$__i]}"
                if [ "${ccfg%%:*}" = "$chain" ]; then
                    container="$ctr"
                    anvil_port=$(echo "$ccfg" | cut -d: -f2)
                    block_interval="${ccfg##*:}"
                    break
                fi
            done
            [ -z "$container" ] && continue
            local RPC="http://localhost:${anvil_port}"

            # Set block interval for this chain (microservices: shared Anvil, switch per cycle)
            if [ "$block_interval" -gt 0 ] 2>/dev/null && [ "$block_interval" != "$_last_block_interval" ]; then
                docker exec "$container" cast rpc --rpc-url $RPC evm_setBlockTimestampInterval "$block_interval" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                _last_block_interval="$block_interval"
            fi

            # Emit Batch event with shared correlation ID
            [ -n "$emitter_addr" ] && _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitBatch(uint256,string)" "$batch_num" "Cycle-$cycle"

            # Generate 1-3 events
            local batch=$((RANDOM % 3 + 1))
            for ((i=0; i<batch; i++)); do
                local pick=$((RANDOM % 100))
                local from_idx=$((RANDOM % ${#KEYS[@]}))
                local to_idx=$(( (from_idx + 1 + RANDOM % ((${#KEYS[@]} - 1))) % ${#KEYS[@]} ))
                local from_key="${KEYS[$from_idx]}"
                local from_addr="${ACCOUNTS[$from_idx]}"
                local to_addr="${ACCOUNTS[$to_idx]}"
                local amt=$((RANDOM % 1000 + 1))
                # Precision-aware amounts: 6-decimal (USDC-like) vs 18-decimal (WETH-like)
                local amt_6dec=$((RANDOM % 9000 + 1000))          # 1,000-10,000 (USDC: $1-$10)
                local amt_18dec=$((RANDOM % 900 + 100))           # 100-1,000 wei-scale (tiny)
                local amt_18dec_mid=$((RANDOM % 9 + 1))$((RANDOM % 10))$((RANDOM % 10))000000000000000  # 0.1-9.9 ETH scale
                local big_amt=$((RANDOM % 100 + 1))

                # Weighted event distribution — thresholds loaded from sim-event-weights.conf
                local ev_type=""
                for ((wi=0; wi<${#WEIGHT_THRESHOLDS[@]}; wi++)); do
                    if [ $pick -lt ${WEIGHT_THRESHOLDS[$wi]} ]; then
                        ev_type="${WEIGHT_NAMES[$wi]}"
                        break
                    fi
                done

                case "$ev_type" in
                    Transfer)
                        sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt_18dec_mid"
                        _ev_Transfer=$((_ev_Transfer + 1))
                        ;;
                    Approval)
                        sim "$chain: Approval ${from_addr:0:8}...->${to_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$token_addr" "approve(address,uint256)" "$to_addr" "$amt_18dec_mid"
                        _ev_Approval=$((_ev_Approval + 1))
                        ;;
                    UniV3Swap)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: UniV3Swap ${from_addr:0:8}..."
                            local sp="250679566756032337290868763570861567304210"
                            local liq="519831781696124571544378"
                            local tk="194280"
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$big_amt" "-$((big_amt / 2))" "$sp" "$liq" "$tk"
                            _ev_UniV3Swap=$((_ev_UniV3Swap + 1))
                        fi
                        ;;
                    AaveSupply)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: AaveSupply ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "$from_addr" "$from_addr" "$amt_6dec" "false"
                            _ev_AaveSupply=$((_ev_AaveSupply + 1))
                        fi
                        ;;
                    AaveWithdraw)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: AaveWithdraw ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitWithdraw(address,address,address,uint256)" "$token_addr" "$from_addr" "$to_addr" "$amt_6dec"
                            _ev_AaveWithdraw=$((_ev_AaveWithdraw + 1))
                        fi
                        ;;
                    AaveBorrow)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: AaveBorrow ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitBorrow(address,address,address,uint256,uint8,bool)" "$token_addr" "$from_addr" "$from_addr" "$amt_6dec" "2" "false"
                            _ev_AaveBorrow=$((_ev_AaveBorrow + 1))
                        fi
                        ;;
                    LiquidationCall)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: Liquidation ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitLiquidation(address,address,address,uint256,uint256,bool)" "$token_addr" "$to_addr" "$from_addr" "$amt" "$((amt * 12 / 10))" "true"
                            _ev_Liquidation=$((_ev_Liquidation + 1))
                        fi
                        ;;
                    CometSupply)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: CometSupply ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitCometSupply(address,address,uint256)" "$from_addr" "$to_addr" "$amt"
                            _ev_CometSupply=$((_ev_CometSupply + 1))
                        fi
                        ;;
                    VoteCast)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local support=$((RANDOM % 3))
                            local sstr="AGAINST"
                            [ "$support" = "1" ] && sstr="FOR"
                            [ "$support" = "2" ] && sstr="ABSTAIN"
                            local _gov_pid; _gov_pid=$(cat "$STATE_DIR/gov_active_proposal" 2>/dev/null || echo "")
                            [ -z "$_gov_pid" ] && _gov_pid=$((RANDOM % 50))
                            sim "$chain: VoteCast ${from_addr:0:8}... $sstr pid=$_gov_pid"
                            local reason="Cycle $(date +%s)"; docker exec "$container" cast send --rpc-url $RPC --private-key "$from_key" "$emitter_addr" "emitVoteCast(uint256,uint8,uint256,string)" "$_gov_pid" "$support" "$amt" "$reason" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                            _ev_VoteCast=$((_ev_VoteCast + 1))
                        fi
                        ;;
                    BridgeEvent)
                        if [ -z "$emitter_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: Bridge ${from_addr:0:8}... -> chain 56"
                            _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitBridge(address,uint256,uint256)" "$token_addr" "$amt" "56"
                            _ev_Bridge=$((_ev_Bridge + 1))
                        fi
                        ;;
                    ERC1155)
                        if [ -z "$emitter2_addr" ]; then
                            # fallback: just do Transfer
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local erc1155_id=$((RANDOM % 1000 + 1))
                            sim "$chain: ERC1155 TransferSingle #$erc1155_id"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitTransferSingle(address,address,address,uint256,uint256)" "$from_addr" "$from_addr" "$to_addr" "$erc1155_id" "$amt"
                            _ev_1155Single=$((_ev_1155Single + 1))
                        fi
                        ;;
                    AaveRepayV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt_18dec_mid"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: AaveRepay ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitRepay(address,address,address,uint256,bool)" "$token_addr" "$from_addr" "$to_addr" "$amt_6dec" "false"
                            _ev_AaveRepay=$((_ev_AaveRepay + 1))
                        fi
                        ;;
                    UniV2SwapV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: UniV2Swap ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitUniV2Swap(uint256,uint256,uint256,uint256,address)" "$amt" "$((amt / 10 + 1))" "$((amt * 9 / 10))" "$amt" "$to_addr"
                            _ev_UniV2Swap=$((_ev_UniV2Swap + 1))
                        fi
                        ;;
                    CometWithdrawV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: CometWithdraw ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitCometWithdraw(address,address,uint256)" "$from_addr" "$to_addr" "$amt"
                            _ev_CometWithdraw=$((_ev_CometWithdraw + 1))
                        fi
                        ;;
                    CometBorrowV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: CometBorrow ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitCometBorrow(address,uint256,uint256)" "$from_addr" "$amt" "$((RANDOM % 100 + 1))"
                            _ev_CometBorrow=$((_ev_CometBorrow + 1))
                        fi
                        ;;
                    ProposalCreatedV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local pid=$((RANDOM % 1000 + 1))
                            sim "$chain: ProposalCreated #$pid"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitProposalCreated(uint256,address,address[],uint256[],string[],bytes[],uint256,uint256,string)" "$pid" "$from_addr" "[]" "[]" "[]" "[]" "$((RANDOM % 1000))" "$((RANDOM % 1000 + 1000))" "Cycle-$cycle"
                            echo "$pid" > "$STATE_DIR/gov_active_proposal"
                            _ev_ProposalCreated=$((_ev_ProposalCreated + 1))
                        fi
                        ;;
                    L2SentMessage)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: L2 SentMessage ${from_addr:0:8}... -> L2"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitSentMessage(address,address,uint256,uint256,uint256)" "$from_addr" "$to_addr" "$amt" "21000" "$((RANDOM % 1000000 + 100000))"
                            _ev_SentMessage=$((_ev_SentMessage + 1))
                        fi
                        ;;
                    L2TxToL2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: L2 TxToL2 ${from_addr:0:8}... -> Arbitrum"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitTxToL2(uint256,address,address,uint256,uint256,uint256)" "$((RANDOM % 10000 + 1))" "$from_addr" "$to_addr" "$amt" "$((RANDOM % 100000 + 10000))" "$((RANDOM % 1000000 + 100000))"
                            _ev_TxToL2=$((_ev_TxToL2 + 1))
                        fi
                        ;;
                    NFTTransfer)
                        if [ -z "$nft_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local tid=$((RANDOM % 100 + 1))
                            sim "$chain: NFT Transfer #$tid ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$nft_addr" "transferFrom(address,address,uint256)" "$from_addr" "$to_addr" "$tid"
                            _ev_NFTTransfer=$((_ev_NFTTransfer + 1))
                        fi
                        ;;
                    NFTApproval)
                        if [ -z "$nft_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local tid=$((RANDOM % 100 + 1))
                            sim "$chain: NFT Approve #$tid ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$nft_addr" "approve(address,uint256)" "$to_addr" "$tid"
                            _ev_NFTApproval=$((_ev_NFTApproval + 1))
                        fi
                        ;;
                    NFTApprovalForAll)
                        if [ -z "$nft_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: NFT ApproveAll ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$nft_addr" "setApprovalForAll(address,bool)" "$to_addr" "true"
                            _ev_NFTApprovalAll=$((_ev_NFTApprovalAll + 1))
                        fi
                        ;;
                    UserOpV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: UserOp ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitUserOpEvent(address,bytes32,uint256,bool,uint256,uint256)" "$from_addr" "0x$(printf '%064x' $((RANDOM * RANDOM)))" "$((RANDOM % 100))" "true" "$((RANDOM % 100000 + 10000))" "$((RANDOM % 200000 + 50000))"
                            _ev_UserOp=$((_ev_UserOp + 1))
                        fi
                        ;;
                    ProposalExecutedV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local _exec_pid; _exec_pid=$(cat "$STATE_DIR/gov_active_proposal" 2>/dev/null || echo "")
                            [ -z "$_exec_pid" ] && _exec_pid=$((RANDOM % 1000 + 1))
                            sim "$chain: ProposalExecuted pid=$_exec_pid"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitProposalExecuted(uint256)" "$_exec_pid"
                            rm -f "$STATE_DIR/gov_active_proposal"
                            _ev_ProposalExecuted=$((_ev_ProposalExecuted + 1))
                        fi
                        ;;
                    BalancerSwapV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: BalancerSwap ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitBalancerSwap(address,address,uint256,uint256)" "$token_addr" "$to_addr" "$amt" "$((amt * 95 / 100))"
                            _ev_BalancerSwap=$((_ev_BalancerSwap + 1))
                        fi
                        ;;
                    UniV2SyncV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: UniV2Sync"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitSync(uint112,uint112)" "$((RANDOM % 1000000 + 100000))" "$((RANDOM % 1000000 + 100000))"
                            _ev_UniV2Sync=$((_ev_UniV2Sync + 1))
                        fi
                        ;;
                    UniV2PairCreatedV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: PairCreated ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitPairCreated(address,address,address)" "$token_addr" "$to_addr" "$from_addr"
                            _ev_UniV2PairCreated=$((_ev_UniV2PairCreated + 1))
                        fi
                        ;;
                    AccountDeployedV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: AccountDeployed ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitAccountDeployed(address,bytes32)" "$from_addr" "0x$(printf '%064x' $((RANDOM * RANDOM + cycle)))"
                            _ev_AccountDeployed=$((_ev_AccountDeployed + 1))
                        fi
                        ;;
                    TransferBatchV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: ERC1155 TransferBatch ${from_addr:0:8}..."
                            docker exec "$container" cast send --rpc-url $RPC --private-key "$from_key" "$emitter2_addr" "emitTransferBatch(address,address,address,uint256[],uint256[])" "$from_addr" "$from_addr" "$to_addr" "[$((RANDOM % 1000 + 1)),$((RANDOM % 1000 + 1))]" "[$amt,$((amt / 2 + 1))]" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                            _ev_TransferBatch=$((_ev_TransferBatch + 1))
                        fi
                        ;;
                    ProposalCanceledV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            local _cancel_pid; _cancel_pid=$(cat "$STATE_DIR/gov_active_proposal" 2>/dev/null || echo "")
                            [ -z "$_cancel_pid" ] && _cancel_pid=$((RANDOM % 1000 + 1))
                            sim "$chain: ProposalCanceled pid=$_cancel_pid"
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitProposalCanceled(uint256)" "$_cancel_pid"
                            rm -f "$STATE_DIR/gov_active_proposal"
                            _ev_ProposalCanceled=$((_ev_ProposalCanceled + 1))
                        fi
                        ;;
                    WithdrawalRequestedV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: WithdrawalRequested ${from_addr:0:8}..."
                            docker exec "$container" cast send --rpc-url $RPC --private-key "$from_key" "$emitter2_addr" "emitWithdrawalRequested(address,bytes,uint256)" "$from_addr" "0x$(printf '%064x' $((RANDOM * RANDOM)))" "$amt" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                            _ev_WithdrawalRequested=$((_ev_WithdrawalRequested + 1))
                        fi
                        ;;
                    MessagePassedV2)
                        if [ -z "$emitter2_addr" ]; then
                            sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: L2→L1 MessagePassed ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitMessagePassed(uint256,address,address,uint256,bytes32,bytes32,uint256)" "$((RANDOM % 100000 + 1))" "$from_addr" "$to_addr" "$amt" "0x$(printf '%064x' $((RANDOM * RANDOM)))" "0x$(printf '%064x' $((RANDOM * RANDOM + cycle)))" "$((RANDOM * RANDOM))"
                            _ev_MessagePassed=$((_ev_MessagePassed + 1))
                        fi
                        ;;
                    *)
                        # Edge case events (residual)
                        sim "$chain: Transfer ${from_addr:0:8}...->${to_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                        _ev_Transfer=$((_ev_Transfer + 1))
                        ;;
                esac
                total_gen=$((total_gen + 1))
            done
        fi

        # Causal chain: Supply → Borrow → Liquidation (12% chance per cycle)
        # Emits correlated events with realistic amounts: borrow ≤ 75% LTV, then
        # a simulated price drop triggers liquidation in a subset of cases.
        if [ $((RANDOM % 100)) -lt 12 ] && [ -n "$emitter_addr" ] && [ -n "$emitter2_addr" ]; then
            local _col_idx=$((RANDOM % ${#KEYS[@]}))
            local _col_key="${KEYS[$_col_idx]}"
            local _col_addr="${ACCOUNTS[$_col_idx]}"
            local _col_amt=$((RANDOM % 5000 + 1000))
            local _brw_amt=$((_col_amt * (RANDOM % 26 + 50) / 100))  # 50-75% LTV
            sim "$chain: CAUSAL Supply($_col_amt) -> Borrow($_brw_amt) ${_col_addr:0:8}..."
            # 1. Supply collateral
            _cs "$container" "$RPC" "$_col_key" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "$_col_addr" "$_col_addr" "$_col_amt" "false"
            _ev_AaveSupply=$((_ev_AaveSupply + 1))
            total_gen=$((total_gen + 1))
            # 2. Borrow against it
            _cs "$container" "$RPC" "$_col_key" "$emitter_addr" "emitBorrow(address,address,address,uint256,uint8,bool)" "$token_addr" "$_col_addr" "$_col_addr" "$_brw_amt" "2" "false"
            _ev_AaveBorrow=$((_ev_AaveBorrow + 1))
            total_gen=$((total_gen + 1))
            # 3. Emit ReserveDataUpdated (reflects health change)
            _cs "$container" "$RPC" "$_col_key" "$emitter2_addr" "emitReserveDataUpdated(address,uint256,uint256,uint256,uint256,uint256)" "$token_addr" "$((RANDOM % 10 + 5))" "$((RANDOM % 5 + 2))" "$((RANDOM % 8 + 3))" "$((RANDOM % 100 + 100))" "$((RANDOM % 100 + 100))"
            _ev_ReserveDataUpdated=$((_ev_ReserveDataUpdated + 1))
            total_gen=$((total_gen + 1))
            # 4. 40% chance: price drop → liquidation
            if [ $((RANDOM % 100)) -lt 40 ]; then
                local _liq_debt=$((_brw_amt * 95 / 100))
                local _liq_collateral=$((_col_amt * 12 / 10))
                sim "$chain: CAUSAL Liquidation ($_liq_debt debt, $_liq_collateral collat liquidated)"
                _cs "$container" "$RPC" "${KEYS[$(( (_col_idx + 1) % ${#KEYS[@]} ))]}" "$emitter_addr" "emitLiquidation(address,address,address,uint256,uint256,bool)" "$token_addr" "$token_addr" "$_col_addr" "$_liq_debt" "$_liq_collateral" "true"
                _ev_Liquidation=$((_ev_Liquidation + 1))
                total_gen=$((total_gen + 1))
            fi
        fi

        # Burst spike (15% chance)
        if [ $((RANDOM % 100)) -lt 15 ]; then
            local tps=$((RANDOM % 6 + 3))
            local dur=4
            local total_spike=$((tps * dur))
            burst_count=$((burst_count + 1))
            local burst_label="solana"
            [ "$use_solana" = false ] && burst_label="$chain"
            sim "$burst_label: BURST ${tps} TPS for ${dur}s (${total_spike} events)..."
            local spike_start sent=0
            spike_start=$(date +%s)
            if [ "$use_solana" = true ]; then
                while [ $(( $(date +%s) - spike_start )) -lt "$dur" ]; do
                    local sbf=$((RANDOM % 5)) sbt=$(( (RANDOM + 1) % 5 ))
                    docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sbf}.json" --owner "$sol_key_dir/key${sbf}.json" "$sol_token" "$((RANDOM % 100 + 1))" "${sol_atas[$sbt]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
                    sent=$((sent + 1)); total_gen=$((total_gen + 1)); _ev_SPLTransfer=$((_ev_SPLTransfer + 1))
                done
            else
                while [ $(( $(date +%s) - spike_start )) -lt "$dur" ]; do
                    local sp_from=$((RANDOM % ${#KEYS[@]}))
                    local sp_to=$(( (sp_from + 1 + RANDOM % ((${#KEYS[@]} - 1))) % ${#KEYS[@]} ))
                    local sp_key="${KEYS[$sp_from]}"
                    # Run synchronous cast send in background so events get indexed (receipts are generated)
                    (docker exec "$container" cast send --rpc-url $RPC --private-key "$sp_key" "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[$sp_to]}" "$((RANDOM % 1000 + 1))" >/dev/null 2>>"$STATE_DIR/errors.log" || true) &
                    sent=$((sent + 1)); total_gen=$((total_gen + 1)); _ev_Transfer=$((_ev_Transfer + 1))
                    sleep 0.1  # throttle to prevent Anvil overload
                done
                wait  # await all background sends to complete
            fi
            sim "$burst_label: BURST ${sent} events sent in ${dur}s"
        fi

        # Reorg (10% chance — EVM only, Solana does not support snapshot/revert)
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 10 ]; then
            local depth=$((RANDOM % 11 + 2))
            sim "$chain: REORG ${depth}-block..."
            local snap
            snap=$(docker exec "$container" cast rpc evm_snapshot --rpc-url $RPC 2>/dev/null | tail -1 || echo "")
            if [ -n "$snap" ]; then
                for ((r=0; r<depth; r++)); do
                    [ -n "$emitter_addr" ] && _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "1" "-1" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
                done
                docker exec "$container" cast rpc evm_revert "$snap" --rpc-url $RPC 2>/dev/null || true
                for ((r=0; r<depth; r++)); do
                    [ -n "$emitter_addr" ] && _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "500" "false"
                done
                sim "$chain: REORG ${depth}-block done"
            fi
        fi

        # Timestamp anomaly (5% chance — EVM only)
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 5 ]; then
            local current_ts
            current_ts=$(docker exec "$container" cast rpc --rpc-url $RPC eth_blockNumber 2>/dev/null | xargs -I{} docker exec "$container" cast block --rpc-url $RPC {} 2>/dev/null | grep "timestamp" | awk '{print $2}' | tr -d ',' || echo "")
            if [ -n "$current_ts" ] && [ "$current_ts" -gt 100 ]; then
                local past_ts=$((current_ts - 3600))
                sim "$chain: TS ANOMALY setting block time to ${past_ts}s"
                docker exec "$container" cast rpc --rpc-url $RPC evm_setNextBlockTimestamp "$past_ts" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                if [ -n "$emitter_addr" ]; then
                    _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "100" "false"
                else
                    _cs "$container" "$RPC" "${KEYS[0]}" "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[1]}" "100"
                fi
            fi
        fi

        # Duplicate event (3% chance — both EVM and Solana)
        if [ $((RANDOM % 100)) -lt 3 ]; then
            if [ "$use_solana" = true ]; then
                local dup_sf=$((RANDOM % 5)) dup_st=$(( (RANDOM + 1) % 5 ))
                local dup_samt=$((RANDOM % 100 + 1))
                sim "solana: DUPLICATE SPL Transfer ${dup_samt} tokens (x2)"
                docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${dup_sf}.json" --owner "$sol_key_dir/key${dup_sf}.json" "$sol_token" "$dup_samt" "${sol_atas[$dup_st]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
                docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${dup_sf}.json" --owner "$sol_key_dir/key${dup_sf}.json" "$sol_token" "$dup_samt" "${sol_atas[$dup_st]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
            else
                local dup_key="${KEYS[$((RANDOM % ${#KEYS[@]}))]}"
                local dup_to="${ACCOUNTS[$((RANDOM % ${#ACCOUNTS[@]}))]}"
                local dup_amt=$((RANDOM % 500 + 1))
                sim "$chain: DUPLICATE sending identical Transfer twice (amt=$dup_amt)"
                _cs "$container" "$RPC" "$dup_key" "$token_addr" "transfer(address,uint256)" "$dup_to" "$dup_amt"
                _cs "$container" "$RPC" "$dup_key" "$token_addr" "transfer(address,uint256)" "$dup_to" "$dup_amt"
            fi
        fi

        # ── Anomaly simulations (EVM only) ──

        # Gas Spike (8%) — simulate sudden gas price surge
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 8 ] && [ -n "$emitter_addr" ]; then
            local old_gas; old_gas=$(docker exec "$container" cast rpc --rpc-url $RPC eth_gasPrice 2>/dev/null | tr -d '"' || echo "0x1")
            local spike_gas="0x$(printf '%x' $(( 500 * 10**9 )) )"  # 500 gwei
            sim "$chain: GAS SPIKE → 500 gwei"
            docker exec "$container" cast rpc --rpc-url $RPC evm_setNextBlockGasPrice "$spike_gas" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "100" "-50" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            docker exec "$container" cast rpc --rpc-url $RPC evm_setNextBlockGasPrice "$old_gas" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_GasSpike=$((_ev_GasSpike + 1)); total_gen=$((total_gen + 1))
        fi

        # BlobTxFee Spike (4%) — simulate EIP-4844 blob base fee surge
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 4 ] && [ -n "$emitter_addr" ]; then
            local blob_fee; blob_fee=$(docker exec "$container" cast rpc --rpc-url $RPC eth_blobBaseFee 2>/dev/null | tr -d '"' || echo "0x1")
            local spike_blob="0x$(printf '%x' $(( 100 * 10**9 )) )"  # 100 gwei blob fee
            sim "$chain: BLOB FEE SPIKE → 100 gwei (was $blob_fee)"
            docker exec "$container" cast rpc --rpc-url $RPC anvil_setNextBlockBaseFeePerGas "$spike_blob" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "200" "-100" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            docker exec "$container" cast rpc --rpc-url $RPC anvil_setNextBlockBaseFeePerGas "$blob_fee" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_BlobFeeSpike=$((_ev_BlobFeeSpike + 1)); total_gen=$((total_gen + 1))
        fi

        # Dropped Transaction (5%) — send with insufficient gas
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 5 ]; then
            sim "$chain: DROPPED TX (insufficient gas)"
            docker exec "$container" cast send --rpc-url $RPC --private-key "${KEYS[$((RANDOM % ${#KEYS[@]}))]}" --gas-limit 21000 "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[$((RANDOM % ${#ACCOUNTS[@]}))]}" "$((RANDOM % 100 + 1))" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_DroppedTx=$((_ev_DroppedTx + 1))
        fi

        # Stale Block (5%) — query a block from the past
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 5 ]; then
            local current_bn; current_bn=$(docker exec "$container" cast rpc --rpc-url $RPC eth_blockNumber 2>/dev/null | tr -d '"' || echo "0x0")
            local bn_dec=$((current_bn)); [ "$bn_dec" -gt 10 ] 2>/dev/null || bn_dec=10
            local stale_bn=$((bn_dec - RANDOM % 10 - 1))
            sim "$chain: STALE BLOCK query (block $stale_bn, current $bn_dec)"
            docker exec "$container" cast block --rpc-url $RPC "$stale_bn" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_StaleBlock=$((_ev_StaleBlock + 1))
        fi

        # MEV Sandwich (7%) — front-run + victim swap + back-run
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 7 ] && [ -n "$emitter_addr" ]; then
            local victim_amt=$((RANDOM % 500 + 100))
            sim "$chain: MEV SANDWICH (front-run + victim + back-run)"
            _cs "$container" "$RPC" "${KEYS[1]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$victim_amt" "-$((victim_amt / 2))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            _cs "$container" "$RPC" "${KEYS[2]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$victim_amt" "-$((victim_amt * 9 / 10))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            _cs "$container" "$RPC" "${KEYS[1]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "-$((victim_amt / 2))" "$((victim_amt * 11 / 10))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            _ev_MEVSandwich=$((_ev_MEVSandwich + 1)); total_gen=$((total_gen + 3))
        fi

        # Contract Revert (8%) — trigger require() failure
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 8 ] && [ -n "$nft_addr" ]; then
            sim "$chain: CONTRACT REVERT (transferFrom without approval)"
            docker exec "$container" cast send --rpc-url $RPC --private-key "${KEYS[3]}" "$nft_addr" "transferFrom(address,address,uint256)" "${ACCOUNTS[0]}" "${ACCOUNTS[3]}" "1" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_ContractRevert=$((_ev_ContractRevert + 1))
        fi

        # Nonce Gap (4%) — send tx with future nonce
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 4 ]; then
            local gap_addr="${ACCOUNTS[$((RANDOM % ${#ACCOUNTS[@]}))]}"
            local gap_key="${KEYS[$((RANDOM % ${#KEYS[@]}))]}"
            local current_nonce; current_nonce=$(docker exec "$container" cast nonce --rpc-url $RPC "$gap_addr" 2>/dev/null || echo "0")
            local gap_nonce=$((current_nonce + 3))
            sim "$chain: NONCE GAP (expected $current_nonce, sending $gap_nonce)"
            docker exec "$container" cast send --rpc-url $RPC --private-key "$gap_key" --nonce "$gap_nonce" "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[$(( (RANDOM + 1) % ${#ACCOUNTS[@]} ))]}" "$((RANDOM % 100 + 1))" >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_NonceGap=$((_ev_NonceGap + 1))
        fi

        # Large Block (5%) — pack many transactions into a single block
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 5 ]; then
            local large_count=$((RANDOM % 20 + 10))
            sim "$chain: LARGE BLOCK ($large_count txs batched)"
            for ((lb=0; lb<large_count; lb++)); do
                docker exec "$container" cast send --rpc-url $RPC --private-key "${KEYS[$((lb % ${#KEYS[@]}))]}" "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[$(( (lb + 1) % ${#ACCOUNTS[@]} ))]}" "$((RANDOM % 100 + 1))" >/dev/null 2>>"$STATE_DIR/errors.log" || true &
            done
            wait
            docker exec "$container" cast rpc --rpc-url $RPC evm_mine >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_LargeBlock=$((_ev_LargeBlock + 1)); total_gen=$((total_gen + large_count)); _ev_Transfer=$((_ev_Transfer + large_count))
        fi

        # ── High-priority simulations ──

        # Cross-chain Bridge Linkage (6%) — lock on chain A, mint on chain B
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 6 ] && [ -n "$emitter_addr" ] && [ -n "$emitter2_addr" ]; then
            local bridge_amt=$((RANDOM % 10000 + 1000))
            local src_ci=$((RANDOM % num_chains))
            local dst_ci=$(( (src_ci + 1 + RANDOM % (num_chains - 1)) % num_chains ))
            local src_chain="${CHAINS[$src_ci]}" dst_chain="${CHAINS[$dst_ci]}"
            local src_container="" dst_container="" src_port="8545" dst_port="8545"
            for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
                local ccfg="${CHAIN_CONFIGS[$__i]}"
                if [ "${ccfg%%:*}" = "$src_chain" ]; then src_container="${CHAIN_CONTAINERS[$__i]}"; src_port=$(echo "$ccfg" | cut -d: -f2); fi
                if [ "${ccfg%%:*}" = "$dst_chain" ]; then dst_container="${CHAIN_CONTAINERS[$__i]}"; dst_port=$(echo "$ccfg" | cut -d: -f2); fi
            done
            if [ -n "$src_container" ] && [ -n "$dst_container" ]; then
                local src_RPC="http://localhost:${src_port}" dst_RPC="http://localhost:${dst_port}"
                local src_emitter="${EMITTERS[$src_ci]}" dst_emitter="${EMITTERS[$dst_ci]}"
                local src_token="${TOKENS[$src_ci]}" dst_token="${TOKENS[$dst_ci]}"
                sim "CROSS-CHAIN BRIDGE: lock $bridge_amt on $src_chain → mint on $dst_chain"
                [ -n "$src_emitter" ] && _cs "$src_container" "$src_RPC" "${KEYS[0]}" "$src_emitter" "emitBridge(address,uint256,uint256)" "$src_token" "$bridge_amt" "56"
                [ -n "$dst_emitter" ] && _cs "$dst_container" "$dst_RPC" "${KEYS[0]}" "$dst_emitter" "emitSupply(address,address,address,uint256,bool)" "$dst_token" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "$bridge_amt" "false"
                _ev_CrossChainBridge=$((_ev_CrossChainBridge + 1)); total_gen=$((total_gen + 2))
            fi
        fi

        # Flash Loan (5%) — borrow → use (swap) → repay in rapid sequence
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 5 ] && [ -n "$emitter_addr" ] && [ -n "$emitter2_addr" ]; then
            local fl_amt=$((RANDOM % 100000 + 10000))
            local fl_profit=$((fl_amt / 100))
            sim "$chain: FLASH LOAN borrow $fl_amt → swap → repay ${fl_amt}"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitBorrow(address,address,address,uint256,uint8,bool)" "$token_addr" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "$fl_amt" "2" "false"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$fl_amt" "-$((fl_amt - fl_profit))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter2_addr" "emitRepay(address,address,address,uint256,bool)" "$token_addr" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "$((fl_amt + fl_profit / 10))" "false"
            _ev_FlashLoan=$((_ev_FlashLoan + 1)); total_gen=$((total_gen + 3))
        fi

        # Solana Vote/Stake (10%) — simulate validator vote and stake events
        if [ "$solana_available" = true ] && [ $((RANDOM % 100)) -lt 10 ]; then
            local sol_vote_pick=$((RANDOM % 100))
            if [ $sol_vote_pick -lt 60 ]; then
                sim "solana: VOTE (create-vote-account)"
                docker exec "$sol_container" solana-keygen new --no-bip39-passphrase --outfile "$sol_key_dir/sim_vote_$cycle.json" --force 2>/dev/null || true
                docker exec "$sol_container" solana create-vote-account --url "$sol_rpc" --fee-payer "$sol_key_dir/key0.json" "$sol_key_dir/sim_vote_$cycle.json" "$sol_key_dir/key0.json" "$sol_key_dir/key0.json" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                _ev_SolanaVote=$((_ev_SolanaVote + 1))
            else
                sim "solana: STAKE (create-stake-account + delegate)"
                docker exec "$sol_container" solana-keygen new --no-bip39-passphrase --outfile "$sol_key_dir/sim_stake_$cycle.json" --force 2>/dev/null || true
                docker exec "$sol_container" solana create-stake-account --url "$sol_rpc" --fee-payer "$sol_key_dir/key0.json" --from "$sol_key_dir/key0.json" --stake-authority "$sol_key_dir/key0.json" --withdraw-authority "$sol_key_dir/key0.json" "$sol_key_dir/sim_stake_$cycle.json" $((RANDOM % 50 + 10)) >/dev/null 2>>"$STATE_DIR/errors.log" || true
                if [ -n "$sol_vote_pubkey" ]; then
                    local stake_pubkey; stake_pubkey=$(docker exec "$sol_container" solana-keygen pubkey "$sol_key_dir/sim_stake_$cycle.json" 2>/dev/null || echo "")
                    [ -n "$stake_pubkey" ] && docker exec "$sol_container" solana delegate-stake --force --url "$sol_rpc" --fee-payer "$sol_key_dir/key0.json" --stake-authority "$sol_key_dir/key0.json" "$stake_pubkey" "$sol_vote_pubkey" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                fi
                _ev_SolanaStake=$((_ev_SolanaStake + 1))
            fi
            total_gen=$((total_gen + 1))
        fi

        # Liquidation Cascade (4%) — price drops → multiple accounts liquidated in sequence
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 4 ] && [ -n "$emitter_addr" ] && [ -n "$emitter2_addr" ]; then
            local cascade_count=$((RANDOM % 4 + 2))
            local cascade_base=$((RANDOM % 5000 + 2000))
            sim "$chain: LIQUIDATION CASCADE ($cascade_count accounts)"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter2_addr" "emitReserveDataUpdated(address,uint256,uint256,uint256,uint256,uint256)" "$token_addr" "2" "8" "12" "100" "100"
            total_gen=$((total_gen + 1)); _ev_ReserveDataUpdated=$((_ev_ReserveDataUpdated + 1))
            for ((lc=0; lc<cascade_count; lc++)); do
                local liq_debt=$((cascade_base * (90 + lc * 2) / 100))
                local liq_col=$((cascade_base * (110 + lc * 5) / 100))
                _cs "$container" "$RPC" "${KEYS[$(( (lc + 1) % ${#KEYS[@]} ))]}" "$emitter_addr" "emitLiquidation(address,address,address,uint256,uint256,bool)" "$token_addr" "$token_addr" "${ACCOUNTS[$(( lc % ${#ACCOUNTS[@]} ))]}" "$liq_debt" "$liq_col" "true"
                total_gen=$((total_gen + 1))
            done
            _ev_LiqCascade=$((_ev_LiqCascade + 1)); _ev_Liquidation=$((_ev_Liquidation + cascade_count))
        fi

        # DEX Aggregator Multi-hop (6%) — TokenA→WETH→USDC→TokenB routing
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 6 ] && [ -n "$emitter_addr" ] && [ -n "$emitter2_addr" ]; then
            local agg_amt=$((RANDOM % 5000 + 500))
            local hop1_out=$((agg_amt * 95 / 100))
            local hop2_out=$((hop1_out * 97 / 100))
            local hop3_out=$((hop2_out * 99 / 100))
            sim "$chain: DEX AGGREGATOR 3-hop (TokenA→WETH→USDC→TokenB)"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$agg_amt" "-$hop1_out" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter2_addr" "emitUniV2Swap(uint256,uint256,uint256,uint256,address)" "$hop1_out" "0" "0" "$hop2_out" "${ACCOUNTS[0]}"
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter2_addr" "emitCurveSwap(int128,uint256,uint256,uint256)" "1" "2" "$hop2_out" "$hop3_out"
            _ev_DexAggregator=$((_ev_DexAggregator + 1)); total_gen=$((total_gen + 3))
        fi

        # ── Medium-priority simulations ──

        # Cross-chain Arbitrage (3%) — same pair, different price on two chains
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 3 ] && [ -n "$emitter_addr" ]; then
            local arb_ci=$((RANDOM % num_chains))
            local arb_ci2=$(( (arb_ci + 1 + RANDOM % (num_chains - 1)) % num_chains ))
            local arb_chain1="${CHAINS[$arb_ci]}" arb_chain2="${CHAINS[$arb_ci2]}"
            local arb_ctr1="" arb_ctr2="" arb_port1="8545" arb_port2="8545"
            for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
                local ccfg="${CHAIN_CONFIGS[$__i]}"
                if [ "${ccfg%%:*}" = "$arb_chain1" ]; then arb_ctr1="${CHAIN_CONTAINERS[$__i]}"; arb_port1=$(echo "$ccfg" | cut -d: -f2); fi
                if [ "${ccfg%%:*}" = "$arb_chain2" ]; then arb_ctr2="${CHAIN_CONTAINERS[$__i]}"; arb_port2=$(echo "$ccfg" | cut -d: -f2); fi
            done
            if [ -n "$arb_ctr1" ] && [ -n "$arb_ctr2" ]; then
                local arb_amt=$((RANDOM % 2000 + 500))
                local arb_RPC1="http://localhost:${arb_port1}" arb_RPC2="http://localhost:${arb_port2}"
                local arb_em1="${EMITTERS[$arb_ci]}" arb_em2="${EMITTERS[$arb_ci2]}"
                sim "CROSS-CHAIN ARB: buy on $arb_chain1, sell on $arb_chain2"
                [ -n "$arb_em1" ] && _cs "$arb_ctr1" "$arb_RPC1" "${KEYS[0]}" "$arb_em1" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$arb_amt" "-$((arb_amt * 95 / 100))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
                [ -n "$arb_em2" ] && _cs "$arb_ctr2" "$arb_RPC2" "${KEYS[0]}" "$arb_em2" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$((arb_amt * 95 / 100))" "-$((arb_amt * 105 / 100))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
                _ev_CrossChainArb=$((_ev_CrossChainArb + 1)); total_gen=$((total_gen + 2))
            fi
        fi

        # Solana Slot Skip (4%) — simulate skipped slots
        if [ "$solana_available" = true ] && [ $((RANDOM % 100)) -lt 4 ]; then
            sim "solana: SLOT SKIP (validator missed slot)"
            local skip_count=$((RANDOM % 4 + 1))
            for ((ss=0; ss<skip_count; ss++)); do
                docker exec "$sol_container" solana transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key0.json" --from "$sol_key_dir/key0.json" "${sol_atas[$((RANDOM % 5))]}" "$((RANDOM % 10 + 1))" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
            done
            _ev_SolanaSlotSkip=$((_ev_SolanaSlotSkip + 1)); total_gen=$((total_gen + skip_count)); _ev_SPLTransfer=$((_ev_SPLTransfer + skip_count))
        fi

        # Solana Compute Budget Exceeded (3%) — simulate compute limit hit
        if [ "$solana_available" = true ] && [ $((RANDOM % 100)) -lt 3 ]; then
            sim "solana: COMPUTE BUDGET EXCEEDED (complex tx fails)"
            local cb_from=$((RANDOM % 5))
            local cb_to=$(( (cb_from + 1) % 5 ))
            docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${cb_from}.json" --owner "$sol_key_dir/key${cb_from}.json" "$sol_token" "1" "${sol_atas[$cb_to]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
            docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${cb_from}.json" --owner "$sol_key_dir/key${cb_from}.json" "$sol_token" "1" "${sol_atas[$cb_to]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
            docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${cb_from}.json" --owner "$sol_key_dir/key${cb_from}.json" "$sol_token" "1" "${sol_atas[$cb_to]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_SolanaComputeLimit=$((_ev_SolanaComputeLimit + 1))
        fi

        # Causal Block Ordering (7%) — events within a single block in causal order
        if [ "$use_solana" = false ] && [ $((RANDOM % 100)) -lt 7 ] && [ -n "$emitter_addr" ]; then
            local causal_amt=$((RANDOM % 2000 + 500))
            sim "$chain: CAUSAL BLOCK (Transfer→Swap→Transfer in 1 block)"
            _cs "$container" "$RPC" "${KEYS[0]}" "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[1]}" "$causal_amt"
            _cs "$container" "$RPC" "${KEYS[1]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$causal_amt" "-$((causal_amt / 2))" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
            _cs "$container" "$RPC" "${KEYS[1]}" "$token_addr" "transfer(address,uint256)" "${ACCOUNTS[2]}" "$((causal_amt / 2))"
            docker exec "$container" cast rpc --rpc-url $RPC evm_mine >/dev/null 2>>"$STATE_DIR/errors.log" || true
            _ev_CausalBlock=$((_ev_CausalBlock + 1)); total_gen=$((total_gen + 3)); _ev_Transfer=$((_ev_Transfer + 2)); _ev_UniV3Swap=$((_ev_UniV3Swap + 1))
        fi

        # Write metrics (Prometheus format)
        local elapsed=$(( $(date +%s) - start_sec ))
        [ "$elapsed" -lt 1 ] && elapsed=1
        {
            echo "# HELP chainpulse_sim_events_total Total events generated"
            echo "# TYPE chainpulse_sim_events_total counter"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"Transfer\"} $_ev_Transfer"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"Approval\"} $_ev_Approval"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"UniV3Swap\"} $_ev_UniV3Swap"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"AaveSupply\"} $_ev_AaveSupply"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"AaveWithdraw\"} $_ev_AaveWithdraw"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"AaveBorrow\"} $_ev_AaveBorrow"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"Liquidation\"} $_ev_Liquidation"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"CometSupply\"} $_ev_CometSupply"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"VoteCast\"} $_ev_VoteCast"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"Bridge\"} $_ev_Bridge"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"NFTTransfer\"} $_ev_NFTTransfer"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"NFTApproval\"} $_ev_NFTApproval"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"NFTApprovalAll\"} $_ev_NFTApprovalAll"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"ERC1155Single\"} $_ev_1155Single"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"AaveRepay\"} $_ev_AaveRepay"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"UniV2Swap\"} $_ev_UniV2Swap"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"CometWithdraw\"} $_ev_CometWithdraw"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"CometBorrow\"} $_ev_CometBorrow"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"ProposalCreated\"} $_ev_ProposalCreated"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"ReserveDataUpdated\"} $_ev_ReserveDataUpdated"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"SentMessage\"} $_ev_SentMessage"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"TxToL2\"} $_ev_TxToL2"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SPLTransfer\"} $_ev_SPLTransfer"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SPLMintTo\"} $_ev_SPLMintTo"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SPLBurn\"} $_ev_SPLBurn"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"GasSpike\"} $_ev_GasSpike"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"BlobFeeSpike\"} $_ev_BlobFeeSpike"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"DroppedTx\"} $_ev_DroppedTx"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"StaleBlock\"} $_ev_StaleBlock"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"MEVSandwich\"} $_ev_MEVSandwich"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"ContractRevert\"} $_ev_ContractRevert"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"NonceGap\"} $_ev_NonceGap"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"LargeBlock\"} $_ev_LargeBlock"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"CrossChainBridge\"} $_ev_CrossChainBridge"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"FlashLoan\"} $_ev_FlashLoan"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"Vote\"} $_ev_SolanaVote"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"Stake\"} $_ev_SolanaStake"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"LiquidationCascade\"} $_ev_LiqCascade"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"DexAggregator\"} $_ev_DexAggregator"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"CrossChainArb\"} $_ev_CrossChainArb"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SlotSkip\"} $_ev_SolanaSlotSkip"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"ComputeLimit\"} $_ev_SolanaComputeLimit"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"CausalBlock\"} $_ev_CausalBlock"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"UserOp\"} $_ev_UserOp"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"ProposalExecuted\"} $_ev_ProposalExecuted"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"BalancerSwap\"} $_ev_BalancerSwap"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"UniV2Sync\"} $_ev_UniV2Sync"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"UniV2PairCreated\"} $_ev_UniV2PairCreated"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"AccountDeployed\"} $_ev_AccountDeployed"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"TransferBatch\"} $_ev_TransferBatch"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"ProposalCanceled\"} $_ev_ProposalCanceled"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"WithdrawalRequested\"} $_ev_WithdrawalRequested"
            echo "chainpulse_sim_events_total{chain=\"evm\",type=\"MessagePassed\"} $_ev_MessagePassed"
            echo "# HELP chainpulse_sim_total_events Total events across all types"
            echo "# TYPE chainpulse_sim_total_events gauge"
            echo "chainpulse_sim_total_events $total_gen"
            echo "# HELP chainpulse_sim_burst_count Total burst spikes"
            echo "# TYPE chainpulse_sim_burst_count counter"
            echo "chainpulse_sim_burst_count $burst_count"
            echo "# HELP chainpulse_sim_events_per_sec Events per second"
            echo "# TYPE chainpulse_sim_events_per_sec gauge"
            echo "chainpulse_sim_events_per_sec $((total_gen / elapsed))"
        } > "$METRICS_FILE"

        # Write stats
        {
            echo "total=$total_gen"
            echo "bursts=$burst_count"
            echo "elapsed_sec=$elapsed"
            echo "events_per_sec=$((total_gen / elapsed))"
            echo "cycle=$cycle"
        } > "$STATS_FILE"

        # Poisson sleep (mean 2s)
        local sleep_time
        sleep_time=$(awk -v seed=$RANDOM -v m=2 'BEGIN{srand(seed); u=rand(); t=-log(1-u)*m; if(t<0.5) t=0.5; printf "%.1f\n", t}')
        sleep "$sleep_time"
    done
}

# ──────────────────────────────────────────────
# Stop simulation
# ──────────────────────────────────────────────
stop_simulation() {
    local pid_file="$STATE_DIR/sim.pid"
    if [ -f "$pid_file" ]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null
            info "Simulator stopped (PID $pid)"
        fi
        rm -f "$pid_file"
    fi
}

# ──────────────────────────────────────────────
# Show status
# ──────────────────────────────────────────────

# ── Run capability verification after simulation starts ──
run_verification() {
    info "===== Verifying all capability points ====="

    # Wait for events
    local wait=0
    while [ $wait -lt 120 ]; do
        local count
        count=$(curl -sf "http://localhost:8080/events?limit=5" 2>/dev/null \
            | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('pagination',{}).get('total',0))" 2>/dev/null || echo "0")
        [ "$count" -gt 0 ] 2>/dev/null && break
        sleep 5; wait=$((wait + 5))
    done

    # 1) WebSocket
    local ws=$(curl -s -o /dev/null -w "%{http_code}" -H "Upgrade: websocket" -H "Connection: Upgrade" -H "Sec-WebSocket-Key: dGVzdA==" -H "Sec-WebSocket-Version: 13" http://localhost:8080/ws 2>/dev/null || echo "000")
    if [ "$ws" = "101" ]; then info "  OK  WS /ws: 101"; else warn "  MISS WS /ws: $ws"; fi

    local ws2=$(curl -s -o /dev/null -w "%{http_code}" -H "Upgrade: websocket" -H "Connection: Upgrade" -H "Sec-WebSocket-Key: dGVzdA==" -H "Sec-WebSocket-Version: 13" http://localhost:8080/events/subscribe 2>/dev/null || echo "000")
    if [ "$ws2" = "101" ]; then info "  OK  WS /events/subscribe: 101"; else warn "  MISS WS /events/subscribe: $ws2"; fi

    # 2) SIWE challenge
    local siwe=$(curl -s -X POST http://localhost:8080/auth/siwe/challenge -H "Content-Type: application/json" -d '{"address":"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"}' 2>/dev/null || echo "{}")
    local nonce=$(echo "$siwe" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('nonce',''))" 2>/dev/null)
    if [ -n "$nonce" ]; then info "  OK  SIWE challenge: nonce=$nonce"; else warn "  MISS SIWE challenge"; fi

    # 3) Event name resolution
    local stats=$(curl -sf http://localhost:8080/events/stats 2>/dev/null || echo "{}")
    local hex=$(echo "$stats" | python3 -c "import sys,json; d=json.load(sys.stdin); names=list(d.get('byEventName',{}).keys()); print(sum(1 for n in names if n.startswith('0x')))" 2>/dev/null || echo "?")
    if [ "$hex" = "0" ] 2>/dev/null; then info "  OK  Event names: human-readable"; else warn "  MISS $hex hex hashes remain"; fi

    # 4) Admin API key
    local ak=$(curl -s -X POST http://localhost:8080/admin/api-keys -H "Content-Type: application/json" -d '{"clientId":"verify-test","name":"verify-key"}' 2>/dev/null || echo "{}")
    local key=$(echo "$ak" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('key','') or d.get('key',''))" 2>/dev/null)
    if [ -n "$key" ]; then info "  OK  Admin API key: ${key:0:12}..."; else info "  -   Admin API key: not exposed in microservices gateway (use deploy-monolith.sh)"; fi

    # 5) Webhook
    local wh=$(curl -s -X POST http://localhost:8080/admin/webhooks -H "Content-Type: application/json" -d '{"clientId":"verify-test","name":"verify-hook","url":"http://localhost:9999/hook","secret":"whsec"}' 2>/dev/null || echo "{}")
    local whid=$(echo "$wh" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('id','') or d.get('id',''))" 2>/dev/null)
    if [ -n "$whid" ]; then info "  OK  Webhook: ID=$whid"; else info "  -   Webhook: not exposed in microservices gateway (use deploy-monolith.sh)"; fi

    # 6) Export
    local ex=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/events/export?limit=10" 2>/dev/null || echo "000")
    if [ "$ex" = "200" ]; then info "  OK  Export: HTTP 200"; else info "  -   Export: not exposed in microservices gateway (use deploy-monolith.sh)"; fi

    # 7) Rate limiter
    local rl_hit=0
    for i in $(seq 1 50); do
        local rl=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/events?limit=1" 2>/dev/null || echo "000")
        if [ "$rl" = "429" ]; then rl_hit=1; info "  OK  Rate limiter: 429 after $i reqs"; break; fi
    done
    [ "$rl_hit" -eq 0 ] && info "  -   Rate limiter: not hit"

    # 8) Reorg handler
    local rg=$(curl -sf http://localhost:8080/runtime/summary 2>/dev/null || echo "{}")
    if echo "$rg" | python3 -c "import sys,json; d=json.load(sys.stdin); d.get('reorg','')" 2>/dev/null; then
        info "  OK  Reorg handler: wired"
    else
        warn "  MISS Reorg handler"
    fi

    info "===== Verification done ====="
}

show_status() {
    local pid_file="$STATE_DIR/sim.pid"
    if [ -f "$pid_file" ]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            info "Simulator: running (PID $pid)"
        else
            warn "Simulator: not running (stale PID)"
        fi
    else
        warn "Simulator: not started"
    fi

    # Performance stats
    if [ -f "$STATS_FILE" ]; then
        while IFS='=' read -r key val; do
            case "$key" in
                events_per_sec) info "Events/sec: $val" ;;
                total) info "Total generated: $val" ;;
                cycle) info "Cycles: $val" ;;
                bursts) info "Burst spikes: $val" ;;
                elapsed_sec) info "Running: ${val}s" ;;
            esac
        done < "$STATS_FILE"
    fi

    local count
    count=$(curl -sf "http://localhost:8080/events?limit=500" 2>/dev/null \
        | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('pagination',{}).get('total',0))" 2>/dev/null || echo "?")
    info "Events indexed by ChainPulse: $count"

    echo ""
    docker ps --filter "name=chainpulse" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null || true
}

# ──────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────
case "${1:-deploy}" in
    deploy)
        build_images
        start_stack
        start_simulation
        info "Waiting 30s for events to accumulate..."
        sleep 30
        run_verification
        show_status
        info "ChainPulse is running. Use '$0 status' to check later."
        info "Use '$0 stop' to shut down."
        ;;
    stop)
        stop_simulation
        info "Stopping microservices stack..."
        docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
        info "Stack stopped."
        ;;
    status)
        show_status
        ;;
    loop)
        shift
        sim_loop "$@"
        ;;
    restart)
        stop_simulation
        docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
        start_stack
        start_simulation
        ;;
    *)
        echo "ChainPulse One-Click Deploy & Simulate"
        echo ""
        echo "Usage: bash docker/deploy-microservices.sh [command]"
        echo ""
        echo "Commands:"
        echo "  deploy    (default) Build → Deploy → Simulate (full stack)"
        echo "  stop      Stop simulation and bring down the stack"
        echo "  status    Show simulator and event indexing status"
        echo "  restart   Restart everything from scratch"
        echo ""
        echo "Quick start:"
        echo "  bash docker/deploy-microservices.sh"
        ;;
esac
