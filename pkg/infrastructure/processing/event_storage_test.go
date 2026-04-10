package processing

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewInMemoryEventStore tests store creation
func TestNewInMemoryEventStore(t *testing.T) {
	store := NewInMemoryEventStore(1000)

	assert.NotNil(t, store)
	assert.NotNil(t, store.events)
	assert.Equal(t, 1000, store.maxSize)
	assert.Equal(t, 0, store.GetSize())
}

// TestStoreEvent tests storing a single event
func TestStoreEvent(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{
		ID:              "event-1",
		EventHash:       "hash-1",
		ChainID:         "ethereum",
		ContractAddress: "0x1234",
		EventName:       "Transfer",
		BlockNumber:     100,
		Timestamp:       time.Now(),
		Status:          "confirmed",
	}

	err := store.StoreEvent(ctx, event)

	assert.NoError(t, err)
	assert.Equal(t, 1, store.GetSize())
	assert.Equal(t, int64(1), store.GetMetrics().EventsStored)
}

// TestStoreEventNil tests storing nil event
func TestStoreEventNil(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	err := store.StoreEvent(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, 0, store.GetSize())
	assert.Equal(t, int64(1), store.GetMetrics().StorageErrors)
}

// TestStoreEventEmptyID tests storing event with empty ID
func TestStoreEventEmptyID(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{
		ID: "",
	}

	err := store.StoreEvent(ctx, event)

	assert.Error(t, err)
	assert.Equal(t, 0, store.GetSize())
}

// TestStoreEventFull tests storing when store is full
func TestStoreEventFull(t *testing.T) {
	store := NewInMemoryEventStore(2)
	ctx := context.Background()

	event1 := &Event{ID: "event-1"}
	event2 := &Event{ID: "event-2"}
	event3 := &Event{ID: "event-3"}

	_ = store.StoreEvent(ctx, event1)
	_ = store.StoreEvent(ctx, event2)

	err := store.StoreEvent(ctx, event3)

	assert.Error(t, err)
	assert.Equal(t, 2, store.GetSize())
}

// TestStoreBatch tests storing a batch of events
func TestStoreBatch(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1"},
		{ID: "event-2"},
		{ID: "event-3"},
	}

	err := store.StoreBatch(ctx, events)

	assert.NoError(t, err)
	assert.Equal(t, 3, store.GetSize())
	assert.Equal(t, int64(3), store.GetMetrics().EventsStored)
	assert.Equal(t, int64(1), store.GetMetrics().BatchesProcessed)
}

// TestStoreBatchEmpty tests storing empty batch
func TestStoreBatchEmpty(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	err := store.StoreBatch(ctx, []*Event{})

	assert.NoError(t, err)
	assert.Equal(t, 0, store.GetSize())
}

// TestStoreBatchExceedsCapacity tests storing batch that exceeds capacity
func TestStoreBatchExceedsCapacity(t *testing.T) {
	store := NewInMemoryEventStore(2)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1"},
		{ID: "event-2"},
		{ID: "event-3"},
	}

	err := store.StoreBatch(ctx, events)

	assert.Error(t, err)
	assert.Equal(t, 0, store.GetSize())
}

// TestGetEvent tests retrieving a single event
func TestGetEvent(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{
		ID:        "event-1",
		EventName: "Transfer",
	}

	_ = store.StoreEvent(ctx, event)

	retrieved, err := store.GetEvent(ctx, "event-1")

	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "event-1", retrieved.ID)
	assert.Equal(t, "Transfer", retrieved.EventName)
	assert.Equal(t, int64(1), store.GetMetrics().EventsRetrieved)
}

// TestGetEventEmptyID tests retrieving with empty ID
func TestGetEventEmptyID(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	_, err := store.GetEvent(ctx, "")

	assert.Error(t, err)
}

// TestGetEventNotFound tests retrieving non-existent event
func TestGetEventNotFound(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	_, err := store.GetEvent(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestQueryEventsNoFilter tests querying with no filter
func TestQueryEventsNoFilter(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", EventName: "Transfer"},
		{ID: "event-2", EventName: "Approval"},
		{ID: "event-3", EventName: "Transfer"},
	}

	_ = store.StoreBatch(ctx, events)

	results, err := store.QueryEvents(ctx, nil)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))
}

// TestQueryEventsByChainID tests querying by chain ID
func TestQueryEventsByChainID(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", ChainID: "ethereum"},
		{ID: "event-2", ChainID: "polygon"},
		{ID: "event-3", ChainID: "ethereum"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{ChainID: "ethereum"}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

// TestQueryEventsByEventName tests querying by event name
func TestQueryEventsByEventName(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", EventName: "Transfer"},
		{ID: "event-2", EventName: "Approval"},
		{ID: "event-3", EventName: "Transfer"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{EventName: "Transfer"}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

// TestQueryEventsByBlockRange tests querying by block range
func TestQueryEventsByBlockRange(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", BlockNumber: 100},
		{ID: "event-2", BlockNumber: 200},
		{ID: "event-3", BlockNumber: 300},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{FromBlock: 150, ToBlock: 250}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "event-2", results[0].ID)
}

// TestQueryEventsByStatus tests querying by status
func TestQueryEventsByStatus(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", Status: "confirmed"},
		{ID: "event-2", Status: "pending"},
		{ID: "event-3", Status: "confirmed"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{Status: "confirmed"}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

// TestQueryEventsWithLimit tests querying with limit
func TestQueryEventsWithLimit(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1"},
		{ID: "event-2"},
		{ID: "event-3"},
		{ID: "event-4"},
		{ID: "event-5"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{Limit: 2}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

// TestQueryEventsWithOffset tests querying with offset
func TestQueryEventsWithOffset(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1"},
		{ID: "event-2"},
		{ID: "event-3"},
		{ID: "event-4"},
		{ID: "event-5"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{Offset: 2, Limit: 2}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

// TestQueryEventsMultipleFilters tests querying with multiple filters
func TestQueryEventsMultipleFilters(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", ChainID: "ethereum", EventName: "Transfer", Status: "confirmed"},
		{ID: "event-2", ChainID: "ethereum", EventName: "Approval", Status: "confirmed"},
		{ID: "event-3", ChainID: "polygon", EventName: "Transfer", Status: "confirmed"},
		{ID: "event-4", ChainID: "ethereum", EventName: "Transfer", Status: "pending"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{
		ChainID:   "ethereum",
		EventName: "Transfer",
		Status:    "confirmed",
	}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "event-1", results[0].ID)
}

// TestDeleteEvent tests deleting an event
func TestDeleteEvent(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{ID: "event-1"}
	_ = store.StoreEvent(ctx, event)

	assert.Equal(t, 1, store.GetSize())

	err := store.DeleteEvent(ctx, "event-1")

	assert.NoError(t, err)
	assert.Equal(t, 0, store.GetSize())
	assert.Equal(t, int64(1), store.GetMetrics().EventsDeleted)
}

// TestDeleteEventEmptyID tests deleting with empty ID
func TestDeleteEventEmptyID(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	err := store.DeleteEvent(ctx, "")

	assert.Error(t, err)
}

// TestDeleteEventNotFound tests deleting non-existent event
func TestDeleteEventNotFound(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	err := store.DeleteEvent(ctx, "nonexistent")

	assert.Error(t, err)
}

// TestGetMetricsEventStorage tests getting metrics
func TestGetMetricsEventStorage(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{ID: "event-1"}
	_ = store.StoreEvent(ctx, event)
	_, _ = store.GetEvent(ctx, "event-1")

	metrics := store.GetMetrics()

	assert.Equal(t, int64(1), metrics.EventsStored)
	assert.Equal(t, int64(1), metrics.EventsRetrieved)
	assert.Equal(t, int64(0), metrics.EventsDeleted)
}

// TestClear tests clearing the store
func TestClear(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1"},
		{ID: "event-2"},
		{ID: "event-3"},
	}

	_ = store.StoreBatch(ctx, events)
	assert.Equal(t, 3, store.GetSize())

	store.Clear()

	assert.Equal(t, 0, store.GetSize())
}

// TestConcurrentStoreEvent tests concurrent storing
func TestConcurrentStoreEvent(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := &Event{ID: "event-" + strconv.Itoa(id)}
			if err := store.StoreEvent(ctx, event); err == nil {
				atomic.AddInt32(&counter, 1)
			}
		}(i)
	}

	wg.Wait()

	// All 100 events should be stored successfully
	assert.Equal(t, int32(100), atomic.LoadInt32(&counter))
}

// TestConcurrentGetEvent tests concurrent retrieval
func TestConcurrentGetEvent(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{ID: "event-1"}
	_ = store.StoreEvent(ctx, event)

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.GetEvent(ctx, "event-1"); err == nil {
				atomic.AddInt32(&counter, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&counter))
}

// TestEventFilterByContractAddress tests filtering by contract address
func TestEventFilterByContractAddress(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1", ContractAddress: "0x1111"},
		{ID: "event-2", ContractAddress: "0x2222"},
		{ID: "event-3", ContractAddress: "0x1111"},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{ContractAddress: "0x1111"}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

// TestEventFilterByTimeRange tests filtering by time range
func TestEventFilterByTimeRange(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	now := time.Now()
	events := []*Event{
		{ID: "event-1", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "event-2", Timestamp: now},
		{ID: "event-3", Timestamp: now.Add(2 * time.Hour)},
	}

	_ = store.StoreBatch(ctx, events)

	filter := &EventFilter{
		FromTime: now.Add(-1 * time.Hour),
		ToTime:   now.Add(1 * time.Hour),
	}
	results, err := store.QueryEvents(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
}

// TestNewTransactionManager tests transaction manager creation
func TestNewTransactionManager(t *testing.T) {
	tm := NewTransactionManager()

	assert.NotNil(t, tm)
	assert.NotNil(t, tm.activeTransactions)
	assert.Equal(t, int64(0), tm.commitCount)
	assert.Equal(t, int64(0), tm.rollbackCount)
}

// TestBeginTransaction tests beginning a transaction
func TestBeginTransaction(t *testing.T) {
	tm := NewTransactionManager()

	tx := tm.BeginTransaction("tx-1")

	assert.NotNil(t, tx)
	assert.Equal(t, "tx-1", tx.ID)
	assert.Equal(t, "active", tx.Status)
	assert.NotZero(t, tx.StartTime)
}

// TestCommitTransaction tests committing a transaction
func TestCommitTransaction(t *testing.T) {
	tm := NewTransactionManager()

	_ = tm.BeginTransaction("tx-1")
	err := tm.CommitTransaction("tx-1")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), tm.commitCount)
}

// TestCommitTransactionNotFound tests committing non-existent transaction
func TestCommitTransactionNotFound(t *testing.T) {
	tm := NewTransactionManager()

	err := tm.CommitTransaction("nonexistent")

	assert.Error(t, err)
}

// TestRollbackTransaction tests rolling back a transaction
func TestRollbackTransaction(t *testing.T) {
	tm := NewTransactionManager()

	_ = tm.BeginTransaction("tx-1")
	err := tm.RollbackTransaction("tx-1")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), tm.rollbackCount)
}

// TestRollbackTransactionNotFound tests rolling back non-existent transaction
func TestRollbackTransactionNotFound(t *testing.T) {
	tm := NewTransactionManager()

	err := tm.RollbackTransaction("nonexistent")

	assert.Error(t, err)
}

// TestTransactionManagerMetrics tests getting transaction metrics
func TestTransactionManagerMetrics(t *testing.T) {
	tm := NewTransactionManager()

	_ = tm.BeginTransaction("tx-1")
	_ = tm.CommitTransaction("tx-1")

	_ = tm.BeginTransaction("tx-2")
	_ = tm.RollbackTransaction("tx-2")

	metrics := tm.GetMetrics()

	assert.Equal(t, int64(1), metrics["commit_count"])
	assert.Equal(t, int64(1), metrics["rollback_count"])
	assert.Equal(t, 0, metrics["active_transactions"])
}

// TestStorageMetricsLatency tests latency recording
func TestStorageMetricsLatency(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{ID: "event-1"}
	_ = store.StoreEvent(ctx, event)

	metrics := store.GetMetrics()

	assert.Greater(t, metrics.AverageLatency, time.Duration(0))
	assert.Greater(t, metrics.TotalStorageTime, time.Duration(0))
}

// TestMultipleTransactions tests multiple transactions
func TestMultipleTransactions(t *testing.T) {
	tm := NewTransactionManager()

	for i := 1; i <= 5; i++ {
		txID := "tx-" + string(rune(48+i))
		_ = tm.BeginTransaction(txID)
		_ = tm.CommitTransaction(txID)
	}

	assert.Equal(t, int64(5), tm.commitCount)
}

// TestStoreBatchWithNilEvents tests batch with nil events
func TestStoreBatchWithNilEvents(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	events := []*Event{
		{ID: "event-1"},
		nil,
		{ID: "event-3"},
	}

	err := store.StoreBatch(ctx, events)

	assert.NoError(t, err)
	// Only non-nil events should be stored
	assert.Equal(t, 2, store.GetSize())
}

// TestQueryEventsEmpty tests querying empty store
func TestQueryEventsEmpty(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	results, err := store.QueryEvents(ctx, nil)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

// TestStoreEventLatency tests event storage latency
func TestStoreEventLatency(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	event := &Event{ID: "event-1"}
	_ = store.StoreEvent(ctx, event)

	metrics := store.GetMetrics()

	assert.NotZero(t, metrics.AverageLatency)
	assert.NotZero(t, metrics.LastStorageTime)
}

// TestGetSize tests getting store size
func TestGetSize(t *testing.T) {
	store := NewInMemoryEventStore(1000)
	ctx := context.Background()

	assert.Equal(t, 0, store.GetSize())

	for i := 1; i <= 5; i++ {
		event := &Event{ID: "event-" + string(rune(48+i))}
		_ = store.StoreEvent(ctx, event)
	}

	assert.Equal(t, 5, store.GetSize())
}

// TestTransactionAddEvents tests adding events to transaction
func TestTransactionAddEvents(t *testing.T) {
	tm := NewTransactionManager()

	tx := tm.BeginTransaction("tx-1")
	event := &Event{ID: "event-1"}
	tx.Events = append(tx.Events, event)

	assert.Equal(t, 1, len(tx.Events))
	assert.Equal(t, "event-1", tx.Events[0].ID)
}
