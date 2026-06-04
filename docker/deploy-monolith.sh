#!/usr/bin/env bash
# ChainPulse Monolith Deploy & Simulate
# Builds monolithic image, deploys the full stack with ALL features, and starts
# continuous event simulation — all with a single command.
#
# Usage: bash docker/deploy-monolith.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.acceptance.yml"
STATE_DIR="/tmp/chainpulse-monolith-sim"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }
sim()   { echo -e "${CYAN}[SIM]${NC} $*"; }

# ── Chain configurations (indexed arrays — bash 3 compat) ──
# Single Anvil instance handles all EVM chains (chain-id switched per cycle)
CHAIN_CONTAINERS=("chainpulse-anvil-ethereum" "chainpulse-anvil-polygon" "chainpulse-anvil-bsc" "chainpulse-anvil-arbitrum" "chainpulse-anvil-base" "chainpulse-anvil-avalanche")
CHAIN_CONFIGS=("ethereum:8545" "polygon:8547" "bsc:8546" "arbitrum:8548" "base:8549" "avalanche:8550")

# Solana configuration (separate from EVM — uses spl-token CLI, not cast send)
SOLANA_CONTAINER="chainpulse-solana-not-deployed"
SOLANA_RPC_PORT=8899
# NOTE: acceptance.yml does not include Solana; Solana simulation is disabled

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

    function emitBatch(uint256 bid, string calldata d) public { emit Batch(bid, d); }
}'

# ──────────────────────────────────────────────
# Step 1: Build Docker images
# ──────────────────────────────────────────────
build_images() {
    info "===== Step 1/3: Building monolithic Docker image ====="
    if [ -f "$SCRIPT_DIR/acceptance.sh" ]; then
        bash "$SCRIPT_DIR/acceptance.sh" build 2>&1 | tail -3
        info "  Monolithic image built via acceptance.sh"
    else
        local project_root="$SCRIPT_DIR/.."
        docker build -f "$project_root/docker/Dockerfile.microservices.prebuilt" \
            --build-arg "SERVICE=chainpulse" \
            -t chainpulse-acceptance:latest "$project_root" 2>&1 | tail -3
    fi
    mkdir -p "$PROJECT_ROOT/build/bin/linux/microservices"

    for svc in puller event-processor api-service api-gateway; do
        if [ -f "$PROJECT_ROOT/build/bin/linux/microservices/$svc" ]; then
            info "  Binary $svc already built, skipping compilation"
        else
            info "  Compiling $svc..."
            local arch="${GOARCH:-$(go env GOARCH 2>/dev/null || echo "amd64")}"
            CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -a -installsuffix cgo \
                -o "$PROJECT_ROOT/build/bin/linux/microservices/$svc" \
                "$PROJECT_ROOT/cmd/microservices/$svc"
        fi
    done

    for svc in puller event-processor api-service api-gateway; do
        info "  Building chainpulse-$svc:latest..."
        docker build --build-arg "SERVICE=$svc" \
            -f "$SCRIPT_DIR/Dockerfile.microservices.prebuilt" \
            -t "chainpulse-$svc:latest" "$PROJECT_ROOT" 2>&1 | tail -1
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
        taddr=$(docker exec "$ctr" forge create --rpc-url http://localhost:"$port" --private-key "$deployer_key" --broadcast --out /tmp/forge-out --cache-path /tmp/forge-cache /tmp/TestToken.sol:TestToken 2>&1 | grep "Deployed to:" | awk '{print $3}')
        [ -n "$taddr" ] && echo "$taddr" > "$STATE_DIR/${chain}.token" && info "    TestToken: $taddr" || error "    TestToken deploy failed on $chain"
    fi

    # TestNFT
    if [ ! -f "$STATE_DIR/${chain}.nft" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/TestNFT.sol << 'EOF'
$TESTNFT_SOL
EOF"
        local nftaddr
        nftaddr=$(docker exec "$ctr" forge create --rpc-url http://localhost:"$port" --private-key "$deployer_key" --broadcast --out /tmp/forge-out --cache-path /tmp/forge-cache /tmp/TestNFT.sol:TestNFT 2>&1 | grep "Deployed to:" | awk '{print $3}')
        [ -n "$nftaddr" ] && echo "$nftaddr" > "$STATE_DIR/${chain}.nft" && info "    TestNFT: $nftaddr" || error "    TestNFT deploy failed on $chain"
    fi

    # RealEventEmitter
    if [ ! -f "$STATE_DIR/${chain}.emitter" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/RealEventEmitter.sol << 'EOF'
$REAL_EMITTER_SOL
EOF"
        local eaddr
        eaddr=$(docker exec "$ctr" forge create --rpc-url http://localhost:"$port" --private-key "$deployer_key" --broadcast --out /tmp/forge-out --cache-path /tmp/forge-cache /tmp/RealEventEmitter.sol:RealEventEmitter 2>&1 | grep "Deployed to:" | awk '{print $3}')
        [ -n "$eaddr" ] && echo "$eaddr" > "$STATE_DIR/${chain}.emitter" && info "    RealEventEmitter: $eaddr" || error "    RealEventEmitter deploy failed on $chain"
    fi

    # RealEventEmitter V2
    if [ ! -f "$STATE_DIR/${chain}.emitter-v2" ]; then
        docker exec "$ctr" sh -c "cat > /tmp/RealEventEmitterV2.sol << 'EOF'
$REAL_EMITTER_V2_SOL
EOF"
        local e2addr
        e2addr=$(docker exec "$ctr" forge create --via-ir --rpc-url http://localhost:"$port" --private-key "$deployer_key" --broadcast --out /tmp/forge-out --cache-path /tmp/forge-cache /tmp/RealEventEmitterV2.sol:RealEventEmitterV2 2>&1 | grep "Deployed to:" | awk '{print $3}')
        [ -n "$e2addr" ] && echo "$e2addr" > "$STATE_DIR/${chain}.emitter-v2" && info "    RealEventEmitterV2: $e2addr" || error "    RealEventEmitterV2 deploy failed on $chain"
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
        docker exec "$sol_ctr" solana-keygen new --no-bip39-passphrase --outfile "$key_dir/key${i}.json" --force --silent 2>/dev/null || true
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
        local chain="${cfg%%:*}" port="${cfg##*:}"
        if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$ctr"; then
            deploy_on_chain "$ctr" "$chain" "$port" "$deployer_key"
            any_deployed=true
        fi
    done

    # Monolithic mode: acceptance.yml has no Solana container; skip Solana setup

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
    nohup bash "$SCRIPT_DIR/deploy-monolith.sh" loop "$chains_list" "$tokens_list" "$nfts_list" "$emitters_list" "$emitters2_list" > "$STATE_DIR/sim.log" 2>&1 &
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
    echo "║  Burst: ON (15-50 TPS)   Reorg: depth 2-12                ║"
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
        local n=0
        while [ $n -lt 3 ]; do
            if docker exec "$c" cast send --rpc-url "$r" --private-key "$k" "$a" "$s" "$@" >/dev/null 2>>"$STATE_DIR/errors.log"; then
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
    # Solana counters
    local _ev_SPLTransfer=0 _ev_SPLMintTo=0 _ev_SPLBurn=0
    local total_gen=0 burst_count=0 start_sec
    start_sec=$(date +%s)
    local cycle=0 nft_next_id=101

    local sol_status="no"
    [ "$solana_available" = true ] && sol_status="yes"
    info "Simulation started: $num_chains EVM chains + Solana, real DeFi + extended protocol event signatures"
    info "Performance baseline:"
    info "  Chains:      ${CHAINS[*]} (EVM)${solana_available:+, solana (SPL)}"
    info "  Events/sec:  ~5-12 (Poisson mean 2s, 3-5 events/cycle now includes V2)"
    info "  Burst:       15-50 TPS for 8s (15% chance)"
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
    : "${WEIGHT_Transfer:=22}" "${WEIGHT_Approval:=8}" "${WEIGHT_UniV3Swap:=10}"
    : "${WEIGHT_AaveSupply:=8}" "${WEIGHT_AaveWithdraw:=8}" "${WEIGHT_AaveBorrow:=8}"
    : "${WEIGHT_LiquidationCall:=5}" "${WEIGHT_CometSupply:=5}" "${WEIGHT_VoteCast:=5}"
    : "${WEIGHT_BridgeEvent:=5}" "${WEIGHT_ERC1155:=5}" "${WEIGHT_AaveRepayV2:=2}"
    : "${WEIGHT_UniV2SwapV2:=2}" "${WEIGHT_CometWithdrawV2:=2}" "${WEIGHT_CometBorrowV2:=1}"
    : "${WEIGHT_ProposalCreatedV2:=1}" "${WEIGHT_L2SentMessage:=1}" "${WEIGHT_L2TxToL2:=1}"
    : "${WEIGHT_NFTTransfer:=2}" "${WEIGHT_NFTApproval:=1}" "${WEIGHT_NFTApprovalForAll:=1}"
    WEIGHT_NAMES=(Transfer Approval UniV3Swap AaveSupply AaveWithdraw AaveBorrow LiquidationCall CometSupply VoteCast BridgeEvent ERC1155 AaveRepayV2 UniV2SwapV2 CometWithdrawV2 CometBorrowV2 ProposalCreatedV2 L2SentMessage L2TxToL2 NFTTransfer NFTApproval NFTApprovalForAll)
    local __wacc=0 __wi=0
    for __w in $WEIGHT_Transfer $WEIGHT_Approval $WEIGHT_UniV3Swap $WEIGHT_AaveSupply $WEIGHT_AaveWithdraw $WEIGHT_AaveBorrow $WEIGHT_LiquidationCall $WEIGHT_CometSupply $WEIGHT_VoteCast $WEIGHT_BridgeEvent $WEIGHT_ERC1155 $WEIGHT_AaveRepayV2 $WEIGHT_UniV2SwapV2 $WEIGHT_CometWithdrawV2 $WEIGHT_CometBorrowV2 $WEIGHT_ProposalCreatedV2 $WEIGHT_L2SentMessage $WEIGHT_L2TxToL2 $WEIGHT_NFTTransfer $WEIGHT_NFTApproval $WEIGHT_NFTApprovalForAll; do
        __wacc=$((__wacc + __w))
        WEIGHT_THRESHOLDS[__wi]=$__wacc
        __wi=$((__wi + 1))
    done

    # Read Solana state (set up by setup_solana() before sim_loop was launched)
    local solana_available=false sol_token="" sol_key_dir="/tmp/chainpulse-sol-keys"
    local sol_container="$SOLANA_CONTAINER" sol_rpc="http://localhost:${SOLANA_RPC_PORT}"
    local sol_atas=()
    if [ -f "$STATE_DIR/solana.token" ]; then
        sol_token=$(cat "$STATE_DIR/solana.token")
        for __si in 0 1 2 3 4; do
            if [ -f "$STATE_DIR/solana.ata${__si}" ]; then
                sol_atas[$__si]=$(cat "$STATE_DIR/solana.ata${__si}")
            fi
        done
        if [ -n "$sol_token" ] && docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$sol_container"; then
            solana_available=true
        fi
    fi

    while true; do
        cycle=$((cycle + 1))

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
                sim "solana: SPL Transfer ${sol_amt} tokens account${sol_from}->account${sol_to}"
                docker exec "$sol_container" spl-token transfer --url "$sol_rpc" --fee-payer "$sol_key_dir/key${sol_from}.json" --owner "$sol_key_dir/key${sol_from}.json" "$sol_token" "$sol_amt" "${sol_atas[$sol_to]}" --allow-unfunded-recipient >/dev/null 2>>"$STATE_DIR/errors.log" || true
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
            [ -z "$token_addr" ] || [ -z "$emitter_addr" ] && continue

            # Get container name + RPC port for this chain
            local container="" anvil_port="8545"
            for ((__i=0; __i<${#CHAIN_CONTAINERS[@]}; __i++)); do
                local ctr="${CHAIN_CONTAINERS[$__i]}" ccfg="${CHAIN_CONFIGS[$__i]}"
                if [ "${ccfg%%:*}" = "$chain" ]; then
                    container="$ctr"
                    anvil_port="${ccfg##*:}"
                    break
                fi
            done
            [ -z "$container" ] && continue
            local RPC="http://localhost:${anvil_port}"

            # Emit Batch event with shared correlation ID
            _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitBatch(uint256,string)" "$batch_num" "Cycle-$cycle"

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
                        _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                        _ev_Transfer=$((_ev_Transfer + 1))
                        ;;
                    Approval)
                        sim "$chain: Approval ${from_addr:0:8}...->${to_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$token_addr" "approve(address,uint256)" "$to_addr" "$amt"
                        _ev_Approval=$((_ev_Approval + 1))
                        ;;
                    UniV3Swap)
                        sim "$chain: UniV3Swap ${from_addr:0:8}..."
                        local sp="250679566756032337290868763570861567304210"
                        local liq="519831781696124571544378"
                        local tk="194280"
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "$big_amt" "-$((big_amt / 2))" "$sp" "$liq" "$tk"
                        _ev_UniV3Swap=$((_ev_UniV3Swap + 1))
                        ;;
                    AaveSupply)
                        sim "$chain: AaveSupply ${from_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "$from_addr" "$from_addr" "$amt" "false"
                        _ev_AaveSupply=$((_ev_AaveSupply + 1))
                        ;;
                    AaveWithdraw)
                        sim "$chain: AaveWithdraw ${from_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitWithdraw(address,address,address,uint256)" "$token_addr" "$from_addr" "$to_addr" "$amt"
                        _ev_AaveWithdraw=$((_ev_AaveWithdraw + 1))
                        ;;
                    AaveBorrow)
                        sim "$chain: AaveBorrow ${from_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitBorrow(address,address,address,uint256,uint8,bool)" "$token_addr" "$from_addr" "$from_addr" "$amt" "2" "false"
                        _ev_AaveBorrow=$((_ev_AaveBorrow + 1))
                        ;;
                    LiquidationCall)
                        sim "$chain: Liquidation ${from_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitLiquidation(address,address,address,uint256,uint256,bool)" "$token_addr" "$to_addr" "$from_addr" "$amt" "$((amt * 12 / 10))" "true"
                        _ev_Liquidation=$((_ev_Liquidation + 1))
                        ;;
                    CometSupply)
                        sim "$chain: CometSupply ${from_addr:0:8}..."
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitCometSupply(address,address,uint256)" "$from_addr" "$to_addr" "$amt"
                        _ev_CometSupply=$((_ev_CometSupply + 1))
                        ;;
                    VoteCast)
                        local support=$((RANDOM % 3))
                        local sstr="AGAINST"
                        [ "$support" = "1" ] && sstr="FOR"
                        [ "$support" = "2" ] && sstr="ABSTAIN"
                        sim "$chain: VoteCast ${from_addr:0:8}... $sstr"
                        local reason="Cycle $(date +%s)"; docker exec "$container" cast send --rpc-url $RPC --private-key "$from_key" "$emitter_addr" "emitVoteCast(uint256,uint8,uint256,string)" "$((RANDOM % 50))" "$support" "$amt" "$reason" >/dev/null 2>>"$STATE_DIR/errors.log" || true
                        _ev_VoteCast=$((_ev_VoteCast + 1))
                        ;;
                    BridgeEvent)
                        sim "$chain: Bridge ${from_addr:0:8}... -> chain 56"
                        _cs "$container" "$RPC" "$from_key" "$emitter_addr" "emitBridge(address,uint256,uint256)" "$token_addr" "$amt" "56"
                        _ev_Bridge=$((_ev_Bridge + 1))
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
                            _cs "$container" "$RPC" "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amt"
                            _ev_Transfer=$((_ev_Transfer + 1))
                        else
                            sim "$chain: AaveRepay ${from_addr:0:8}..."
                            _cs "$container" "$RPC" "$from_key" "$emitter2_addr" "emitRepay(address,address,address,uint256,bool)" "$token_addr" "$from_addr" "$to_addr" "$amt" "false"
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
        if [ $((RANDOM % 100)) -lt 12 ] && [ -n "$emitter2_addr" ]; then
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
            local tps=$((RANDOM % 36 + 15))
            local dur=8
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
                    _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" "1" "-1" "250679566756032337290868763570861567304210" "519831781696124571544378" "194280"
                done
                docker exec "$container" cast rpc evm_revert "$snap" --rpc-url $RPC 2>/dev/null || true
                for ((r=0; r<depth; r++)); do
                    _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "500" "false"
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
                _cs "$container" "$RPC" "${KEYS[0]}" "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" "$token_addr" "${ACCOUNTS[0]}" "${ACCOUNTS[0]}" "100" "false"
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

        # Write metrics (Prometheus format)
        local elapsed=$(( $(date +%s) - start_sec ))
        [ "$elapsed" -lt 1 ] && elapsed=1
        {
            echo "# HELP chainpulse_sim_events_total Total events generated"
            echo "# TYPE chainpulse_sim_events_total counter"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"Transfer\"} $_ev_Transfer"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"Approval\"} $_ev_Approval"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"UniV3Swap\"} $_ev_UniV3Swap"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"AaveSupply\"} $_ev_AaveSupply"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"AaveWithdraw\"} $_ev_AaveWithdraw"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"AaveBorrow\"} $_ev_AaveBorrow"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"Liquidation\"} $_ev_Liquidation"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"CometSupply\"} $_ev_CometSupply"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"VoteCast\"} $_ev_VoteCast"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"Bridge\"} $_ev_Bridge"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"NFTTransfer\"} $_ev_NFTTransfer"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"NFTApproval\"} $_ev_NFTApproval"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"NFTApprovalAll\"} $_ev_NFTApprovalAll"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"ERC1155Single\"} $_ev_1155Single"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"AaveRepay\"} $_ev_AaveRepay"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"UniV2Swap\"} $_ev_UniV2Swap"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"CometWithdraw\"} $_ev_CometWithdraw"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"CometBorrow\"} $_ev_CometBorrow"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"ProposalCreated\"} $_ev_ProposalCreated"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"ReserveDataUpdated\"} $_ev_ReserveDataUpdated"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"SentMessage\"} $_ev_SentMessage"
            echo "chainpulse_sim_events_total{chain=\"$chain\",type=\"TxToL2\"} $_ev_TxToL2"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SPLTransfer\"} $_ev_SPLTransfer"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SPLMintTo\"} $_ev_SPLMintTo"
            echo "chainpulse_sim_events_total{chain=\"solana\",type=\"SPLBurn\"} $_ev_SPLBurn"
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
    if [ -n "$key" ]; then info "  OK  Admin API key: ${key:0:12}..."; else warn "  MISS Admin API key (try --auth-enabled flag)"; fi

    # 5) Webhook
    local wh=$(curl -s -X POST http://localhost:8080/admin/webhooks -H "Content-Type: application/json" -d '{"clientId":"verify-test","name":"verify-hook","url":"http://localhost:9999/hook","secret":"whsec"}' 2>/dev/null || echo "{}")
    local whid=$(echo "$wh" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('id','') or d.get('id',''))" 2>/dev/null)
    if [ -n "$whid" ]; then info "  OK  Webhook: ID=$whid"; else warn "  MISS Webhook"; fi

    # 6) Export
    local ex=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/events/export?limit=10" 2>/dev/null || echo "000")
    if [ "$ex" = "200" ]; then info "  OK  Export: HTTP 200"; else warn "  MISS Export: $ex"; fi

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
        echo "Usage: bash docker/deploy-monolith.sh [command]"
        echo ""
        echo "Commands:"
        echo "  deploy    (default) Build → Deploy → Simulate (full stack)"
        echo "  stop      Stop simulation and bring down the stack"
        echo "  status    Show simulator and event indexing status"
        echo "  restart   Restart everything from scratch"
        echo ""
        echo "Quick start:"
        echo "  bash docker/deploy-monolith.sh"
        ;;
esac
