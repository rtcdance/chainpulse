package bootstrap

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func TestMonolithicIndexingEventStoreFiltersByContractAndName(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))

	first := &blockchain.BlockchainEvent{
		ID:              "evt-1",
		ChainID:         "ethereum",
		BlockNumber:     12,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0x1"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000011"),
		EventName:       "Transfer",
		CreatedAt:       time.Unix(1700000000, 0).UTC(),
	}
	second := &blockchain.BlockchainEvent{
		ID:              "evt-2",
		ChainID:         "polygon",
		BlockNumber:     20,
		LogIndex:        2,
		TransactionHash: common.HexToHash("0x2"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000022"),
		EventName:       "Approval",
		CreatedAt:       time.Unix(1700000001, 0).UTC(),
	}
	require.NoError(t, db.StoreEvent(context.Background(), first))
	require.NoError(t, db.StoreEvent(context.Background(), second))

	store := NewMonolithicIndexingEventStore(db, testhelpers.NewTestLogger(), core.NewDefaultMetricsCollector())
	require.NoError(t, store.Initialize(context.Background()))

	contractEvents, err := store.GetEventsByContract(context.Background(), "0x0000000000000000000000000000000000000011", 10, 0)
	require.NoError(t, err)
	require.Len(t, contractEvents, 1)
	require.Equal(t, "evt-1", contractEvents[0].ID)

	nameEvents, err := store.GetEventsByEventName(context.Background(), "approval", 10, 0)
	require.NoError(t, err)
	require.Len(t, nameEvents, 1)
	require.Equal(t, "evt-2", nameEvents[0].ID)
}

func TestMonolithicIndexingDomainQueryServiceReadsFromIndexingDatabase(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))

	event := &blockchain.BlockchainEvent{
		ID:              "evt-3",
		ChainID:         "ethereum",
		BlockNumber:     42,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x123"),
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		EventName:       "Transfer",
		CreatedAt:       time.Unix(1700000002, 0).UTC(),
	}
	require.NoError(t, db.StoreEvent(context.Background(), event))

	service := NewMonolithicIndexingDomainQueryService(db, testhelpers.NewTestLogger(), core.NewDefaultMetricsCollector())
	result, err := service.Query(context.Background(), &domainquery.Request{
		Collection: "events",
		Filter: map[string]any{
			"eventName": "Transfer",
		},
		Limit:  10,
		Offset: 0,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Events, 1)
	require.Equal(t, "monolithic-indexing", result.Source)
	require.Equal(t, "evt-3", result.Events[0].ID)

	found, err := service.QueryByHash(context.Background(), event.TransactionHash.Hex())
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, "evt-3", found.ID)
}

func TestSaturatingUint64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint64
		want  int64
	}{
		{"zero", 0, 0},
		{"normal_value", 42, 42},
		{"max_int64", math.MaxInt64, math.MaxInt64},
		{"max_int64_plus_one", uint64(math.MaxInt64) + 1, math.MaxInt64},
		{"max_uint64", math.MaxUint64, math.MaxInt64},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := saturatingUint64ToInt64(tc.input)
			if got != tc.want {
				t.Fatalf("saturatingUint64ToInt64(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestSafeInt64ToSliceIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  int64
		length int
		want   int
		wantOk bool
	}{
		{"valid_index_0", 0, 10, 0, true},
		{"valid_index_middle", 5, 10, 5, true},
		{"valid_index_last", 9, 10, 9, true},
		{"negative_value", -1, 10, 0, false},
		{"exceeds_MaxInt", math.MaxInt64, 10, 0, false},
		{"index_equals_length", 10, 10, 0, false},
		{"index_exceeds_length", 100, 10, 0, false},
		{"empty_slice", 0, 0, 0, false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := safeInt64ToSliceIndex(tc.value, tc.length)
			if ok != tc.wantOk {
				t.Fatalf("safeInt64ToSliceIndex(%d, %d) ok = %v, want %v", tc.value, tc.length, ok, tc.wantOk)
			}
			if got != tc.want {
				t.Fatalf("safeInt64ToSliceIndex(%d, %d) = %d, want %d", tc.value, tc.length, got, tc.want)
			}
		})
	}
}

func TestSafeInt64ToSliceBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		limit  int64
		offset int
		length int
		want   int
		wantOk bool
	}{
		{"normal_range", 5, 0, 10, 5, true},
		{"limit_exceeds_length", 15, 0, 10, 10, true},
		{"offset_plus_limit_within", 3, 5, 10, 8, true},
		{"offset_plus_limit_exceeds", 10, 5, 10, 10, true},
		{"zero_limit", 0, 0, 10, 0, false},
		{"negative_limit", -1, 0, 10, 0, false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := safeInt64ToSliceBound(tc.limit, tc.offset, tc.length)
			if ok != tc.wantOk {
				t.Fatalf("safeInt64ToSliceBound(%d, %d, %d) ok = %v, want %v", tc.limit, tc.offset, tc.length, ok, tc.wantOk)
			}
			if got != tc.want {
				t.Fatalf("safeInt64ToSliceBound(%d, %d, %d) = %d, want %d", tc.limit, tc.offset, tc.length, got, tc.want)
			}
		})
	}
}

func TestBuildSyntheticMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("fully populated event", func(t *testing.T) {
		t.Parallel()
		event := &blockchain.BlockchainEvent{
			ID:              "evt-abc",
			ChainID:         "1",
			BlockNumber:     100,
			TransactionHash: common.HexToHash("0xabcdef"),
			LogIndex:        2,
			ContractAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			EventName:       "Transfer",
			ProcessedAt:     now,
			CreatedAt:       now.Add(-time.Hour),
			IndexedAt:       now.Add(-30 * time.Minute),
			Status:          "confirmed",
		}

		meta := buildSyntheticMetadata(event)
		if meta.EventID != "evt-abc" {
			t.Fatalf("EventID = %q, want evt-abc", meta.EventID)
		}
		if meta.ChainID != 1 {
			t.Fatalf("ChainID = %d, want 1", meta.ChainID)
		}
		if meta.BlockNumber != 100 {
			t.Fatalf("BlockNumber = %d, want 100", meta.BlockNumber)
		}
		if meta.TransactionHash != "0x0000000000000000000000000000000000000000000000000000000000abcdef" {
			t.Fatalf("TransactionHash = %q", meta.TransactionHash)
		}
		if meta.LogIndex != 2 {
			t.Fatalf("LogIndex = %d, want 2", meta.LogIndex)
		}
		if meta.EventName != "Transfer" {
			t.Fatalf("EventName = %q, want Transfer", meta.EventName)
		}
		if !meta.ProcessedAt.Equal(now) {
			t.Fatalf("ProcessedAt = %v, want %v", meta.ProcessedAt, now)
		}
		if meta.ProcessingStatus != "confirmed" {
			t.Fatalf("ProcessingStatus = %q, want confirmed", meta.ProcessingStatus)
		}
	})

	t.Run("event with zero times falls back", func(t *testing.T) {
		t.Parallel()
		event := &blockchain.BlockchainEvent{
			ID:              "evt-zero-times",
			ChainID:         "ethereum",
			BlockNumber:     50,
			TransactionHash: common.HexToHash("0xdead"),
			ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
			EventName:       "Approval",
			Status:          "pending",
			BlockTimestamp:  1700000000,
		}

		meta := buildSyntheticMetadata(event)
		if meta.ChainID != 0 {
			t.Fatalf("ChainID for non-numeric chain = %d, want 0", meta.ChainID)
		}
		if meta.EventName != "Approval" {
			t.Fatalf("EventName = %q, want Approval", meta.EventName)
		}
	})
}

func TestPaginateEvents(t *testing.T) {
	t.Parallel()

	makeEvents := func(n int) []*blockchain.BlockchainEvent {
		events := make([]*blockchain.BlockchainEvent, n)
		for i := 0; i < n; i++ {
			events[i] = &blockchain.BlockchainEvent{ID: fmt.Sprintf("evt-%d", i)}
		}
		return events
	}

	t.Run("limit within range", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(10)
		result := paginateEvents(events, 3, 0)
		if len(result) != 3 {
			t.Fatalf("len = %d, want 3", len(result))
		}
		if result[2].ID != "evt-2" {
			t.Fatalf("last = %q, want evt-2", result[2].ID)
		}
	})

	t.Run("offset beyond length", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(5)
		result := paginateEvents(events, 3, 10)
		if len(result) != 0 {
			t.Fatalf("len = %d, want 0", len(result))
		}
	})

	t.Run("limit zero returns all from offset", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(5)
		result := paginateEvents(events, 0, 2)
		if len(result) != 3 {
			t.Fatalf("len = %d, want 3", len(result))
		}
	})

	t.Run("negative limit returns all from offset", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(5)
		result := paginateEvents(events, -1, 0)
		if len(result) != 5 {
			t.Fatalf("len = %d, want 5", len(result))
		}
	})
}

func TestPaginateDomainEvents(t *testing.T) {
	t.Parallel()

	makeEvents := func(n int) []blockchain.BlockchainEvent {
		events := make([]blockchain.BlockchainEvent, n)
		for i := 0; i < n; i++ {
			events[i] = blockchain.BlockchainEvent{ID: fmt.Sprintf("evt-%d", i)}
		}
		return events
	}

	t.Run("normal pagination", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(10)
		result := paginateDomainEvents(events, 5, 0)
		if len(result) != 5 {
			t.Fatalf("len = %d, want 5", len(result))
		}
	})

	t.Run("offset beyond length", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(3)
		result := paginateDomainEvents(events, 5, 10)
		if len(result) != 0 {
			t.Fatalf("len = %d, want 0", len(result))
		}
	})

	t.Run("limit exceeds available", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(3)
		result := paginateDomainEvents(events, 10, 0)
		if len(result) != 3 {
			t.Fatalf("len = %d, want 3", len(result))
		}
	})

	t.Run("offset and limit", func(t *testing.T) {
		t.Parallel()
		events := makeEvents(10)
		result := paginateDomainEvents(events, 3, 5)
		if len(result) != 3 {
			t.Fatalf("len = %d, want 3", len(result))
		}
		if result[0].ID != "evt-5" {
			t.Fatalf("first = %q, want evt-5", result[0].ID)
		}
	})
}

func TestMatchesDomainQueryFilter(t *testing.T) {
	t.Parallel()

	event := &blockchain.BlockchainEvent{
		ChainID:         "1",
		ContractAddress: common.HexToAddress("0x1234567890ABCDEF1234567890ABCDEF12345678"),
		EventName:       "Transfer",
	}

	t.Run("empty filter matches all", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{}) {
			t.Fatal("empty filter should match")
		}
	})

	t.Run("chainId string match", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{"chainId": "1"}) {
			t.Fatal("chainId string should match")
		}
	})

	t.Run("chainId int match", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{"chainId": 1}) {
			t.Fatal("chainId int should match")
		}
	})

	t.Run("chainId int64 match", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{"chainId": int64(1)}) {
			t.Fatal("chainId int64 should match")
		}
	})

	t.Run("chainId case insensitive match", func(t *testing.T) {
		t.Parallel()
		evt := &blockchain.BlockchainEvent{ChainID: "ethereum", ContractAddress: common.HexToAddress("0x1234567890ABCDEF1234567890ABCDEF12345678"), EventName: "Transfer"}
		if !matchesDomainQueryFilter(evt, map[string]any{"chainId": "Ethereum"}) {
			t.Fatal("chainId case insensitive should match")
		}
	})

	t.Run("contractAddress match", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{"contractAddress": "0x1234567890ABCDEF1234567890ABCDEF12345678"}) {
			t.Fatal("contractAddress should match")
		}
	})

	t.Run("eventName match", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{"eventName": "Transfer"}) {
			t.Fatal("eventName should match")
		}
	})

	t.Run("eventName case insensitive match", func(t *testing.T) {
		t.Parallel()
		if !matchesDomainQueryFilter(event, map[string]any{"eventName": "transfer"}) {
			t.Fatal("eventName case insensitive should match")
		}
	})

	t.Run("chainId mismatch", func(t *testing.T) {
		t.Parallel()
		if matchesDomainQueryFilter(event, map[string]any{"chainId": "137"}) {
			t.Fatal("chainId mismatch should not match")
		}
	})

	t.Run("contractAddress mismatch", func(t *testing.T) {
		t.Parallel()
		if matchesDomainQueryFilter(event, map[string]any{"contractAddress": "0xDEAD"}) {
			t.Fatal("contractAddress mismatch should not match")
		}
	})

	t.Run("eventName mismatch", func(t *testing.T) {
		t.Parallel()
		if matchesDomainQueryFilter(event, map[string]any{"eventName": "Approval"}) {
			t.Fatal("eventName mismatch should not match")
		}
	})
}

func TestMonolithicIndexingEventStoreFullCoverage(t *testing.T) {
	t.Parallel()

	db := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))

	store := NewMonolithicIndexingEventStore(db, testhelpers.NewTestLogger(), core.NewDefaultMetricsCollector())
	require.NoError(t, store.Initialize(context.Background()))

	now := time.Now().UTC()

	event1 := &blockchain.BlockchainEvent{
		ID:              "evt-101",
		ChainID:         "1",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0xa1"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000011"),
		EventName:       "Transfer",
		CorrelationID:   "corr-1",
		CreatedAt:       now,
		ProcessedAt:     now,
		Status:          "confirmed",
	}
	event2 := &blockchain.BlockchainEvent{
		ID:              "evt-102",
		ChainID:         "137",
		BlockNumber:     100,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0xa2"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000011"),
		EventName:       "Transfer",
		CorrelationID:   "corr-1",
		CreatedAt:       now.Add(time.Minute),
		ProcessedAt:     now.Add(time.Minute),
		Status:          "confirmed",
	}
	event3 := &blockchain.BlockchainEvent{
		ID:              "evt-103",
		ChainID:         "1",
		BlockNumber:     200,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0xa3"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000022"),
		EventName:       "Approval",
		CreatedAt:       now.Add(2 * time.Minute),
		ProcessedAt:     now.Add(2 * time.Minute),
		Status:          "pending",
	}

	t.Run("insert event and batch", func(t *testing.T) {
		require.NoError(t, store.InsertEvent(context.Background(), event1))
		require.NoError(t, store.InsertEventBatch(context.Background(), []*blockchain.BlockchainEvent{event2, event3}))
	})

	t.Run("get event by id", func(t *testing.T) {
		found, err := store.GetEvent(context.Background(), "evt-101")
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, "evt-101", found.ID)

		notFound, err := store.GetEvent(context.Background(), "nonexistent")
		require.NoError(t, err)
		require.Nil(t, notFound)
	})

	t.Run("get events by chain", func(t *testing.T) {
		events, err := store.GetEventsByChain(context.Background(), 1, 10, 0)
		require.NoError(t, err)
		require.Len(t, events, 2)
	})

	t.Run("get events by chain with chainID zero returns all", func(t *testing.T) {
		events, err := store.GetEventsByChain(context.Background(), 0, 10, 0)
		require.NoError(t, err)
		require.Len(t, events, 3)
	})

	t.Run("get events by block", func(t *testing.T) {
		events, err := store.GetEventsByBlock(context.Background(), 100)
		require.NoError(t, err)
		require.Len(t, events, 2)

		events, err = store.GetEventsByBlock(context.Background(), 200)
		require.NoError(t, err)
		require.Len(t, events, 1)
	})

	t.Run("get events by address", func(t *testing.T) {
		events, err := store.GetEventsByAddress(context.Background(), "0x0000000000000000000000000000000000000011", 10)
		require.NoError(t, err)
		require.Len(t, events, 2)
	})

	t.Run("get events by name", func(t *testing.T) {
		events, err := store.GetEventsByName(context.Background(), "Transfer", 10)
		require.NoError(t, err)
		require.Len(t, events, 2)
	})

	t.Run("get events paginated", func(t *testing.T) {
		events, hasMore, err := store.GetEventsPaginated(context.Background(), "", 2)
		require.NoError(t, err)
		require.Len(t, events, 2)
		require.True(t, hasMore)

		events, hasMore, err = store.GetEventsPaginated(context.Background(), "", 5)
		require.NoError(t, err)
		require.Len(t, events, 3)
		require.False(t, hasMore)
	})

	t.Run("get events paginated with cursor", func(t *testing.T) {
		cursor := domainquery.EncodePageCursor(200, 0, "evt-103")
		events, _, err := store.GetEventsPaginated(context.Background(), cursor, 5)
		require.NoError(t, err)
		require.True(t, len(events) >= 2)
	})

	t.Run("get events by correlation id", func(t *testing.T) {
		events, err := store.GetEventsByCorrelationID(context.Background(), "corr-1", 10, 0)
		require.NoError(t, err)
		require.Len(t, events, 2)
	})

	t.Run("count events", func(t *testing.T) {
		count, err := store.CountEvents(context.Background())
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
	})

	t.Run("delete expired events", func(t *testing.T) {
		deleted, err := store.DeleteExpiredEvents(context.Background())
		require.NoError(t, err)
		require.Equal(t, int64(0), deleted)
	})

	t.Run("health check", func(t *testing.T) {
		health := store.Health(context.Background())
		require.NotNil(t, health)
		require.Equal(t, "healthy", health.Status)
	})

	t.Run("close", func(t *testing.T) {
		require.NoError(t, store.Close(context.Background()))
	})
}

func TestMonolithicIndexingMetadataStore_Lifecycle(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))
	store := NewMonolithicIndexingMetadataStore(db)

	// Initialize
	require.NoError(t, store.Initialize(context.Background()))
	require.True(t, store.initialized)

	// Double init should fail
	err := store.Initialize(context.Background())
	require.ErrorContains(t, err, "already initialized")

	// InsertMetadata (no-op)
	require.NoError(t, store.InsertMetadata(context.Background(), nil))

	// InsertMetadataBatch (no-op)
	require.NoError(t, store.InsertMetadataBatch(context.Background(), nil))

	// GetMetadata — not found
	meta, err := store.GetMetadata(context.Background(), "nonexistent")
	require.NoError(t, err)
	require.Nil(t, meta)

	// GetMetadataByChain — empty result with valid chain
	metas, err := store.GetMetadataByChain(context.Background(), 1, 10, 0)
	require.NoError(t, err)
	require.Empty(t, metas)

	// GetMetadataBatch (no-op)
	batch, err := store.GetMetadataBatch(context.Background(), []string{"a", "b"})
	require.NoError(t, err)
	require.Nil(t, batch)

	// UpdateMetadata (no-op)
	require.NoError(t, store.UpdateMetadata(context.Background(), nil))

	// Health
	health := store.Health(context.Background())
	require.Equal(t, "healthy", health.Status)

	// Close (no-op)
	require.NoError(t, store.Close(context.Background()))
}

func TestMonolithicIndexingMetadataStore_Initialize_NilDatabase(t *testing.T) {
	t.Parallel()
	store := NewMonolithicIndexingMetadataStore(nil)
	err := store.Initialize(context.Background())
	require.ErrorContains(t, err, "database plugin is required")
}

func TestMonolithicIndexingMetadataStore_Health_Unhealthy(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))
	store := NewMonolithicIndexingMetadataStore(db)

	// Without Initialize, GetMetadata should return error
	storeNoInit := NewMonolithicIndexingMetadataStore(db)
	_, err := storeNoInit.GetMetadata(context.Background(), "x")
	require.ErrorContains(t, err, "not initialized")

	// GetMetadataByChain without Initialize
	_, err = storeNoInit.GetMetadataByChain(context.Background(), 0, 10, 0)
	require.ErrorContains(t, err, "not initialized")

	// Healthy store works
	require.NoError(t, store.Initialize(context.Background()))
	hc := store.Health(context.Background())
	require.Equal(t, "healthy", hc.Status)
}

func TestMonolithicIndexingMetadataStore_GetMetadataByChain_Limits(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))
	store := NewMonolithicIndexingMetadataStore(db)
	require.NoError(t, store.Initialize(context.Background()))

	// Insert some events first
	for i := 0; i < 5; i++ {
		evt := &blockchain.BlockchainEvent{
			ID:        fmt.Sprintf("evt-metadata-%d", i),
			ChainID:   "1",
			CreatedAt: time.Now(),
		}
		require.NoError(t, db.StoreEvent(context.Background(), evt))
	}

	// Limit = 2
	metas, err := store.GetMetadataByChain(context.Background(), 1, 2, 0)
	require.NoError(t, err)
	require.Len(t, metas, 2)

	// Offset beyond result
	metas, err = store.GetMetadataByChain(context.Background(), 1, 10, 100)
	require.NoError(t, err)
	require.Empty(t, metas)

	// Limit = 0 (no limit)
	metas, err = store.GetMetadataByChain(context.Background(), 0, 0, 0)
	require.NoError(t, err)
	require.NotEmpty(t, metas)
}

func TestMonolithicIndexingDomainQueryService_QueryByHash(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewTestLogger()
	db := NewMonolithicMemoryDatabase(log)
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))

	srv := NewMonolithicIndexingDomainQueryService(db, log, nil)

	// Empty hash
	_, err := srv.QueryByHash(context.Background(), "")
	require.ErrorContains(t, err, "hash is required")

	// Not found — empty database
	evt, err := srv.QueryByHash(context.Background(), "nonexistent")
	require.NoError(t, err)
	require.Nil(t, evt)

	evt = &blockchain.BlockchainEvent{
		ID:              "evt-qbh-1",
		TransactionHash: common.HexToHash("0xabcd"),
		EventHash:       "event-hash-abc",
	}
	require.NoError(t, db.StoreEvent(context.Background(), evt))

	// Find by TransactionHash
	txHash := common.HexToHash("0xabcd")
	found, err := srv.QueryByHash(context.Background(), txHash.Hex())
	require.NoError(t, err)
	require.NotNil(t, found)

	// Find by EventHash
	found, err = srv.QueryByHash(context.Background(), "event-hash-abc")
	require.NoError(t, err)
	require.NotNil(t, found)

	// Find by ID
	found, err = srv.QueryByHash(context.Background(), "evt-qbh-1")
	require.NoError(t, err)
	require.NotNil(t, found)
}

func TestMonolithicIndexingDomainQueryService_Query_NilRequest(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewTestLogger()
	db := NewMonolithicMemoryDatabase(log)
	srv := NewMonolithicIndexingDomainQueryService(db, log, nil)
	_, err := srv.Query(context.Background(), nil)
	require.ErrorContains(t, err, "query request is required")
}

func TestMonolithicIndexingDomainQueryService_InvalidateCacheAndHealth(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewTestLogger()
	db := NewMonolithicMemoryDatabase(log)
	require.NoError(t, db.Initialize(context.Background(), core.Config{}))
	require.NoError(t, db.Start(context.Background()))
	srv := NewMonolithicIndexingDomainQueryService(db, log, nil)

	require.NoError(t, srv.InvalidateCache(context.Background(), "any"))

	health := srv.Health(context.Background())
	require.Equal(t, "healthy", health.Status)
}
