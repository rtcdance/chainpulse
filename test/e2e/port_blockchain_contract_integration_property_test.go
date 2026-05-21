package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestPortAllocationReleaseRoundTrip tests that allocate/release is a round trip
// Property: For any port allocation, releasing and re-allocating should work
func TestPortAllocationReleaseRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	portMgr, err := NewPortManager(10000, 10100)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	// Run property test 100 times
	for iteration := 0; iteration < 100; iteration++ {
		// Allocate port
		port1, err := portMgr.Allocate(ctx)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to allocate port: %v", iteration, err)
		}

		// Release port
		err = portMgr.Release(port1)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to release port: %v", iteration, err)
		}

		// Allocate again
		port2, err := portMgr.Allocate(ctx)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to re-allocate port: %v", iteration, err)
		}

		// Release again
		err = portMgr.Release(port2)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to re-release port: %v", iteration, err)
		}

		// Verify port is available
		if !portMgr.IsAvailable(port2) {
			t.Fatalf("Iteration %d: Port %d should be available", iteration, port2)
		}
	}

	t.Logf("✓ Port allocation/release round trip property test passed (100 iterations)")
}

// TestContractRegistrationUnregistrationRoundTrip tests that register/unregister is a round trip
// Property: For any contract registration, unregistering and re-registering should work
func TestContractRegistrationUnregistrationRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := NewContractRegistry()

	// Run property test 100 times
	for iteration := 0; iteration < 100; iteration++ {
		address := fmt.Sprintf("0x%040d", iteration)
		name := fmt.Sprintf("Contract_%d", iteration)

		// Register contract
		contract := &RegisteredContract{
			Address:      address,
			Name:         name,
			ChainID:      "31337",
			ABI:          `[]`,
			Bytecode:     "0x",
			DeploymentTx: fmt.Sprintf("0x%064d", iteration),
		}

		err := registry.Register(ctx, contract)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to register contract: %v", iteration, err)
		}

		// Verify contract exists
		retrieved, err := registry.Get(ctx, address)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to get contract: %v", iteration, err)
		}
		if retrieved.Name != name {
			t.Fatalf("Iteration %d: Expected name %s, got %s", iteration, name, retrieved.Name)
		}

		// Unregister contract
		err = registry.Unregister(ctx, address)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to unregister contract: %v", iteration, err)
		}

		// Verify contract doesn't exist
		_, err = registry.Get(ctx, address)
		if err == nil {
			t.Fatalf("Iteration %d: Contract should not exist after unregistration", iteration)
		}

		// Re-register contract
		err = registry.Register(ctx, contract)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to re-register contract: %v", iteration, err)
		}

		// Verify contract exists again
		retrieved, err = registry.Get(ctx, address)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to get re-registered contract: %v", iteration, err)
		}
		if retrieved.Name != name {
			t.Fatalf("Iteration %d: Expected name %s after re-registration, got %s", iteration, name, retrieved.Name)
		}

		// Clean up
		err = registry.Unregister(ctx, address)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to cleanup: %v", iteration, err)
		}
	}

	t.Logf("✓ Contract registration/unregistration round trip property test passed (100 iterations)")
}

// TestPortAllocationConsistency tests that port allocation is consistent
// Property: For any set of allocations, the total allocated + available should equal total
func TestPortAllocationConsistency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	portMgr, err := NewPortManager(10200, 10300)
	if err != nil {
		t.Fatalf("Failed to create port manager: %v", err)
	}

	// Run property test 50 times
	for iteration := 0; iteration < 50; iteration++ {
		// Allocate random number of ports (1-20)
		numToAllocate := (iteration % 20) + 1
		allocatedPorts := make([]int, 0)

		for i := 0; i < numToAllocate; i++ {
			port, err := portMgr.Allocate(ctx)
			if err != nil {
				// Port exhaustion is expected, break
				break
			}
			allocatedPorts = append(allocatedPorts, port)
		}

		// Check consistency
		stats := portMgr.GetStats()
		total := stats.AllocatedPorts + stats.AvailablePorts
		if total != stats.TotalPorts {
			t.Fatalf("Iteration %d: Inconsistent stats: allocated=%d, available=%d, total=%d",
				iteration, stats.AllocatedPorts, stats.AvailablePorts, stats.TotalPorts)
		}

		// Release all allocated ports
		for _, port := range allocatedPorts {
			err := portMgr.Release(port)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to release port: %v", iteration, err)
			}
		}

		// Check consistency again
		stats = portMgr.GetStats()
		total = stats.AllocatedPorts + stats.AvailablePorts
		if total != stats.TotalPorts {
			t.Fatalf("Iteration %d: Inconsistent stats after release: allocated=%d, available=%d, total=%d",
				iteration, stats.AllocatedPorts, stats.AvailablePorts, stats.TotalPorts)
		}
	}

	t.Logf("✓ Port allocation consistency property test passed (50 iterations)")
}

// TestContractRegistryListConsistency tests that list operations are consistent
// Property: For any set of registrations, ListByChain should be consistent with List
func TestContractRegistryListConsistency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := NewContractRegistry()

	// Run property test 50 times
	for iteration := 0; iteration < 50; iteration++ {
		// Register contracts on different chains
		chains := []string{"1", "137", "31337"}
		contractsPerChain := 3

		for _, chainID := range chains {
			for i := 0; i < contractsPerChain; i++ {
				contract := &RegisteredContract{
					Address:      fmt.Sprintf("0x%040d", iteration*100+len(chainID)*10+i),
					Name:         fmt.Sprintf("Contract_%s_%d", chainID, i),
					ChainID:      chainID,
					ABI:          `[]`,
					Bytecode:     "0x",
					DeploymentTx: fmt.Sprintf("0x%064d", iteration*100+len(chainID)*10+i),
				}

				err := registry.Register(ctx, contract)
				if err != nil {
					t.Fatalf("Iteration %d: Failed to register contract: %v", iteration, err)
				}
			}
		}

		// Get all contracts
		allContracts, err := registry.List(ctx)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to list all contracts: %v", iteration, err)
		}

		// Get contracts by chain and verify consistency
		totalByChain := 0
		for _, chainID := range chains {
			chainContracts, err := registry.ListByChain(ctx, chainID)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to list contracts by chain: %v", iteration, err)
			}
			totalByChain += len(chainContracts)

			// Verify all contracts in chain list are in all list
			for _, contract := range chainContracts {
				found := false
				for _, allContract := range allContracts {
					if allContract.Address == contract.Address {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Iteration %d: Contract %s not found in all list", iteration, contract.Address)
				}
			}
		}

		// Verify total consistency
		if totalByChain != len(allContracts) {
			t.Fatalf("Iteration %d: Total by chain (%d) != all contracts (%d)", iteration, totalByChain, len(allContracts))
		}

		// Clean up for next iteration
		for _, contract := range allContracts {
			err := registry.Unregister(ctx, contract.Address)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to cleanup: %v", iteration, err)
			}
		}
	}

	t.Logf("✓ Contract registry list consistency property test passed (50 iterations)")
}

// TestMetadataIsolation tests that metadata updates don't affect other contracts
// Property: For any contract, updating its metadata should not affect other contracts
func TestMetadataIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := NewContractRegistry()

	// Run property test 50 times
	for iteration := 0; iteration < 50; iteration++ {
		// Register multiple contracts
		numContracts := 5
		contracts := make([]*RegisteredContract, numContracts)

		for i := 0; i < numContracts; i++ {
			contract := &RegisteredContract{
				Address:      fmt.Sprintf("0x%040d", iteration*100+i),
				Name:         fmt.Sprintf("Contract_%d", i),
				ChainID:      "31337",
				ABI:          `[]`,
				Bytecode:     "0x",
				DeploymentTx: fmt.Sprintf("0x%064d", iteration*100+i),
			}
			contracts[i] = contract

			err := registry.Register(ctx, contract)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to register contract %d: %v", iteration, i, err)
			}
		}

		// Update metadata for first contract
		err := registry.UpdateMetadata(ctx, contracts[0].Address, map[string]any{
			"version": "2.0",
			"owner":   "0xowner",
		})
		if err != nil {
			t.Fatalf("Iteration %d: Failed to update metadata: %v", iteration, err)
		}

		// Verify other contracts' metadata is unchanged
		for i := 1; i < numContracts; i++ {
			contract, err := registry.Get(ctx, contracts[i].Address)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to get contract %d: %v", iteration, i, err)
			}

			// Metadata should be empty or not contain the updated values
			if len(contract.Metadata) > 0 {
				if contract.Metadata["version"] == "2.0" {
					t.Fatalf("Iteration %d: Contract %d metadata was affected", iteration, i)
				}
			}
		}

		// Verify first contract's metadata was updated
		updated, err := registry.Get(ctx, contracts[0].Address)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to get updated contract: %v", iteration, err)
		}
		if updated.Metadata["version"] != "2.0" {
			t.Fatalf("Iteration %d: Expected version 2.0, got %v", iteration, updated.Metadata["version"])
		}

		// Clean up
		for _, contract := range contracts {
			err := registry.Unregister(ctx, contract.Address)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to cleanup: %v", iteration, err)
			}
		}
	}

	t.Logf("✓ Metadata isolation property test passed (50 iterations)")
}

// TestContractChainIsolation tests that contracts are properly isolated by chain
// Property: For any contract on a chain, it should only appear in that chain's list
func TestContractChainIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registry := NewContractRegistry()

	// Run property test 50 times
	for iteration := 0; iteration < 50; iteration++ {
		chains := []string{"1", "137", "31337", "43114"}

		// Register contracts on each chain
		for chainIdx, chainID := range chains {
			// Generate unique address for each iteration and chain combination
			contract := &RegisteredContract{
				Address:      fmt.Sprintf("0x%040x", iteration*10000+chainIdx),
				Name:         fmt.Sprintf("Contract_%s_%d", chainID, iteration),
				ChainID:      chainID,
				ABI:          `[]`,
				Bytecode:     "0x",
				DeploymentTx: fmt.Sprintf("0x%064x", iteration*10000+chainIdx),
			}

			err := registry.Register(ctx, contract)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to register contract on chain %s: %v", iteration, chainID, err)
			}
		}

		// Verify isolation
		for _, chainID := range chains {
			chainContracts, err := registry.ListByChain(ctx, chainID)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to list contracts on chain %s: %v", iteration, chainID, err)
			}

			// Should have exactly 1 contract
			if len(chainContracts) != 1 {
				t.Fatalf("Iteration %d: Expected 1 contract on chain %s, got %d", iteration, chainID, len(chainContracts))
			}

			// Contract should have correct chain ID
			if chainContracts[0].ChainID != chainID {
				t.Fatalf("Iteration %d: Contract on chain %s has wrong chain ID: %s", iteration, chainID, chainContracts[0].ChainID)
			}
		}

		// Clean up
		allContracts, err := registry.List(ctx)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to list all contracts: %v", iteration, err)
		}
		for _, contract := range allContracts {
			err := registry.Unregister(ctx, contract.Address)
			if err != nil {
				t.Fatalf("Iteration %d: Failed to cleanup: %v", iteration, err)
			}
		}
	}

	t.Logf("✓ Chain isolation property test passed (50 iterations)")
}
