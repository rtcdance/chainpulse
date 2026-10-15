//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

func TestPostgresQueryScopesChainAndExcludesOrphans(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}
	a := newAdapter(t, dsn)
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`DELETE FROM blockchain_events WHERE chain_id IN ($1,$2)`, "query-delivery-a", "query-delivery-b")
	require.NoError(t, err)
	for i, chain := range []string{"query-delivery-a", "query-delivery-b"} {
		event := &blockchain.BlockchainEvent{ChainID: chain, BlockNumber: 10, BlockHash: common.HexToHash("0x01"), TransactionHash: common.BigToHash(common.Big1), LogIndex: uint64(i), EventName: "Transfer"}
		event.ID = core.ComputeEventHash(event)
		event.EventHash = event.ID
		require.NoError(t, a.StoreEvent(ctx, event))
	}
	rows, err := a.QueryEvents(ctx, &core.EventFilter{Network: "query-delivery-a", Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "query-delivery-a", rows[0].(*blockchain.BlockchainEvent).ChainID)
	id := rows[0].(*blockchain.BlockchainEvent).ID
	_, err = a.GetEvent(ctx, id)
	require.NoError(t, err)
	_, err = a.MarkEventsAsReorgedForChain(ctx, "query-delivery-a", 10, 10)
	require.NoError(t, err)
	audited, err := a.GetEvent(ctx, id)
	require.NoError(t, err)
	require.Equal(t, blockchain.EventStatusReorged, audited.Status, "ID reads must not reuse stale pre-reorg cache entries")
	rows, err = a.QueryEvents(ctx, &core.EventFilter{Network: "query-delivery-a", Limit: 10})
	require.NoError(t, err)
	require.Empty(t, rows)
	rows, err = a.QueryEvents(ctx, &core.EventFilter{Network: "query-delivery-a", Status: []string{"reorged"}, Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1, "explicit historical queries retain orphan auditability")
	a.WithChainScope("query-delivery-a", "query-delivery-b")
	_, err = a.QueryDomainEvents(ctx, &domainquery.Request{Limit: 10})
	require.Error(t, err, "multi-chain queries must specify their chain")
	page, err := a.QueryDomainEvents(ctx, &domainquery.Request{Filter: map[string]any{"chainId": "query-delivery-a"}, Limit: 1})
	require.NoError(t, err)
	require.Empty(t, page.Events, "authoritative API reads exclude orphan effects")
	page, err = a.QueryDomainEvents(ctx, &domainquery.Request{Filter: map[string]any{"chainId": "query-delivery-b"}, Limit: 1})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total)
	require.Len(t, page.Events, 1)
	page, err = a.QueryDomainEvents(ctx, &domainquery.Request{Filter: map[string]any{"chainId": "query-delivery-b"}, Limit: 1, Offset: 1})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total, "total count is not merely current-page length")
	require.Empty(t, page.Events)
}
