#!/bin/bash
# ChainPulse Container Event Simulator
# Runs inside a foundry container, connects to 6 anvil chains.
# Compiles Solidity contracts via forge build, deploys via forge create,
# and continuously generates 30+ event types (V1 + V2 + L2).
set -euo pipefail

# ── RPC Endpoints ──
ANVIL_RPC="${ANVIL_RPC:-http://chainpulse-anvil:8545}"
ANVIL_RPC_BSC="${ANVIL_RPC_BSC:-http://chainpulse-anvil-bsc:8546}"
ANVIL_RPC_POLYGON="${ANVIL_RPC_POLYGON:-http://chainpulse-anvil-polygon:8547}"
ANVIL_RPC_ARBITRUM="${ANVIL_RPC_ARBITRUM:-http://chainpulse-anvil-arbitrum:8548}"
ANVIL_RPC_BASE="${ANVIL_RPC_BASE:-http://chainpulse-anvil-base:8549}"
ANVIL_RPC_AVALANCHE="${ANVIL_RPC_AVALANCHE:-http://chainpulse-anvil-avalanche:8550}"
DEPLOYER_KEY="${DEPLOYER_KEY:-0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80}"

# ── Traffic control ──
SIM_EVENTS_MIN="${SIM_EVENTS_MIN:-1}"
SIM_EVENTS_MAX="${SIM_EVENTS_MAX:-3}"
SIM_POISSON_MEAN="${SIM_POISSON_MEAN:-4}"
SIM_REORG_CHANCE="${SIM_REORG_CHANCE:-10}"
SIM_BURST_ENABLED="${SIM_BURST_ENABLED:-true}"
SIM_BURST_MIN_TPS="${SIM_BURST_MIN_TPS:-15}"
SIM_BURST_MAX_TPS="${SIM_BURST_MAX_TPS:-50}"
SIM_BURST_DURATION_SEC="${SIM_BURST_DURATION_SEC:-8}"
SIM_BURST_PROBABILITY="${SIM_BURST_PROBABILITY:-15}"
SIM_REORG_MIN_DEPTH="${SIM_REORG_MIN_DEPTH:-2}"
SIM_REORG_MAX_DEPTH="${SIM_REORG_MAX_DEPTH:-6}"
SIM_TIMESTAMP_ANOMALY_CHANCE="${SIM_TIMESTAMP_ANOMALY_CHANCE:-5}"
SIM_DUPLICATE_CHANCE="${SIM_DUPLICATE_CHANCE:-3}"

# ── Accounts (5 anvil default) ──
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

# Colors
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[SIM]${NC} $*" >&2; }
warn()  { echo -e "${YELLOW}[SIM]${NC} $*" >&2; }
sim()   { echo -e "${CYAN}[SIM]${NC} $*" >&2; }

CACHE_DIR="/tmp/chainpulse-sim"
mkdir -p "$CACHE_DIR"

# ── Solidty source contracts ──
write_sources() {
    local dir="$1"
    mkdir -p "$dir/src"

    # TestToken
    cat > "$dir/src/TestToken.sol" << 'TESTTOKEN_EOF'
// SPDX-License-Identifier: MIT
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
}
TESTTOKEN_EOF

    # TestNFT (ERC-721)
    cat > "$dir/src/TestNFT.sol" << 'TESTNFT_EOF'
// SPDX-License-Identifier: MIT
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
}
TESTNFT_EOF

    # RealEventEmitter (V1 - real DeFi protocol signatures)
    cat > "$dir/src/RealEventEmitter.sol" << 'RE_V1_EOF'
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract RealEventEmitter {
    event Swap(address indexed sender, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick);
    event Supply(address indexed reserve, address user, address indexed onBehalfOf, uint256 amount, bool indexed referral);
    event Withdraw(address indexed reserve, address indexed user, address indexed to, uint256 amount);
    event Borrow(address indexed reserve, address user, address indexed onBehalfOf, uint256 amount, uint8 interestRateMode, bool indexed referral);
    event LiquidationCall(address indexed collateralAsset, address indexed debtAsset, address indexed user, uint256 debtToCover, uint256 liquidatedCollateralAmount, bool receiveAToken);
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
}
RE_V1_EOF

    # RealEventEmitter V2 — extended protocol events
    cat > "$dir/src/RealEventEmitterV2.sol" << 'RE_V2_EOF'
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;
contract RealEventEmitterV2 {
    event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value);
    event TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values);
    event URI(string value, uint256 indexed id);
    event Repay(address indexed reserve, address indexed user, address indexed repayer, uint256 amount, bool useATokens);
    event ReserveDataUpdated(address indexed reserve, uint256 liquidityRate, uint256 stableBorrowRate, uint256 variableBorrowRate, uint256 liquidityIndex, uint256 variableBorrowIndex);
    event WithdrawCompound(address indexed from, address indexed to, uint256 amount);
    event BorrowCompound(address indexed account, uint256 amount, uint256 index);
    event RepayCompound(address indexed from, address indexed to, uint256 amount);
    event LiquidateCompound(address indexed liquidator, address indexed victim, uint256 amount, address indexed asset, bool isSupply);
    event SwapV2(address indexed sender, uint256 amount0In, uint256 amount1In, uint256 amount0Out, uint256 amount1Out, address indexed to);
    event Sync(uint112 reserve0, uint112 reserve1);
    event PairCreated(address indexed token0, address indexed token1, address pair, uint256);
    event TokenExchangeCurve(address indexed buyer, int128 sold_id, int128 bought_id, uint256 tokens_sold, uint256 tokens_bought);
    event SwapBalancer(address indexed tokenIn, address indexed tokenOut, uint256 amountIn, uint256 amountOut);
    event ProposalCreated(uint256 proposalId, address indexed proposer, address[] targets, uint256[] values, string[] signatures, bytes[] calldatas, uint256 voteStart, uint256 voteEnd, string description);
    event ProposalExecuted(uint256 proposalId);
    event ProposalCanceled(uint256 proposalId);
    event UserOperationEvent(address indexed sender, bytes32 userOpHash, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed);
    event AccountDeployed(address indexed sender, bytes32 userOpHash);
    event WithdrawalRequested(address indexed source, bytes pubkey, uint256 amount);
    event SentMessage(address indexed target, address indexed sender, uint256 value, uint256 gasLimit, uint256 nonce);
    event TxToL2(uint256 indexed callValue, address indexed destination, address indexed sender, uint256 amount, uint256 maxSubmissionCost, uint256 maxGas);
    event BatchV2(uint256 indexed batchId, string description);

    function emitTransferSingle(address op, address from, address to, uint256 id, uint256 val) public { emit TransferSingle(op, from, to, id, val); }
    function emitTransferBatch(address op, address from, address to, uint256[] calldata ids, uint256[] calldata vals) public { emit TransferBatch(op, from, to, ids, vals); }
    function emitURI(string calldata val, uint256 id) public { emit URI(val, id); }
    function emitRepay(address r, address u, address rp, uint256 a, bool ua) public { emit Repay(r, u, rp, a, ua); }
    function emitReserveDataUpdated(address r, uint256 lr, uint256 sbr, uint256 vbr, uint256 li, uint256 vbi) public { emit ReserveDataUpdated(r, lr, sbr, vbr, li, vbi); }
    function emitCometWithdraw(address f, address t, uint256 a) public { emit WithdrawCompound(f, t, a); }
    function emitCometBorrow(address a, uint256 amt, uint256 idx) public { emit BorrowCompound(a, amt, idx); }
    function emitCometRepay(address f, address t, uint256 a) public { emit RepayCompound(f, t, a); }
    function emitCometLiquidate(address liq, address vic, uint256 a, address asset, bool isS) public { emit LiquidateCompound(liq, vic, a, asset, isS); }
    function emitUniV2Swap(uint256 a0in, uint256 a1in, uint256 a0out, uint256 a1out, address to) public { emit SwapV2(msg.sender, a0in, a1in, a0out, a1out, to); }
    function emitSync(uint112 r0, uint112 r1) public { emit Sync(r0, r1); }
    function emitPairCreated(address t0, address t1, address pair) public { emit PairCreated(t0, t1, pair, 0); }
    function emitCurveSwap(int128 sid, int128 bid, uint256 sold, uint256 bought) public { emit TokenExchangeCurve(msg.sender, sid, bid, sold, bought); }
    function emitBalancerSwap(address tin, address tout, uint256 ain, uint256 aout) public { emit SwapBalancer(tin, tout, ain, aout); }
    function emitProposalCreated(uint256 pid, address proposer, address[] calldata targets, uint256[] calldata values, string[] calldata sigs, bytes[] calldata cds, uint256 vs, uint256 ve, string calldata desc) public { emit ProposalCreated(pid, proposer, targets, values, sigs, cds, vs, ve, desc); }
    function emitProposalExecuted(uint256 pid) public { emit ProposalExecuted(pid); }
    function emitProposalCanceled(uint256 pid) public { emit ProposalCanceled(pid); }
    function emitUserOpEvent(address sender, bytes32 uoh, uint256 nonce, bool success, uint256 agc, uint256 agu) public { emit UserOperationEvent(sender, uoh, nonce, success, agc, agu); }
    function emitAccountDeployed(address sender, bytes32 uoh) public { emit AccountDeployed(sender, uoh); }
    function emitWithdrawalRequested(address src, bytes calldata pubkey, uint256 amt) public { emit WithdrawalRequested(src, pubkey, amt); }
    function emitSentMessage(address target, address sender, uint256 val, uint256 gl, uint256 nonce) public { emit SentMessage(target, sender, val, gl, nonce); }
    function emitTxToL2(uint256 cv, address dest, address sender, uint256 amt, uint256 msc, uint256 mg) public { emit TxToL2(cv, dest, sender, amt, msc, mg); }
    function emitBatchV2(uint256 bid, string calldata d) public { emit BatchV2(bid, d); }
}
RE_V2_EOF
}

# ── Build and deploy contracts via forge ──
BUILD_DONE=0
TOKEN_ADDR=""
EMITTER1_ADDR=""
EMITTER2_ADDR=""
NFT_ADDR=""

build_contracts() {
    local sol_dir="$1"
    info "Compiling Solidity contracts with forge build..."
    (cd "$sol_dir" && forge build --via-ir --optimize --optimizer-runs 200 2>&1) | grep -E '^$|Error|Warning|Compiling|Solc|generated' || true
    BUILD_DONE=1
}

deploy_contract() {
    local sol_dir="$1"
    local contract_name="$2"
    local rpc="$3"
    local cache_file="$4"

    if [ -f "$cache_file" ]; then
        cat "$cache_file"
        return
    fi

    info "Deploying $contract_name on $rpc..."
    # Read bytecode directly from forge build artifact to avoid re-compilation
    local artifact
    artifact=$(find "$sol_dir/out" -name "${contract_name}.json" -maxdepth 3 2>/dev/null | head -1)
    local bytecode
    if [ -n "$artifact" ]; then
        bytecode=$(grep -o '"object":"[^"]*"' "$artifact" | head -1 | sed 's/"object":"//;s/"//' || echo "")
    fi
    if [ -z "$bytecode" ] || [ "$bytecode" = "0x" ]; then
        warn "  $contract_name — empty bytecode in artifact $artifact"
        echo ""
        return
    fi
    local out
    out=$(cast send --rpc-url "$rpc" --private-key "$DEPLOYER_KEY" \
        --create "$bytecode" 2>&1) || true
    local addr
    addr=$(echo "$out" | grep -E '^contractAddress|^deployedContract' | awk '{print $NF}' || echo "")
    if [ -z "$addr" ]; then
        addr=$(echo "$out" | grep -oiE '0x[0-9a-fA-F]{40}' | head -1 || echo "")
    fi
    if [ -n "$addr" ]; then
        echo "$addr" > "$cache_file"
        info "  $contract_name deployed at $addr"
        echo "$addr"
    else
        warn "  $contract_name deploy failed:"
        echo "$out" | head -5 >&2
        echo ""
    fi
}

# ── Wait for anvil readiness ──
wait_for_anvil() {
    local rpc="$1"
    local max=60 n=0
    while [ $n -lt $max ]; do
        n=$((n+1))
        if cast rpc --rpc-url "$rpc" eth_chainId >/dev/null 2>&1; then
            return 0
        fi
        sleep 2
    done
    warn "Anvil at $rpc not reachable after ${max}s, skipping"
    return 1
}

# ── Power-law random amount (Pareto α=2.0, scale=1e18) ──
random_amount() {
    awk -v seed=$RANDOM -v s=1000000000000000000 -v a=2.0 \
        'BEGIN{srand(seed); u=rand(); printf "%.0f\n", s / ((1-u)^(1/a))}'
}

# ── Poisson sleep ──
poisson_sleep() {
    local m=${1:-$SIM_POISSON_MEAN}
    awk -v seed=$RANDOM -v m="$m" \
        'BEGIN{srand(seed); u=rand(); t=-log(1-u)*m; if(t<0.5) t=0.5; if(t>35) t=35; printf "%.1f\n", t}'
}

# ── Generate a single random event ──
generate_event() {
    local rpc="$1" chain_name="$2"
    local token_addr="$3" emitter1_addr="$4"
    local nft_addr="$5" emitter2_addr="$6"

    local from_idx=$((RANDOM % ${#KEYS[@]}))
    local to_idx=$(( (from_idx + 1 + RANDOM % ((${#KEYS[@]} - 1)) ) % ${#KEYS[@]}))
    local from_key="${KEYS[$from_idx]}"
    local from_addr="${ACCOUNTS[$from_idx]}"
    local to_addr="${ACCOUNTS[$to_idx]}"
    local amount=$(random_amount)
    local one_ether=$((10**18))

    local pick=$((RANDOM % 100))

    # Weighted distribution (100% total):
    #  0-21: Transfer (22%)
    # 22-29: Approval (8%)
    # 30-37: UniV3Swap (8%) [V1]
    # 38-45: Aave Supply (8%) [V1]
    # 46-52: Aave Withdraw (7%) [V1]
    # 53-59: Aave Borrow (7%) [V1]
    # 60-64: LiquidationCall (5%) [V1]
    # 65-69: CometSupply (5%) [V1]
    # 70-74: VoteCast (5%) [V1]
    # 75-79: Bridge (5%) [V1]
    # 80-84: ERC-1155 TransferSingle (5%) [V2]
    # 85-86: Aave Repay (2%) [V2]
    # 87-88: UniV2Swap (2%) [V2]
    # 89-90: CometWithdraw (2%) [V2]
    # 91: CometBorrow (1%) [V2]
    # 92: ProposalCreated (1%) [V2]
    # 93: L2 SentMessage (1%) [V2]
    # 94: L2 TxToL2 (1%) [V2]
    # 95-96: NFT Transfer (2%) [nft_addr]
    # 97: NFT ApprovalForAll (1%) [nft_addr]
    # 98-99: Edge cases (2%)

    # Quick helper
    send_cast() {
        cast send --rpc-url "$rpc" --private-key "$1" "$2" "$3" ${4:+"$4"} ${5:+"$5"} ${6:+"$6"} ${7:+"$7"} ${8:+"$8"} >/dev/null 2>&1 || true
    }

    # ── V1: ERC-20 (30%) ──
    if [ $pick -lt 22 ]; then
        sim "$chain_name: Transfer ${from_addr:0:8}... -> ${to_addr:0:8}... ($(( amount / one_ether )) TTK)"
        send_cast "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount"
    elif [ $pick -lt 30 ]; then
        sim "$chain_name: Approval ${from_addr:0:8}... -> ${to_addr:0:8}..."
        send_cast "$from_key" "$token_addr" "approve(address,uint256)" "$to_addr" "$amount"

    # ── V1: RealEventEmitter DeFi (35%) ──
    elif [ $pick -lt 38 ]; then
        sim "$chain_name: UniV3Swap ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter1_addr" "emitUniSwap(int256,int256,uint160,uint128,int24)" \
            "$((RANDOM % 1000 * -1))" "$((RANDOM % 1000))" "0" "0" "0"
    elif [ $pick -lt 46 ]; then
        sim "$chain_name: AaveSupply ${from_addr:0:8}... ($(( amount / one_ether )))"
        send_cast "$from_key" "$emitter1_addr" "emitSupply(address,address,address,uint256,bool)" \
            "$token_addr" "$from_addr" "$from_addr" "$amount" "false"
    elif [ $pick -lt 53 ]; then
        sim "$chain_name: AaveWithdraw ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter1_addr" "emitWithdraw(address,address,address,uint256)" \
            "$token_addr" "$from_addr" "$to_addr" "$amount"
    elif [ $pick -lt 60 ]; then
        sim "$chain_name: AaveBorrow ${from_addr:0:8}... ($(( amount / one_ether )))"
        send_cast "$from_key" "$emitter1_addr" "emitBorrow(address,address,address,uint256,uint8,bool)" \
            "$token_addr" "$from_addr" "$from_addr" "$amount" "2" "false"
    elif [ $pick -lt 65 ]; then
        sim "$chain_name: LiquidationCall ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter1_addr" "emitLiquidation(address,address,address,uint256,uint256,bool)" \
            "$token_addr" "$token_addr" "$to_addr" "$amount" "$((amount * 8 / 10))" "true"
    elif [ $pick -lt 70 ]; then
        sim "$chain_name: CometSupply ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter1_addr" "emitCometSupply(address,address,uint256)" \
            "$from_addr" "$to_addr" "$amount"
    elif [ $pick -lt 75 ]; then
        local support=$((RANDOM % 3))
        local sstr="AGAINST"
        [ "$support" = "1" ] && sstr="FOR"
        [ "$support" = "2" ] && sstr="ABSTAIN"
        sim "$chain_name: VoteCast #$((RANDOM % 50)) $sstr"
        send_cast "$from_key" "$emitter1_addr" "emitVoteCast(uint256,uint8,uint256,string)" \
            "$((RANDOM % 50))" "$support" "$amount" "cycle-$(date +%s)"
    elif [ $pick -lt 80 ]; then
        sim "$chain_name: Bridge -> chain $((RANDOM % 5 + 1))"
        send_cast "$from_key" "$emitter1_addr" "emitBridge(address,uint256,uint256)" \
            "$token_addr" "$amount" "$((RANDOM % 5 + 1))"

    # ── V2: Extended protocol events (14%, needs emitter2_addr) ──
    elif [ $pick -lt 85 ] && [ -n "$emitter2_addr" ]; then
        local erc1155_id=$((RANDOM % 1000 + 1))
        sim "$chain_name: ERC1155 TransferSingle #$erc1155_id"
        send_cast "$from_key" "$emitter2_addr" "emitTransferSingle(address,address,address,uint256,uint256)" \
            "$from_addr" "$from_addr" "$to_addr" "$erc1155_id" "$amount"
    elif [ $pick -lt 87 ] && [ -n "$emitter2_addr" ]; then
        sim "$chain_name: AaveRepay ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter2_addr" "emitRepay(address,address,address,uint256,bool)" \
            "$token_addr" "$from_addr" "$to_addr" "$amount" "false"
    elif [ $pick -lt 89 ] && [ -n "$emitter2_addr" ]; then
        sim "$chain_name: UniV2Swap ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter2_addr" "emitUniV2Swap(uint256,uint256,uint256,uint256,address)" \
            "$amount" "$((amount / 10 + 1))" "$((amount * 9 / 10))" "$amount" "$to_addr"
    elif [ $pick -lt 91 ] && [ -n "$emitter2_addr" ]; then
        sim "$chain_name: CometWithdraw ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter2_addr" "emitCometWithdraw(address,address,uint256)" \
            "$from_addr" "$to_addr" "$amount"
    elif [ $pick -lt 92 ] && [ -n "$emitter2_addr" ]; then
        sim "$chain_name: CometBorrow ${from_addr:0:8}..."
        send_cast "$from_key" "$emitter2_addr" "emitCometBorrow(address,uint256,uint256)" \
            "$from_addr" "$amount" "$((RANDOM % 100 + 1))"
    elif [ $pick -lt 93 ] && [ -n "$emitter2_addr" ]; then
        local pid=$((RANDOM % 1000 + 1))
        sim "$chain_name: ProposalCreated #$pid"
        send_cast "$from_key" "$emitter2_addr" "emitProposalCreated(uint256,address,address[],uint256[],string[],bytes[],uint256,uint256,string)" \
            "$pid" "$from_addr" "[]" "[]" "[]" "[]" "$((RANDOM % 1000))" "$((RANDOM % 1000 + 1000))" "cycle-$(date +%s)"
    elif [ $pick -lt 94 ] && [ -n "$emitter2_addr" ]; then
        sim "$chain_name: L2 SentMessage -> L2"
        send_cast "$from_key" "$emitter2_addr" "emitSentMessage(address,address,uint256,uint256,uint256)" \
            "$from_addr" "$to_addr" "$amount" "21000" "$((RANDOM % 1000000 + 100000))"
    elif [ $pick -lt 95 ] && [ -n "$emitter2_addr" ]; then
        sim "$chain_name: L2 TxToL2 -> Arbitrum"
        send_cast "$from_key" "$emitter2_addr" "emitTxToL2(uint256,address,address,uint256,uint256,uint256)" \
            "$((RANDOM % 10000 + 1))" "$from_addr" "$to_addr" "$amount" "$((RANDOM % 100000 + 10000))" "$((RANDOM % 1000000 + 100000))"

    # ── NFT (4%) ──
    elif [ $pick -lt 98 ] && [ -n "$nft_addr" ]; then
        if [ $((RANDOM % 2)) -eq 0 ]; then
            sim "$chain_name: NFT Transfer #$((RANDOM % 100 + 1))"
            send_cast "$from_key" "$nft_addr" "transferFrom(address,address,uint256)" \
                "${ACCOUNTS[0]}" "$to_addr" "$((RANDOM % 100 + 1))"
        else
            sim "$chain_name: NFT ApprovalForAll"
            send_cast "$from_key" "$nft_addr" "setApprovalForAll(address,bool)" "$to_addr" "true"
        fi
    else
        # Edge cases (2%)
        local edge_case=$((RANDOM % 3))
        case $edge_case in
            0) sim "$chain_name: EDGE Zero-value Transfer"
               send_cast "$from_key" "$token_addr" "transfer(address,uint256)" "$to_addr" 0 ;;
            1) sim "$chain_name: EDGE Gas-exhaustion"
               cast send --rpc-url "$rpc" --private-key "$from_key" --gas-limit 5000 \
                   "$emitter1_addr" "emitBatch(uint256,string)" "$(date +%s)" "gas-exhaust" >/dev/null 2>&1 || true ;;
            2) sim "$chain_name: EDGE Max Approval"
               send_cast "$from_key" "$token_addr" "approve(address,uint256)" "$to_addr" "$((2**256 - 1))" ;;
        esac
    fi
}

# ── Burst spike ──
burst_spike() {
    local rpc="$1" chain_name="$2"
    local token_addr="$3" emitter1_addr="$4"
    local nft_addr="$5" emitter2_addr="$6"
    local tps=$(( RANDOM % (SIM_BURST_MAX_TPS - SIM_BURST_MIN_TPS + 1) + SIM_BURST_MIN_TPS ))
    local total=$(( tps * SIM_BURST_DURATION_SEC ))
    sim "$chain_name BURST: ${tps} TPS for ${SIM_BURST_DURATION_SEC}s (${total} events)"
    local start_sec sent=0
    start_sec=$(date +%s)
    while [ $(( $(date +%s) - start_sec )) -lt "$SIM_BURST_DURATION_SEC" ]; do
        generate_event "$rpc" "$chain_name" "$token_addr" "$emitter1_addr" "$nft_addr" "$emitter2_addr"
        sent=$((sent + 1))
    done
    sim "$chain_name BURST: ${sent} events sent"
}

# ── Reorg ──
simulate_reorg() {
    local rpc="$1" chain_name="$2" token_addr="$3" emitter1_addr="$4"
    local snap
    snap=$(cast rpc evm_snapshot --rpc-url "$rpc" 2>/dev/null | tail -1 || echo "")
    [ -z "$snap" ] && return
    local depth=$((RANDOM % (SIM_REORG_MAX_DEPTH - SIM_REORG_MIN_DEPTH + 1) + SIM_REORG_MIN_DEPTH))
    sim "$chain_name REORG: Mining ${depth} discardable blocks..."
    for ((i=0; i<depth; i++)); do
        cast send --rpc-url "$rpc" --private-key "${KEYS[0]}" \
            "$emitter1_addr" "emitBatch(uint256,string)" "$(date +%s)" "reorg-orphan" >/dev/null 2>&1 || true
    done
    cast rpc evm_revert "$snap" --rpc-url "$rpc" 2>/dev/null || true
    for ((i=0; i<depth; i++)); do
        cast send --rpc-url "$rpc" --private-key "${KEYS[0]}" \
            "$emitter1_addr" "emitBatch(uint256,string)" "$(date +%s)" "reorg-replace" >/dev/null 2>&1 || true
    done
    sim "$chain_name REORG: ${depth}-block reorg complete"
}

# ── Timestamp anomaly ──
simulate_timestamp_anomaly() {
    local rpc="$1" chain_name="$2"
    local current_ts
    current_ts=$(cast rpc --rpc-url "$rpc" eth_blockNumber 2>/dev/null | \
        xargs -I{} cast block --rpc-url "$rpc" {} 2>/dev/null | \
        grep "timestamp" | awk '{print $2}' | tr -d ',' || echo "")
    [ -z "$current_ts" ] || [ "$current_ts" -lt 100 ] && return
    local past_ts=$((current_ts - 3600))
    sim "$chain_name TIMESTAMP: ${past_ts}s (-1h)"
    cast rpc --rpc-url "$rpc" evm_setNextBlockTimestamp "$past_ts" 2>/dev/null || true
    cast send --rpc-url "$rpc" --private-key "${KEYS[$((RANDOM % ${#KEYS[@]}))]}" \
        "$emitter1_addr" "emitBatch(uint256,string)" "$(date +%s)" "timestamp-anomaly" >/dev/null 2>&1 || true
}

# ── Duplicate injection ──
inject_duplicate() {
    local rpc="$1" chain_name="$2" token_addr="$3"
    local from_key="${KEYS[$((RANDOM % ${#KEYS[@]}))]}"
    local to_addr="${ACCOUNTS[$((RANDOM % ${#ACCOUNTS[@]}))]}"
    local amount=$(random_amount)
    sim "$chain_name DUPLICATE: Two identical transfers"
    cast send --rpc-url "$rpc" --private-key "$from_key" \
        "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" >/dev/null 2>&1 || true
    cast send --rpc-url "$rpc" --private-key "$from_key" \
        "$token_addr" "transfer(address,uint256)" "$to_addr" "$amount" >/dev/null 2>&1 || true
}

# ── Main event generation loop ──
run_loop() {
    local rpc_list="$1" chain_names_csv="$2"
    local token_addrs_csv="$3" emitter1_addrs_csv="$4"
    local nft_addrs_csv="$5" emitter2_addrs_csv="$6"

    IFS=',' read -ra rpcs <<< "$rpc_list"
    IFS=',' read -ra names <<< "$chain_names_csv"
    IFS=',' read -ra tokens <<< "$token_addrs_csv"
    IFS=',' read -ra em1s <<< "$emitter1_addrs_csv"
    IFS=',' read -ra nfts <<< "$nft_addrs_csv"
    IFS=',' read -ra em2s <<< "$emitter2_addrs_csv"
    local n=${#rpcs[@]}

    info "Starting event generation loop ($n chains)..."
    for ((i=0; i<n; i++)); do
        info "  ${names[$i]}: token=${tokens[$i]} em1=${em1s[$i]} nft=${nfts[$i]-} em2=${em2s[$i]-}"
    done
    info "  Event types: Transfer, Approval, UniV3Swap, AaveSupply/Withdraw/Borrow,"
    info "    LiquidationCall, CometSupply, VoteCast, Bridge, ERC-1155, AaveRepay,"
    info "    UniV2Swap, CometWithdraw, CometBorrow, ProposalCreated,"
    info "    L2 SentMessage, L2 TxToL2, NFT, Edge cases"
    info "  Causal chains: Supply→Borrow→ReserveDataUpdated→Liquidation (12%)"
    info "  Burst: ${SIM_BURST_ENABLED}, Reorg: ${SIM_REORG_CHANCE}%, Edge: ts=${SIM_TIMESTAMP_ANOMALY_CHANCE}% dup=${SIM_DUPLICATE_CHANCE}%"

    local cycle=0
    while true; do
        if [ "$n" -eq 0 ]; then
            warn "No chains available — sleeping 30s..."
            sleep 30
            continue
        fi
        cycle=$((cycle + 1))
        local ci=$((RANDOM % n))
        local rpc="${rpcs[$ci]}"
        local chain_name="${names[$ci]}"
        local token_addr="${tokens[$ci]}"
        local emitter1_addr="${em1s[$ci]}"
        local nft_addr="${nfts[$ci]-}"
        local emitter2_addr="${em2s[$ci]-}"

        # Batch event for correlation
        cast send --rpc-url "$rpc" --private-key "$DEPLOYER_KEY" \
            "$emitter1_addr" "emitBatch(uint256,string)" "$(date +%s)" "Cycle-$cycle" >/dev/null 2>&1 || true
        if [ -n "$emitter2_addr" ]; then
            cast send --rpc-url "$rpc" --private-key "$DEPLOYER_KEY" \
                "$emitter2_addr" "emitBatchV2(uint256,string)" "$(date +%s)" "CycleV2-$cycle" >/dev/null 2>&1 || true
        fi

        # Normal events (1-3 per cycle)
        local batch=$((RANDOM % (SIM_EVENTS_MAX - SIM_EVENTS_MIN + 1) + SIM_EVENTS_MIN))
        for ((i=0; i<batch; i++)); do
            generate_event "$rpc" "$chain_name" "$token_addr" "$emitter1_addr" "$nft_addr" "$emitter2_addr"
        done

        # Causal chain: Supply→Borrow→ReserveDataUpdated→Liquidation (12%)
        if [ $((RANDOM % 100)) -lt 12 ] && [ -n "$emitter2_addr" ]; then
            local col_idx=$((RANDOM % ${#KEYS[@]}))
            local col_key="${KEYS[$col_idx]}"
            local col_addr="${ACCOUNTS[$col_idx]}"
            local col_amt=$((RANDOM % 5000 + 1000))
            local brw_amt=$((col_amt * (RANDOM % 26 + 50) / 100))
            sim "$chain_name CAUSAL: Supply(${col_amt}) -> Borrow(${brw_amt}) ${col_addr:0:8}..."
            cast send --rpc-url "$rpc" --private-key "$col_key" "$emitter1_addr" \
                "emitSupply(address,address,address,uint256,bool)" "$token_addr" "$col_addr" "$col_addr" "$col_amt" "false" >/dev/null 2>&1 || true
            cast send --rpc-url "$rpc" --private-key "$col_key" "$emitter1_addr" \
                "emitBorrow(address,address,address,uint256,uint8,bool)" "$token_addr" "$col_addr" "$col_addr" "$brw_amt" "2" "false" >/dev/null 2>&1 || true
            cast send --rpc-url "$rpc" --private-key "$col_key" "$emitter2_addr" \
                "emitReserveDataUpdated(address,uint256,uint256,uint256,uint256,uint256)" "$token_addr" \
                "$((RANDOM % 10 + 5))" "$((RANDOM % 5 + 2))" "$((RANDOM % 8 + 3))" "$((RANDOM % 100 + 100))" "$((RANDOM % 100 + 100))" >/dev/null 2>&1 || true
            if [ $((RANDOM % 100)) -lt 40 ]; then
                local liq_debt=$((brw_amt * 95 / 100))
                local liq_col=$((col_amt * 12 / 10))
                sim "$chain_name CAUSAL: Liquidation (debt=${liq_debt})"
                cast send --rpc-url "$rpc" --private-key "${KEYS[$(( (col_idx + 1) % ${#KEYS[@]} ))]}" "$emitter1_addr" \
                    "emitLiquidation(address,address,address,uint256,uint256,bool)" "$token_addr" "$token_addr" "$col_addr" "$liq_debt" "$liq_col" "true" >/dev/null 2>&1 || true
            fi
        fi

        # Burst spike
        if [ "$SIM_BURST_ENABLED" = "true" ] && [ $((RANDOM % 100)) -lt "$SIM_BURST_PROBABILITY" ]; then
            # Use --async during burst for throughput
            local tps=$((RANDOM % 36 + 15))
            local dur=8
            local spike_start sent=0
            spike_start=$(date +%s)
            sim "$chain_name BURST: ${tps} TPS for ${dur}s"
            while [ $(( $(date +%s) - spike_start )) -lt "$dur" ]; do
                local sp_from=$((RANDOM % ${#KEYS[@]}))
                local sp_to=$(( (sp_from + 1 + RANDOM % ((${#KEYS[@]} - 1))) % ${#KEYS[@]} ))
                local sp_key="${KEYS[$sp_from]}"
                cast send --async --rpc-url "$rpc" --private-key "$sp_key" "$token_addr" \
                    "transfer(address,uint256)" "${ACCOUNTS[$sp_to]}" "$((RANDOM % 1000 + 1))" >/dev/null 2>&1 || true
                sent=$((sent + 1))
            done
            sim "$chain_name BURST: ${sent} events"
        fi

        # Reorg
        if [ $((RANDOM % 100)) -lt "$SIM_REORG_CHANCE" ]; then
            simulate_reorg "$rpc" "$chain_name" "$token_addr" "$emitter1_addr"
        fi

        # Timestamp anomaly
        if [ $((RANDOM % 100)) -lt "$SIM_TIMESTAMP_ANOMALY_CHANCE" ]; then
            simulate_timestamp_anomaly "$rpc" "$chain_name"
        fi

        # Duplicate
        if [ $((RANDOM % 100)) -lt "$SIM_DUPLICATE_CHANCE" ]; then
            inject_duplicate "$rpc" "$chain_name" "$token_addr"
        fi

        # Cross-chain bridge
        if [ $n -gt 1 ] && [ $((RANDOM % 100)) -lt 15 ]; then
            local other_i=$(( (ci + 1) % n ))
            local other_name="${names[$other_i]}"
            local bridge_amt=$(random_amount)
            sim "$chain_name CROSS-CHAIN: -> $other_name"
            cast send --rpc-url "$rpc" --private-key "$DEPLOYER_KEY" \
                "$emitter1_addr" "emitBridge(address,uint256,uint256)" "$token_addr" "$bridge_amt" "56" >/dev/null 2>&1 || true
        fi

        # Metrics
        mkdir -p "$CACHE_DIR"
        {
            echo "# HELP chainpulse_sim_events_total Total event generation cycles"
            echo "# TYPE chainpulse_sim_events_total counter"
            echo "chainpulse_sim_events_total $cycle"
            echo "# HELP chainpulse_sim_active_chains Number of active chains"
            echo "# TYPE chainpulse_sim_active_chains gauge"
            echo "chainpulse_sim_active_chains $n"
        } > "$CACHE_DIR/metrics.prom.tmp" && mv "$CACHE_DIR/metrics.prom.tmp" "$CACHE_DIR/metrics.prom" 2>/dev/null || true

        sleep "$(poisson_sleep "$SIM_POISSON_MEAN")"
    done
}

# ════════════════════════════════════════════
# Main
# ════════════════════════════════════════════
info "ChainPulse Event Simulator"
info ""

# Wait for anvil instances
CHAINS=()
[ -n "${ANVIL_RPC:-}" ] && CHAINS+=("ethereum|$ANVIL_RPC")
[ -n "${ANVIL_RPC_BSC:-}" ] && CHAINS+=("bsc|$ANVIL_RPC_BSC")
[ -n "${ANVIL_RPC_POLYGON:-}" ] && CHAINS+=("polygon|$ANVIL_RPC_POLYGON")
[ -n "${ANVIL_RPC_ARBITRUM:-}" ] && CHAINS+=("arbitrum|$ANVIL_RPC_ARBITRUM")
[ -n "${ANVIL_RPC_BASE:-}" ] && CHAINS+=("base|$ANVIL_RPC_BASE")
[ -n "${ANVIL_RPC_AVALANCHE:-}" ] && CHAINS+=("avalanche|$ANVIL_RPC_AVALANCHE")

for entry in "${CHAINS[@]}"; do
    IFS='|' read -r name rpc <<< "$entry"
    wait_for_anvil "$rpc" || true
done

# Write Solidity sources and compile
SOL_DIR=$(mktemp -d)
write_sources "$SOL_DIR"
build_contracts "$SOL_DIR"

# Deploy contracts on each chain
CHAIN_NAMES=()
CHAIN_RPCS=()
TOKEN_ADDRS=()
EMITTER1_ADDRS=()
NFT_ADDRS=()
EMITTER2_ADDRS=()

for entry in "${CHAINS[@]}"; do
    IFS='|' read -r name rpc <<< "$entry"
    info "Setting up contracts on [$name]..."

    # TestToken
    token_addr=$(deploy_contract "$SOL_DIR" "TestToken" "$rpc" "$CACHE_DIR/token-${name}.addr")

    # RealEventEmitter (V1)
    em1_addr=$(deploy_contract "$SOL_DIR" "RealEventEmitter" "$rpc" "$CACHE_DIR/emitter1-${name}.addr")

    # TestNFT (ERC-721)
    nft_addr=$(deploy_contract "$SOL_DIR" "TestNFT" "$rpc" "$CACHE_DIR/nft-${name}.addr")

    # RealEventEmitter V2
    em2_addr=$(deploy_contract "$SOL_DIR" "RealEventEmitterV2" "$rpc" "$CACHE_DIR/emitter2-${name}.addr")

    if [ -n "$token_addr" ] && [ -n "$em1_addr" ]; then
        CHAIN_NAMES+=("$name")
        CHAIN_RPCS+=("$rpc")
        TOKEN_ADDRS+=("$token_addr")
        EMITTER1_ADDRS+=("$em1_addr")
        NFT_ADDRS+=("${nft_addr:-}")
        EMITTER2_ADDRS+=("${em2_addr:-}")
        info "  $name ready: token=$token_addr em1=$em1_addr nft=${nft_addr:-none} em2=${em2_addr:-none}"
    else
        warn "  $name: Skipping (missing token or emitter1)"
    fi
done

# Build comma-separated args for loop
rpc_csv=$(IFS=,; echo "${CHAIN_RPCS[*]}")
name_csv=$(IFS=,; echo "${CHAIN_NAMES[*]}")
token_csv=$(IFS=,; echo "${TOKEN_ADDRS[*]}")
em1_csv=$(IFS=,; echo "${EMITTER1_ADDRS[*]}")
nft_csv=$(IFS=,; echo "${NFT_ADDRS[*]}")
em2_csv=$(IFS=,; echo "${EMITTER2_ADDRS[*]}")

run_loop "$rpc_csv" "$name_csv" "$token_csv" "$em1_csv" "$nft_csv" "$em2_csv"
