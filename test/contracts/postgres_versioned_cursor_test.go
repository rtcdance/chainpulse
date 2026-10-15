//go:build integration

package contracts

import (
	"context"
	"database/sql"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
	"github.com/stretchr/testify/require"
)

func TestVersionedCursorBindsChainFilterAndCanonicalGeneration(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set PostgreSQL integration gate and isolated DSN")
	}
	truncateIndexingTables(t, dsn)
	ctx := context.Background()
	store := newAdapter(t, dsn).WithChainScope("31337", "31338")
	write := func(height uint64) {
		event := txTestEvent("31337", height)
		event.BlockHash = common.BigToHash(new(big.Int).SetUint64(height))
		event.ID = core.ComputeEventHash(event)
		event.EventHash = event.ID
		require.NoError(t, store.StoreEvent(ctx, event))
	}
	for i := uint64(1); i <= 3; i++ {
		write(i)
	}
	filter := map[string]any{"chainId": "31337", "eventName": "Transfer"}
	first, err := store.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, first.NextCursor)
	write(4)
	second, err := store.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: 1, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Equal(t, uint64(2), second.Events[0].BlockNumber)
	_, err = store.QueryDomainEvents(ctx, &domainquery.Request{Filter: map[string]any{"chainId": "31338", "eventName": "Transfer"}, Limit: 1, Cursor: first.NextCursor})
	require.Error(t, err)
	_, err = store.QueryDomainEvents(ctx, &domainquery.Request{Filter: map[string]any{"chainId": "31337", "eventName": "Swap"}, Limit: 1, Cursor: first.NextCursor})
	require.Error(t, err)
	_, err = store.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: 1, Offset: 1, Cursor: first.NextCursor})
	require.Error(t, err)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `INSERT INTO reorg_generations(chain_id,generation)VALUES('31337',1) ON CONFLICT(chain_id)DO UPDATE SET generation=reorg_generations.generation+1`)
	require.NoError(t, err)
	_, err = store.QueryDomainEvents(ctx, &domainquery.Request{Filter: filter, Limit: 1, Cursor: first.NextCursor})
	require.ErrorContains(t, err, "generation")
}
