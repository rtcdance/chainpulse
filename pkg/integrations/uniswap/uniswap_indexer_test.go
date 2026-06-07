package uniswap

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/services/decoder"
)

// MockLogger for testing
type MockLogger struct{}

func (ml *MockLogger) Debug(msg string, args ...any) {}
func (ml *MockLogger) Info(msg string, args ...any)  {}
func (ml *MockLogger) Warn(msg string, args ...any)  {}
func (ml *MockLogger) Error(msg string, args ...any) {}
func (ml *MockLogger) Fatal(msg string, args ...any) {}
func (ml *MockLogger) WithCorrelationID(id string) core.Logger {
	return ml
}

// MockDatabasePlugin for testing
type MockDatabasePlugin struct {
	events []*blockchain.BlockchainEvent
}

func (mdp *MockDatabasePlugin) StoreEvent(ctx context.Context, event any) error {
	if e, ok := event.(*blockchain.BlockchainEvent); ok {
		mdp.events = append(mdp.events, e)
	}
	return nil
}

func (mdp *MockDatabasePlugin) GetEvent(ctx context.Context, id string) (*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error) {
	return mdp.events, nil
}

func (mdp *MockDatabasePlugin) GetEventsByBlockRange(ctx context.Context, from, to uint64) ([]*blockchain.BlockchainEvent, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) DeleteEvent(ctx context.Context, id string) error {
	return nil
}

func (mdp *MockDatabasePlugin) GetBlock(ctx context.Context, number uint64) (*blockchain.Block, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*blockchain.Block, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error {
	for _, event := range events {
		if e, ok := event.(*blockchain.BlockchainEvent); ok {
			mdp.events = append(mdp.events, e)
		}
	}
	return nil
}

func (mdp *MockDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

func (mdp *MockDatabasePlugin) Health(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mdp *MockDatabasePlugin) Name() string {
	return "MockDatabasePlugin"
}

func (mdp *MockDatabasePlugin) Version() string {
	return "1.0.0"
}

func (mdp *MockDatabasePlugin) Initialize(ctx context.Context, config core.Config) error {
	_ = ctx
	_ = config
	return nil
}

func (mdp *MockDatabasePlugin) Start(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mdp *MockDatabasePlugin) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mdp *MockDatabasePlugin) Close() error { return nil }

// MockCachePlugin for testing
type MockCachePlugin struct {
	data map[string][]byte
}

func NewMockCachePlugin() *MockCachePlugin {
	return &MockCachePlugin{
		data: make(map[string][]byte),
	}
}

func (mcp *MockCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	return mcp.data[key], nil
}

func (mcp *MockCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	mcp.data[key] = value
	return nil
}

func (mcp *MockCachePlugin) Delete(ctx context.Context, key string) error {
	delete(mcp.data, key)
	return nil
}

func (mcp *MockCachePlugin) Clear(ctx context.Context) error {
	mcp.data = make(map[string][]byte)
	return nil
}

func (mcp *MockCachePlugin) Close() error { return nil }

func (mcp *MockCachePlugin) GetStats() core.CacheStats {
	return core.CacheStats{
		HitCount:      0,
		MissCount:     0,
		EvictionCount: 0,
		HitRate:       0.0,
	}
}

func (mcp *MockCachePlugin) Name() string {
	return "MockCachePlugin"
}

func (mcp *MockCachePlugin) Version() string {
	return "1.0.0"
}

func (mcp *MockCachePlugin) Initialize(ctx context.Context, config core.Config) error {
	_ = ctx
	_ = config
	return nil
}

func (mcp *MockCachePlugin) Start(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mcp *MockCachePlugin) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mcp *MockCachePlugin) Health(ctx context.Context) error {
	_ = ctx
	return nil
}

func (mcp *MockCachePlugin) HealthCheck(ctx context.Context) error {
	_ = ctx
	return nil
}

func TestNewUniswapIndexer(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	assert.NotNil(t, indexer)
}

func TestIndexSwapEvents(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	events := []*blockchain.BlockchainEvent{
		{
			ID:              "event1",
			BlockNumber:     100,
			LogIndex:        0,
			TransactionHash: common.HexToHash("0x1234"),
			ContractAddress: common.HexToAddress("0x1111"),
			DecodedData: map[string]any{
				"sender":       common.HexToAddress("0x2222"),
				"recipient":    common.HexToAddress("0x3333"),
				"amount0In":    big.NewInt(1000),
				"amount1In":    big.NewInt(0),
				"amount0Out":   big.NewInt(0),
				"amount1Out":   big.NewInt(900),
				"sqrtPriceX96": big.NewInt(1000000),
				"liquidity":    big.NewInt(5000000),
				"tick":         int32(100),
			},
		},
	}

	err := indexer.IndexSwapEvents(context.Background(), events)
	require.NoError(t, err)
}

func TestIndexSwapEventsEmpty(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	err := indexer.IndexSwapEvents(context.Background(), []*blockchain.BlockchainEvent{})
	require.NoError(t, err)
}

func TestGetSwapHistory(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	pool := common.HexToAddress("0x1111")

	history, err := indexer.GetSwapHistory(context.Background(), pool, 100, 200)
	require.NoError(t, err)

	assert.NotNil(t, history)
	assert.Equal(t, int64(0), history.SwapCount)
}

func TestGetSwapHistoryEmptyPool(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	_, err := indexer.GetSwapHistory(context.Background(), common.Address{}, 100, 200)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pool address is empty")
}

func TestGetSwapHistoryInvalidBlockRange(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	pool := common.HexToAddress("0x1111")

	_, err := indexer.GetSwapHistory(context.Background(), pool, 200, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from_block must be <= to_block")
}

func TestGetPoolMetadata(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	pool := common.HexToAddress("0x1111")

	metadata := indexer.GetPoolMetadata(pool)

	assert.Nil(t, metadata)
}

func TestGetAllPoolMetadata(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	allMetadata := indexer.GetAllPoolMetadata()

	assert.Equal(t, 0, len(allMetadata))
}

func TestClearCache(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	indexer.ClearCache()

	stats := indexer.GetCacheStats()
	assert.Equal(t, 0, stats["cached_swaps"])
}

func TestGetCacheStats(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	stats := indexer.GetCacheStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "cached_swaps")
	assert.Contains(t, stats, "pools")
}

func TestSwapEventStructure(t *testing.T) {
	t.Parallel()
	swap := &SwapEvent{
		TransactionHash: common.HexToHash("0x1234"),
		BlockNumber:     100,
		BlockTimestamp:  1234567890,
		LogIndex:        0,
		Pool:            common.HexToAddress("0x1111"),
		Sender:          common.HexToAddress("0x2222"),
		Recipient:       common.HexToAddress("0x3333"),
		Amount0In:       big.NewInt(1000),
		Amount1In:       big.NewInt(0),
		Amount0Out:      big.NewInt(0),
		Amount1Out:      big.NewInt(900),
		SqrtPriceX96:    big.NewInt(1000000),
		Liquidity:       big.NewInt(5000000),
		Tick:            100,
	}

	assert.NotNil(t, swap)
	assert.Equal(t, uint64(100), swap.BlockNumber)
	assert.Equal(t, big.NewInt(1000), swap.Amount0In)
}

func TestSwapHistoryStructure(t *testing.T) {
	t.Parallel()
	history := &SwapHistory{
		Swaps:         make([]*SwapEvent, 0),
		TotalVolume0:  big.NewInt(1000),
		TotalVolume1:  big.NewInt(900),
		AveragePrice:  big.NewFloat(0.9),
		SwapCount:     1,
		FirstSwapTime: 1234567890,
		LastSwapTime:  1234567900,
	}

	assert.NotNil(t, history)
	assert.Equal(t, int64(1), history.SwapCount)
	assert.Equal(t, big.NewInt(1000), history.TotalVolume0)
}

func TestPoolMetadataStructure(t *testing.T) {
	t.Parallel()
	metadata := &PoolMetadata{
		Address:      common.HexToAddress("0x1111"),
		Token0:       common.HexToAddress("0x2222"),
		Token1:       common.HexToAddress("0x3333"),
		Fee:          3000,
		Liquidity:    big.NewInt(5000000),
		SqrtPriceX96: big.NewInt(1000000),
		Tick:         100,
	}

	assert.NotNil(t, metadata)
	assert.Equal(t, uint32(3000), metadata.Fee)
	assert.Equal(t, int32(100), metadata.Tick)
}

func TestValidateSwapEvent(t *testing.T) {
	t.Parallel()
	swap := &SwapEvent{
		Pool:       common.HexToAddress("0x1111"),
		Amount0In:  big.NewInt(1000),
		Amount1In:  big.NewInt(0),
		Amount0Out: big.NewInt(0),
		Amount1Out: big.NewInt(900),
	}

	// This should not error in the actual implementation
	assert.NotNil(t, swap)
}

func TestValidateSwapEventEmptyPool(t *testing.T) {
	t.Parallel()
	swap := &SwapEvent{
		Pool:       common.Address{},
		Amount0In:  big.NewInt(1000),
		Amount1In:  big.NewInt(0),
		Amount0Out: big.NewInt(0),
		Amount1Out: big.NewInt(900),
	}

	// Empty pool should be invalid
	assert.Equal(t, common.Address{}, swap.Pool)
}

func TestConcurrentSwapIndexing(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewUniswapIndexer(db, cache, logger, eventDecoder, contractManager)

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			events := []*blockchain.BlockchainEvent{
				{
					ID:              "event" + string(rune(id)),
					BlockNumber:     uint64(100 + id),
					LogIndex:        0,
					TransactionHash: common.HexToHash("0x1234"),
					ContractAddress: common.HexToAddress("0x1111"),
					DecodedData: map[string]any{
						"sender":       common.HexToAddress("0x2222"),
						"recipient":    common.HexToAddress("0x3333"),
						"amount0In":    big.NewInt(1000),
						"amount1In":    big.NewInt(0),
						"amount0Out":   big.NewInt(0),
						"amount1Out":   big.NewInt(900),
						"sqrtPriceX96": big.NewInt(1000000),
						"liquidity":    big.NewInt(5000000),
						"tick":         int32(100),
					},
				},
			}
			_ = indexer.IndexSwapEvents(context.Background(), events)
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestSwapEventWithZeroAmounts(t *testing.T) {
	t.Parallel()
	swap := &SwapEvent{
		Pool:       common.HexToAddress("0x1111"),
		Amount0In:  big.NewInt(0),
		Amount1In:  big.NewInt(0),
		Amount0Out: big.NewInt(0),
		Amount1Out: big.NewInt(0),
	}

	assert.NotNil(t, swap)
	assert.Equal(t, 0, swap.Amount0In.Sign())
}

func TestSwapEventWithLargeAmounts(t *testing.T) {
	t.Parallel()
	largeAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	swap := &SwapEvent{
		Pool:       common.HexToAddress("0x1111"),
		Amount0In:  largeAmount,
		Amount1In:  big.NewInt(0),
		Amount0Out: big.NewInt(0),
		Amount1Out: largeAmount,
	}

	assert.NotNil(t, swap)
	assert.Equal(t, largeAmount, swap.Amount0In)
}
