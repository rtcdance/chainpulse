package erc20

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
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
	events []*core.BlockchainEvent
}

func (mdp *MockDatabasePlugin) StoreEvent(ctx context.Context, event any) error {
	if e, ok := event.(*core.BlockchainEvent); ok {
		mdp.events = append(mdp.events, e)
	}
	return nil
}

func (mdp *MockDatabasePlugin) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	return mdp.events, nil
}

func (mdp *MockDatabasePlugin) GetEventsByBlockRange(ctx context.Context, from, to uint64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) DeleteEvent(ctx context.Context, id string) error {
	return nil
}

func (mdp *MockDatabasePlugin) GetBlock(ctx context.Context, number uint64) (*core.Block, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	return nil, nil
}

func (mdp *MockDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error {
	for _, event := range events {
		if e, ok := event.(*core.BlockchainEvent); ok {
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

func TestNewERC20Indexer(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	assert.NotNil(t, indexer)
}

func TestIndexTransfers(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			BlockNumber:     100,
			LogIndex:        0,
			TransactionHash: common.HexToHash("0x1234"),
			ContractAddress: common.HexToAddress("0x1111"),
			DecodedData: map[string]any{
				"from":  common.HexToAddress("0x2222"),
				"to":    common.HexToAddress("0x3333"),
				"value": big.NewInt(1000),
			},
		},
	}

	err := indexer.IndexTransfers(context.Background(), events)
	require.NoError(t, err)
}

func TestIndexTransfersEmpty(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	err := indexer.IndexTransfers(context.Background(), []*core.BlockchainEvent{})
	require.NoError(t, err)
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	token := common.HexToAddress("0x1111")
	account := common.HexToAddress("0x2222")

	balance := indexer.GetBalance(token, account)

	assert.NotNil(t, balance)
	assert.Equal(t, 0, balance.Sign())
}

func TestGetBalanceEmptyAddress(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	balance := indexer.GetBalance(common.Address{}, common.HexToAddress("0x2222"))

	assert.NotNil(t, balance)
	assert.Equal(t, 0, balance.Sign())
}

func TestGetTokenMetadata(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	token := common.HexToAddress("0x1111")

	metadata := indexer.GetTokenMetadata(token)

	assert.Nil(t, metadata)
}

func TestSetTokenMetadata(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	token := common.HexToAddress("0x1111")
	metadata := &TokenMetadata{
		Address:  token,
		Name:     "Test Token",
		Symbol:   "TEST",
		Decimals: 18,
	}

	err := indexer.SetTokenMetadata(metadata)
	require.NoError(t, err)

	retrieved := indexer.GetTokenMetadata(token)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Test Token", retrieved.Name)
	assert.Equal(t, "TEST", retrieved.Symbol)
}

func TestSetTokenMetadataNil(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	err := indexer.SetTokenMetadata(nil)
	assert.Error(t, err)
}

func TestSetTokenMetadataEmptyAddress(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	metadata := &TokenMetadata{
		Address: common.Address{},
		Name:    "Test Token",
	}

	err := indexer.SetTokenMetadata(metadata)
	assert.Error(t, err)
}

func TestGetAllTokenMetadata(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	token1 := common.HexToAddress("0x1111")
	token2 := common.HexToAddress("0x2222")

	metadata1 := &TokenMetadata{
		Address: token1,
		Name:    "Token 1",
	}
	metadata2 := &TokenMetadata{
		Address: token2,
		Name:    "Token 2",
	}

	_ = indexer.SetTokenMetadata(metadata1)
	_ = indexer.SetTokenMetadata(metadata2)

	allMetadata := indexer.GetAllTokenMetadata()

	assert.Equal(t, 2, len(allMetadata))
	assert.Contains(t, allMetadata, token1)
	assert.Contains(t, allMetadata, token2)
}

func TestClearCache(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	indexer.ClearCache()

	stats := indexer.GetCacheStats()
	assert.Equal(t, 0, stats["cached_transfers"])
}

func TestGetCacheStats(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	stats := indexer.GetCacheStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "cached_transfers")
	assert.Contains(t, stats, "tracked_balances")
	assert.Contains(t, stats, "tokens")
}

func TestTransferEventStructure(t *testing.T) {
	t.Parallel()
	transfer := &TransferEvent{
		TransactionHash: common.HexToHash("0x1234"),
		BlockNumber:     100,
		BlockTimestamp:  1234567890,
		LogIndex:        0,
		Token:           common.HexToAddress("0x1111"),
		From:            common.HexToAddress("0x2222"),
		To:              common.HexToAddress("0x3333"),
		Value:           big.NewInt(1000),
	}

	assert.NotNil(t, transfer)
	assert.Equal(t, uint64(100), transfer.BlockNumber)
	assert.Equal(t, big.NewInt(1000), transfer.Value)
}

func TestTokenBalanceStructure(t *testing.T) {
	t.Parallel()
	balance := &TokenBalance{
		Token:       common.HexToAddress("0x1111"),
		Account:     common.HexToAddress("0x2222"),
		Balance:     big.NewInt(5000),
		BlockNumber: 100,
		BlockTime:   1234567890,
	}

	assert.NotNil(t, balance)
	assert.Equal(t, big.NewInt(5000), balance.Balance)
}

func TestTransferHistoryStructure(t *testing.T) {
	t.Parallel()
	history := &TransferHistory{
		Transfers:     make([]*TransferEvent, 0),
		TotalIncoming: big.NewInt(1000),
		TotalOutgoing: big.NewInt(500),
		NetChange:     big.NewInt(500),
		TransferCount: 2,
		FirstTransfer: 1234567890,
		LastTransfer:  1234567900,
	}

	assert.NotNil(t, history)
	assert.Equal(t, int64(2), history.TransferCount)
	assert.Equal(t, big.NewInt(500), history.NetChange)
}

func TestConcurrentIndexing(t *testing.T) {
	t.Parallel()
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewERC20Indexer(db, cache, logger, eventDecoder, contractManager)

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			events := []*core.BlockchainEvent{
				{
					ID:              "event" + string(rune(id)),
					BlockNumber:     uint64(100 + id),
					LogIndex:        0,
					TransactionHash: common.HexToHash("0x1234"),
					ContractAddress: common.HexToAddress("0x1111"),
					DecodedData: map[string]any{
						"from":  common.HexToAddress("0x2222"),
						"to":    common.HexToAddress("0x3333"),
						"value": big.NewInt(1000),
					},
				},
			}
			_ = indexer.IndexTransfers(context.Background(), events)
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
