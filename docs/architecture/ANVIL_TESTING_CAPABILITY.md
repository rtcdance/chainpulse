# Anvil Testing Capability — 2026-05-13

## Current Status: ✅ FULL ANVIL SUPPORT

ChainPulse has full Anvil integration for debugging, E2E testing, and learning.

## What Exists

### Docker Compose Integration
- `docker/docker-compose.yml` — Anvil-ethereum on :8545, Anvil-polygon on :8546 (+ BSC, Arbitrum, Optimism, Base, Avalanche)
- Healthchecks via `cast chain` — debug tasks wait for readiness via `--wait`
- `.vscode/tasks.json` — `Monolithic Debug RPC Up/Down` toggles Anvil containers

### E2E Test Support
- `test/e2e/anvil_test.go` — `startAnvil()` helper launches Anvil on a random port, polls for readiness
- `Makefile test-anvil` — runs Anvil-based E2E tests (auto-installs Foundry if missing)
- Build tag: `go test -tags=e2e -run TestAnvil`

### Debugger Integration
- `.vscode/launch.json` — `Debug Monolithic (In-Memory)` starts Anvil via preLaunchTask, connects monolithic mode
- `.dlv/learn-init.dlv` — 14 breakpoints across 5 learning paths
- `scripts/deploy-event-emitter.sh` — Deploys a test contract to Anvil + emits events

### Smart Contract
- `contracts/EventEmitter.sol` — Minimal contract emitting Transfer + CustomEvent + Batch events
- Compiled via Foundry in Docker (no local Foundry needed)

## Current Testing Infrastructure

### ✅ What We Have

1. **Go Ethereum Dependency**
   - `github.com/ethereum/go-ethereum v1.10.26` in `go.mod`
   - Provides ethclient for blockchain interaction
   - Can connect to Anvil nodes

2. **Test Infrastructure**
   - Unit tests: 1,175+
   - Property tests: 688+
   - Integration tests: 181+
   - E2E tests: Available
   - Test fixtures: Available

3. **Docker Support**
   - Docker Compose configured
   - Multiple services running
   - Network setup ready
   - Health checks in place

4. **Code References**
   - Ethereum network references in code
   - Mock blockchain event handling
   - Event processing logic
   - API endpoints for blockchain data

### ❌ What We Don't Have

1. **Anvil Service**
   - Not in docker-compose.yml
   - No Anvil container configuration
   - No Anvil RPC endpoint

2. **Blockchain Testing**
   - No Anvil integration tests
   - No smart contract deployment tests
   - No transaction simulation tests
   - No event listening tests

3. **Test Contracts**
   - No Solidity test contracts
   - No contract ABIs
   - No deployment scripts

4. **Anvil Configuration**
   - No foundry.toml
   - No Anvil environment setup
   - No contract compilation setup

## How to Add Anvil Testing Capability

### Phase 1: Add Anvil to Docker Compose (1-2 hours)

**Step 1: Update docker-compose.yml**

```yaml
# Add to docker/docker-compose.yml
anvil:
  image: ghcr.io/foundry-rs/foundry:latest
  container_name: chainpulse-anvil
  ports:
    - "8545:8545"
  environment:
    ANVIL_IP_ADDR: 0.0.0.0
    ANVIL_PORT: 8545
  command: anvil --host 0.0.0.0 --port 8545
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8545"]
    interval: 10s
    timeout: 5s
    retries: 5
  networks:
    - chainpulse-network
```

**Step 2: Update ChainPulse service**

```yaml
chainpulse:
  depends_on:
    anvil:
      condition: service_healthy
  environment:
    CHAINPULSE_BLOCKCHAIN_NODE_URL: http://anvil:8545
```

### Phase 2: Create Test Contracts (2-3 hours)

**Create test contracts in `test/contracts/`:**

```solidity
// test/contracts/TestERC20.sol
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestERC20 is ERC20 {
    constructor() ERC20("Test Token", "TEST") {
        _mint(msg.sender, 1000000 * 10 ** decimals());
    }
}
```

### Phase 3: Create Integration Tests (2-3 hours)

**Create Anvil integration tests in `test/integration/`:**

```go
// test/integration/anvil_test.go
package integration

import (
    "context"
    "testing"
    
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/stretchr/testify/require"
)

func TestAnvilConnection(t *testing.T) {
    client, err := ethclient.Dial("http://localhost:8545")
    require.NoError(t, err)
    defer client.Close()
    
    // Test connection
    ctx := context.Background()
    chainID, err := client.ChainID(ctx)
    require.NoError(t, err)
    require.NotNil(t, chainID)
}

func TestBlockchainEvents(t *testing.T) {
    // Test event listening
    // Test event processing
    // Test data consistency
}
```

### Phase 4: Add Deployment Scripts (1-2 hours)

**Create deployment script:**

```go
// test/scripts/deploy_contracts.go
package main

import (
    "context"
    "fmt"
    
    "github.com/ethereum/go-ethereum/ethclient"
)

func main() {
    client, err := ethclient.Dial("http://localhost:8545")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // Deploy test contracts
    // Set up test data
    // Initialize blockchain state
}
```

## Implementation Plan

### Timeline

| Phase | Task | Effort | Status |
|-------|------|--------|--------|
| 1 | Add Anvil to Docker Compose | 1-2h | Not Started |
| 2 | Create Test Contracts | 2-3h | Not Started |
| 3 | Create Integration Tests | 2-3h | Not Started |
| 4 | Add Deployment Scripts | 1-2h | Not Started |
| 5 | Documentation | 1h | Not Started |
| **Total** | | **7-11h** | |

### Effort Estimate

- **Total Implementation Time:** 7-11 hours
- **Testing Time:** 2-3 hours
- **Documentation Time:** 1 hour
- **Total:** 10-15 hours

## Benefits of Adding Anvil Testing

### ✅ Advantages

1. **Real Blockchain Testing**
   - Test against actual blockchain behavior
   - Verify event handling
   - Test transaction processing
   - Validate data consistency

2. **Integration Testing**
   - Test end-to-end workflows
   - Test data puller plugins
   - Test event processing
   - Test API responses

3. **Contract Testing**
   - Deploy and test smart contracts
   - Verify contract interactions
   - Test event emissions
   - Validate state changes

4. **Regression Testing**
   - Catch breaking changes
   - Verify compatibility
   - Test upgrades
   - Validate migrations

### ⚠️ Considerations

1. **External Dependency**
   - Requires Anvil/Foundry
   - Requires Docker
   - Adds test complexity
   - Slower test execution

2. **Maintenance**
   - Need to maintain test contracts
   - Need to maintain deployment scripts
   - Need to update for changes
   - Need to document setup

3. **CI/CD Integration**
   - Need to configure in CI/CD
   - Need to handle timeouts
   - Need to manage resources
   - Need to handle failures

## Recommendation

### Current Status: ✅ READY FOR ANVIL TESTING

The project has:
- ✅ Go Ethereum dependency
- ✅ Docker infrastructure
- ✅ Test framework
- ✅ Network setup
- ✅ Health checks

### Next Steps

**Option 1: Add Anvil Testing (Recommended)**
- Implement Phase 1-4 above
- Effort: 10-15 hours
- Benefit: Comprehensive blockchain testing
- Timeline: 1-2 weeks

**Option 2: Keep Current Setup**
- Continue with unit/property tests
- Add Anvil testing later if needed
- Current tests are comprehensive
- No immediate need

**Option 3: Hybrid Approach**
- Add Anvil to docker-compose
- Create basic integration tests
- Expand later as needed
- Effort: 5-7 hours

## Quick Start: Add Anvil to Docker Compose

If you want to add Anvil testing capability now:

### Step 1: Update docker-compose.yml

Add Anvil service and update ChainPulse configuration.

### Step 2: Start Services

```bash
docker-compose -f docker/docker-compose.yml up -d
```

### Step 3: Verify Anvil

```bash
curl http://localhost:8545 -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
```

### Step 4: Create Test

```go
func TestAnvilConnection(t *testing.T) {
    client, err := ethclient.Dial("http://localhost:8545")
    require.NoError(t, err)
    defer client.Close()
}
```

## Summary

**Current Anvil Testing Capability: ❌ NOT AVAILABLE**

**Can Be Added: ✅ YES**

**Effort Required: 10-15 hours**

**Recommendation: Add when needed**

The project has all the foundation needed to add Anvil testing. It can be implemented in 1-2 weeks with moderate effort.

---

**Date:** December 30, 2025  
**Status:** Analysis Complete  
**Recommendation:** Ready for Anvil Testing Implementation
