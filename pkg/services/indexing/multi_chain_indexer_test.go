package indexing

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/integrations/generic"
)

func TestNewMultiChainIndexer(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}

	indexer := NewMultiChainIndexer(logger, config)

	assert.NotNil(t, indexer)
	assert.Equal(t, 0, indexer.GetIndexerCount())
	assert.False(t, indexer.IsMultiChain())
}

func TestRegisterChainIndexer(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	err := indexer.RegisterChainIndexer("ethereum", chainIndexer)
	require.NoError(t, err)

	assert.Equal(t, 1, indexer.GetIndexerCount())
	assert.True(t, indexer.IsMultiChain() == false) // Only 1 chain
}

func TestRegisterChainIndexerEmptyChainID(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	err := indexer.RegisterChainIndexer("", chainIndexer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chain ID cannot be empty")
}

func TestRegisterChainIndexerNil(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	err := indexer.RegisterChainIndexer("ethereum", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "indexer cannot be nil")
}

func TestRegisterChainIndexerDuplicate(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)

	err := indexer.RegisterChainIndexer("ethereum", chainIndexer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestIndexEventsFromChain(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)

	events := []*blockchain.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	err := indexer.IndexEventsFromChain(context.Background(), "ethereum", events)
	require.NoError(t, err)
}

func TestIndexEventsFromChainNotRegistered(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	events := []*blockchain.BlockchainEvent{
		{
			ID:      "event1",
			ChainID: "ethereum",
		},
	}

	err := indexer.IndexEventsFromChain(context.Background(), "ethereum", events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no indexer registered")
}

func TestIndexEventsFromChainEmptyChainID(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	events := []*blockchain.BlockchainEvent{
		{
			ID: "event1",
		},
	}

	err := indexer.IndexEventsFromChain(context.Background(), "", events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chain ID cannot be empty")
}

func TestIndexEventsFromAllChains(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db1 := NewMockDatabasePlugin()
	cache1 := NewMockCachePlugin()
	genericIndexer1 := generic.NewGenericContractIndexer(db1, cache1, logger, nil, nil)
	chainIndexer1 := NewDefaultChainIndexer("ethereum", db1, cache1, logger, genericIndexer1)

	db2 := NewMockDatabasePlugin()
	cache2 := NewMockCachePlugin()
	genericIndexer2 := generic.NewGenericContractIndexer(db2, cache2, logger, nil, nil)
	chainIndexer2 := NewDefaultChainIndexer("polygon", db2, cache2, logger, genericIndexer2)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer1)
	_ = indexer.RegisterChainIndexer("polygon", chainIndexer2)

	eventsByChain := map[string][]*blockchain.BlockchainEvent{
		"ethereum": {
			{
				ID:              "event1",
				ChainID:         "ethereum",
				BlockNumber:     100,
				TransactionHash: common.HexToHash("0x1234"),
				EventSignature:  common.HexToHash("0x5678"),
			},
		},
		"polygon": {
			{
				ID:              "event2",
				ChainID:         "polygon",
				BlockNumber:     200,
				TransactionHash: common.HexToHash("0x1235"),
				EventSignature:  common.HexToHash("0x5679"),
			},
		},
	}

	err := indexer.IndexEventsFromAllChains(context.Background(), eventsByChain)
	require.NoError(t, err)
}

func TestIndexEventsFromAllChainsUnregisteredChain(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)

	eventsByChain := map[string][]*blockchain.BlockchainEvent{
		"polygon": {
			{
				ID:      "event1",
				ChainID: "polygon",
			},
		},
	}

	err := indexer.IndexEventsFromAllChains(context.Background(), eventsByChain)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no indexer registered")
}

func TestGetChainIndexer(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)

	retrieved, err := indexer.GetChainIndexer("ethereum")
	require.NoError(t, err)

	assert.NotNil(t, retrieved)
	assert.Equal(t, "ethereum", retrieved.GetChainID())
}

func TestGetChainIndexerNotFound(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	_, err := indexer.GetChainIndexer("ethereum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no indexer registered")
}

func TestGetRegisteredChains(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db1 := NewMockDatabasePlugin()
	cache1 := NewMockCachePlugin()
	genericIndexer1 := generic.NewGenericContractIndexer(db1, cache1, logger, nil, nil)
	chainIndexer1 := NewDefaultChainIndexer("ethereum", db1, cache1, logger, genericIndexer1)

	db2 := NewMockDatabasePlugin()
	cache2 := NewMockCachePlugin()
	genericIndexer2 := generic.NewGenericContractIndexer(db2, cache2, logger, nil, nil)
	chainIndexer2 := NewDefaultChainIndexer("polygon", db2, cache2, logger, genericIndexer2)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer1)
	_ = indexer.RegisterChainIndexer("polygon", chainIndexer2)

	chains := indexer.GetRegisteredChains()

	assert.Equal(t, 2, len(chains))
	assert.Contains(t, chains, "ethereum")
	assert.Contains(t, chains, "polygon")
}

func TestMultiChainGetStatus(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)

	status := indexer.GetStatus()

	assert.NotNil(t, status)
	assert.Contains(t, status, "ethereum")
	assert.NotNil(t, status["ethereum"])
}

func TestMultiChainClose(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)

	err := indexer.Close()
	require.NoError(t, err)
}

func TestIsMultiChain(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	assert.False(t, indexer.IsMultiChain())

	db1 := NewMockDatabasePlugin()
	cache1 := NewMockCachePlugin()
	genericIndexer1 := generic.NewGenericContractIndexer(db1, cache1, logger, nil, nil)
	chainIndexer1 := NewDefaultChainIndexer("ethereum", db1, cache1, logger, genericIndexer1)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer1)
	assert.False(t, indexer.IsMultiChain())

	db2 := NewMockDatabasePlugin()
	cache2 := NewMockCachePlugin()
	genericIndexer2 := generic.NewGenericContractIndexer(db2, cache2, logger, nil, nil)
	chainIndexer2 := NewDefaultChainIndexer("polygon", db2, cache2, logger, genericIndexer2)

	_ = indexer.RegisterChainIndexer("polygon", chainIndexer2)
	assert.True(t, indexer.IsMultiChain())
}

func TestGetIndexerCount(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	assert.Equal(t, 0, indexer.GetIndexerCount())

	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	chainIndexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer)
	assert.Equal(t, 1, indexer.GetIndexerCount())
}

func TestMultipleChainIndexing(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	db1 := NewMockDatabasePlugin()
	cache1 := NewMockCachePlugin()
	genericIndexer1 := generic.NewGenericContractIndexer(db1, cache1, logger, nil, nil)
	chainIndexer1 := NewDefaultChainIndexer("ethereum", db1, cache1, logger, genericIndexer1)

	db2 := NewMockDatabasePlugin()
	cache2 := NewMockCachePlugin()
	genericIndexer2 := generic.NewGenericContractIndexer(db2, cache2, logger, nil, nil)
	chainIndexer2 := NewDefaultChainIndexer("polygon", db2, cache2, logger, genericIndexer2)

	_ = indexer.RegisterChainIndexer("ethereum", chainIndexer1)
	_ = indexer.RegisterChainIndexer("polygon", chainIndexer2)

	// Index events from both chains
	ethEvents := []*blockchain.BlockchainEvent{
		{
			ID:              "eth1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	polyEvents := []*blockchain.BlockchainEvent{
		{
			ID:              "poly1",
			ChainID:         "polygon",
			BlockNumber:     200,
			TransactionHash: common.HexToHash("0x1235"),
			EventSignature:  common.HexToHash("0x5679"),
		},
	}

	_ = indexer.IndexEventsFromChain(context.Background(), "ethereum", ethEvents)
	_ = indexer.IndexEventsFromChain(context.Background(), "polygon", polyEvents)

	assert.Equal(t, 2, indexer.GetIndexerCount())
	assert.True(t, indexer.IsMultiChain())
}

// MockConfigManager for testing
type MockConfigManager struct{}

func (mcm *MockConfigManager) Load() (core.Config, error) {
	return core.Config{}, nil
}

func (mcm *MockConfigManager) Validate(config core.Config) error {
	return nil
}

func (mcm *MockConfigManager) Get(key string) (any, error) {
	return nil, nil
}

func (mcm *MockConfigManager) Set(key string, value any) error {
	return nil
}

func (mcm *MockConfigManager) GetConfig(key string) (any, error) {
	return nil, nil
}

func (mcm *MockConfigManager) SetConfig(key string, value any) error {
	return nil
}

func (mcm *MockConfigManager) GetString(key string) string {
	return ""
}

func (mcm *MockConfigManager) GetInt(key string) int {
	return 0
}

func (mcm *MockConfigManager) GetBool(key string) bool {
	return false
}

func (mcm *MockConfigManager) Close() error {
	return nil
}

type MockReorgConfirmationChecker struct {
	confirmed bool
	head      uint64
}

func (m *MockReorgConfirmationChecker) IsConfirmed(blockNumber uint64) bool {
	return m.confirmed
}

func (m *MockReorgConfirmationChecker) UpdateChainHead(head uint64) {
	m.head = head
}

func TestSetReorgHandler(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	handler := &MockReorgConfirmationChecker{confirmed: true}
	indexer.SetReorgHandler(handler)

	assert.NotNil(t, indexer.reorgHandler)
	assert.NotNil(t, indexer.chainHeads)
}

func TestUpdateChainHead(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	handler := &MockReorgConfirmationChecker{}
	indexer.SetReorgHandler(handler)

	indexer.UpdateChainHead("ethereum", 200)

	assert.Equal(t, uint64(200), handler.head)
	assert.Equal(t, uint64(200), indexer.chainHeads["ethereum"])
}

func TestUpdateChainHead_NoReorgHandler(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	config := &MockConfigManager{}
	indexer := NewMultiChainIndexer(logger, config)

	indexer.UpdateChainHead("polygon", 100)

	assert.Equal(t, uint64(100), indexer.chainHeads["polygon"])
}
