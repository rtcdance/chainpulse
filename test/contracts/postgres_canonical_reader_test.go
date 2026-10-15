//go:build integration

package contracts

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestCanonicalReaderDetailsAndStatsExcludeOrphans(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set PostgreSQL integration gate and isolated DSN")
	}
	truncateIndexingTables(t, dsn)
	ctx := context.Background()
	a := newAdapter(t, dsn).WithChainScope("31337", "31338")
	one := txTestEvent("31337", 1)
	two := txTestEvent("31338", 1)
	orphan := txTestEvent("31337", 2)
	orphan.LogIndex = 1
	orphan.ID = core.ComputeEventHash(orphan)
	orphan.EventHash = orphan.ID
	orphan.Status = blockchain.EventStatusReorged
	for _, event := range []*blockchain.BlockchainEvent{one, two, orphan} {
		require.NoError(t, a.StoreEvent(ctx, event))
	}
	reader := bootstrap.NewMonolithicIndexingEventStore(a, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector())
	require.NoError(t, reader.Initialize(ctx))
	got, err := reader.GetEvent(ctx, one.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = reader.GetEvent(ctx, orphan.ID)
	require.NoError(t, err)
	require.Nil(t, got)
	count, err := reader.CountEvents(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	byChain, byName, reorged, err := reader.GetEventStats(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"31337": 1, "31338": 1}, byChain)
	require.Equal(t, int64(2), byName["Transfer"])
	require.Equal(t, int64(1), reorged)
	events, err := reader.GetEventsByChain(ctx, 31338, 10, 0)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "31338", events[0].ChainID)
	a.WithChainScope("31337")
	foreign, err := a.QueryDomainEvents(ctx, &core.QueryRequest{Filter: map[string]any{"chainId": "31338"}, Limit: 10})
	require.NoError(t, err)
	require.Empty(t, foreign.Events, "request filters cannot widen the adapter's declared chain scope")
	got, err = reader.GetEvent(ctx, two.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}
