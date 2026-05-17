package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestPortBlockchainContractIntegration tests the integration of Port Manager, Blockchain Manager, and Contract Registry
func TestPortBlockchainContractIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Create Port Manager
	portMgr, err := NewPortManager(9000, 9100)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	// Test 2: Allocate a port
	port, err := portMgr.Allocate(ctx)
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}
	if port < 9000 || port >= 9100 {
		t.Fatalf("Port out of range: %d", port)
	}

	// Test 3: Create Contract Registry
	registry := NewContractRegistry()
	if registry == nil {
		t.Fatal("Failed to create contract registry")
	}

	// Test 4: Register a contract
	contract := &RegisteredContract{
		Address:      "0x1234567890123456789012345678901234567890",
		Name:         "TestToken",
		ChainID:      "31337",
		ABI:          `[{"type":"function","name":"transfer"}]`,
		Bytecode:     "0xabcdef",
		DeploymentTx: "0xdeadbeef",
	}

	err = registry.Register(ctx, contract)
	if err != nil {
		t.Fatalf("Failed to register contract: %v", err)
	}

	// Test 5: Retrieve contract by address
	retrieved, err := registry.Get(ctx, contract.Address)
	if err != nil {
		t.Fatalf("Failed to get contract: %v", err)
	}
	if retrieved.Name != "TestToken" {
		t.Fatalf("Expected TestToken, got %s", retrieved.Name)
	}

	// Test 6: Retrieve contract by name
	byName, err := registry.GetByName(ctx, "TestToken")
	if err != nil {
		t.Fatalf("Failed to get contract by name: %v", err)
	}
	if byName.Address != contract.Address {
		t.Fatalf("Expected %s, got %s", contract.Address, byName.Address)
	}

	// Test 7: Update metadata
	err = registry.UpdateMetadata(ctx, contract.Address, map[string]any{
		"version": "1.0",
		"owner":   "0xowner",
	})
	if err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	// Test 8: Verify metadata was updated
	updated, err := registry.Get(ctx, contract.Address)
	if err != nil {
		t.Fatalf("Failed to get updated contract: %v", err)
	}
	if updated.Metadata["version"] != "1.0" {
		t.Fatalf("Expected version 1.0, got %v", updated.Metadata["version"])
	}

	// Test 9: Register multiple contracts on different chains
	contract2 := &RegisteredContract{
		Address:      "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		Name:         "TestUSDC",
		ChainID:      "1",
		ABI:          `[{"type":"function","name":"approve"}]`,
		Bytecode:     "0x123456",
		DeploymentTx: "0xcafebabe",
	}

	err = registry.Register(ctx, contract2)
	if err != nil {
		t.Fatalf("Failed to register second contract: %v", err)
	}

	// Test 10: List contracts by chain
	chain31337Contracts, err := registry.ListByChain(ctx, "31337")
	if err != nil {
		t.Fatalf("Failed to list contracts by chain: %v", err)
	}
	if len(chain31337Contracts) != 1 {
		t.Fatalf("Expected 1 contract on chain 31337, got %d", len(chain31337Contracts))
	}

	chain1Contracts, err := registry.ListByChain(ctx, "1")
	if err != nil {
		t.Fatalf("Failed to list contracts on chain 1: %v", err)
	}
	if len(chain1Contracts) != 1 {
		t.Fatalf("Expected 1 contract on chain 1, got %d", len(chain1Contracts))
	}

	// Test 11: Get registry statistics
	stats := registry.GetStats()
	if stats.TotalContracts != 2 {
		t.Fatalf("Expected 2 total contracts, got %d", stats.TotalContracts)
	}
	if stats.ContractsByChain["31337"] != 1 {
		t.Fatalf("Expected 1 contract on chain 31337, got %d", stats.ContractsByChain["31337"])
	}
	if stats.ContractsByChain["1"] != 1 {
		t.Fatalf("Expected 1 contract on chain 1, got %d", stats.ContractsByChain["1"])
	}

	// Test 12: Release port
	err = portMgr.Release(port)
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Test 13: Verify port is available again
	if !portMgr.IsAvailable(port) {
		t.Fatalf("Port %d should be available after release", port)
	}

	// Test 14: Get port manager stats
	stats2 := portMgr.GetStats()
	if stats2.AllocatedPorts != 0 {
		t.Fatalf("Expected 0 allocated ports, got %d", stats2.AllocatedPorts)
	}
	if stats2.AvailablePorts != stats2.TotalPorts {
		t.Fatalf("Expected all ports available, got %d/%d", stats2.AvailablePorts, stats2.TotalPorts)
	}

	t.Logf("✓ All integration tests passed")
}

// TestConcurrentPortAllocationWithContractRegistry tests concurrent port allocation with contract registration
func TestConcurrentPortAllocationWithContractRegistry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	portMgr, err := NewPortManager(9200, 9300)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	registry := NewContractRegistry()

	// Allocate 10 ports concurrently and register contracts
	numGoroutines := 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			// Allocate port
			port, err := portMgr.Allocate(ctx)
			if err != nil {
				done <- err
				return
			}

			// Register contract
			contract := &RegisteredContract{
				Address:      generateAddress(index),
				Name:         generateContractName(index),
				ChainID:      "31337",
				ABI:          `[]`,
				Bytecode:     "0x",
				DeploymentTx: generateTxHash(index),
			}

			err = registry.Register(ctx, contract)
			if err != nil {
				done <- err
				return
			}

			// Release port
			err = portMgr.Release(port)
			if err != nil {
				done <- err
				return
			}

			done <- nil
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Goroutine %d failed: %v", i, err)
		}
	}

	// Verify all contracts were registered
	contracts, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list contracts: %v", err)
	}
	if len(contracts) != numGoroutines {
		t.Fatalf("Expected %d contracts, got %d", numGoroutines, len(contracts))
	}

	// Verify all ports were released
	stats := portMgr.GetStats()
	if stats.AllocatedPorts != 0 {
		t.Fatalf("Expected 0 allocated ports, got %d", stats.AllocatedPorts)
	}

	t.Logf("✓ Concurrent integration test passed")
}

// TestMultiChainContractDeploymentWithPorts tests multi-chain contract deployment with dynamic port allocation
func TestMultiChainContractDeploymentWithPorts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	portMgr, err := NewPortManager(9400, 9500)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	registry := NewContractRegistry()

	// Simulate deployment on multiple chains
	chains := []string{"1", "137", "43114", "31337"}
	contractsPerChain := 3

	for _, chainID := range chains {
		// Allocate port for this chain
		port, err := portMgr.Allocate(ctx)
		if err != nil {
			t.Fatalf("Failed to allocate port for chain %s: %v", chainID, err)
		}

		// Deploy contracts on this chain
		for i := 0; i < contractsPerChain; i++ {
			contract := &RegisteredContract{
				Address:      generateAddressForChain(chainID, i),
				Name:         generateContractNameForChain(chainID, i),
				ChainID:      chainID,
				ABI:          `[]`,
				Bytecode:     "0x",
				DeploymentTx: generateTxHashForChain(chainID, i),
			}

			err := registry.Register(ctx, contract)
			if err != nil {
				t.Fatalf("Failed to register contract on chain %s: %v", chainID, err)
			}
		}

		// Release port after deployment
		err = portMgr.Release(port)
		if err != nil {
			t.Fatalf("Failed to release port for chain %s: %v", chainID, err)
		}
	}

	// Verify contracts on each chain
	for _, chainID := range chains {
		contracts, err := registry.ListByChain(ctx, chainID)
		if err != nil {
			t.Fatalf("Failed to list contracts on chain %s: %v", chainID, err)
		}
		if len(contracts) != contractsPerChain {
			t.Fatalf("Expected %d contracts on chain %s, got %d", contractsPerChain, chainID, len(contracts))
		}
	}

	// Verify total contracts
	allContracts, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list all contracts: %v", err)
	}
	expectedTotal := len(chains) * contractsPerChain
	if len(allContracts) != expectedTotal {
		t.Fatalf("Expected %d total contracts, got %d", expectedTotal, len(allContracts))
	}

	// Verify all ports released
	stats := portMgr.GetStats()
	if stats.AllocatedPorts != 0 {
		t.Fatalf("Expected 0 allocated ports, got %d", stats.AllocatedPorts)
	}

	t.Logf("✓ Multi-chain deployment test passed")
}

// TestPortExhaustionHandling tests handling of port exhaustion
func TestPortExhaustionHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create port manager with small range
	portMgr, err := NewPortManager(9600, 9605)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	registry := NewContractRegistry()

	// Allocate all available ports
	ports := make([]int, 0)
	for i := 0; i < 5; i++ {
		port, err := portMgr.Allocate(ctx)
		if err != nil {
			t.Fatalf("Failed to allocate port %d: %v", i, err)
		}
		ports = append(ports, port)

		// Register a contract for this port
		contract := &RegisteredContract{
			Address:      generateAddress(i),
			Name:         generateContractName(i),
			ChainID:      "31337",
			ABI:          `[]`,
			Bytecode:     "0x",
			DeploymentTx: generateTxHash(i),
		}
		err = registry.Register(ctx, contract)
		if err != nil {
			t.Fatalf("Failed to register contract: %v", err)
		}
	}

	// Try to allocate one more port (should fail)
	_, err = portMgr.Allocate(ctx)
	if err == nil {
		t.Fatal("Expected error when allocating port from exhausted pool")
	}

	// Release one port
	err = portMgr.Release(ports[0])
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Now allocation should succeed
	port, err := portMgr.Allocate(ctx)
	if err != nil {
		t.Fatalf("Failed to allocate port after release: %v", err)
	}

	// Release all ports
	for _, p := range ports[1:] {
		err = portMgr.Release(p)
		if err != nil {
			t.Fatalf("Failed to release port: %v", err)
		}
	}
	err = portMgr.Release(port)
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Verify all ports available
	stats := portMgr.GetStats()
	if stats.AllocatedPorts != 0 {
		t.Fatalf("Expected 0 allocated ports, got %d", stats.AllocatedPorts)
	}

	t.Logf("✓ Port exhaustion handling test passed")
}

// Helper functions
func generateAddress(index int) string {
	hex := fmt.Sprintf("%040x", index+1)
	return "0x" + hex
}

func generateAddressForChain(chainID string, index int) string {
	// Generate unique address using chainID hash and index
	chainHash := 0
	for _, c := range chainID {
		chainHash = chainHash*31 + int(c)
	}
	// Ensure uniqueness by combining chain hash and index
	uniqueValue := (chainHash%10000)*100000 + index
	hex := fmt.Sprintf("%040x", uniqueValue)
	return "0x" + hex
}

func generateContractName(index int) string {
	names := []string{"Token", "USDC", "DAI", "USDT", "WETH", "LINK", "UNI", "AAVE", "COMP", "MKR"}
	return names[index%len(names)]
}

func generateContractNameForChain(chainID string, index int) string {
	return generateContractName(index) + "_" + chainID
}

func generateTxHash(index int) string {
	hex := fmt.Sprintf("%064x", index+1)
	return "0x" + hex
}

func generateTxHashForChain(chainID string, index int) string {
	// Generate unique tx hash using chainID hash and index
	chainHash := 0
	for _, c := range chainID {
		chainHash = chainHash*31 + int(c)
	}
	// Ensure uniqueness by combining chain hash and index
	uniqueValue := (chainHash%1000000)*1000 + index
	hex := fmt.Sprintf("%064x", uniqueValue)
	return "0x" + hex
}
