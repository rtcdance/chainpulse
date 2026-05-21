package indexing

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/integrations/generic"
)

// MockDatabasePlugin implements core.DatabasePlugin for testing
type MockDatabasePlugin struct {
	data       map[string]any
	events     map[string]*core.BlockchainEvent
	mu         sync.RWMutex
	storeCount int
}

func NewMockDatabasePlugin() *MockDatabasePlugin {
	return &MockDatabasePlugin{
		data:   make(map[string]any),
		events: make(map[string]*core.BlockchainEvent),
	}
}

func (m *MockDatabasePlugin) StoreEvent(ctx context.Context, event any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storeCount++
	if e, ok := event.(*core.BlockchainEvent); ok {
		m.events[e.ID] = e
	}
	return nil
}

func (m *MockDatabasePlugin) GetStoreCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.storeCount
}

func (m *MockDatabasePlugin) GetEvent(ctx context.Context, id string) (*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return nil, nil
}

func (m *MockDatabasePlugin) QueryEvents(ctx context.Context, filter any) ([]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]any, 0)
	for _, e := range m.events {
		result = append(result, e)
	}
	return result, nil
}

func (m *MockDatabasePlugin) BatchStoreEvents(ctx context.Context, events []any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, event := range events {
		if e, ok := event.(*core.BlockchainEvent); ok {
			m.events[e.ID] = e
		}
	}
	return nil
}

func (m *MockDatabasePlugin) GetAllEvents(ctx context.Context) ([]*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*core.BlockchainEvent, 0)
	for _, e := range m.events {
		result = append(result, e)
	}
	return result, nil
}

func (m *MockDatabasePlugin) GetAllBlocks(ctx context.Context) ([]*core.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return make([]*core.Block, 0), nil
}

func (m *MockDatabasePlugin) DeleteEvent(ctx context.Context, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.events, eventID)
	return nil
}

func (m *MockDatabasePlugin) GetEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) ([]*core.BlockchainEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*core.BlockchainEvent, 0)
	for _, e := range m.events {
		if e.BlockNumber >= fromBlock && e.BlockNumber <= toBlock {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockDatabasePlugin) GetBlock(ctx context.Context, blockNumber uint64) (*core.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return nil, nil
}

func (m *MockDatabasePlugin) GetLatestBlock(ctx context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var maxBlock uint64
	for _, e := range m.events {
		if e.BlockNumber > maxBlock {
			maxBlock = e.BlockNumber
		}
	}
	return maxBlock, nil
}

func (m *MockDatabasePlugin) DeleteEventsByBlockRange(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := int64(0)
	for id, e := range m.events {
		if e.BlockNumber >= fromBlock && e.BlockNumber <= toBlock {
			delete(m.events, id)
			count++
		}
	}
	return count, nil
}

func (m *MockDatabasePlugin) MarkEventsAsReorged(ctx context.Context, fromBlock, toBlock uint64) (int64, error) {
	return 0, nil
}

func (m *MockDatabasePlugin) GetReorgStats(ctx context.Context) (*core.ReorgStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &core.ReorgStats{}, nil
}

func (m *MockDatabasePlugin) Health(ctx context.Context) error {
	return nil
}

func (m *MockDatabasePlugin) Name() string {
	return "MockDatabasePlugin"
}

func (m *MockDatabasePlugin) Version() string {
	return "1.0.0"
}

func (m *MockDatabasePlugin) Initialize(ctx context.Context, config core.Config) error {
	return nil
}

func (m *MockDatabasePlugin) Start(ctx context.Context) error {
	return nil
}

func (m *MockDatabasePlugin) Stop(ctx context.Context) error {
	return nil
}

func (m *MockDatabasePlugin) Close() error {
	return nil
}

// MockLogger implements core.Logger for testing
type MockLogger struct {
	logs []string
	mu   sync.RWMutex
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]string, 0),
	}
}

func (m *MockLogger) Info(msg string, _ ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Error(msg string, _ ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Debug(msg string, _ ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Warn(msg string, _ ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Fatal(msg string, _ ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) WithCorrelationID(id string) core.Logger {
	return m
}

func (m *MockLogger) Contains(substring string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, log := range m.logs {
		if log == substring {
			return true
		}
	}
	return false
}

// MockCachePlugin implements core.CachePlugin for testing
type MockCachePlugin struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMockCachePlugin() *MockCachePlugin {
	return &MockCachePlugin{
		data: make(map[string][]byte),
	}
}

func (mcp *MockCachePlugin) Get(ctx context.Context, key string) ([]byte, error) {
	mcp.mu.RLock()
	defer mcp.mu.RUnlock()
	if val, ok := mcp.data[key]; ok {
		return val, nil
	}
	return nil, nil
}

func (mcp *MockCachePlugin) Set(ctx context.Context, key string, value []byte, ttl int) error {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	mcp.data[key] = value
	return nil
}

func (mcp *MockCachePlugin) Delete(ctx context.Context, key string) error {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	delete(mcp.data, key)
	return nil
}

func (mcp *MockCachePlugin) Clear(ctx context.Context) error {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	mcp.data = make(map[string][]byte)
	return nil
}

func (mcp *MockCachePlugin) Close() error {
	return nil
}

func (mcp *MockCachePlugin) GetStats() core.CacheStats {
	mcp.mu.RLock()
	defer mcp.mu.RUnlock()
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

type stubSharedBatchRuntime struct {
	mu          sync.Mutex
	calls       int
	lastChainID string
	lastEvents  []core.EventEnvelope
	err         error
}

func (s *stubSharedBatchRuntime) ProcessBatch(ctx context.Context, chainID string, events []core.EventEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.lastChainID = chainID
	s.lastEvents = append([]core.EventEnvelope(nil), events...)
	return s.err
}

func TestNewDefaultChainIndexer(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	assert.NotNil(t, indexer)
	assert.Equal(t, "ethereum", indexer.GetChainID())
	assert.Equal(t, uint64(0), indexer.GetLastIndexedBlock())
	assert.Equal(t, int64(0), indexer.GetTotalEventsIndexed())
}

func TestIndexEvents(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
		{
			ID:              "event2",
			ChainID:         "ethereum",
			BlockNumber:     101,
			TransactionHash: common.HexToHash("0x1235"),
			EventSignature:  common.HexToHash("0x5679"),
		},
	}

	err := indexer.IndexEvents(context.Background(), events)
	require.NoError(t, err)

	assert.Equal(t, int64(2), indexer.GetTotalEventsIndexed())
	assert.Equal(t, uint64(101), indexer.GetLastIndexedBlock())
}

func TestIndexEventsEmptyList(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	err := indexer.IndexEvents(context.Background(), []*core.BlockchainEvent{})
	require.NoError(t, err)

	assert.Equal(t, int64(0), indexer.GetTotalEventsIndexed())
}

func TestIndexEventsChainIDMismatch(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	events := []*core.BlockchainEvent{
		{
			ID:      "event1",
			ChainID: "polygon", // Wrong chain
		},
	}

	err := indexer.IndexEvents(context.Background(), events)
	require.NoError(t, err)

	assert.Equal(t, int64(1), indexer.GetTotalErrors())
}

func TestGetChainID(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	assert.Equal(t, "ethereum", indexer.GetChainID())
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events)

	status := indexer.GetStatus()

	assert.NotNil(t, status)
	assert.Equal(t, "ethereum", status["chain_id"])
	assert.Equal(t, uint64(100), status["last_indexed_block"])
	assert.Equal(t, int64(1), status["total_events_indexed"])
	assert.Equal(t, int64(0), status["shadow_owned_events"])
	assert.Equal(t, int64(1), status["legacy_owned_events"])
}

func TestClose(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	err := indexer.Close()
	require.NoError(t, err)
}

func TestGetLastIndexedBlock(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	assert.Equal(t, uint64(0), indexer.GetLastIndexedBlock())

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events)

	assert.Equal(t, uint64(100), indexer.GetLastIndexedBlock())
}

func TestGetTotalEventsIndexed(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	assert.Equal(t, int64(0), indexer.GetTotalEventsIndexed())

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
		{
			ID:              "event2",
			ChainID:         "ethereum",
			BlockNumber:     101,
			TransactionHash: common.HexToHash("0x1235"),
			EventSignature:  common.HexToHash("0x5679"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events)

	assert.Equal(t, int64(2), indexer.GetTotalEventsIndexed())
}

func TestGetTotalErrors(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	assert.Equal(t, int64(0), indexer.GetTotalErrors())

	events := []*core.BlockchainEvent{
		{
			ID:      "event1",
			ChainID: "polygon", // Wrong chain
		},
	}

	_ = indexer.IndexEvents(context.Background(), events)

	assert.Equal(t, int64(1), indexer.GetTotalErrors())
}

func TestResetStats(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events)

	assert.Equal(t, int64(1), indexer.GetTotalEventsIndexed())

	indexer.ResetStats()

	assert.Equal(t, int64(0), indexer.GetTotalEventsIndexed())
	assert.Equal(t, uint64(0), indexer.GetLastIndexedBlock())
	assert.Equal(t, int64(0), indexer.GetTotalErrors())
}

func TestMultipleIndexing(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	// First batch
	events1 := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events1)

	// Second batch
	events2 := []*core.BlockchainEvent{
		{
			ID:              "event2",
			ChainID:         "ethereum",
			BlockNumber:     101,
			TransactionHash: common.HexToHash("0x1235"),
			EventSignature:  common.HexToHash("0x5679"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events2)

	assert.Equal(t, int64(2), indexer.GetTotalEventsIndexed())
	assert.Equal(t, uint64(101), indexer.GetLastIndexedBlock())
}

func TestStatusMetrics(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	_ = indexer.IndexEvents(context.Background(), events)

	status := indexer.GetStatus()

	assert.Contains(t, status, "chain_id")
	assert.Contains(t, status, "last_indexed_block")
	assert.Contains(t, status, "total_events_indexed")
	assert.Contains(t, status, "shadow_owned_events")
	assert.Contains(t, status, "legacy_owned_events")
	assert.Contains(t, status, "total_errors")
	assert.Contains(t, status, "uptime_seconds")
	assert.Contains(t, status, "events_per_second")
	assert.Contains(t, status, "error_rate")
}

func TestIndexEventsForwardsShadowBatchToSharedRuntime(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	metrics := core.NewDefaultMetricsCollector()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	sharedRuntime := &stubSharedBatchRuntime{}

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)
	indexer.SetSharedRuntime(sharedRuntime, metrics)

	createdAt := time.Unix(1710000000, 0)
	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			LogIndex:        2,
			CreatedAt:       createdAt,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	err := indexer.IndexEvents(context.Background(), events)
	require.NoError(t, err)

	require.Equal(t, 1, sharedRuntime.calls)
	require.Equal(t, "ethereum", sharedRuntime.lastChainID)
	require.Len(t, sharedRuntime.lastEvents, 1)
	assert.Equal(t, "event1", sharedRuntime.lastEvents[0].EventKey)
	assert.Equal(t, "ethereum:100:2", sharedRuntime.lastEvents[0].CheckpointCursor)
	assert.Equal(t, createdAt, sharedRuntime.lastEvents[0].ReceivedAt)
}

func TestIndexEventsSharedRuntimeFailureDoesNotBlockLegacyIndexing(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	metrics := core.NewDefaultMetricsCollector()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	sharedRuntime := &stubSharedBatchRuntime{err: errors.New("shadow boom")}

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)
	indexer.SetSharedRuntime(sharedRuntime, metrics)

	events := []*core.BlockchainEvent{
		{
			ID:              "event1",
			ChainID:         "ethereum",
			BlockNumber:     100,
			LogIndex:        1,
			TransactionHash: common.HexToHash("0x1234"),
			EventSignature:  common.HexToHash("0x5678"),
		},
	}

	err := indexer.IndexEvents(context.Background(), events)
	require.NoError(t, err)

	stored, getErr := db.GetEvent(context.Background(), "event1")
	require.NoError(t, getErr)
	require.NotNil(t, stored)
	assert.Equal(t, int64(1), metrics.GetCounter("indexing_runtime_shadow_batch_errors_total", map[string]string{
		"chain_id":  "ethereum",
		"service":   "monolithic",
		"operation": "shadow_batch",
	}))
	assert.True(t, logger.Contains("shared runtime shadow batch failed"))
}

func TestIndexEventsSkipsDuplicateLegacyWriteAfterShadowPersistence(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	metrics := core.NewDefaultMetricsCollector()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	sharedRuntime := &stubSharedBatchRuntime{}

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)
	indexer.SetSharedRuntime(sharedRuntime, metrics)

	event := &core.BlockchainEvent{
		ID:              "event1",
		ChainID:         "ethereum",
		BlockNumber:     100,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}

	markShadowWrite(event)

	err := indexer.IndexEvents(context.Background(), []*core.BlockchainEvent{event})
	require.NoError(t, err)

	assert.Equal(t, 0, db.GetStoreCount())
	assert.Equal(t, int64(1), indexer.GetTotalEventsIndexed())
	assert.Equal(t, int64(1), metrics.GetCounter("indexing_runtime_shadow_owned_events_total", map[string]string{
		"chain_id":  "ethereum",
		"service":   "monolithic",
		"operation": "shadow_owned_write",
	}))
	status := indexer.GetStatus()
	assert.Equal(t, int64(1), status["shadow_owned_events"])
	assert.Equal(t, int64(0), status["legacy_owned_events"])
}

func TestIndexEventsDoesNotEmitShadowOwnedMetricOnLegacyFallback(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()
	metrics := core.NewDefaultMetricsCollector()
	genericIndexer := generic.NewGenericContractIndexer(db, cache, logger, nil, nil)
	sharedRuntime := &stubSharedBatchRuntime{err: errors.New("shadow boom")}

	indexer := NewDefaultChainIndexer("ethereum", db, cache, logger, genericIndexer)
	indexer.SetSharedRuntime(sharedRuntime, metrics)

	event := &core.BlockchainEvent{
		ID:              "event1",
		ChainID:         "ethereum",
		BlockNumber:     100,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0x1234"),
		EventSignature:  common.HexToHash("0x5678"),
	}

	err := indexer.IndexEvents(context.Background(), []*core.BlockchainEvent{event})
	require.NoError(t, err)

	assert.Equal(t, 1, db.GetStoreCount())
	assert.Equal(t, int64(0), metrics.GetCounter("indexing_runtime_shadow_owned_events_total", map[string]string{
		"chain_id":  "ethereum",
		"service":   "monolithic",
		"operation": "shadow_owned_write",
	}))
	status := indexer.GetStatus()
	assert.Equal(t, int64(0), status["shadow_owned_events"])
	assert.Equal(t, int64(1), status["legacy_owned_events"])
}
