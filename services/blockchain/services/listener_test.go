package blockchain

import (
	"context"
	"math/big"
	"testing"
	"time"

	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestNewEventListener(t *testing.T) {
	listener := NewEventListener()
	
	if listener == nil {
		t.Error("Expected EventListener instance, got nil")
	}
	
	if listener.contractAddresses == nil {
		t.Error("Expected contractAddresses to be initialized")
	}
	
	if listener.eventProcessor == nil {
		t.Error("Expected eventProcessor to be initialized")
	}
}

func TestEventListener_AddContractAddress(t *testing.T) {
	listener := NewEventListener()
	
	addr := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	listener.AddContractAddress(addr)
	
	if len(listener.contractAddresses) != 1 {
		t.Errorf("Expected 1 contract address, got %d", len(listener.contractAddresses))
	}
	
	if listener.contractAddresses[0] != addr {
		t.Errorf("Expected address %s, got %s", addr.Hex(), listener.contractAddresses[0].Hex())
	}
}

func TestEventListener_AddEventFilter(t *testing.T) {
	listener := NewEventListener()
	
	contractAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	eventSignature := "Transfer(address,address,uint256)"
	listener.AddEventFilter(contractAddr, eventSignature)
	
	if len(listener.eventProcessor.eventFilters) != 1 {
		t.Errorf("Expected 1 event filter, got %d", len(listener.eventProcessor.eventFilters))
	}
	
	filter, exists := listener.eventProcessor.eventFilters[contractAddr]
	if !exists {
		t.Error("Expected event filter to exist for contract address")
		return
	}
	
	if len(filter) != 1 {
		t.Errorf("Expected 1 filter for contract, got %d", len(filter))
	}
	
	if filter[0] != eventSignature {
		t.Errorf("Expected event signature %s, got %s", eventSignature, filter[0])
	}
}

func TestEventListener_ListenForEvents(t *testing.T) {
	listener := NewEventListener()
	
	// Add a contract to listen to
	addr := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	listener.AddContractAddress(addr)
	
	// Create a mock client
	mockClient := &MockEthClient{}
	
	// Create a context with timeout to avoid hanging
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Create channels for events and errors
	events := make(chan *types.IndexedEvent, 10)
	errors := make(chan error, 10)
	
	// Start listening (this will return immediately in our mock implementation)
	listener.ListenForEvents(ctx, mockClient, events, errors)
	
	// Give some time for potential goroutines to start
	time.Sleep(100 * time.Millisecond)
	
	// At this point, we just want to ensure the function doesn't panic
	// The actual event listening would require a real Ethereum node connection
	select {
	case <-ctx.Done():
		// Expected behavior in mock
	default:
		// Function should have returned or be waiting
	}
}

func TestEventListener_FetchHistoricalEvents(t *testing.T) {
	listener := NewEventListener()
	
	// Add a contract to listen to
	addr := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	listener.AddContractAddress(addr)
	
	// Create a mock client
	mockClient := &MockEthClient{}
	
	fromBlock := big.NewInt(1000)
	toBlock := big.NewInt(1010)
	
	// Create channels for events and errors
	events := make(chan *types.IndexedEvent, 10)
	errors := make(chan error, 10)
	
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Start fetching historical events
	listener.FetchHistoricalEvents(ctx, mockClient, fromBlock, toBlock, events, errors)
	
	// Give some time for potential goroutines to start
	time.Sleep(100 * time.Millisecond)
	
	// At this point, we just want to ensure the function doesn't panic
	select {
	case <-ctx.Done():
		// Expected behavior in mock
	default:
		// Function should have returned or be waiting
	}
}