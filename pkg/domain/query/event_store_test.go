package query

import (
	"context"
	"testing"
	"time"

	"chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type mockEventStoreDomain struct {
	events     map[string]*core.BlockchainEvent
	initialize bool
	close      bool
}

func NewMockEventStoreDomain() *mockEventStoreDomain {
	return &mockEventStoreDomain{events: make(map[string]*core.BlockchainEvent)}
}

func (m *mockEventStoreDomain) Initialize(ctx context.Context) error {
	m.initialize = true
	return nil
}

func (m *mockEventStoreDomain) Close(ctx context.Context) error {
	m.close = true
	return nil
}

func (m *mockEventStoreDomain) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if event == nil || event.ID == "" {
		return nil
	}
	m.events[event.ID] = event
	return nil
}

func (m *mockEventStoreDomain) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	for _, e := range events {
		if err := m.InsertEvent(ctx, e); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockEventStoreDomain) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	event, ok := m.events[eventID]
	if !ok {
		return nil, nil
	}
	return event, nil
}

func (m *mockEventStoreDomain) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	chainIDStr := "1"
	if chainID == 2 {
		chainIDStr = "2"
	}
	for _, e := range m.events {
		if e.ChainID == chainIDStr {
			result = append(result, e)
		}
		if len(result) >= offset+limit {
			break
		}
	}
	if offset > len(result) {
		return result[offset:], nil
	}
	return result, nil
}

func (m *mockEventStoreDomain) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	addr := common.HexToAddress(contractAddress)
	for _, e := range m.events {
		if e.ContractAddress == addr {
			result = append(result, e)
		}
	}
	if offset > len(result) {
		return result[offset : offset+limit], nil
	}
	return result[offset:minInt(offset+limit, len(result))], nil
}

func (m *mockEventStoreDomain) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	for _, e := range m.events {
		if e.EventName == eventName {
			result = append(result, e)
		}
	}
	if offset > len(result) {
		return result[offset : offset+limit], nil
	}
	return result[offset:minInt(offset+limit, len(result))], nil
}

func (m *mockEventStoreDomain) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	for _, e := range m.events {
		if e.BlockNumber == uint64(blockNumber) {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockEventStoreDomain) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	var result []*core.BlockchainEvent
	addr := common.HexToAddress(address)
	count := 0
	for _, e := range m.events {
		if e.ContractAddress == addr {
			result = append(result, e)
			count++
			if count >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockEventStoreDomain) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	return m.GetEventsByEventName(ctx, eventName, limit, 0)
}

func (m *mockEventStoreDomain) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	var result []*core.BlockchainEvent
	count := 0
	hasMore := false
	for _, e := range m.events {
		if count >= limit {
			hasMore = true
			break
		}
		result = append(result, e)
		count++
	}
	return result, hasMore, nil
}

func (m *mockEventStoreDomain) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockEventStoreDomain) CountEvents(ctx context.Context) (int64, error) {
	return int64(len(m.events)), nil
}

func (m *mockEventStoreDomain) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy", Timestamp: time.Now()}
}

func TestEventStoreInterface(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	// Verify the mock implements EventStore interface at compile time
	_ = EventStore(store)
}

func TestEventStoreInitialize(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	err := store.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() unexpected error: %v", err)
	}
	err = store.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() second call should not fail: %v", err)
	}
}

func TestEventStoreClose(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	err := store.Close(ctx)
	if err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

func TestEventStoreInsertEvent(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	event := &core.BlockchainEvent{
		ID:              "test-event-1",
		EventName:       "Transfer",
		ChainID:         "1",
		BlockNumber:     100,
		BlockHash:       common.HexToHash("0x123"),
		TransactionHash: common.HexToHash("0x456"),
		ContractAddress: common.HexToAddress("0x789"),
	}
	err := store.InsertEvent(ctx, event)
	if err != nil {
		t.Errorf("InsertEvent() unexpected error: %v", err)
	}
	retrieved, err := store.GetEvent(ctx, "test-event-1")
	if err != nil {
		t.Errorf("GetEvent() unexpected error: %v", err)
	}
	if retrieved == nil || retrieved.ID != "test-event-1" {
		t.Error("GetEvent() returned wrong event")
	}
}

func TestEventStoreInsertInvalidEvent(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	_ = store.InsertEvent(ctx, nil)
	_ = store.InsertEvent(ctx, &core.BlockchainEvent{ID: ""})
}

func TestEventStoreInsertBatch(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	events := []*core.BlockchainEvent{
		{ID: "batch-1", EventName: "Transfer", ChainID: "1"},
		{ID: "batch-2", EventName: "Transfer", ChainID: "1"},
		{ID: "batch-3", EventName: "Approval", ChainID: "1"},
	}
	err := store.InsertEventBatch(ctx, events)
	if err != nil {
		t.Errorf("InsertEventBatch() unexpected error: %v", err)
	}
}

func TestEventStoreGetEventsByChain(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	_ = store.InsertEvent(ctx, &core.BlockchainEvent{ID: "chain1-1", ChainID: "1"})
	_ = store.InsertEvent(ctx, &core.BlockchainEvent{ID: "chain1-2", ChainID: "1"})
	_ = store.InsertEvent(ctx, &core.BlockchainEvent{ID: "chain2-1", ChainID: "2"})
	events, err := store.GetEventsByChain(ctx, 1, 10, 0)
	if err != nil {
		t.Errorf("GetEventsByChain() unexpected error: %v", err)
	}
	_ = events
}

func TestEventStorePagination(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	for i := 0; i < 10; i++ {
		_ = store.InsertEvent(ctx, &core.BlockchainEvent{ID: "event-" + string(rune('0'+i))})
	}
	events, hasMore, err := store.GetEventsPaginated(ctx, "", 3)
	if err != nil {
		t.Errorf("GetEventsPaginated() unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("GetEventsPaginated() returned %d events, expected 3", len(events))
	}
	if !hasMore {
		t.Error("GetEventsPaginated() should have more events")
	}
}

func TestEventStoreHealth(t *testing.T) {
	t.Parallel()
	store := NewMockEventStoreDomain()
	ctx := context.Background()
	_ = store.Initialize(ctx)
	health := store.Health(ctx)
	if health == nil {
		t.Fatal("Health() returned nil")
	}
	if health.Status != "healthy" {
		t.Errorf("Health() returned status %s, expected healthy", health.Status)
	}
}

func TestRequestStruct(t *testing.T) {
	t.Parallel()
	req := &Request{
		QueryType:  "events",
		Collection: "transfers",
		Filter:     map[string]any{"chain_id": "1"},
		Limit:      100,
		Offset:     0,
		CacheKey:   "cache-key",
		CacheTTL:   5 * time.Minute,
		Sort:       map[string]int{"timestamp": -1},
	}
	if req.QueryType != "events" {
		t.Errorf("Request.QueryType = %s, expected events", req.QueryType)
	}
	if req.Limit != 100 {
		t.Errorf("Request.Limit = %d, expected 100", req.Limit)
	}
}

func TestResultStruct(t *testing.T) {
	t.Parallel()
	result := &Result{
		Events:       []core.BlockchainEvent{{ID: "1"}},
		Total:        1,
		CacheHit:     false,
		ResponseTime: 100,
		Source:       "database",
	}
	if result.Total != 1 {
		t.Errorf("Result.Total = %d, expected 1", result.Total)
	}
	if result.CacheHit {
		t.Error("Result.CacheHit should be false")
	}
	if result.Source != "database" {
		t.Errorf("Result.Source = %s, expected database", result.Source)
	}
}
