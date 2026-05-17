package e2e

import (
	"context"
	"fmt"
	"testing"
	"testing/quick"
)

// TestContractRegistry_PropertyRegisterUnregister tests that register/unregister are inverses
func TestContractRegistry_PropertyRegisterUnregister(t *testing.T) {
	f := func(index uint32) bool {
		registry := NewContractRegistry()
		ctx := context.Background()

		contract := &RegisteredContract{
			Address: fmt.Sprintf("0x%040x", index),
			Name:    fmt.Sprintf("Token%d", index),
			ChainID: "31337",
		}

		// Register
		if err := registry.Register(ctx, contract); err != nil {
			t.Logf("Register failed: %v", err)
			return false
		}

		// Verify it exists
		if _, err := registry.Get(ctx, contract.Address); err != nil {
			t.Logf("Get after register failed: %v", err)
			return false
		}

		// Unregister
		if err := registry.Unregister(ctx, contract.Address); err != nil {
			t.Logf("Unregister failed: %v", err)
			return false
		}

		// Verify it doesn't exist
		if _, err := registry.Get(ctx, contract.Address); err == nil {
			t.Logf("Get after unregister should fail but didn't")
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestContractRegistry_PropertyListConsistency tests that List is consistent with individual Gets
func TestContractRegistry_PropertyListConsistency(t *testing.T) {
	f := func(indices []uint32) bool {
		if len(indices) == 0 || len(indices) > 50 {
			return true // Skip edge cases
		}

		registry := NewContractRegistry()
		ctx := context.Background()

		// Register contracts
		for i := range indices {
			contract := &RegisteredContract{
				Address: fmt.Sprintf("0x%040x", i),
				Name:    fmt.Sprintf("Token%d", i),
				ChainID: "31337",
			}
			if err := registry.Register(ctx, contract); err != nil {
				t.Logf("Register failed: %v", err)
				return false
			}
		}

		// Get list
		list, err := registry.List(ctx)
		if err != nil {
			t.Logf("List failed: %v", err)
			return false
		}

		// Verify count
		if len(list) != len(indices) {
			t.Logf("List count mismatch: expected %d, got %d", len(indices), len(list))
			return false
		}

		// Verify each contract in list can be retrieved
		for _, contract := range list {
			if _, err := registry.Get(ctx, contract.Address); err != nil {
				t.Logf("Get failed for contract in list: %v", err)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestContractRegistry_PropertyMetadataIsolation tests that metadata updates don't affect other contracts
func TestContractRegistry_PropertyMetadataIsolation(t *testing.T) {
	f := func(index1, index2 uint32) bool {
		if index1 == index2 {
			return true // Skip if same index
		}

		registry := NewContractRegistry()
		ctx := context.Background()

		contract1 := &RegisteredContract{
			Address: fmt.Sprintf("0x%040x", index1),
			Name:    fmt.Sprintf("Token%d", index1),
			ChainID: "31337",
		}

		contract2 := &RegisteredContract{
			Address: fmt.Sprintf("0x%040x", index2),
			Name:    fmt.Sprintf("Token%d", index2),
			ChainID: "31337",
		}

		// Register both
		if err := registry.Register(ctx, contract1); err != nil {
			t.Logf("Register contract1 failed: %v", err)
			return false
		}
		if err := registry.Register(ctx, contract2); err != nil {
			t.Logf("Register contract2 failed: %v", err)
			return false
		}

		// Update metadata for contract1
		metadata := map[string]any{
			"version": "1.0",
		}
		if err := registry.UpdateMetadata(ctx, contract1.Address, metadata); err != nil {
			t.Logf("UpdateMetadata failed: %v", err)
			return false
		}

		// Verify contract1 has metadata
		c1, err := registry.Get(ctx, contract1.Address)
		if err != nil {
			t.Logf("Get contract1 failed: %v", err)
			return false
		}
		if c1.Metadata["version"] != "1.0" {
			t.Logf("Contract1 metadata not updated")
			return false
		}

		// Verify contract2 doesn't have metadata
		c2, err := registry.Get(ctx, contract2.Address)
		if err != nil {
			t.Logf("Get contract2 failed: %v", err)
			return false
		}
		if len(c2.Metadata) > 0 {
			t.Logf("Contract2 metadata should be empty")
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestContractRegistry_PropertyChainIsolation tests that contracts are properly isolated by chain
func TestContractRegistry_PropertyChainIsolation(t *testing.T) {
	f := func(index uint32) bool {
		registry := NewContractRegistry()
		ctx := context.Background()

		contract1 := &RegisteredContract{
			Address: fmt.Sprintf("0x%040x", index),
			Name:    fmt.Sprintf("Token%d", index),
			ChainID: "31337",
		}

		contract2 := &RegisteredContract{
			Address: fmt.Sprintf("0x%040x", index+1000),
			Name:    fmt.Sprintf("Token%d", index+1000),
			ChainID: "1",
		}

		// Register both
		if err := registry.Register(ctx, contract1); err != nil {
			t.Logf("Register contract1 failed: %v", err)
			return false
		}
		if err := registry.Register(ctx, contract2); err != nil {
			t.Logf("Register contract2 failed: %v", err)
			return false
		}

		// List by chain 31337
		list31337, err := registry.ListByChain(ctx, "31337")
		if err != nil {
			t.Logf("ListByChain 31337 failed: %v", err)
			return false
		}

		// List by chain 1
		list1, err := registry.ListByChain(ctx, "1")
		if err != nil {
			t.Logf("ListByChain 1 failed: %v", err)
			return false
		}

		// Verify isolation
		if len(list31337) != 1 {
			t.Logf("Expected 1 contract for chain 31337, got %d", len(list31337))
			return false
		}
		if len(list1) != 1 {
			t.Logf("Expected 1 contract for chain 1, got %d", len(list1))
			return false
		}

		// Verify correct contracts
		if list31337[0].Address != contract1.Address {
			t.Logf("Wrong contract for chain 31337")
			return false
		}
		if list1[0].Address != contract2.Address {
			t.Logf("Wrong contract for chain 1")
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestContractRegistry_PropertyStatsAccuracy tests that stats are accurate
func TestContractRegistry_PropertyStatsAccuracy(t *testing.T) {
	f := func(indices []uint32) bool {
		if len(indices) == 0 || len(indices) > 50 {
			return true // Skip edge cases
		}

		registry := NewContractRegistry()
		ctx := context.Background()

		chainCounts := make(map[string]int)

		// Register contracts
		for i := range indices {
			chainID := "31337"
			if i%2 == 0 {
				chainID = "1"
			}

			contract := &RegisteredContract{
				Address: fmt.Sprintf("0x%040x", i),
				Name:    fmt.Sprintf("Token%d", i),
				ChainID: chainID,
			}

			if err := registry.Register(ctx, contract); err != nil {
				t.Logf("Register failed: %v", err)
				return false
			}

			chainCounts[chainID]++
		}

		// Get stats
		stats := registry.GetStats()

		// Verify total count
		if stats.TotalContracts != len(indices) {
			t.Logf("Total contracts mismatch: expected %d, got %d", len(indices), stats.TotalContracts)
			return false
		}

		// Verify chain counts
		for chainID, expectedCount := range chainCounts {
			if stats.ContractsByChain[chainID] != expectedCount {
				t.Logf("Chain %s count mismatch: expected %d, got %d", chainID, expectedCount, stats.ContractsByChain[chainID])
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestContractRegistry_PropertyGetByNameConsistency tests that GetByName returns same as Get
func TestContractRegistry_PropertyGetByNameConsistency(t *testing.T) {
	f := func(index uint32) bool {
		registry := NewContractRegistry()
		ctx := context.Background()

		contract := &RegisteredContract{
			Address: fmt.Sprintf("0x%040x", index),
			Name:    fmt.Sprintf("Token%d", index),
			ChainID: "31337",
		}

		// Register
		if err := registry.Register(ctx, contract); err != nil {
			t.Logf("Register failed: %v", err)
			return false
		}

		// Get by address
		byAddr, err := registry.Get(ctx, contract.Address)
		if err != nil {
			t.Logf("Get by address failed: %v", err)
			return false
		}

		// Get by name
		byName, err := registry.GetByName(ctx, contract.Name)
		if err != nil {
			t.Logf("Get by name failed: %v", err)
			return false
		}

		// Verify they're the same
		if byAddr.Address != byName.Address {
			t.Logf("Address mismatch: %s vs %s", byAddr.Address, byName.Address)
			return false
		}
		if byAddr.Name != byName.Name {
			t.Logf("Name mismatch: %s vs %s", byAddr.Name, byName.Name)
			return false
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
