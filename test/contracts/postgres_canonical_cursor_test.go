//go:build integration

package contracts

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestCanonicalCursorContinuesBelowAnchorAndRejectsOrphan(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set PostgreSQL integration gate and isolated DSN")
	}
	truncateIndexingTables(t, dsn)
	ctx := context.Background()
	store := newAdapter(t, dsn).WithChainScope("31337")
	write := func(height uint64) {
		event := txTestEvent("31337", height)
		event.BlockHash = common.BigToHash(new(big.Int).SetUint64(height))
		event.ID = core.ComputeEventHash(event)
		event.EventHash = event.ID
		require.NoError(t, store.StoreEvent(ctx, event))
	}
	for height := uint64(1); height <= 3; height++ {
		write(height)
	}
	reader := bootstrap.NewMonolithicIndexingEventStore(store, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector())
	require.NoError(t, reader.Initialize(ctx))
	first, more, err := reader.GetEventsPaginated(ctx, "", 1)
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, uint64(3), first[0].BlockNumber)
	cursor := domainquery.EncodePageCursor(first[0].BlockNumber, first[0].LogIndex, first[0].ID)
	write(4)
	second, more, err := reader.GetEventsPaginated(ctx, cursor, 1)
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, uint64(2), second[0].BlockNumber)
	next := domainquery.EncodePageCursor(second[0].BlockNumber, second[0].LogIndex, second[0].ID)
	last, more, err := reader.GetEventsPaginated(ctx, next, 1)
	require.NoError(t, err)
	require.False(t, more)
	require.Equal(t, uint64(1), last[0].BlockNumber)
	_, err = store.MarkEventsAsReorgedForChain(ctx, "31337", 3, 3)
	require.NoError(t, err)
	_, _, err = reader.GetEventsPaginated(ctx, cursor, 1)
	require.ErrorContains(t, err, "canonical event")
	legacy, _, err := reader.GetEventsPaginated(ctx, "1", 1)
	require.NoError(t, err)
	require.Equal(t, uint64(2), legacy[0].BlockNumber)
	_, _, err = reader.GetEventsPaginated(ctx, "100001", 1)
	require.Error(t, err)
}
