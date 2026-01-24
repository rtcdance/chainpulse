package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestContractRegistry_Register(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract := &RegisteredContract{
		Address:      "0x1234567890123456789012345678901234567890",
		Name:         "TestToken",
		ChainID:      "31337",
		ABI:          `[{"type":"function"}]`,
		Bytecode:     "0xabcd",
		DeploymentTx: "0xdeadbeef",
	}

	err := registry.Register(ctx, contract)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify contract was registered
	retrieved, err := registry.Get(ctx, contract.Address)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != contract.Name {
		t.Errorf("Expected name %s, got %s", contract.Name, retrieved.Name)
	}
}

func TestContractRegistry_RegisterDuplicateAddress(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract1 := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "Token1",
		ChainID: "31337",
	}

	contract2 := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "Token2",
		ChainID: "31337",
	}

	err := registry.Register(ctx, contract1)
	if err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	err = registry.Register(ctx, contract2)
	if err == nil {
		t.Fatal("Expected error for duplicate address, got nil")
	}
}

func TestContractRegistry_RegisterDuplicateName(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract1 := &RegisteredContract{
		Address: "0x1111111111111111111111111111111111111111",
		Name:    "TestToken",
		ChainID: "31337",
	}

	contract2 := &RegisteredContract{
		Address: "0x2222222222222222222222222222222222222222",
		Name:    "TestToken",
		ChainID: "31337",
	}

	err := registry.Register(ctx, contract1)
	if err != nil {
		t.Fatalf("First register failed: %v", err)
	}

	err = registry.Register(ctx, contract2)
	if err == nil {
		t.Fatal("Expected error for duplicate name, got nil")
	}
}

func TestContractRegistry_Unregister(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "TestToken",
		ChainID: "31337",
	}

	err := registry.Register(ctx, contract)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = registry.Unregister(ctx, contract.Address)
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	// Verify contract was removed
	_, err = registry.Get(ctx, contract.Address)
	if err == nil {
		t.Fatal("Expected error for unregistered contract, got nil")
	}
}

func TestContractRegistry_GetByName(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "TestToken",
		ChainID: "31337",
	}

	err := registry.Register(ctx, contract)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	retrieved, err := registry.GetByName(ctx, "TestToken")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}

	if retrieved.Address != contract.Address {
		t.Errorf("Expected address %s, got %s", contract.Address, retrieved.Address)
	}
}

func TestContractRegistry_List(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contracts := []*RegisteredContract{
		{
			Address: "0x1111111111111111111111111111111111111111",
			Name:    "Token1",
			ChainID: "31337",
		},
		{
			Address: "0x2222222222222222222222222222222222222222",
			Name:    "Token2",
			ChainID: "31337",
		},
		{
			Address: "0x3333333333333333333333333333333333333333",
			Name:    "Token3",
			ChainID: "1",
		},
	}

	for _, contract := range contracts {
		err := registry.Register(ctx, contract)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	list, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != len(contracts) {
		t.Errorf("Expected %d contracts, got %d", len(contracts), len(list))
	}
}

func TestContractRegistry_ListByChain(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contracts := []*RegisteredContract{
		{
			Address: "0x1111111111111111111111111111111111111111",
			Name:    "Token1",
			ChainID: "31337",
		},
		{
			Address: "0x2222222222222222222222222222222222222222",
			Name:    "Token2",
			ChainID: "31337",
		},
		{
			Address: "0x3333333333333333333333333333333333333333",
			Name:    "Token3",
			ChainID: "1",
		},
	}

	for _, contract := range contracts {
		err := registry.Register(ctx, contract)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	// List contracts for chain 31337
	list, err := registry.ListByChain(ctx, "31337")
	if err != nil {
		t.Fatalf("ListByChain failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 contracts for chain 31337, got %d", len(list))
	}

	// List contracts for chain 1
	list, err = registry.ListByChain(ctx, "1")
	if err != nil {
		t.Fatalf("ListByChain failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 contract for chain 1, got %d", len(list))
	}
}

func TestContractRegistry_UpdateMetadata(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "TestToken",
		ChainID: "31337",
	}

	err := registry.Register(ctx, contract)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	metadata := map[string]interface{}{
		"version": "1.0",
		"author":  "test",
	}

	err = registry.UpdateMetadata(ctx, contract.Address, metadata)
	if err != nil {
		t.Fatalf("UpdateMetadata failed: %v", err)
	}

	// Verify metadata was updated
	retrieved, err := registry.Get(ctx, contract.Address)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Metadata["version"] != "1.0" {
		t.Errorf("Expected version 1.0, got %v", retrieved.Metadata["version"])
	}
	if retrieved.Metadata["author"] != "test" {
		t.Errorf("Expected author test, got %v", retrieved.Metadata["author"])
	}
}

func TestContractRegistry_GetStats(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contracts := []*RegisteredContract{
		{
			Address: "0x1111111111111111111111111111111111111111",
			Name:    "Token1",
			ChainID: "31337",
		},
		{
			Address: "0x2222222222222222222222222222222222222222",
			Name:    "Token2",
			ChainID: "31337",
		},
		{
			Address: "0x3333333333333333333333333333333333333333",
			Name:    "Token3",
			ChainID: "1",
		},
	}

	for _, contract := range contracts {
		err := registry.Register(ctx, contract)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	stats := registry.GetStats()

	if stats.TotalContracts != 3 {
		t.Errorf("Expected 3 total contracts, got %d", stats.TotalContracts)
	}

	if stats.ContractsByChain["31337"] != 2 {
		t.Errorf("Expected 2 contracts for chain 31337, got %d", stats.ContractsByChain["31337"])
	}

	if stats.ContractsByChain["1"] != 1 {
		t.Errorf("Expected 1 contract for chain 1, got %d", stats.ContractsByChain["1"])
	}

	if stats.LastRegistered.IsZero() {
		t.Error("Expected LastRegistered to be set")
	}
}

func TestContractRegistry_ContextCancellation(t *testing.T) {
	registry := NewContractRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	contract := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "TestToken",
		ChainID: "31337",
	}

	err := registry.Register(ctx, contract)
	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}
}

func TestContractRegistry_ValidationErrors(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	tests := []struct {
		name     string
		contract *RegisteredContract
		wantErr  bool
	}{
		{
			name:     "nil contract",
			contract: nil,
			wantErr:  true,
		},
		{
			name: "empty address",
			contract: &RegisteredContract{
				Name:    "Token",
				ChainID: "31337",
			},
			wantErr: true,
		},
		{
			name: "empty name",
			contract: &RegisteredContract{
				Address: "0x1234567890123456789012345678901234567890",
				ChainID: "31337",
			},
			wantErr: true,
		},
		{
			name: "empty chain ID",
			contract: &RegisteredContract{
				Address: "0x1234567890123456789012345678901234567890",
				Name:    "Token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(ctx, tt.contract)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContractRegistry_ConcurrentOperations(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	// Register contracts concurrently
	done := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			contract := &RegisteredContract{
				Address: fmt.Sprintf("0x%040x", index),
				Name:    fmt.Sprintf("Token%d", index),
				ChainID: "31337",
			}
			done <- registry.Register(ctx, contract)
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Fatalf("Concurrent register failed: %v", err)
		}
	}

	// Verify all contracts were registered
	list, err := registry.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 10 {
		t.Errorf("Expected 10 contracts, got %d", len(list))
	}
}

func TestContractRegistry_DeploymentTimeTracking(t *testing.T) {
	registry := NewContractRegistry()
	ctx := context.Background()

	contract := &RegisteredContract{
		Address: "0x1234567890123456789012345678901234567890",
		Name:    "TestToken",
		ChainID: "31337",
	}

	before := time.Now()
	err := registry.Register(ctx, contract)
	after := time.Now()

	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	retrieved, err := registry.Get(ctx, contract.Address)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.DeployedAt.Before(before) || retrieved.DeployedAt.After(after) {
		t.Errorf("DeployedAt time not within expected range")
	}
}
