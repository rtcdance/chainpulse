#!/usr/bin/env bash
# ChainPulse Event Simulator - Continuous blockchain event generation
# Deploys ERC-20 tokens + MultiEventEmitter on Anvil chains
# Generates 8 event types: Swap, Mint, Burn, VoteCast, Deposit, Withdrawal, Stake, Unstake, Transfer, Approval
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

# Minimal ERC-20 Solidity source
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

# Multi-event emitter — simulates DeFi activity with 8 distinct event types
EMITTER_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract MultiEventEmitter {
    event Swap(address indexed sender, address tokenIn, address tokenOut, uint256 amountIn, uint256 amountOut);
    event Mint(address indexed sender, uint256 amount0, uint256 amount1);
    event Burn(address indexed sender, uint256 amount0, uint256 amount1, address indexed to);
    event VoteCast(address indexed voter, uint256 proposalId, bool support, uint256 votes);
    event Deposit(address indexed sender, uint256 amount);
    event Withdrawal(address indexed sender, uint256 amount);
    event Stake(address indexed user, uint256 amount);
    event Unstake(address indexed user, uint256 amount);
    event Batch(uint256 indexed batchId, string description);
    function emitSwap(address tokenIn, address tokenOut, uint256 amountIn, uint256 amountOut) public { emit Swap(msg.sender, tokenIn, tokenOut, amountIn, amountOut); }
    function emitMint(uint256 a0, uint256 a1) public { emit Mint(msg.sender, a0, a1); }
    function emitBurn(uint256 a0, uint256 a1, address to) public { emit Burn(msg.sender, a0, a1, to); }
    function emitVoteCast(uint256 pid, bool support, uint256 votes) public { emit VoteCast(msg.sender, pid, support, votes); }
    function emitDeposit(uint256 amount) public { emit Deposit(msg.sender, amount); }
    function emitWithdrawal(uint256 amount) public { emit Withdrawal(msg.sender, amount); }
    function emitStake(uint256 amount) public { emit Stake(msg.sender, amount); }
    function emitUnstake(uint256 amount) public { emit Unstake(msg.sender, amount); }
    function emitBatch(uint256 batchId, string calldata description) public { emit Batch(batchId, description); }
}'

# ERC-721 TestNFT (100 pre-minted NFTs Transfer/Approval/ApprovalForAll)
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

# RealEventEmitter — real DeFi protocol event signatures (Uniswap V3 Swap, Aave, Compound, VoteCast, Bridge)
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

# RealEventEmitter V2 — extends coverage with 20+ additional protocol event types
REAL_EMITTER_V2_SOL='// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract RealEventEmitterV2 {
    event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value);
    event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values);
    event URI(string value, uint256 indexed id);
    event Repay(address indexed reserve, address indexed user, address indexed repayer, uint256 amount, bool useATokens);
    event ReserveDataUpdated(address indexed reserve, uint256 liquidityRate, uint256 stableBorrowRate, uint256 variableBorrowRate, uint256 liquidityIndex, uint256 variableBorrowIndex);
    event Withdraw(address indexed from, address indexed to, uint256 amount);
    event Borrow(address indexed account, uint256 amount, uint256 index);
    event Repay(address indexed from, address indexed to, uint256 amount);
    event Liquidate(address indexed liquidator, address indexed victim, uint256 amount, address indexed asset, bool isSupply);
    event Swap(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to);
    event Sync(uint112 reserve0, uint112 reserve1);
    event PairCreated(address indexed token0, address indexed token1, address pair, uint256);
    event TokenExchange(address indexed buyer, int128 sold_id, int128 bought_id, uint256 tokens_sold, uint256 tokens_bought);
    event Swap(address indexed tokenIn, address indexed tokenOut, uint256 amountIn, uint256 amountOut);
    event ProposalCreated(uint256 proposalId, address indexed proposer, address[] targets, uint256[] values, string[] signatures, bytes[] calldatas, uint256 voteStart, uint256 voteEnd, string description);
    event ProposalExecuted(uint256 proposalId);
    event ProposalCanceled(uint256 proposalId);
    event UserOperationEvent(address indexed sender, bytes32 userOpHash, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed);
    event AccountDeployed(address indexed sender, bytes32 userOpHash);
    event WithdrawalRequested(address indexed source, bytes pubkey, uint256 amount);
    event SentMessage(address indexed target, address indexed sender, uint256 value, uint256 gasLimit, uint256 nonce);
    event TxToL2(address indexed callValue, address indexed destination, address indexed sender, uint256 amount, uint256 maxSubmissionCost, uint256 maxGas);
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
    function emitSentMessage(address target, address sender, uint256 val, uint256 gl, uint256 nonce) public { emit SentMessage(target, sender, val, gl, nonce); }
    function emitTxToL2(uint256 cv, address dest, address sender, uint256 amt, uint256 msc, uint256 mg) public { emit TxToL2(cv, dest, sender, amt, msc, mg); }
    function emitBatch(uint256 bid, string calldata d) public { emit Batch(bid, d); }
}'

# Compose file paths
COMPOSE_FILE_MONO="docker/docker-compose.yml"
COMPOSE_FILE_MS="docker/docker-compose.microservices.yml"
COMPOSE_FILE_UI="docker/docker-compose.with-ui.yml"

# Chain configurations (container_name:chain_name:host_port)
CHAINS_MONO="chainpulse-anvil-ethereum:ethereum:8545 chainpulse-anvil-polygon:polygon:8546 chainpulse-anvil-bsc:bsc:8547"
CHAINS_MS="chainpulse-ms-anvil-ethereum:ethereum:18545 chainpulse-ms-anvil-polygon:polygon:18546 chainpulse-ms-anvil-bsc:bsc:18547 chainpulse-ms-anvil-arbitrum:arbitrum:18548 chainpulse-ms-anvil-optimism:optimism:18549 chainpulse-ms-anvil-base:base:18550 chainpulse-ms-anvil-avalanche:avalanche:18551"
# Single-anvil config for microservices compose (uses 1 Anvil with multi-chain-id)
CHAINS_MS_SINGLE="chainpulse-anvil:ethereum:8545"

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

detect_stack() {
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "chainpulse-puller"; then
        echo "microservices"
    elif docker ps --format "{{.Names}}" 2>/dev/null | grep -q -E "chainpulse-app|chainpulse-anvil-ethereum"; then
        echo "monolithic"
    else
        echo "none"
    fi
}

get_chains() {
    local stack=$(detect_stack)
    if [ "$stack" = "microservices" ]; then
        # Use single-anvil config if ms compose (1 anvil named chainpulse-anvil)
        echo "$CHAINS_MS_SINGLE"
    elif [ "$stack" = "monolithic" ]; then
        echo "$CHAINS_MONO"
    else
        echo ""
    fi
}

# ============================================
# Deploy a contract via forge create
# Uses: FORGE_SOL (source), CONTRACT_NAME, state_file
# ============================================
deploy_contract() {
    local container="$1"
    local chain="$2"
    local contract_name="$3"
    local sol_source="$4"
    local state_file="$5"

    if [ -f "$state_file" ]; then
        cat "$state_file"
        return
    fi

    info "Deploying $contract_name on $chain (container: $container)..."
    local deployer_key="${KEYS[0]}"

    local sol_tmp="/tmp/${contract_name}_${chain}.sol"
    docker exec "$container" sh -c "cat > '$sol_tmp' << 'SOLEOF'
$(echo "$sol_source")
SOLEOF"

    docker exec "$container" sh -c "mkdir -p /tmp/forge-out /tmp/forge-cache && chmod 777 /tmp/forge-out /tmp/forge-cache" 2>/dev/null || true

    local result
    result=$(docker exec "$container" forge create \
        "${sol_tmp}:${contract_name}" \
        --rpc-url http://localhost:8545 \
        --private-key "$deployer_key" \
        --out /tmp/forge-out \
        --cache-path /tmp/forge-cache \
        --broadcast 2>&1)

    local contract_address
    contract_address=$(echo "$result" | grep "Deployed to:" | awk '{print $3}' || echo "")

    if [ -z "$contract_address" ]; then
        contract_address=$(echo "$result" | grep -oE "0x[0-9a-fA-F]{40}" | tail -1 || echo "")
    fi

    if [ -n "$contract_address" ]; then
        echo "$contract_address" > "$state_file"
        info "  $chain: $contract_name deployed at $contract_address"
        echo "$contract_address"
    else
        error "  $chain: Failed to deploy $contract_name"
        echo "$result" | tail -5
        echo ""
    fi
}

# ============================================
# Deploy TestToken (ERC-20)
# ============================================
deploy_token() {
    local container="$1"
    local chain="$2"
    deploy_contract "$container" "$chain" "TestToken" "$ERC20_SOL" "$STATE_DIR/${chain}.token"
}

# ============================================
# Deploy MultiEventEmitter
# ============================================
deploy_emitter() {
    local container="$1"
    local chain="$2"
    deploy_contract "$container" "$chain" "MultiEventEmitter" "$EMITTER_SOL" "$STATE_DIR/${chain}.emitter"
}

# ============================================
# Deploy RealEventEmitter (real DeFi signatures)
# ============================================
deploy_realeventemitter() {
    local container="$1"
    local chain="$2"
    deploy_contract "$container" "$chain" "RealEventEmitter" "$REAL_EMITTER_SOL" "$STATE_DIR/${chain}.realeventemitter"
}

# ============================================
# Deploy TestNFT (ERC-721)
# ============================================
deploy_nft() {
    local container="$1"
    local chain="$2"
    deploy_contract "$container" "$chain" "TestNFT" "$TESTNFT_SOL" "$STATE_DIR/${chain}.nft"
}

# ============================================
# Deploy RealEventEmitter V2 (extended protocol events)
# ============================================
deploy_realeventemiterv2() {
    local container="$1"
    local chain="$2"
    deploy_contract "$container" "$chain" "RealEventEmitterV2" "$REAL_EMITTER_V2_SOL" "$STATE_DIR/${chain}.realeventemiterv2"
}

# ============================================
# Generate a power-law distributed integer amount (most small, few large)
# Pareto(α=1.5, scale=1e17) → most values 0.1-10 tokens, tail up to ~5000
# ============================================
random_amount() {
    awk -v seed=$RANDOM -v s=1000000000000000000 -v a=2.0 'BEGIN{srand(seed); u=rand(); printf "%.0f\n", s / ((1-u)^(1/a))}'
}

# ============================================
# Generate a Poisson (exponential) sleep interval in seconds
# mean=5s, range ~0.1-35s
# ============================================
poisson_sleep() {
    awk -v seed=$RANDOM -v m=5 'BEGIN{srand(seed); u=rand(); t=-log(1-u)*m; if(t<0.5) t=0.5; printf "%.1f\n", t}'
}

# ============================================
# Generate a single random event on a chain
# ============================================
generate_event() {
    local container="$1"
    local chain="$2"
    local token_addr="$3"
    local emitter_addr="$4"
    local nft_addr="${5:-}"
    local emitter2_addr="${6:-}"

    local from_idx=$((RANDOM % ${#KEYS[@]}))
    local to_idx=$(( (from_idx + 1 + RANDOM % ( (${#KEYS[@]} - 1) ) ) % ${#KEYS[@]}))
    local from_key="${KEYS[$from_idx]}"
    local from_addr="${ACCOUNTS[$from_idx]}"
    local to_addr="${ACCOUNTS[$to_idx]}"
    local amount=$(random_amount)
    local one_ether=$((10**18))
    local rpc_url="http://localhost:8545"

    # Weighted event distribution — thresholds loaded from sim-event-weights.conf
    local pick=$((RANDOM % 100))
    local ev_type=""
    for ((__wi=0; __wi<${#WEIGHT_THRESHOLDS[@]}; __wi++)); do
        if [ $pick -lt ${WEIGHT_THRESHOLDS[$__wi]} ]; then
            ev_type="${WEIGHT_NAMES[$__wi]}"
            break
        fi
    done

    case "$ev_type" in
        Transfer)
            sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        Approval)
            sim "$chain: Approval ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$token_addr" "approve(address,uint256)" "$to_addr" "$amount" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        UniV3Swap)
            local amount_in=$(random_amount)
            local amount_out=$(( amount_in * (80 + RANDOM % 40) / 100 ))
            sim "$chain: Swap ($(( amount_in / 10**17 )) -> $(( amount_out / 10**17 )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" \
                "$(( RANDOM % 1000 * -1 ))" "$(( RANDOM % 1000 ))" "0" "0" "0" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        AaveSupply)
            sim "$chain: Aave Supply ${from_addr:0:10}... ($(( amount / one_ether )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitSupply(address,address,address,uint256,bool)" \
                "$token_addr" "$from_addr" "$from_addr" "$amount" "true" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        AaveWithdraw)
            sim "$chain: Aave Withdraw ${from_addr:0:10}... ($(( amount / one_ether )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitWithdraw(address,address,address,uint256)" \
                "$token_addr" "$from_addr" "$to_addr" "$amount" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        AaveBorrow)
            sim "$chain: Aave Borrow ${from_addr:0:10}... ($(( amount / one_ether )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitBorrow(address,address,address,uint256,uint8,bool)" \
                "$token_addr" "$from_addr" "$from_addr" "$amount" "2" "false" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        LiquidationCall)
            sim "$chain: LiquidationCall ${from_addr:0:10}... ($(( amount / one_ether )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitLiquidation(address,address,address,uint256,uint256,bool)" \
                "$token_addr" "$token_addr" "$to_addr" "$amount" "$(( amount * 80 / 100 ))" "true" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        CometSupply)
            sim "$chain: CometSupply ${from_addr:0:10}... ($(( amount / one_ether )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitCometSupply(address,address,uint256)" \
                "$from_addr" "$to_addr" "$amount" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        VoteCast)
            local pid=$((RANDOM % 50 + 1))
            local support=$((RANDOM % 3))
            local support_str="AGAINST"
            [ "$support" = "1" ] && support_str="FOR"
            [ "$support" = "2" ] && support_str="ABSTAIN"
            local reason_str="sim-cycle-$(date +%s)"
            sim "$chain: VoteCast proposal #$pid $support_str"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitVoteCast(uint256,uint8,uint256,string)" "$pid" "$support" "$amount" "$reason_str" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        BridgeEvent)
            local dest_chain=$((RANDOM % 5 + 1))
            sim "$chain: Bridge -> chain $dest_chain ($(( amount / one_ether )))"
            docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                "$emitter_addr" "emitBridge(address,uint256,uint256)" \
                "$token_addr" "$amount" "$dest_chain" \
                2>&1 | grep -E "blockNumber|transactionHash" || true
            ;;
        ERC1155)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                local erc1155_id=$((RANDOM % 1000 + 1))
                sim "$chain: ERC1155 TransferSingle #$erc1155_id"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitTransferSingle(address,address,address,uint256,uint256)" \
                    "$from_addr" "$from_addr" "$to_addr" "$erc1155_id" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        AaveRepayV2)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                sim "$chain: AaveRepay ${from_addr:0:10}... ($(( amount / one_ether )))"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitRepay(address,address,address,uint256,bool)" \
                    "$token_addr" "$from_addr" "$to_addr" "$amount" "false" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        UniV2SwapV2)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                sim "$chain: UniV2Swap ${from_addr:0:10}..."
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitUniV2Swap(uint256,uint256,uint256,uint256,address)" \
                    "$amount" "$((amount / 10 + 1))" "$((amount * 9 / 10))" "$amount" "$to_addr" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        CometWithdrawV2)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                sim "$chain: CometWithdraw ${from_addr:0:10}... ($(( amount / one_ether )))"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitCometWithdraw(address,address,uint256)" \
                    "$from_addr" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        CometBorrowV2)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                sim "$chain: CometBorrow ${from_addr:0:10}... ($(( amount / one_ether )))"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitCometBorrow(address,uint256,uint256)" \
                    "$from_addr" "$amount" "$((RANDOM % 100 + 1))" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        ProposalCreatedV2)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                local gpid=$((RANDOM % 1000 + 1))
                sim "$chain: ProposalCreated #$gpid"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitProposalCreated(uint256,address,address[],uint256[],string[],bytes[],uint256,uint256,string)" \
                    "$gpid" "$from_addr" "[]" "[]" "[]" "[]" "$((RANDOM % 1000))" "$((RANDOM % 1000 + 1000))" "sim-cycle-$(date +%s)" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        L2SentMessage)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                sim "$chain: L2 SentMessage ${from_addr:0:10}... -> L2"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitSentMessage(address,address,uint256,uint256,uint256)" \
                    "$from_addr" "$to_addr" "$amount" "21000" "$((RANDOM % 1000000 + 100000))" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        L2TxToL2)
            if [ -z "$emitter2_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                sim "$chain: L2 TxToL2 ${from_addr:0:10}... -> Arbitrum"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$emitter2_addr" "emitTxToL2(uint256,address,address,uint256,uint256,uint256)" \
                    "$((RANDOM % 10000 + 1))" "$from_addr" "$to_addr" "$amount" "$((RANDOM % 100000 + 10000))" "$((RANDOM % 1000000 + 100000))" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            fi
            ;;
        NFTTransfer|NFTApproval|NFTApprovalForAll)
            if [ -z "$nft_addr" ]; then
                sim "$chain: Transfer ${from_addr:0:10}... -> ${to_addr:0:10}... ($(( amount / one_ether )) TTK)"
                docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                    "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" \
                    2>&1 | grep -E "blockNumber|transactionHash" || true
            else
                if [ "$ev_type" = "NFTTransfer" ]; then
                    sim "$chain: NFT Transfer token #$((RANDOM % 100 + 1)) -> ${to_addr:0:10}..."
                    docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                        "$nft_addr" "transferFrom(address,address,uint256)" \
                        "${ACCOUNTS[0]}" "$to_addr" "$((RANDOM % 100 + 1))" \
                        2>&1 | grep -E "blockNumber|transactionHash" || true
                else
                    sim "$chain: NFT ApprovalForAll ${from_addr:0:10}... -> ${to_addr:0:10}..."
                    docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                        "$nft_addr" "setApprovalForAll(address,bool)" "$to_addr" "true" \
                        2>&1 | grep -E "blockNumber|transactionHash" || true
                fi
            fi
            ;;
        *)
            # Edge case events (residual)
            local edge_case=$((RANDOM % 3))
            case $edge_case in
                0)
                    sim "$chain: EDGE Zero-value Transfer"
                    docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                        "$token_addr" "transfer(address,uint256)" "$to_addr" 0 \
                        2>&1 | grep -E "blockNumber|transactionHash" || true ;;
                1)
                    sim "$chain: EDGE Gas-exhaustion (gas-limit=5000)"
                    docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                        --gas-limit 5000 "$emitter_addr" "emitDeposit(uint256)" "$amount" \
                        2>&1 | grep -E "blockNumber|transactionHash|reverted|out of gas" || true ;;
                2)
                    sim "$chain: EDGE Max-value Approval"
                    docker exec "$container" cast send --rpc-url "$rpc_url" --private-key "$from_key" \
                        "$token_addr" "approve(address,uint256)" "$to_addr" "$((2**256 - 1))" \
                        2>&1 | grep -E "blockNumber|transactionHash" || true ;;
            esac
            ;;
    esac
}

# Load event type weights from config (fallback to hardcoded defaults)
__SIMDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__wconf="$__SIMDIR/sim-event-weights.conf"
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
__wacc=0 __wi=0
for __w in $WEIGHT_Transfer $WEIGHT_Approval $WEIGHT_UniV3Swap $WEIGHT_AaveSupply $WEIGHT_AaveWithdraw $WEIGHT_AaveBorrow $WEIGHT_LiquidationCall $WEIGHT_CometSupply $WEIGHT_VoteCast $WEIGHT_BridgeEvent $WEIGHT_ERC1155 $WEIGHT_AaveRepayV2 $WEIGHT_UniV2SwapV2 $WEIGHT_CometWithdrawV2 $WEIGHT_CometBorrowV2 $WEIGHT_ProposalCreatedV2 $WEIGHT_L2SentMessage $WEIGHT_L2TxToL2 $WEIGHT_NFTTransfer $WEIGHT_NFTApproval $WEIGHT_NFTApprovalForAll; do
    __wacc=$((__wacc + __w))
    WEIGHT_THRESHOLDS[__wi]=$__wacc
    __wi=$((__wi + 1))
done

# Metrics & stats files
METRICS_FILE="$STATE_DIR/metrics.prom"
STATS_FILE="$STATE_DIR/sim.stats"

# Burst mode config
SIM_BURST_ENABLED="${SIM_BURST_ENABLED:-true}"
SIM_BURST_MIN_TPS="${SIM_BURST_MIN_TPS:-15}"
SIM_BURST_MAX_TPS="${SIM_BURST_MAX_TPS:-50}"
SIM_BURST_DURATION_SEC="${SIM_BURST_DURATION_SEC:-8}"
SIM_BURST_PROBABILITY="${SIM_BURST_PROBABILITY:-15}"
SIM_REORG_MIN_DEPTH="${SIM_REORG_MIN_DEPTH:-2}"
SIM_REORG_MAX_DEPTH="${SIM_REORG_MAX_DEPTH:-12}"
SIM_TIMESTAMP_ANOMALY_CHANCE="${SIM_TIMESTAMP_ANOMALY_CHANCE:-5}"
SIM_DUPLICATE_CHANCE="${SIM_DUPLICATE_CHANCE:-3}"
SIM_REORG_CHANCE="${SIM_REORG_CHANCE:-10}"

# ============================================
# Burst spike — simulate sudden high TPS (whale move, popular mint)
# ============================================
burst_spike() {
    local container="$1" chain="$2" token_addr="$3" emitter_addr="$4"
    local emitter2_addr="${5:-}"
    local tps=$(( RANDOM % (SIM_BURST_MAX_TPS - SIM_BURST_MIN_TPS + 1) + SIM_BURST_MIN_TPS ))
    local total_events=$(( tps * SIM_BURST_DURATION_SEC ))
    local delay_ms=$(awk "BEGIN{printf \"%.4f\", 1000 / $tps}")
    sim "${chain} BURST: ${tps} TPS for ${SIM_BURST_DURATION_SEC}s (${total_events} events)"
    local spike_start=$(date +%s)
    local sent=0
    while [ $(( $(date +%s) - spike_start )) -lt "$SIM_BURST_DURATION_SEC" ]; do
        generate_event "$container" "$chain" "$token_addr" "$emitter_addr" "" "$emitter2_addr"
        sent=$((sent + 1))
        awk -v d="$delay_ms" 'BEGIN{for(i=0;i<d*1000;i++){}}}' 2>/dev/null || true
    done
    sim "${chain} BURST: ${sent} events sent in $(( $(date +%s) - spike_start ))s"
    echo "$(( $(date +%s) ))" > "$STATE_DIR/last_burst"
}

# ============================================
# Timestamp anomaly — mine a block with timestamp in the past
# ============================================
simulate_timestamp_anomaly() {
    local container="$1" chain="$2" token_addr="$3" emitter_addr="$4"
    local current_ts
    current_ts=$(docker exec "$container" cast block-number --rpc-url http://localhost:8545 2>/dev/null | \
        xargs -I{} docker exec "$container" cast block --rpc-url http://localhost:8545 {} 2>/dev/null | \
        grep "timestamp" | awk '{print $2}' | tr -d ',' || echo "")
    [ -z "$current_ts" ] || [ "$current_ts" -lt 100 ] && return
    local past_ts=$(( current_ts - 3600 ))
    sim "${chain} TIMESTAMP ANOMALY: Setting block timestamp to ${past_ts}s (-1h)..."
    docker exec "$container" cast rpc --rpc-url http://localhost:8545 evm_setNextBlockTimestamp "$past_ts" 2>/dev/null || true
    local from_key="${KEYS[$((RANDOM % ${#KEYS[@]}))]}"
    local amount=$(random_amount)
    docker exec "$container" cast send --rpc-url http://localhost:8545 --private-key "$from_key" \
        "$emitter_addr" "emitDeposit(uint256)" "$amount" >/dev/null 2>&1 || true
}

# ============================================
# Duplicate injection — send same TX twice to test dedup
# ============================================
inject_duplicate() {
    local container="$1" chain="$2" token_addr="$3"
    local from_key="${KEYS[$((RANDOM % ${#KEYS[@]}))]}"
    local to_addr="${ACCOUNTS[$((RANDOM % ${#ACCOUNTS[@]}))]}"
    local amount=$(random_amount)
    local one_ether=$((10**18))
    sim "${chain} DUPLICATE: Sending identical Transfer twice (${amount} wei)..."
    docker exec "$container" cast send --rpc-url http://localhost:8545 --private-key "$from_key" \
        "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" >/dev/null 2>&1 || true
    docker exec "$container" cast send --rpc-url http://localhost:8545 --private-key "$from_key" \
        "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" >/dev/null 2>&1 || true
}

# ============================================
# Start continuous simulation
# ============================================
# ============================================
# Continuous event generation loop (runs detached)
# ============================================
sim_loop() {
    local pid_file="$STATE_DIR/sim.pid"
    echo "$$" > "$pid_file"

    local chains=$(get_chains)

    # Track block progress per chain for stall detection (bash 3.x compat)
    local -a chain_names=()
    local -a last_blocks=()
    local -a missed_cycles=()
    for ce in $chains; do
        local cn
        cn=$(echo "$ce" | cut -d: -f2)
        chain_names+=("$cn")
        last_blocks+=("")
        missed_cycles+=(0)
    done

    # Register cleanup on exit
    trap 'rm -f "$pid_file"' EXIT

    while true; do
        # Generate correlation ID for this batch cycle
        local batch_id=$(date +%s)_${RANDOM}
        local batch_num=$(date +%s)

        # Emit Batch event on ethereum for cross-cycle correlation tracing
        for first_ce in $chains; do
            local first_ctr="${first_ce%%:*}"
            local first_chain
            first_chain=$(echo "$first_ce" | cut -d: -f2)
            local fe_file="$STATE_DIR/${first_chain}.emitter"
            if [ -f "$fe_file" ]; then
                local fe_addr=$(cat "$fe_file")
                if [ -n "$fe_addr" ]; then
                    docker exec "$first_ctr" cast send \
                        --rpc-url http://localhost:8545 \
                        --private-key "${KEYS[0]}" \
                        "$fe_addr" \
                        "emitBatch(uint256,string)" "$batch_num" "Cycle $batch_id" 2>&1 | grep -E "blockNumber|transactionHash" || true
                fi
            fi
            break
        done

        # Auto-stop if no Anvil containers are running (Docker Compose was torn down)
        local any_alive=false
        for chain_entry in $chains; do
            local ctr="${chain_entry%%:*}"
            if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "$ctr"; then
                any_alive=true
                break
            fi
        done
        if [ "$any_alive" = false ]; then
            warn "No Anvil containers detected. Docker Compose likely stopped. Exiting..."
            rm -f "$pid_file"
            exit 0
        fi

        local idx=0 cycle_events=0
        for chain_entry in $chains; do
            local container="${chain_entry%%:*}"
            local chain
            chain=$(echo "$chain_entry" | cut -d: -f2)
            local token_file="$STATE_DIR/${chain}.token"
            local emitter_file="$STATE_DIR/${chain}.emitter"
            local nft_file="$STATE_DIR/${chain}.nft"
            local re_file="$STATE_DIR/${chain}.realeventemitter"

            local token_addr="" emitter_addr="" nft_addr="" realemitter_addr="" rev2_addr=""
            [ -f "$token_file" ] && token_addr=$(cat "$token_file")
            [ -f "$emitter_file" ] && emitter_addr=$(cat "$emitter_file")
            [ -f "$nft_file" ] && nft_addr=$(cat "$nft_file")
            [ -f "$re_file" ] && realemitter_addr=$(cat "$re_file")
            local rev2_file="$STATE_DIR/${chain}.realeventemiterv2"
            [ -f "$rev2_file" ] && rev2_addr=$(cat "$rev2_file")

            # Primary: use RealEventEmitter if available, fall back to MultiEventEmitter
            local primary_emitter="$realemitter_addr"
            [ -z "$primary_emitter" ] && primary_emitter="$emitter_addr"

            if [ -n "$token_addr" ] && [ -n "$primary_emitter" ]; then
                local batch=$((RANDOM % 3 + 1))
                for ((i=0; i<batch; i++)); do
                    generate_event "$container" "$chain" "$token_addr" "$primary_emitter" "$nft_addr" "$rev2_addr"
                    cycle_events=$((cycle_events + 1))
                done
            fi

            # Burst spike check (15% chance)
            if [ "$SIM_BURST_ENABLED" = "true" ] && [ $((RANDOM % 100)) -lt "$SIM_BURST_PROBABILITY" ] && [ -n "$token_addr" ] && [ -n "$primary_emitter" ]; then
                burst_spike "$container" "$chain" "$token_addr" "$primary_emitter" "$rev2_addr"
            fi

            # Timestamp anomaly check (5% chance)
            if [ $((RANDOM % 100)) -lt "$SIM_TIMESTAMP_ANOMALY_CHANCE" ] && [ -n "$token_addr" ] && [ -n "$primary_emitter" ]; then
                simulate_timestamp_anomaly "$container" "$chain" "$token_addr" "$primary_emitter"
            fi

            # Duplicate injection check (3% chance)
            if [ $((RANDOM % 100)) -lt "$SIM_DUPLICATE_CHANCE" ] && [ -n "$token_addr" ]; then
                inject_duplicate "$container" "$chain" "$token_addr"
            fi

            # Reorg check — deeper depth (2-12 blocks)
            if [ $((RANDOM % 100)) -lt "$SIM_REORG_CHANCE" ] && [ -n "$token_addr" ] && [ -n "$primary_emitter" ]; then
                simulate_reorg
            fi

            # Track block progress
            local current_block
            current_block=$(docker exec "$container" cast block-number --rpc-url http://localhost:8545 2>/dev/null || echo "")
            if [ -n "$current_block" ]; then
                if [ "${last_blocks[$idx]:-}" != "" ] && [ "${last_blocks[$idx]:-}" = "$current_block" ]; then
                    missed_cycles[$idx]=$(( missed_cycles[$idx] + 1 ))
                    if [ ${missed_cycles[$idx]} -ge 3 ]; then
                        warn "${chain}: No block progress for ${missed_cycles[$idx]} cycles. RPC may be stuck."
                    fi
                else
                    missed_cycles[$idx]=0
                fi
                last_blocks[$idx]=$current_block
            fi
            idx=$((idx + 1))
        done

        # Write Prometheus metrics
        mkdir -p "$STATE_DIR"
        {
            echo "# HELP chainpulse_sim_total_events Total events generated"
            echo "# TYPE chainpulse_sim_total_events counter"
            echo "chainpulse_sim_total_events $cycle_events"
            echo "# HELP chainpulse_sim_cycles Simulation cycles"
            echo "# TYPE chainpulse_sim_cycles counter"
            echo "chainpulse_sim_cycles 1"
        } > "$METRICS_FILE.tmp" && mv "$METRICS_FILE.tmp" "$METRICS_FILE"

        # Write stats
        {
            echo "total=$(cat "$STATE_DIR/total_events" 2>/dev/null || echo 0)"
            echo "cycle=$(date +%s)"
        } > "$STATS_FILE" 2>/dev/null || true

        # Poisson sleep (mean ~2s for higher throughput)
        local sleep_time
        sleep_time=$(poisson_sleep)
        sleep "$sleep_time"
    done
}

# ============================================
# Simulate a chain reorg via evm_snapshot + evm_revert
# Mines events, reverts, mines different events → different block hashes
# ChainPulse's reorg detection should pick this up
# ============================================
simulate_reorg() {
    local chains=$(get_chains)
    local chain_list=()
    for ce in $chains; do
        local c
        c=$(echo "$ce" | cut -d: -f2)
        chain_list+=("$c")
    done

    local pick=$((RANDOM % ${#chain_list[@]}))
    local target_chain="${chain_list[$pick]}"
    local container=""
    for ce in $chains; do
        local ch=$(echo "$ce" | cut -d: -f2)
        if [ "$ch" = "$target_chain" ]; then
            container="${ce%%:*}"
            break
        fi
    done
    [ -z "$container" ] && return

    local token_file="$STATE_DIR/${target_chain}.token"
    local emitter_file="$STATE_DIR/${target_chain}.emitter"
    [ ! -f "$token_file" ] || [ ! -f "$emitter_file" ] && return

    local token_addr=$(cat "$token_file")
    local emitter_addr=$(cat "$emitter_file")

    # Snapshot current state
    local snap
    snap=$(docker exec "$container" cast rpc evm_snapshot --rpc-url http://localhost:8545 2>/dev/null | tail -1 || echo "")
    [ -z "$snap" ] && return

    local from_key="${KEYS[0]}"
    local to_key="${KEYS[1]}"
    local from_addr="${ACCOUNTS[0]}"
    local to_addr="${ACCOUNTS[1]}"
    local one_ether=$((10**18))

    local depth=$(( RANDOM % (SIM_REORG_MAX_DEPTH - SIM_REORG_MIN_DEPTH + 1) + SIM_REORG_MIN_DEPTH ))
    sim "${target_chain} REORG: Mining ${depth} discardable blocks (snapshot=$snap)..."
    local pre_block
    pre_block=$(docker exec "$container" cast block-number --rpc-url http://localhost:8545 2>/dev/null || echo "?")
    for ((i=0; i<depth; i++)); do
        local amt=$(( (RANDOM % 500 + 1) * 10**17 ))
        docker exec "$container" cast send \
            --rpc-url http://localhost:8545 \
            --private-key "$from_key" \
            "$emitter_addr" \
            "emitSwap(address,address,uint256,uint256)" \
            "$token_addr" "$to_addr" "$amt" "$amt" 2>&1 | grep -E "blockNumber" || true
    done

    local post_block
    post_block=$(docker exec "$container" cast block-number --rpc-url http://localhost:8545 2>/dev/null || echo "?")
    sim "${target_chain} REORG: Discardable events mined (blocks ${pre_block}→${post_block}). Reverting..."

    # Revert to snapshot — this restores chain to pre-event state
    docker exec "$container" cast rpc evm_revert "$snap" --rpc-url http://localhost:8545 2>/dev/null || true

    # Now mine DIFFERENT events at the same block heights → different block hashes
    sim "${target_chain} REORG: Mining ${depth} replacement events..."
    for ((i=0; i<depth; i++)); do
        local amt=$(( (RANDOM % 500 + 1) * 10**17 ))
        # Use different event type/params for different data
        docker exec "$container" cast send \
            --rpc-url http://localhost:8545 \
            --private-key "$from_key" \
            "$emitter_addr" \
            "emitDeposit(uint256)" "$amt" 2>&1 | grep -E "blockNumber" || true
    done

    sim "${target_chain} REORG: Replacement events mined. ChainPulse should detect reorg on next poll."
}

# ============================================
# Docker Compose health wait
# ============================================
await_healthy() {
    local container="$1"
    local label="$2"
    local max_attempts=60
    local n=0
    info "  Waiting for $label ($container) to become healthy..."
    while [ $n -lt $max_attempts ]; do
        n=$((n + 1))
        local status
        status=$(docker inspect "$container" --format '{{.State.Health.Status}}' 2>/dev/null || echo "")
        if [ "$status" = "healthy" ]; then
            info "  $label: healthy (${n}s)"
            return 0
        fi
        sleep 2
    done
    warn "  $label: not healthy after ${max_attempts}s — continuing anyway"
    return 1
}

# ============================================
# Auto-start Docker Compose stack
# ============================================
start_stack() {
    local mode="${1:-mono}"
    info "No running ChainPulse stack detected. Starting Docker Compose ($mode)..."
    info ""

    # Check .env exists
    if [ ! -f "docker/.env" ]; then
        if [ -f "docker/.env.example" ]; then
            cp docker/.env.example docker/.env
            info "  Created docker/.env from .env.example"
        fi
    fi

    if [ "$mode" = "ms" ]; then
        # --- Microservices mode ---
        local compose_flags="-f $COMPOSE_FILE_MS"
        info "  Compose: $COMPOSE_FILE_MS"
        docker compose $compose_flags up -d 2>&1 || {
            error "Docker Compose failed. Check your Docker daemon and try again."
            exit 1
        }

        info ""
        info "Waiting for microservices infrastructure..."

        await_healthy "chainpulse-postgres" "PostgreSQL"
        await_healthy "chainpulse-redis" "Redis"
        await_healthy "chainpulse-kafka" "Kafka"
        await_healthy "chainpulse-anvil" "Anvil"
        await_healthy "chainpulse-api-gateway" "API Gateway"
        await_healthy "chainpulse-api-service" "API Service"
        await_healthy "chainpulse-event-processor" "Event Processor"
        await_healthy "chainpulse-puller" "Puller"
        await_healthy "chainpulse-ms-frontend" "Dashboard UI"

        # Verify backend
        info ""
        info "Verifying backend API..."
        local n=0
        while [ $n -lt 15 ]; do
            n=$((n + 1))
            local health
            health=$(curl -sf http://localhost:8080/health 2>/dev/null || echo "")
            if [ -n "$health" ]; then
                info "  API Gateway: ready (${n}s)"
                break
            fi
            sleep 2
        done

        info ""
        info "=== Microservices stack is ready ==="
        info "  API Gateway:      http://localhost:8080"
        info "  API Service:      http://localhost:8081"
        info "  Event Processor:  http://localhost:8082"
        info "  Puller:           http://localhost:8083"
        info "  Dashboard UI:     http://localhost:13000"
        info "  Grafana:          http://localhost:3000"
        info "  Prometheus:       http://localhost:9090"
        info "  Jaeger:           http://localhost:16686"
        info ""

    else
        # --- Monolithic mode (default) ---
        local compose_flags="-f $COMPOSE_FILE_MONO"

        if [ -f "$COMPOSE_FILE_UI" ]; then
            compose_flags="$compose_flags -f $COMPOSE_FILE_UI"
            info "  Dashboard UI: enabled ($COMPOSE_FILE_UI)"
        fi

        info "  Compose: $compose_flags"
        docker compose $compose_flags up -d 2>&1 || {
            error "Docker Compose failed. Check your Docker daemon and try again."
            exit 1
        }

        info ""
        info "Waiting for monolithic infrastructure..."

        await_healthy "chainpulse-postgres" "PostgreSQL"
        await_healthy "chainpulse-redis" "Redis"
        await_healthy "chainpulse-kafka" "Kafka"

        # Wait for all 7 Anvil chains
        for ctr in \
            chainpulse-anvil-ethereum chainpulse-anvil-polygon chainpulse-anvil-bsc \
            chainpulse-anvil-arbitrum chainpulse-anvil-optimism \
            chainpulse-anvil-base chainpulse-anvil-avalanche; do
            await_healthy "$ctr" "Anvil ($ctr)"
        done

        await_healthy "chainpulse-app" "ChainPulse backend"
        await_healthy "chainpulse-frontend" "Dashboard UI"

        # Verify backend
        info ""
        info "Verifying backend API..."
        local n=0
        while [ $n -lt 15 ]; do
            n=$((n + 1))
            local health
            health=$(curl -sf http://localhost:8080/health 2>/dev/null || echo "")
            if [ -n "$health" ]; then
                info "  Backend API: ready (${n}s)"
                break
            fi
            sleep 2
        done

        info ""
        info "=== Monolithic stack is ready ==="
        info "  Backend API:  http://localhost:8080"
        info "  Dashboard UI: http://localhost:3000"
        info "  Health:       http://localhost:8080/health"
        info ""
    fi
}

# ============================================
# Start simulation (auto-starts stack if needed)
# ============================================
sim_start() {
    local mode="${1:-mono}"
    local stack=$(detect_stack)
    if [ "$stack" = "none" ]; then
        # Convert mode shorthand: ms → microservices, mono → monolithic
        if [ "$mode" = "ms" ]; then
            start_stack "ms"
        else
            start_stack "mono"
        fi
        # Re-detect using container names
        if [ "$mode" = "ms" ]; then
            stack="microservices"
        else
            stack="monolithic"
        fi
        info "Stack: $stack"
    else
        info "Detected stack: $stack"
        # Override mode to match detected stack
        if [ "$stack" = "microservices" ]; then mode="ms"; else mode="mono"; fi
    fi

    # Kill existing simulator if running
    local pid_file="$STATE_DIR/sim.pid"
    if [ -f "$pid_file" ]; then
        local old_pid
        old_pid=$(cat "$pid_file")
        if kill -0 "$old_pid" 2>/dev/null; then
            warn "Simulator already running (PID $old_pid). Stopping..."
            kill "$old_pid" 2>/dev/null || true
            sleep 1
        fi
    fi

    local chains=$(get_chains)
    local needs_deploy=false

    # Validate existing contracts — redeploy only if Anvil state was wiped
    for chain_entry in $chains; do
        local container="${chain_entry%%:*}"
        local chain
        chain=$(echo "$chain_entry" | cut -d: -f2)

        # Token + emitters + NFT
        local token_file="$STATE_DIR/${chain}.token"
        local emitter_file="$STATE_DIR/${chain}.emitter"
        local nft_file="$STATE_DIR/${chain}.nft"
        local re_file="$STATE_DIR/${chain}.realeventemitter"

        if [ -f "$token_file" ] && [ -f "$emitter_file" ]; then
            local token_addr=$(cat "$token_file")
            local emitter_addr=$(cat "$emitter_file")
            local token_code
            token_code=$(docker exec "$container" cast code "$token_addr" --rpc-url http://localhost:8545 2>/dev/null || echo "")
            local emitter_code
            emitter_code=$(docker exec "$container" cast code "$emitter_addr" --rpc-url http://localhost:8545 2>/dev/null || echo "")
            if [ "${#token_code}" -gt 4 ] && [ "${#emitter_code}" -gt 4 ]; then
                info "$chain: Core contracts valid"
            else
                warn "$chain: Contracts missing (Anvil restarted?). Redeploying..."
                rm -f "$token_file" "$emitter_file" "$nft_file" "$re_file"
                needs_deploy=true
            fi
        else
            needs_deploy=true
        fi
    done

    if [ "$needs_deploy" = true ]; then
        info "Deploying contracts (TestToken, MultiEventEmitter, RealEventEmitter, RealEventEmitterV2, TestNFT)..."

        for chain_entry in $chains; do
            local container="${chain_entry%%:*}"
            local chain
            chain=$(echo "$chain_entry" | cut -d: -f2)

            # ERC-20 TestToken
            local token_file="$STATE_DIR/${chain}.token"
            if [ ! -f "$token_file" ]; then
                deploy_token "$container" "$chain"
            else
                info "  $chain: token already deployed, skipping"
            fi

            # MultiEventEmitter (legacy)
            local emitter_file="$STATE_DIR/${chain}.emitter"
            if [ ! -f "$emitter_file" ]; then
                deploy_emitter "$container" "$chain"
            else
                info "  $chain: emitter already deployed, skipping"
            fi

            # RealEventEmitter (new - real DeFi signatures)
            local re_file="$STATE_DIR/${chain}.realeventemitter"
            if [ ! -f "$re_file" ]; then
                deploy_realeventemitter "$container" "$chain"
            else
                info "  $chain: RealEventEmitter already deployed, skipping"
            fi

            # TestNFT (ERC-721)
            local nft_file="$STATE_DIR/${chain}.nft"
            if [ ! -f "$nft_file" ]; then
                deploy_nft "$container" "$chain"
            else
                info "  $chain: TestNFT already deployed, skipping"
            fi

            # RealEventEmitter V2 (extended protocol events)
            local rev2_file="$STATE_DIR/${chain}.realeventemiterv2"
            if [ ! -f "$rev2_file" ]; then
                deploy_realeventemiterv2 "$container" "$chain"
            else
                info "  $chain: RealEventEmitterV2 already deployed, skipping"
            fi
        done
    fi

    # Launch loop in fully detached process via nohup (log to file for diagnostics)
    local log_file="$STATE_DIR/sim.log"
    nohup bash "$0" loop > "$log_file" 2>&1 &

    local pid_file="$STATE_DIR/sim.pid"

    # Wait briefly for PID file to appear
    for _ in 1 2 3 4 5; do
        if [ -f "$pid_file" ]; then
            local sim_pid
            sim_pid=$(cat "$pid_file")
            if kill -0 "$sim_pid" 2>/dev/null; then
                local mode_label="Monolithic"
                local api_url="http://localhost:8080"
                local ui_url="http://localhost:3000"
                if [ "$mode" = "ms" ]; then
                    mode_label="Microservices"
                fi
                echo ""
                echo "╔══════════════════════════════════════════════════════════════╗"
                echo "║         ChainPulse  —  $mode_label One-Click Start          ║"
                echo "╠══════════════════════════════════════════════════════════════╣"
                echo "║  Simulator running   (PID $sim_pid)                        ║"
                echo "║  Backend API:        $api_url                 ║"
                echo "║  Live Events:        $api_url/events          ║"
                if [ "$mode" = "ms" ]; then
                    echo "║  Dashboard UI:    http://localhost:13000                 ║"
                    echo "║  Grafana:          http://localhost:3000                 ║"
                    echo "║  API Service:      http://localhost:8081                 ║"
                    echo "║  Event Processor:  http://localhost:8082                 ║"
                else
                    echo "║  Dashboard UI:       $ui_url                 ║"
                fi
                echo "║                                                              ║"
                echo "║  Events: Transfer, Approval, Swap, Mint, Burn, VoteCast,   ║"
                echo "║          Deposit, Withdrawal, Stake, Unstake, Batch         ║"
                echo "║  Reorg:  ✓   Edge Cases: ✓   Power-Law: ✓                  ║"
                echo "╚══════════════════════════════════════════════════════════════╝"
                return 0
            fi
        fi
        sleep 0.5
    done

    error "Failed to start simulator loop"
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
# Show status
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

    # Show deployed contracts with on-chain validation
    local chains=$(get_chains)
    for chain_entry in $chains; do
        local container="${chain_entry%%:*}"
        local chain
        chain=$(echo "$chain_entry" | cut -d: -f2)
        local token_file="$STATE_DIR/${chain}.token"
        local emitter_file="$STATE_DIR/${chain}.emitter"

        if [ -f "$token_file" ]; then
            local addr=$(cat "$token_file")
            local code
            code=$(docker exec "$container" cast code "$addr" --rpc-url http://localhost:8545 2>/dev/null || echo "")
            if [ "${#code}" -gt 4 ]; then
                info "  $chain.token: $addr (valid)"
            else
                warn "  $chain.token: $addr (MISSING - Anvil state lost)"
            fi
        else
            warn "  $chain.token: not deployed"
        fi

        if [ -f "$emitter_file" ]; then
            local addr=$(cat "$emitter_file")
            local code
            code=$(docker exec "$container" cast code "$addr" --rpc-url http://localhost:8545 2>/dev/null || echo "")
            if [ "${#code}" -gt 4 ]; then
                info "  $chain.emitter: $addr (valid)"
            else
                warn "  $chain.emitter: $addr (MISSING - Anvil state lost)"
            fi
        else
            warn "  $chain.emitter: not deployed"
        fi
    done

    # Show events indexed
    if [ "$stack" = "microservices" ]; then
        local event_count
        event_count=$(curl -sf "http://localhost:8080/events?limit=500" 2>/dev/null \
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
        sim_start "${2:-mono}"
        ;;
    stop)
        sim_stop
        ;;
    status)
        sim_status
        ;;
    loop)
        sim_loop
        ;;
    help|*)
        echo "ChainPulse Event Simulator - Real-time blockchain event generation"
        echo ""
        echo "Usage: bash docker/simulate-events.sh <command>"
        echo ""
        echo "Commands:"
        echo "  start [mono|ms]  One-click: start stack + deploy + simulate events"
        echo "                  mono = monolithic (default), ms = microservices"
        echo "  stop    Stop the background event generator"
        echo "  status  Show simulation status and event counts"
        echo ""
        echo "Event types:"
        echo "  ERC-20:   Transfer, Approval"
        echo "  DEX:      Swap, Mint, Burn"
        echo "  Gov:      VoteCast"
        echo "  Lending:  Deposit, Withdrawal"
        echo "  Staking:  Stake, Unstake"
        echo "  Tracing:  Batch (correlation ID per cycle)"
        echo ""
        echo "Capabilities:"
        echo "  ✓ Auto-start Docker Compose (if not running)"
        echo "  ✓ DB migrations"
        echo "  ✓ Auto-redeploy contracts on Anvil restart"
        echo "  ✓ Reorg simulation (10% probability)"
        echo "  ✓ Edge case events (zero-value, gas-exhaust, max-approval)"
        echo "  ✓ Power-law traffic distribution (Pareto amounts + Poisson timing)"
        echo "  ✓ Correlation ID tracing (Batch events)"
        echo "  ✓ Block stall detection"
        echo "  ✓ Docker lifecycle auto-stop"
        echo "  ✓ Dashboard UI (http://localhost:3000 / http://localhost:13000 for ms)"
        echo ""
        echo "Quick start:"
        echo "  bash docker/simulate-events.sh start"
        echo "  bash docker/simulate-events.sh status"
        ;;
esac
