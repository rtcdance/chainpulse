package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

func TestMonolithicIndexingEventStoreFiltersByContractAndName(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(core.NewTestLogger())
	require.NoError(t, db.Initialize(core.Config{}))
	require.NoError(t, db.Start())

	first := &core.BlockchainEvent{
		ID:              "evt-1",
		ChainID:         "ethereum",
		BlockNumber:     12,
		LogIndex:        1,
		TransactionHash: common.HexToHash("0x1"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000011"),
		EventName:       "Transfer",
		CreatedAt:       time.Unix(1700000000, 0).UTC(),
	}
	second := &core.BlockchainEvent{
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

	store := NewMonolithicIndexingEventStore(db, core.NewTestLogger(), core.NewDefaultMetricsCollector())
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
	db := NewMonolithicMemoryDatabase(core.NewTestLogger())
	require.NoError(t, db.Initialize(core.Config{}))
	require.NoError(t, db.Start())

	event := &core.BlockchainEvent{
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

	service := NewMonolithicIndexingDomainQueryService(db, core.NewTestLogger(), core.NewDefaultMetricsCollector())
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
