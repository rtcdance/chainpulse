package generic

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/decoder"
)

// MockLogger for testing
type MockLogger struct{}

func (ml *MockLogger) Debug(msg string, args ...interface{}) {}
func (ml *MockLogger) Info(msg string, args ...interface{})  {}
func (ml *MockLogger) Warn(msg string, args ...interface{})  {}
func (ml *MockLogger) Error(msg string, args ...interface{}) {}
func (ml *MockLogger) Fatal(msg string, args ...interface{}) {}
func (ml *MockLogger) WithCorrelationID(id string) core.Logger {
	return ml
}

// MockDatabasePlugin for testing
type MockDatabasePlugin struct {
	events []*core.BlockchainEvent
}

func (mdp *MockDatabasePlugin) StoreEvent(ctx context.Context, event interface{}) error {
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

func (mdp *MockDatabasePlugin) QueryEvents(ctx context.Context, filter interface{}) ([]interface{}, error) {
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

func (mdp *MockDatabasePlugin) BatchStoreEvents(ctx context.Context, events []interface{}) error {
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

func (mdp *MockDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	return 0, nil
}

func (mdp *MockDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	return &core.ReorgStats{}, nil
}

func (mdp *MockDatabasePlugin) Health() error {
	return nil
}

func (mdp *MockDatabasePlugin) Name() string {
	return "MockDatabasePlugin"
}

func (mdp *MockDatabasePlugin) Version() string {
	return "1.0.0"
}

func (mdp *MockDatabasePlugin) Initialize(config core.Config) error {
	return nil
}

func (mdp *MockDatabasePlugin) Start() error {
	return nil
}

func (mdp *MockDatabasePlugin) Stop() error {
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

func (mcp *MockCachePlugin) Initialize(config core.Config) error {
	return nil
}

func (mcp *MockCachePlugin) Start() error {
	return nil
}

func (mcp *MockCachePlugin) Stop() error {
	return nil
}

func (mcp *MockCachePlugin) Health() error {
	return nil
}

func (mcp *MockCachePlugin) HealthCheck(ctx context.Context) error {
	_ = ctx
	return nil
}

func TestNewGenericContractIndexer(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	assert.NotNil(t, indexer)
}

func TestRegisterContractABI(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	var contractABI abi.ABI
	err := indexer.RegisterContractABI("ERC20", contractABI)
	require.NoError(t, err)
}

func TestRegisterContractABIEmptyName(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	var contractABI abi.ABI
	err := indexer.RegisterContractABI("", contractABI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contract name cannot be empty")
}

func TestRegisterEventHandler(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	handler := &MockEventHandler{eventName: "Transfer"}

	err := indexer.RegisterEventHandler("Transfer", handler)
	require.NoError(t, err)
}

func TestRegisterEventHandlerEmptyName(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	handler := &MockEventHandler{eventName: "Transfer"}

	err := indexer.RegisterEventHandler("", handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event name cannot be empty")
}

func TestRegisterEventHandlerNil(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	err := indexer.RegisterEventHandler("Transfer", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler cannot be nil")
}

func TestIndexEvents(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	var contractABI abi.ABI
	_ = indexer.RegisterContractABI("ERC20", contractABI)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			ContractAddress: common.HexToAddress("0x1111"),
		},
	}

	err := indexer.IndexEvents(context.Background(), "ERC20", events)
	require.NoError(t, err)
}

func TestIndexEventsNotRegistered(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	events := []*core.BlockchainEvent{
		{
			ID: "event1",
		},
	}

	err := indexer.IndexEvents(context.Background(), "NonExistent", events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestGetEventsByName(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	events := indexer.GetEventsByName("Transfer")

	assert.Equal(t, 0, len(events))
}

func TestGetEventsByContract(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	contractAddr := common.HexToAddress("0x1111")
	events := indexer.GetEventsByContract(contractAddr)

	assert.Equal(t, 0, len(events))
}

func TestGetContractMetadata(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	contractAddr := common.HexToAddress("0x1111")
	metadata := indexer.GetContractMetadata(contractAddr)

	assert.Nil(t, metadata)
}

func TestSetContractMetadata(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	contractAddr := common.HexToAddress("0x1111")
	metadata := &ContractMetadata{
		Address: contractAddr,
		Name:    "ERC20",
	}

	err := indexer.SetContractMetadata(metadata)
	require.NoError(t, err)

	retrieved := indexer.GetContractMetadata(contractAddr)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "ERC20", retrieved.Name)
}

func TestSetContractMetadataNil(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	err := indexer.SetContractMetadata(nil)
	assert.Error(t, err)
}

func TestSetContractMetadataEmptyAddress(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	metadata := &ContractMetadata{
		Address: common.Address{},
		Name:    "ERC20",
	}

	err := indexer.SetContractMetadata(metadata)
	assert.Error(t, err)
}

func TestGetCacheStats(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	stats := indexer.GetCacheStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "cached_events")
	assert.Contains(t, stats, "registered_abis")
	assert.Contains(t, stats, "event_handlers")
	assert.Contains(t, stats, "tracked_contracts")
}

func TestClearCache(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	indexer.ClearCache()

	stats := indexer.GetCacheStats()
	assert.Equal(t, 0, stats["cached_events"])
}

func TestGetRegisteredContracts(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	var contractABI abi.ABI
	_ = indexer.RegisterContractABI("ERC20", contractABI)
	_ = indexer.RegisterContractABI("ERC721", contractABI)

	contracts := indexer.GetRegisteredContracts()

	assert.Equal(t, 2, len(contracts))
	assert.Contains(t, contracts, "ERC20")
	assert.Contains(t, contracts, "ERC721")
}

func TestContractMetadataStructure(t *testing.T) {
	var contractABI abi.ABI
	metadata := &ContractMetadata{
		Address:     common.HexToAddress("0x1111"),
		Name:        "ERC20",
		ABI:         contractABI,
		EventCount:  10,
		LastUpdated: 1234567890,
	}

	assert.NotNil(t, metadata)
	assert.Equal(t, "ERC20", metadata.Name)
	assert.Equal(t, int64(10), metadata.EventCount)
}

func TestDecodedContractEventStructure(t *testing.T) {
	event := &DecodedContractEvent{
		ContractAddress:  common.HexToAddress("0x1111"),
		EventName:        "Transfer",
		EventSignature:   common.HexToHash("0x1234"),
		BlockNumber:      100,
		BlockTimestamp:   1234567890,
		TransactionHash:  common.HexToHash("0x5678"),
		LogIndex:         0,
		Parameters:       make(map[string]interface{}),
		IndexedParams:    make(map[string]interface{}),
		NonIndexedParams: make(map[string]interface{}),
	}

	assert.Equal(t, "Transfer", event.EventName)
	assert.Equal(t, uint64(100), event.BlockNumber)
}

// MockEventHandler for testing
type MockEventHandler struct {
	eventName string
}

func (meh *MockEventHandler) Handle(ctx context.Context, event *DecodedContractEvent) error {
	return nil
}

func (meh *MockEventHandler) GetEventName() string {
	return meh.eventName
}

func TestConcurrentIndexing(t *testing.T) {
	db := &MockDatabasePlugin{}
	cache := NewMockCachePlugin()
	logger := &MockLogger{}
	eventDecoder := decoder.NewEventDecoder(nil, logger)
	contractManager := decoder.NewContractManager(logger)

	indexer := NewGenericContractIndexer(db, cache, logger, eventDecoder, contractManager)

	var contractABI abi.ABI
	_ = indexer.RegisterContractABI("ERC20", contractABI)

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			events := []*core.BlockchainEvent{
				{
					ID:              "event1",
					BlockNumber:     100,
					TransactionHash: common.HexToHash("0x1234"),
					ContractAddress: common.HexToAddress("0x1111"),
				},
			}
			_ = indexer.IndexEvents(context.Background(), "ERC20", events)
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
