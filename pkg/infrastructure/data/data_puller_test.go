package data

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBlockchainTypeConstants tests blockchain type constants
func TestBlockchainTypeConstants(t *testing.T) {
	assert.Equal(t, BlockchainType("evm"), EVM)
	assert.Equal(t, BlockchainType("cosmos"), Cosmos)
	assert.Equal(t, BlockchainType("solana"), Solana)
}

// TestBlockchainEventCreation tests blockchain event creation
func TestBlockchainEventCreation(t *testing.T) {
	event := &BlockchainEvent{
		ID:              "event-1",
		EventHash:       "0x123",
		BlockNumber:     100,
		TransactionHash: "0x456",
		LogIndex:        0,
		ContractAddress: "0xabc",
		EventName:       "Transfer",
		EventData:       map[string]interface{}{"from": "0x123", "to": "0x456"},
		ChainID:         "ethereum",
		Timestamp:       time.Now(),
		ProcessedAt:     time.Now(),
		Status:          "confirmed",
	}

	assert.Equal(t, "event-1", event.ID)
	assert.Equal(t, uint64(100), event.BlockNumber)
	assert.Equal(t, "Transfer", event.EventName)
	assert.Equal(t, "confirmed", event.Status)
}

// TestDataPullerConfigCreation tests data puller config creation
func TestDataPullerConfigCreation(t *testing.T) {
	config := DataPullerConfig{
		ChainType:      EVM,
		ChainID:        "ethereum",
		BlockchainNode: "http://localhost:8545",
		StartBlock:     0,
		BatchSize:      100,
		PollInterval:   5 * time.Second,
	}

	assert.Equal(t, EVM, config.ChainType)
	assert.Equal(t, "ethereum", config.ChainID)
	assert.Equal(t, uint64(100), config.BatchSize)
	assert.Equal(t, 5*time.Second, config.PollInterval)
}

// TestDataPullerCreation tests data puller creation
func TestDataPullerCreation(t *testing.T) {
	config := DataPullerConfig{
		ChainType:      EVM,
		ChainID:        "ethereum",
		BlockchainNode: "http://localhost:8545",
		StartBlock:     0,
		BatchSize:      100,
		PollInterval:   5 * time.Second,
	}

	puller := &DataPuller{
		config:       config,
		currentBlock: 0,
	}

	assert.NotNil(t, puller)
	assert.Equal(t, config, puller.config)
	assert.Equal(t, uint64(0), puller.currentBlock)
}

// TestBlockchainEventWithEmptyData tests blockchain event with empty data
func TestBlockchainEventWithEmptyData(t *testing.T) {
	event := &BlockchainEvent{
		ID:        "event-1",
		EventName: "Transfer",
		EventData: map[string]interface{}{},
	}

	assert.Equal(t, 0, len(event.EventData))
}

// TestBlockchainEventWithMultipleData tests blockchain event with multiple data
func TestBlockchainEventWithMultipleData(t *testing.T) {
	event := &BlockchainEvent{
		ID:        "event-1",
		EventName: "Transfer",
		EventData: map[string]interface{}{
			"from":   "0x123",
			"to":     "0x456",
			"value":  "1000",
			"index":  5,
			"status": "confirmed",
		},
	}

	assert.Equal(t, 5, len(event.EventData))
	assert.Equal(t, "0x123", event.EventData["from"])
	assert.Equal(t, "0x456", event.EventData["to"])
	assert.Equal(t, "1000", event.EventData["value"])
	assert.Equal(t, 5, event.EventData["index"])
	assert.Equal(t, "confirmed", event.EventData["status"])
}

// TestDataPullerConfigWithDifferentChainTypes tests config with different chain types
func TestDataPullerConfigWithDifferentChainTypes(t *testing.T) {
	chainTypes := []BlockchainType{EVM, Cosmos, Solana}

	for _, chainType := range chainTypes {
		config := DataPullerConfig{
			ChainType: chainType,
			ChainID:   "test-chain",
		}
		assert.Equal(t, chainType, config.ChainType)
	}
}

// TestDataPullerConfigWithDifferentBatchSizes tests config with different batch sizes
func TestDataPullerConfigWithDifferentBatchSizes(t *testing.T) {
	batchSizes := []uint64{10, 100, 1000, 10000}

	for _, batchSize := range batchSizes {
		config := DataPullerConfig{
			ChainType: EVM,
			BatchSize: batchSize,
		}
		assert.Equal(t, batchSize, config.BatchSize)
	}
}

// TestDataPullerConfigWithDifferentPollIntervals tests config with different poll intervals
func TestDataPullerConfigWithDifferentPollIntervals(t *testing.T) {
	intervals := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		10 * time.Second,
		1 * time.Minute,
	}

	for _, interval := range intervals {
		config := DataPullerConfig{
			ChainType:    EVM,
			PollInterval: interval,
		}
		assert.Equal(t, interval, config.PollInterval)
	}
}

// TestBlockchainEventTimestamps tests blockchain event timestamps
func TestBlockchainEventTimestamps(t *testing.T) {
	now := time.Now()
	event := &BlockchainEvent{
		ID:          "event-1",
		Timestamp:   now,
		ProcessedAt: now.Add(1 * time.Second),
	}

	assert.True(t, event.ProcessedAt.After(event.Timestamp))
}

// TestBlockchainEventBlockNumber tests blockchain event block number
func TestBlockchainEventBlockNumber(t *testing.T) {
	blockNumbers := []uint64{0, 1, 100, 1000, 10000000}

	for _, blockNum := range blockNumbers {
		event := &BlockchainEvent{
			BlockNumber: blockNum,
		}
		assert.Equal(t, blockNum, event.BlockNumber)
	}
}

// TestBlockchainEventLogIndex tests blockchain event log index
func TestBlockchainEventLogIndex(t *testing.T) {
	logIndices := []uint{0, 1, 5, 10, 100}

	for _, logIndex := range logIndices {
		event := &BlockchainEvent{
			LogIndex: logIndex,
		}
		assert.Equal(t, logIndex, event.LogIndex)
	}
}

// TestBlockchainEventStatus tests blockchain event status
func TestBlockchainEventStatus(t *testing.T) {
	statuses := []string{"pending", "confirmed", "failed", "reorged"}

	for _, status := range statuses {
		event := &BlockchainEvent{
			Status: status,
		}
		assert.Equal(t, status, event.Status)
	}
}

// TestDataPullerConfigStartBlock tests data puller config start block
func TestDataPullerConfigStartBlock(t *testing.T) {
	startBlocks := []uint64{0, 1, 100, 1000, 10000000}

	for _, startBlock := range startBlocks {
		config := DataPullerConfig{
			ChainType:  EVM,
			StartBlock: startBlock,
		}
		assert.Equal(t, startBlock, config.StartBlock)
	}
}

// TestBlockchainEventChainID tests blockchain event chain ID
func TestBlockchainEventChainID(t *testing.T) {
	chainIDs := []string{"ethereum", "polygon", "arbitrum", "optimism"}

	for _, chainID := range chainIDs {
		event := &BlockchainEvent{
			ChainID: chainID,
		}
		assert.Equal(t, chainID, event.ChainID)
	}
}

// TestDataPullerConfigChainID tests data puller config chain ID
func TestDataPullerConfigChainID(t *testing.T) {
	chainIDs := []string{"ethereum", "polygon", "arbitrum", "optimism"}

	for _, chainID := range chainIDs {
		config := DataPullerConfig{
			ChainType: EVM,
			ChainID:   chainID,
		}
		assert.Equal(t, chainID, config.ChainID)
	}
}

// TestBlockchainEventEventName tests blockchain event event name
func TestBlockchainEventEventName(t *testing.T) {
	eventNames := []string{"Transfer", "Approval", "Swap", "Mint", "Burn"}

	for _, eventName := range eventNames {
		event := &BlockchainEvent{
			EventName: eventName,
		}
		assert.Equal(t, eventName, event.EventName)
	}
}

// TestBlockchainEventAddresses tests blockchain event addresses
func TestBlockchainEventAddresses(t *testing.T) {
	event := &BlockchainEvent{
		ContractAddress: "0x1234567890123456789012345678901234567890",
	}

	assert.Equal(t, "0x1234567890123456789012345678901234567890", event.ContractAddress)
}

// TestBlockchainEventHashes tests blockchain event hashes
func TestBlockchainEventHashes(t *testing.T) {
	event := &BlockchainEvent{
		EventHash:       "0xevent123",
		TransactionHash: "0xtx456",
	}

	assert.Equal(t, "0xevent123", event.EventHash)
	assert.Equal(t, "0xtx456", event.TransactionHash)
}

// TestDataPullerConfigBlockchainNode tests data puller config blockchain node
func TestDataPullerConfigBlockchainNode(t *testing.T) {
	nodes := []string{
		"http://localhost:8545",
		"https://mainnet.infura.io/v3/YOUR-PROJECT-ID",
		"https://eth-mainnet.g.alchemy.com/v2/YOUR-API-KEY",
	}

	for _, node := range nodes {
		config := DataPullerConfig{
			ChainType:      EVM,
			BlockchainNode: node,
		}
		assert.Equal(t, node, config.BlockchainNode)
	}
}

// TestBlockchainEventID tests blockchain event ID
func TestBlockchainEventID(t *testing.T) {
	event := &BlockchainEvent{
		ID: "event-12345",
	}

	assert.Equal(t, "event-12345", event.ID)
}

// TestDataPullerCurrentBlock tests data puller current block tracking
func TestDataPullerCurrentBlock(t *testing.T) {
	config := DataPullerConfig{
		ChainType:  EVM,
		StartBlock: 100,
	}

	puller := &DataPuller{
		config:       config,
		currentBlock: 100,
	}

	assert.Equal(t, uint64(100), puller.currentBlock)
}

// TestBlockchainEventComparison tests blockchain event comparison
func TestBlockchainEventComparison(t *testing.T) {
	event1 := &BlockchainEvent{
		ID:          "event-1",
		BlockNumber: 100,
		EventName:   "Transfer",
	}

	event2 := &BlockchainEvent{
		ID:          "event-1",
		BlockNumber: 100,
		EventName:   "Transfer",
	}

	assert.Equal(t, event1.ID, event2.ID)
	assert.Equal(t, event1.BlockNumber, event2.BlockNumber)
	assert.Equal(t, event1.EventName, event2.EventName)
}

// TestDataPullerConfigComparison tests data puller config comparison
func TestDataPullerConfigComparison(t *testing.T) {
	config1 := DataPullerConfig{
		ChainType:    EVM,
		ChainID:      "ethereum",
		BatchSize:    100,
		PollInterval: 5 * time.Second,
	}

	config2 := DataPullerConfig{
		ChainType:    EVM,
		ChainID:      "ethereum",
		BatchSize:    100,
		PollInterval: 5 * time.Second,
	}

	assert.Equal(t, config1.ChainType, config2.ChainType)
	assert.Equal(t, config1.ChainID, config2.ChainID)
	assert.Equal(t, config1.BatchSize, config2.BatchSize)
	assert.Equal(t, config1.PollInterval, config2.PollInterval)
}
