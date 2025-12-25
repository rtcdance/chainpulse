#!/bin/bash

# Anvil Testing Automation Script for chainpulse
# This script automates the complete Anvil testing workflow

set -e  # Exit immediately if a command exits with a non-zero status

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
ANVIL_CONFIG_DIR="$PROJECT_ROOT/test/anvil/config"
ANVIL_CONTRACTS_DIR="$PROJECT_ROOT/test/anvil/contracts"
ANVIL_INTEGRATION_DIR="$PROJECT_ROOT/test/anvil/integration"

# Default values that can be overridden by environment variables
ANVIL_RPC_URL="${ANVIL_RPC_URL:-http://localhost:8545}"
ANVIL_CHAIN_ID="${ANVIL_CHAIN_ID:-31337}"
ANVIL_PORT="${ANVIL_PORT:-8545}"
DEFAULT_PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
ANVIL_PRIVATE_KEY="${ANVIL_PRIVATE_KEY:-$DEFAULT_PRIVATE_KEY}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    if ! command_exists anvil; then
        log_error "anvil is not installed. Please install Foundry first:"
        log_error "curl -L https://foundry.paradigm.xyz | bash && foundryup"
        exit 1
    fi

    if ! command_exists forge; then
        log_error "forge is not installed. Please install Foundry first:"
        log_error "curl -L https://foundry.paradigm.xyz | bash && foundryup"
        exit 1
    fi

    if ! command_exists go; then
        log_error "go is not installed. Please install Go 1.21+"
        exit 1
    fi

    if ! command_exists docker; then
        log_warn "docker is not installed. Docker-based tests will be skipped."
    fi

    if ! command_exists docker-compose; then
        log_warn "docker-compose is not installed. Docker-based tests will be skipped."
    fi

    log_success "All prerequisites checked."
}

# Function to start Anvil in the background
start_anvil() {
    log_info "Starting Anvil on port $ANVIL_PORT with chain ID $ANVIL_CHAIN_ID..."

    # Kill any existing Anvil process on the same port
    pkill -f "anvil.*--port $ANVIL_PORT" || true
    sleep 2

    # Start Anvil in the background
    anvil --host 0.0.0.0 --port $ANVIL_PORT --chain-id $ANVIL_CHAIN_ID --block-time 1 > /tmp/anvil.log 2>&1 &
    ANVIL_PID=$!

    # Wait a bit for Anvil to start
    sleep 3

    # Check if Anvil started successfully
    if ! kill -0 $ANVIL_PID 2>/dev/null; then
        log_error "Failed to start Anvil. Check /tmp/anvil.log for details."
        cat /tmp/anvil.log
        exit 1
    fi

    # Test connection to Anvil
    if ! curl -s "$ANVIL_RPC_URL" -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' > /dev/null 2>&1; then
        log_error "Failed to connect to Anvil at $ANVIL_RPC_URL"
        exit 1
    fi

    log_success "Anvil started successfully with PID $ANVIL_PID"
}

# Function to stop Anvil
stop_anvil() {
    if [ ! -z "$ANVIL_PID" ] && kill -0 $ANVIL_PID 2>/dev/null; then
        log_info "Stopping Anvil (PID: $ANVIL_PID)..."
        kill $ANVIL_PID
        sleep 2
        log_success "Anvil stopped."
    else
        log_info "Anvil is not running."
    fi
}

# Function to compile contracts
compile_contracts() {
    log_info "Compiling test contracts..."
    
    if [ ! -d "$ANVIL_CONTRACTS_DIR" ]; then
        log_error "Contracts directory does not exist: $ANVIL_CONTRACTS_DIR"
        exit 1
    fi

    cd "$ANVIL_CONTRACTS_DIR"
    
    # Install OpenZeppelin if not already installed
    if [ ! -d "lib/openzeppelin-contracts" ]; then
        log_info "Installing OpenZeppelin contracts..."
        mkdir -p lib
        git clone https://github.com/OpenZeppelin/openzeppelin-contracts.git lib/openzeppelin-contracts
    fi

    # Build contracts
    forge build
    
    log_success "Contracts compiled successfully."
    cd - > /dev/null
}

# Function to run contract tests
run_contract_tests() {
    log_info "Running contract tests..."

    if [ ! -d "$ANVIL_CONTRACTS_DIR" ]; then
        log_error "Contracts directory does not exist: $ANVIL_CONTRACTS_DIR"
        exit 1
    fi

    cd "$ANVIL_CONTRACTS_DIR"
    forge test
    cd - > /dev/null

    log_success "Contract tests completed."
}

# Function to run Go integration tests
run_integration_tests() {
    log_info "Running Go integration tests..."

    if [ ! -d "$ANVIL_INTEGRATION_DIR" ]; then
        log_error "Integration tests directory does not exist: $ANVIL_INTEGRATION_DIR"
        exit 1
    fi

    # Set environment variables for the tests
    export ANVIL_RPC_URL
    export ANVIL_CHAIN_ID
    export ANVIL_PRIVATE_KEY

    cd "$PROJECT_ROOT"
    go test -v ./test/anvil/integration/...
    cd - > /dev/null

    log_success "Go integration tests completed."
}

# Function to run Docker-based tests
run_docker_tests() {
    if ! command_exists docker || ! command_exists docker-compose; then
        log_warn "Docker or docker-compose not available. Skipping Docker tests."
        return
    fi

    log_info "Running Docker-based Anvil tests..."

    if [ ! -f "$ANVIL_CONFIG_DIR/docker-compose.test.yml" ]; then
        log_error "Docker Compose config not found: $ANVIL_CONFIG_DIR/docker-compose.test.yml"
        return
    fi

    cd "$PROJECT_ROOT"
    docker-compose -f "$ANVIL_CONFIG_DIR/docker-compose.test.yml" up -d --build
    
    # Wait for services to start
    sleep 15

    # Run tests against the Docker environment
    ANVIL_RPC_URL=http://localhost:8545 go test -v ./test/anvil/integration/...

    # Stop the Docker services
    docker-compose -f "$ANVIL_CONFIG_DIR/docker-compose.test.yml" down
    
    log_success "Docker-based tests completed."
}

# Function to run a complete test suite
run_complete_tests() {
    log_info "Starting complete Anvil test suite..."

    # Compile contracts
    compile_contracts

    # Start Anvil
    start_anvil

    # Run contract tests
    run_contract_tests

    # Run Go integration tests
    run_integration_tests

    # Stop Anvil
    stop_anvil

    # Run Docker tests (optional)
    run_docker_tests

    log_success "Complete Anvil test suite completed successfully!"
}

# Function to show help
show_help() {
    echo "Anvil Testing Automation Script for chainpulse"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  help           Show this help message"
    echo "  check          Check prerequisites"
    echo "  start          Start Anvil locally"
    echo "  stop           Stop Anvil"
    echo "  compile        Compile test contracts"
    echo "  test-contracts Run contract tests"
    echo "  test-go        Run Go integration tests"
    echo "  test-docker    Run Docker-based tests"
    echo "  test-all       Run complete test suite"
    echo ""
    echo "Environment Variables:"
    echo "  ANVIL_RPC_URL      Anvil RPC URL (default: http://localhost:8545)"
    echo "  ANVIL_CHAIN_ID     Anvil chain ID (default: 31337)"
    echo "  ANVIL_PORT         Anvil port (default: 8545)"
    echo "  ANVIL_PRIVATE_KEY  Private key for testing (default: Anvil default)"
    echo ""
}

# Main execution
case "${1:-help}" in
    "help")
        show_help
        ;;
    "check")
        check_prerequisites
        ;;
    "start")
        check_prerequisites
        start_anvil
        echo "Anvil is running. Press Ctrl+C to stop."
        # Keep the script running so Anvil continues
        while kill -0 $ANVIL_PID 2>/dev/null; do
            sleep 1
        done
        ;;
    "stop")
        stop_anvil
        ;;
    "compile")
        check_prerequisites
        compile_contracts
        ;;
    "test-contracts")
        check_prerequisites
        run_contract_tests
        ;;
    "test-go")
        check_prerequisites
        run_integration_tests
        ;;
    "test-docker")
        check_prerequisites
        run_docker_tests
        ;;
    "test-all")
        check_prerequisites
        run_complete_tests
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac