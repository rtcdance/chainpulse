# chainpulse Anvil Testing Environment

This directory contains the complete Anvil testing environment for the chainpulse Web3 project, including configuration files, test contracts, integration tests, and automation scripts.

## Overview

The Anvil testing environment provides:

- Local blockchain simulation using Anvil (Foundry)
- Test smart contracts for various Web3 scenarios
- Go integration tests for contract interactions
- Automated testing workflows
- Docker-based test environment

## Directory Structure

```
test/anvil/
├── config/                    # Configuration files
│   ├── foundry.toml           # Foundry configuration
│   ├── anvil.env              # Environment variables for Anvil
│   └── docker-compose.test.yml # Docker Compose for test environment
├── contracts/                 # Test smart contracts
│   ├── TestERC20.sol          # ERC20 token test contract
│   ├── TestERC721.sol         # ERC721 NFT test contract
│   ├── TestDeFiPool.sol       # DeFi pool test contract
│   └── TestGovernance.sol     # Governance test contract
├── integration/               # Go integration tests
│   ├── blockchain_test.go     # Blockchain interaction tests
│   ├── contract_test.go       # Contract interaction tests
│   └── event_test.go          # Event listening tests
├── scripts/                   # Automation scripts
│   ├── deploy_contracts.go    # Contract deployment script
│   └── run_tests.sh           # Test automation script
└── README.md                  # This file
```

## Prerequisites

Before running the tests, ensure you have:

- [Go 1.21+](https://golang.org/dl/)
- [Foundry](https://github.com/foundry-rs/foundry) (includes Anvil and Forge)
- [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) (optional, for containerized tests)

Install Foundry:
```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

## Quick Start

### 1. Using Makefile (Recommended)

```bash
# Check prerequisites
make anvil-test-deps

# Run all Anvil tests
make anvil-run-all-tests

# Run complete test suite (contracts + Go integration)
make anvil-full-test

# View all available commands
make help
```

### 2. Using the Automation Script

```bash
# Check prerequisites
./test/anvil/scripts/run_tests.sh check

# Run complete test suite
./test/anvil/scripts/run_tests.sh test-all

# Run specific test types
./test/anvil/scripts/run_tests.sh test-contracts  # Contract tests only
./test/anvil/scripts/run_tests.sh test-go         # Go integration tests only
./test/anvil/scripts/run_tests.sh test-docker     # Docker-based tests

# Manually start/stop Anvil
./test/anvil/scripts/run_tests.sh start
./test/anvil/scripts/run_tests.sh stop
```

## Test Contracts

The test environment includes several smart contracts for different Web3 scenarios:

- **TestERC20.sol**: Standard ERC20 token implementation for testing token transfers and balances
- **TestERC721.sol**: Standard ERC721 NFT implementation for testing NFT minting and transfers
- **TestDeFiPool.sol**: DeFi liquidity pool for testing yield farming and staking
- **TestGovernance.sol**: DAO governance contract for testing proposal and voting mechanisms

## Integration Tests

Go integration tests verify interactions with the blockchain:

- **blockchain_test.go**: Tests basic blockchain operations (connection, transactions, blocks)
- **contract_test.go**: Tests interactions with deployed contracts (ERC20, ERC721, DeFi)
- **event_test.go**: Tests event listening and filtering capabilities

## Configuration

### Environment Variables

The test environment uses the following environment variables:

```bash
ANVIL_RPC_URL=http://localhost:8545    # Anvil RPC endpoint
ANVIL_CHAIN_ID=31337                   # Anvil chain ID
ANVIL_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80  # Default Anvil private key
```

### Foundry Configuration

See `config/foundry.toml` for Foundry-specific settings including compiler version, optimizer settings, and remappings.

## Docker Environment

For containerized testing, use the provided Docker Compose configuration:

```bash
# Start the complete test environment
docker-compose -f test/anvil/config/docker-compose.test.yml up -d

# Run tests against the containerized environment
ANVIL_RPC_URL=http://localhost:8545 go test -v ./test/anvil/integration/...

# Stop the test environment
docker-compose -f test/anvil/config/docker-compose.test.yml down
```

## Development Workflow

When adding new test contracts or integration tests:

1. Add your Solidity contract to the `contracts/` directory
2. Run `forge build` to compile the contract
3. Create Go bindings using `abigen` (if needed for integration tests)
4. Add integration tests to the `integration/` directory
5. Update the test automation scripts if necessary

## Troubleshooting

### Common Issues

- **Anvil not found**: Make sure Foundry is installed and `anvil` is in your PATH
- **Port already in use**: Kill existing Anvil processes with `pkill -f anvil`
- **Contract compilation errors**: Check that OpenZeppelin contracts are installed in the `lib/` directory
- **Connection refused**: Ensure Anvil is running before running integration tests

### Useful Commands

```bash
# Check if Anvil is running
curl -s http://localhost:8545 -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# View Anvil logs if running in background
tail -f /tmp/anvil.log

# Run a single integration test
go test -v ./test/anvil/integration/ -run TestBlockchainConnection
```

## Best Practices

- Always run `make anvil-test-deps` before running tests to check prerequisites
- Use the automation scripts for consistent test execution
- Write both contract tests (using Forge) and Go integration tests
- Mock external dependencies in test contracts to ensure deterministic tests
- Use the default Anvil accounts and private keys for consistent test results