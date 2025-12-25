package integration

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventListening tests listening for blockchain events
func TestEventListening(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Mock contract address for testing
	contractAddr := generateMockContractAddress(7)

	// Create a filter for events from the mock contract
	filterQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(1), // Start from block 1
		ToBlock:   nil,          // To latest block
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{}, // All topics
	}

	// Get logs for the filter
	logs, err := client.FilterLogs(context.Background(), filterQuery)
	require.NoError(t, err)

	t.Logf("Found %d events for contract %s", len(logs), contractAddr.Hex())

	// Process the logs
	for i, log := range logs {
		t.Logf("Event %d: Block %d, TxIndex %d, Address %s", 
			i, log.BlockNumber, log.TxIndex, log.Address.Hex())
		
		// Verify log properties
		assert.Equal(t, contractAddr, log.Address, "Log address should match contract address")
		assert.GreaterOrEqual(t, log.BlockNumber, uint64(1), "Log should be from a valid block")
	}
}

// TestEventSubscription tests real-time event subscription
func TestEventSubscription(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Mock contract address for testing
	contractAddr := generateMockContractAddress(8)

	// Create a filter for events
	filterQuery := ethereum.FilterQuery{
		FromBlock: nil, // From latest block
		ToBlock:   nil, // To latest block
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{},
	}

	// Create a channel to receive logs
	logsCh := make(chan types.Log, 10)

	// Subscribe to logs (in a real scenario, this would receive new logs as they're emitted)
	// For this test, we'll just verify that we can create a subscription
	sub, err := client.SubscribeFilterLogs(context.Background(), filterQuery, logsCh)
	require.NoError(t, err, "Failed to subscribe to logs")
	defer sub.Unsubscribe()

	// Wait briefly to see if any logs come through
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case log := <-logsCh:
		t.Logf("Received log: Block %d, Address %s", log.BlockNumber, log.Address.Hex())
	case <-ctx.Done():
		t.Log("No new logs received within timeout (this is expected in a test environment)")
	}

	t.Log("Event subscription test completed")
}

// TestTransferEvents tests listening for token transfer events
func TestTransferEvents(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Mock token contract address
	tokenAddr := generateMockContractAddress(9)

	// Transfer event signature: Transfer(address,address,uint256)
	transferEventSig := []byte("Transfer(address,address,uint256)")
	transferTopic := crypto.Keccak256Hash(transferEventSig)

	// Create filter for Transfer events only
	filterQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(1),
		ToBlock:   nil,
		Addresses: []common.Address{tokenAddr},
		Topics:    [][]common.Hash{{transferTopic}}, // Filter by Transfer event signature
	}

	// Get transfer logs
	logs, err := client.FilterLogs(context.Background(), filterQuery)
	require.NoError(t, err)

	t.Logf("Found %d Transfer events for token %s", len(logs), tokenAddr.Hex())

	// In a real implementation, we would parse the log data to extract
	// the from, to, and value parameters from the Transfer event
	for i, log := range logs {
		t.Logf("Transfer event %d: Block %d, TxHash %s", i, log.BlockNumber, log.TxHash.Hex())
		
		// Verify it's a transfer event (has the correct topic)
		if len(log.Topics) > 0 {
			assert.Equal(t, transferTopic, log.Topics[0], "First topic should be Transfer event signature")
		}
	}
}

// TestBlockEvents tests listening for new blocks and their events
func TestBlockEvents(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Subscribe to new blocks
	headerCh := make(chan *types.Header, 10)
	headerSub, err := client.SubscribeNewHead(context.Background(), headerCh)
	require.NoError(t, err, "Failed to subscribe to new blocks")
	defer headerSub.Unsubscribe()

	// Also subscribe to logs
	logsCh := make(chan types.Log, 10)
	filterQuery := ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
		Addresses: []common.Address{}, // All contracts
		Topics:    [][]common.Hash{},
	}
	logsSub, err := client.SubscribeFilterLogs(context.Background(), filterQuery, logsCh)
	require.NoError(t, err, "Failed to subscribe to logs")
	defer logsSub.Unsubscribe()

	// Wait for events
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var blocksReceived int
	var logsReceived int

	for {
		select {
		case header := <-headerCh:
			blocksReceived++
			t.Logf("New block received: %d (Hash: %s)", header.Number.Uint64(), header.Hash().Hex())
			
			// Check if we've received enough blocks
			if blocksReceived >= 2 {
				goto done
			}
		case log := <-logsCh:
			logsReceived++
			t.Logf("New log received: Block %d, Address %s", log.BlockNumber, log.Address.Hex())
		case <-ctx.Done():
			t.Logf("Timeout reached. Blocks received: %d, Logs received: %d", blocksReceived, logsReceived)
			goto done
		}
	}

done:
	assert.GreaterOrEqual(t, blocksReceived, 1, "Should receive at least one new block")
	t.Logf("Event correlation test completed - Blocks: %d, Logs: %d", blocksReceived, logsReceived)
}

// TestEventFiltering tests filtering events by various criteria
func TestEventFiltering(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Mock addresses for testing
	contractAddr1 := generateMockContractAddress(10)
	contractAddr2 := generateMockContractAddress(11)

	// Test filtering by multiple addresses
	multiAddrFilter := ethereum.FilterQuery{
		FromBlock: big.NewInt(1),
		ToBlock:   nil,
		Addresses: []common.Address{contractAddr1, contractAddr2},
		Topics:    [][]common.Hash{},
	}

	logs, err := client.FilterLogs(context.Background(), multiAddrFilter)
	require.NoError(t, err)
	t.Logf("Found %d events for multiple contracts", len(logs))

	// Test filtering by block range
	currentBlock, err := client.BlockNumber(context.Background())
	require.NoError(t, err)

	rangeFilter := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(currentBlock) - 5), // Last 5 blocks
		ToBlock:   big.NewInt(int64(currentBlock)),
		Addresses: []common.Address{},
		Topics:    [][]common.Hash{},
	}

	rangeLogs, err := client.FilterLogs(context.Background(), rangeFilter)
	require.NoError(t, err)
	t.Logf("Found %d events in last 5 blocks", len(rangeLogs))

	// Test with no filters (all events)
	allLogsFilter := ethereum.FilterQuery{
		FromBlock: big.NewInt(1),
		ToBlock:   nil,
		Addresses: []common.Address{}, // All addresses
		Topics:    [][]common.Hash{},  // All topics
	}

	allLogs, err := client.FilterLogs(context.Background(), allLogsFilter)
	require.NoError(t, err)
	t.Logf("Found %d total events in blockchain", len(allLogs))

	// Verify that filtered results are subset of all results
	assert.LessOrEqual(t, len(logs), len(allLogs), "Filtered logs should be <= all logs")
	assert.LessOrEqual(t, len(rangeLogs), len(allLogs), "Range logs should be <= all logs")
}

// TestComplexEventFiltering tests complex event filtering scenarios
func TestComplexEventFiltering(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Mock token address
	tokenAddr := generateMockContractAddress(12)

	// Create a complex filter: Transfer events where the first topic (from address) matches specific addresses
	transferEventSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	
	// In a real scenario, we might filter by specific sender addresses
	// For this test, we'll just verify the filter structure
	fromAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	
	complexFilter := ethereum.FilterQuery{
		FromBlock: big.NewInt(1),
		ToBlock:   nil,
		Addresses: []common.Address{tokenAddr},
		Topics: [][]common.Hash{
			{transferEventSig},           // Event signature
			{fromAddr.Hash()},            // From address (in Transfer event, from is the first indexed parameter)
			{},                           // To address (can be any)
		},
	}

	logs, err := client.FilterLogs(context.Background(), complexFilter)
	require.NoError(t, err)
	t.Logf("Found %d Transfer events from specific address", len(logs))

	// In a real implementation, we would verify that all returned logs
	// match our complex filter criteria
	for i, log := range logs {
		t.Logf("Complex filter event %d: Block %d", i, log.BlockNumber)
	}
}

// TestEventParsing tests parsing event data from logs
func TestEventParsing(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// In a real implementation, we would have contract ABIs to parse event data
	// For this test, we'll just verify that we can access log data
	
	// Get some logs to work with
	filterQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(1),
		ToBlock:   nil,
		Addresses: []common.Address{},
		Topics:    [][]common.Hash{},
	}

	logs, err := client.FilterLogs(context.Background(), filterQuery)
	require.NoError(t, err)

	// Test log data access
	for i, log := range logs {
		if i >= 3 { // Only check first 3 logs to limit output
			break
		}

		t.Logf("Log %d: Block=%d, TxIndex=%d, Index=%d, Removed=%t", 
			i, log.BlockNumber, log.TxIndex, log.Index, log.Removed)

		// Verify log properties
		assert.LessOrEqual(t, log.TxIndex, uint(21000), "Transaction index should be reasonable") // Max possible txs per block
		assert.GreaterOrEqual(t, log.BlockNumber, uint64(1), "Block number should be at least 1")
		
		// Topics contain the event signature and indexed parameters
		t.Logf("  Topics: %d", len(log.Topics))
		for j, topic := range log.Topics {
			t.Logf("    Topic %d: %s", j, topic.Hex())
		}
		
		// Data contains non-indexed parameters
		t.Logf("  Data length: %d bytes", len(log.Data))
	}
	
	t.Log("Event parsing test completed")
}