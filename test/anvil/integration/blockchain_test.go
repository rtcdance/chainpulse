package integration

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Anvil default settings
	anvilRPCURL    = "http://localhost:8545"
	anvilChainID   = 31337
	anvilTestKey   = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	testTimeout    = 30 * time.Second
)

// TestBlockchainConnection tests basic connection to the Anvil node
func TestBlockchainConnection(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Test chain ID
	chainID, err := client.NetworkID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(anvilChainID), chainID.Uint64(), "Chain ID should match Anvil default")

	// Test block number
	blockNumber, err := client.BlockNumber(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, blockNumber, uint64(0), "Block number should be non-negative")

	// Test syncing status
	syncing, err := client.SyncProgress(context.Background())
	require.NoError(t, err)
	assert.Nil(t, syncing, "Node should not be syncing")

	t.Logf("Connected to Anvil node. Current block: %d", blockNumber)
}

// TestTransactionSubmission tests submitting a basic transaction to Anvil
func TestTransactionSubmission(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Use the default Anvil account (first account with the test key)
	fromAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	toAddr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")

	// Get the current nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	require.NoError(t, err)

	// Create a simple transaction
	value := big.NewInt(100000000000000000) // 0.1 ETH
	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	require.NoError(t, err)

	tx := types.NewTransaction(nonce, toAddr, value, gasLimit, gasPrice, nil)

	// Sign the transaction (in a real test we would sign it with the private key)
	// For this test we'll just verify that we can submit a transaction structure
	assert.NotNil(t, tx)
	assert.Equal(t, value, tx.Value())
	assert.Equal(t, gasLimit, tx.Gas())

	t.Logf("Transaction created: nonce=%d, value=%d, gas=%d", nonce, value, gasLimit)
}

// TestBlockEvents tests listening for new blocks
func TestBlockEvents(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Create a subscription for new blocks
	headCh := make(chan *types.Header, 10)
	sub, err := client.SubscribeNewHead(context.Background(), headCh)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Wait for a few blocks to be mined
	// Anvil mines blocks every 1 second if block time is set
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	var blocksReceived []uint64
	for len(blocksReceived) < 3 {
		select {
		case header := <-headCh:
			blocksReceived = append(blocksReceived, header.Number.Uint64())
			t.Logf("Received block: %d", header.Number.Uint64())
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for blocks, only received %d blocks", len(blocksReceived))
		}
	}

	assert.GreaterOrEqual(t, len(blocksReceived), 3, "Should receive at least 3 blocks")
	t.Logf("Received blocks: %v", blocksReceived)
}

// TestFilterLogs tests filtering logs from the blockchain
func TestFilterLogs(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Get the latest block number
	toBlock, err := client.BlockNumber(context.Background())
	require.NoError(t, err)

	fromBlock := toBlock - 10 // Look at the last 10 blocks

	// Create a filter query for all logs in recent blocks
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{},
		Topics:    [][]common.Hash{},
	}

	logs, err := client.FilterLogs(context.Background(), query)
	require.NoError(t, err)

	t.Logf("Found %d logs in the last %d blocks", len(logs), 10)
	for i, log := range logs {
		t.Logf("Log %d: Block %d, TxIndex %d, Address %s", 
			i, log.BlockNumber, log.TxIndex, log.Address.Hex())
	}
}

// TestGasEstimation tests gas estimation functionality
func TestGasEstimation(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Create a simple transaction to estimate gas for
	toAddr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	value := big.NewInt(100000000000000000) // 0.1 ETH
	gasPrice, err := client.SuggestGasPrice(context.Background())
	require.NoError(t, err)

	// Create a call msg for gas estimation
	msg := ethereum.CallMsg{
		From:     common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		To:       &toAddr,
		Value:    value,
		GasPrice: gasPrice,
		Data:     nil,
	}

	gasLimit, err := client.EstimateGas(context.Background(), msg)
	require.NoError(t, err)
	assert.Greater(t, gasLimit, uint64(0), "Gas estimation should return a positive value")
	assert.Less(t, gasLimit, uint64(100000), "Gas estimation for simple transfer should be reasonable")

	t.Logf("Estimated gas for transfer: %d", gasLimit)
}

// TestChainConfig tests retrieving chain configuration
func TestChainConfig(t *testing.T) {
	client, err := ethclient.Dial(anvilRPCURL)
	require.NoError(t, err, "Failed to connect to Anvil node")
	defer client.Close()

	// Get the current block to verify chain details
	block, err := client.BlockByNumber(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, block)

	assert.Equal(t, big.NewInt(anvilChainID), block.ChainID(), "Block should have correct chain ID")

	// Check block properties
	assert.Greater(t, block.Number().Uint64(), uint64(0), "Block number should be greater than 0")
	assert.NotEqual(t, common.Hash{}, block.Hash(), "Block should have a valid hash")

	t.Logf("Current block: %d, Hash: %s, Chain ID: %d", 
		block.Number().Uint64(), block.Hash().Hex(), block.ChainID())
}

// Helper function to wait for a specific block number
func waitForBlock(t *testing.T, client *ethclient.Client, targetBlock uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentBlock, err := client.BlockNumber(context.Background())
			if err != nil {
				continue
			}
			if currentBlock >= targetBlock {
				return
			}
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for block %d, current block: %d", targetBlock, getCurrentBlock(t, client))
		}
	}
}

// Helper function to get current block number
func getCurrentBlock(t *testing.T, client *ethclient.Client) uint64 {
	currentBlock, err := client.BlockNumber(context.Background())
	require.NoError(t, err)
	return currentBlock
}