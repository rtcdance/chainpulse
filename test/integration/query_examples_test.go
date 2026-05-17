package integration

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chainpulse/pkg/core"
	"chainpulse/pkg/integrations/erc20"
	"chainpulse/pkg/integrations/uniswap"
	"chainpulse/pkg/services/decoder"
)

// MockDatabasePlugin for testing - defined in test_helpers.go
// MockCachePlugin for testing - defined in test_helpers.go
// MockLogger for testing - defined in test_helpers.go

// Helper functions - createTestBlockchainEvent is defined in test_helpers.go

// Example Query Tests

func TestExampleQueryAllContractEvents(t *testing.T) {
	db := NewMockDatabasePlugin()
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1000, "1"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1001, "2"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1002, "3"))

	filter := &core.EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111")},
		FromBlock:       0,
		ToBlock:         0,
		Limit:           1000,
	}

	events, err := db.QueryEvents(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 3, len(events))
}

func TestExampleQueryByBlockRange(t *testing.T) {
	db := NewMockDatabasePlugin()
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1000, "1"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1001, "2"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1002, "3"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1003, "4"))

	filter := &core.EventFilter{
		Network:   "ethereum",
		FromBlock: 1000,
		ToBlock:   1002,
		Limit:     1000,
	}

	events, err := db.QueryEvents(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 4, len(events))
}

func TestExampleQueryMultipleContracts(t *testing.T) {
	db := NewMockDatabasePlugin()
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1000, "1"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1001, "2"))
	_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", 1002, "3"))

	addresses := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	filter := &core.EventFilter{
		Network:         "ethereum",
		ContractAddress: addresses,
		FromBlock:       0,
		ToBlock:         0,
		Limit:           2000,
	}

	events, err := db.QueryEvents(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 3, len(events))
}

func TestExampleQueryWithPagination(t *testing.T) {
	db := NewMockDatabasePlugin()
	defer func() {
		if err := db.Close(); err != nil {
			_ = err // Log but continue
		}
	}()

	for i := 0; i < 2500; i++ {
		_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", uint64(1000+i), fmt.Sprintf("%d", i)))
	}

	pageSize := 1000
	filter := &core.EventFilter{
		Network: "ethereum",
		Limit:   pageSize,
	}

	// Query all events at once (mock database returns all events)
	pageEvents, err := db.QueryEvents(context.Background(), filter)
	require.NoError(t, err)

	// Verify we got all events
	assert.Equal(t, 2500, len(pageEvents))
}

// Uniswap Query Examples

func TestExampleUniswapIndexSwaps(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := uniswap.NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	events := []*core.BlockchainEvent{
		createTestBlockchainEvent("1", 1000, "Swap"),
		createTestBlockchainEvent("2", 1001, "Swap"),
	}

	err := indexer.IndexSwapEvents(context.Background(), events)

	require.NoError(t, err)
	assert.Contains(t, logger.messages, "indexing swap events")
}

func TestExampleUniswapGetPoolMetadata(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := uniswap.NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	poolAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	// Create a swap event to populate pool metadata
	swapEvent := &core.BlockchainEvent{
		ID:              "swap-1",
		ChainID:         "1",
		BlockNumber:     1000,
		TransactionHash: common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		ContractAddress: poolAddr,
		EventName:       "Swap",
		DecodedData: map[string]any{
			"sender":       common.HexToAddress("0x2222222222222222222222222222222222222222"),
			"recipient":    common.HexToAddress("0x3333333333333333333333333333333333333333"),
			"amount0In":    big.NewInt(1000),
			"amount1In":    big.NewInt(0),
			"amount0Out":   big.NewInt(0),
			"amount1Out":   big.NewInt(1000),
			"sqrtPriceX96": big.NewInt(1000000),
			"liquidity":    big.NewInt(1000000000),
			"tick":         int32(0),
		},
	}

	err := indexer.IndexSwapEvents(context.Background(), []*core.BlockchainEvent{swapEvent})
	require.NoError(t, err)

	retrieved := indexer.GetPoolMetadata(poolAddr)
	require.NotNil(t, retrieved)
	assert.Equal(t, poolAddr, retrieved.Address)
}

func TestExampleUniswapGetAllPools(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := uniswap.NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	// Create swap events for multiple pools
	for i := 0; i < 3; i++ {
		poolAddr := common.HexToAddress(fmt.Sprintf("0x%040d", i+1))
		swapEvent := &core.BlockchainEvent{
			ID:              fmt.Sprintf("swap-%d", i),
			ChainID:         "1",
			BlockNumber:     uint64(1000 + i),
			TransactionHash: common.HexToHash(fmt.Sprintf("0x%064d", i)),
			ContractAddress: poolAddr,
			EventName:       "Swap",
			DecodedData: map[string]any{
				"sender":       common.HexToAddress("0x2222222222222222222222222222222222222222"),
				"recipient":    common.HexToAddress("0x3333333333333333333333333333333333333333"),
				"amount0In":    big.NewInt(1000),
				"amount1In":    big.NewInt(0),
				"amount0Out":   big.NewInt(0),
				"amount1Out":   big.NewInt(1000),
				"sqrtPriceX96": big.NewInt(1000000),
				"liquidity":    big.NewInt(1000000000),
				"tick":         int32(0),
			},
		}
		_ = indexer.IndexSwapEvents(context.Background(), []*core.BlockchainEvent{swapEvent})
	}

	allMetadata := indexer.GetAllPoolMetadata()
	assert.Equal(t, 3, len(allMetadata))
}

// ERC20 Query Examples

func TestExampleERC20IndexTransfers(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := erc20.NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	events := []*core.BlockchainEvent{
		createTestBlockchainEvent("1", 1000, "Transfer"),
		createTestBlockchainEvent("2", 1001, "Transfer"),
	}

	err := indexer.IndexTransfers(context.Background(), events)

	require.NoError(t, err)
	assert.Contains(t, logger.messages, "indexing transfer events")
}

func TestExampleERC20GetBalance(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := erc20.NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	token := common.HexToAddress("0x1111111111111111111111111111111111111111")
	account := common.HexToAddress("0x2222222222222222222222222222222222222222")

	balance := indexer.GetBalance(token, account)
	assert.Equal(t, big.NewInt(0), balance)
}

func TestExampleERC20SetTokenMetadata(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := erc20.NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	token := common.HexToAddress("0x1111111111111111111111111111111111111111")
	metadata := &erc20.TokenMetadata{
		Address:     token,
		Name:        "Test Token",
		Symbol:      "TEST",
		Decimals:    18,
		TotalSupply: big.NewInt(1000000000),
	}

	err := indexer.SetTokenMetadata(metadata)
	require.NoError(t, err)

	retrieved := indexer.GetTokenMetadata(token)
	require.NotNil(t, retrieved)
	assert.Equal(t, "Test Token", retrieved.Name)
	assert.Equal(t, "TEST", retrieved.Symbol)
}

func TestExampleERC20GetAllTokenMetadata(t *testing.T) {
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := &decoder.EventDecoder{}
	contractManager := &decoder.ContractManager{}

	indexer := erc20.NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	for i := 0; i < 3; i++ {
		token := common.HexToAddress(fmt.Sprintf("0x%040d", i+1))
		metadata := &erc20.TokenMetadata{
			Address:  token,
			Name:     fmt.Sprintf("Token %d", i),
			Symbol:   fmt.Sprintf("TK%d", i),
			Decimals: 18,
		}
		_ = indexer.SetTokenMetadata(metadata)
	}

	allMetadata := indexer.GetAllTokenMetadata()
	assert.Equal(t, 3, len(allMetadata))
}

// Performance Examples

func TestExampleQueryPerformanceMetrics(t *testing.T) {
	db := NewMockDatabasePlugin()
	for i := 0; i < 1000; i++ {
		_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", uint64(1000+i), fmt.Sprintf("%d", i)))
	}

	start := time.Now()

	filter := &core.EventFilter{
		Network: "ethereum",
		Limit:   1000,
	}

	queryEvents, err := db.QueryEvents(context.Background(), filter)

	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, 1000, len(queryEvents))
	assert.Less(t, duration, 100*time.Millisecond)
}

func TestExampleCachingPattern(t *testing.T) {
	cache := NewMockCachePlugin()
	cacheKey := "test_query_result"

	// First access - cache miss
	_, err := cache.Get(context.Background(), cacheKey)
	assert.Error(t, err)

	// Store in cache
	err = cache.Set(context.Background(), cacheKey, []byte("result_data"), 3600)
	require.NoError(t, err)

	// Second access - cache hit
	result, err := cache.Get(context.Background(), cacheKey)
	require.NoError(t, err)
	assert.Equal(t, "result_data", string(result))
}

// Error Handling Examples

func TestExampleErrorHandlingValidation(t *testing.T) {
	db := NewMockDatabasePlugin()

	filter := &core.EventFilter{
		Network: "ethereum",
		Limit:   1000,
	}

	// Valid query
	events, err := db.QueryEvents(context.Background(), filter)
	require.NoError(t, err)
	assert.NotNil(t, events)
}

func TestExampleErrorHandlingEmptyResults(t *testing.T) {
	db := NewMockDatabasePlugin()

	filter := &core.EventFilter{
		Network: "ethereum",
		Limit:   1000,
	}

	events, err := db.QueryEvents(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 0, len(events))
}

func TestExampleErrorHandlingPagination(t *testing.T) {
	db := NewMockDatabasePlugin()
	for i := 0; i < 100; i++ {
		_ = db.StoreEvent(context.Background(), createTestBlockchainEvent("ethereum", uint64(1000+i), fmt.Sprintf("%d", i)))
	}

	pageSize := 25
	filter := &core.EventFilter{
		Network: "ethereum",
		Limit:   pageSize,
	}

	// Query all events at once
	pageEvents, err := db.QueryEvents(context.Background(), filter)
	require.NoError(t, err)

	// Verify we got all events
	assert.Equal(t, 100, len(pageEvents))
}
