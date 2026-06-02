#!/usr/bin/env bash
# ChainPulse Solana Event Simulator
# Generates SPL Token events: Transfer, MintTo, Burn, InitializeMint, CloseAccount
set -euo pipefail

SOLANA_RPC_URL="${SOLANA_RPC_URL:-http://localhost:8899}"
SIM_EVENTS_MIN="${SIM_EVENTS_MIN:-1}"
SIM_EVENTS_MAX="${SIM_EVENTS_MAX:-3}"
SIM_POISSON_MEAN="${SIM_POISSON_MEAN:-6}"

export SOLANA_METRICS_CONFIG="host=http://localhost:8080"
solana config set --url "$SOLANA_RPC_URL" >/dev/null 2>&1

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[SOL-SIM]${NC} $*"; }
warn()  { echo -e "${YELLOW}[SOL-SIM]${NC} $*"; }
error() { echo -e "${RED}[SOL-SIM]${NC} $*"; }
sim()   { echo -e "${CYAN}[SOL-SIM]${NC} $*"; }

# ── Wait for validator to be ready ──
info "Waiting for Solana validator at $SOLANA_RPC_URL..."
for i in $(seq 1 60); do
    if solana cluster-version --url "$SOLANA_RPC_URL" >/dev/null 2>&1; then
        info "Validator is ready"
        break
    fi
    sleep 2
done

# ── Generate keypairs ──
KEY_DIR="/tmp/chainpulse-sol-keys"
mkdir -p "$KEY_DIR"

info "Generating keypairs..."
for i in $(seq 0 4); do
    if [ ! -f "$KEY_DIR/key${i}.json" ]; then
        solana-keygen new --no-bip39-passphrase --outfile "$KEY_DIR/key${i}.json" --force --silent 2>/dev/null
    fi
done

# ── Airdrop SOL to each keypair ──
info "Airdropping SOL..."
for i in $(seq 0 4); do
    solana airdrop 100 --url "$SOLANA_RPC_URL" --keypair "$KEY_DIR/key${i}.json" >/dev/null 2>&1 || true
    sleep 0.5
done

# ── Create SPL Token ──
info "Creating SPL Token..."
TOKEN_MINT=""
for i in $(seq 1 10); do
    TOKEN_MINT=$(spl-token create-token --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key0.json" --mint-authority "$KEY_DIR/key0.json" 2>/dev/null | grep "Creating token" | awk '{print $3}' || true)
    if [ -n "$TOKEN_MINT" ]; then
        info "SPL Token created: $TOKEN_MINT"
        break
    fi
    sleep 2
done

if [ -z "$TOKEN_MINT" ]; then
    error "Failed to create SPL Token"
    exit 1
fi

# ── Create token accounts for each keypair ──
info "Creating token accounts..."
declare -a TOKEN_ACCOUNTS
for i in $(seq 0 4); do
    ACCOUNT=$(spl-token create-account --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key${i}.json" --owner "$KEY_DIR/key${i}.json" "$TOKEN_MINT" 2>/dev/null | grep "Creating account" | awk '{print $3}' || true)
    if [ -n "$ACCOUNT" ]; then
        TOKEN_ACCOUNTS[$i]="$ACCOUNT"
        info "  Token account $i: $ACCOUNT"
    else
        TOKEN_ACCOUNTS[$i]=""
    fi
    sleep 1
done

# ── Mint initial tokens to account 0 ──
info "Minting initial tokens..."
spl-token mint --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key0.json" --mint-authority "$KEY_DIR/key0.json" "$TOKEN_MINT" 1000000 "${TOKEN_ACCOUNTS[0]}" >/dev/null 2>&1 || true

# ── Main event loop ──
info "Starting Solana SPL event simulation..."
info "  RPC:        $SOLANA_RPC_URL"
info "  Token Mint: $TOKEN_MINT"
info "  Events/cycle: ${SIM_EVENTS_MIN}-${SIM_EVENTS_MAX}"
info "  Poisson mean: ${SIM_POISSON_MEAN}s"

CYCLE=0
START_SEC=$(date +%s)

while true; do
    CYCLE=$((CYCLE + 1))

    # Generate 1-3 events per cycle
    BATCH=$((RANDOM % ${SIM_EVENTS_MAX} + ${SIM_EVENTS_MIN}))
    for ((i=0; i<BATCH; i++)); do
        PICK=$((RANDOM % 100))
        FROM_IDX=$((RANDOM % 5))
        TO_IDX=$(( (FROM_IDX + 1 + RANDOM % 4) % 5 ))
        AMT=$((RANDOM % 100 + 1))

        if [ "${TOKEN_ACCOUNTS[$FROM_IDX]}" = "" ] || [ "${TOKEN_ACCOUNTS[$TO_IDX]}" = "" ]; then
            continue
        fi

        if [ $PICK -lt 55 ]; then
            # 55% SPL Transfer
            sim "SPL:Transfer ${AMT} tokens from account${FROM_IDX} -> account${TO_IDX}"
            spl-token transfer --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key${FROM_IDX}.json" \
                --owner "$KEY_DIR/key${FROM_IDX}.json" "$TOKEN_MINT" "$AMT" "${TOKEN_ACCOUNTS[$TO_IDX]}" \
                --allow-unfunded-recipient >/dev/null 2>&1 || true

        elif [ $PICK -lt 75 ]; then
            # 20% SPL MintTo
            sim "SPL:MintTo ${AMT} tokens to account${TO_IDX}"
            spl-token mint --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key0.json" \
                --mint-authority "$KEY_DIR/key0.json" "$TOKEN_MINT" "$AMT" "${TOKEN_ACCOUNTS[$TO_IDX]}" >/dev/null 2>&1 || true

        elif [ $PICK -lt 90 ]; then
            # 15% SPL Burn
            sim "SPL:Burn ${AMT} tokens from account${FROM_IDX}"
            spl-token burn --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key${FROM_IDX}.json" \
                --owner "$KEY_DIR/key${FROM_IDX}.json" "${TOKEN_ACCOUNTS[$FROM_IDX]}" "$AMT" >/dev/null 2>&1 || true

        else
            # 10% SPL Transfer with new account creation
            sim "SPL:Transfer + CreateAccount ${AMT} tokens to new account"
            NEW_ACCOUNT=$(spl-token create-account --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key${FROM_IDX}.json" \
                --owner "$KEY_DIR/key${FROM_IDX}.json" "$TOKEN_MINT" 2>/dev/null | grep "Creating account" | awk '{print $3}' || true)
            if [ -n "$NEW_ACCOUNT" ]; then
                spl-token transfer --url "$SOLANA_RPC_URL" --fee-payer "$KEY_DIR/key${FROM_IDX}.json" \
                    --owner "$KEY_DIR/key${FROM_IDX}.json" "$TOKEN_MINT" "$AMT" "$NEW_ACCOUNT" \
                    --allow-unfunded-recipient >/dev/null 2>&1 || true
            fi
        fi
    done

    # Poisson sleep
    SLEEP_TIME=$(awk -v seed=$RANDOM -v m="$SIM_POISSON_MEAN" 'BEGIN{srand(seed); u=rand(); t=-log(1-u)*m; if(t<1) t=1; printf "%.1f\n", t}')
    sleep "$SLEEP_TIME"
done