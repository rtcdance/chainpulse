//go:build integration

package contracts

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// postgresDSN returns the DSN to use, or "" when the integration gate is off
// or no reachable database was provided.
//
// Gate: CHAINPULSE_RUN_POSTGRES_INTEGRATION=1
// DSN:  CHAINPULSE_TEST_DATABASE_URL  (required)
//
// Why the DSN is required rather than defaulted: on this machine a native
// PostgreSQL listens on 127.0.0.1:5432 and shadows the published
// chainpulse-postgres port, while Docker container IPs are not routable from
// the host. Point CHAINPULSE_TEST_DATABASE_URL at whatever is reachable, e.g.
//
//	CHAINPULSE_TEST_DATABASE_URL='postgres://chainpulse:chainpulse_dev@<host>:5432/chainpulse_dev?sslmode=disable'
func postgresDSN() string {
	if os.Getenv("CHAINPULSE_RUN_POSTGRES_INTEGRATION") != "1" {
		return ""
	}
	return os.Getenv("CHAINPULSE_TEST_DATABASE_URL")
}

// TestPostgresEventRoundTrip is the T04 acceptance test for the Postgres write
// path. It is EXPECTED TO BE RED until T04 aligns the adapter with the real
// schema (see docs/project/2026-10-12-T04-postgres-alignment-spec.md): the
// adapter currently INSERTs into column `event_hash`, which does not exist, so
// WriteEvent fails outright.
//
// It is gated behind CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 so the default short
// lane is unaffected, per the plan's rule that real-service tests may be
// skipped in the short lane but MUST run in the full lane.
func TestPostgresEventRoundTrip(t *testing.T) {
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL to run the Postgres round-trip")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pg := database.NewPostgreSQLDatabase(
		testhelpers.NewTestLogger(),
		testhelpers.NewTestMetricsCollector(),
	)
	require.NoError(t, pg.Initialize(ctx, core.Config{DatabaseURL: dsn}), "Initialize")
	require.NoError(t, pg.Start(ctx), "Start")
	t.Cleanup(func() { _ = pg.Stop(context.Background()) })

	// Use a chain id unique to this test so the round-trip does not collide
	// with anything else in the shared dev database.
	chainID := "t04-roundtrip"
	blockHash := common.HexToHash("0xbbbb0000000000000000000000000000000000000000000000000000000000bb")
	txHash := common.HexToHash("0x7777000000000000000000000000000000000000000000000000000000000077")
	contractAddr := common.HexToAddress("0xcccc0000000000000000000000000000000000cc")

	// The `id` column holds the canonical event hash (that is what
	// GetEventByHash assumes), so ID and EventHash are the same value here.
	evHash := core.ComputeEventHash(&blockchain.BlockchainEvent{
		ChainID:         chainID,
		BlockNumber:     100,
		TransactionHash: txHash,
		LogIndex:        7,
	})

	event := &blockchain.BlockchainEvent{
		ID:              evHash,
		EventHash:       evHash,
		ChainID:         chainID,
		BlockNumber:     100,
		BlockHash:       blockHash,
		BlockTimestamp:  1700000000,
		TransactionHash: txHash,
		LogIndex:        7,
		ContractAddress: contractAddr,
		EventName:       "Transfer",
		EventData:       []byte("0xdeadbeef"),
	}

	require.NoError(t, pg.WriteEvent(ctx, event), "WriteEvent must succeed against the real schema")

	// Read back via the range query (does not need an id lookup) and via hash.
	events, err := pg.GetEventsByBlockRange(ctx, 100, 100)
	require.NoError(t, err, "GetEventsByBlockRange")

	var found *blockchain.BlockchainEvent
	for _, e := range events {
		if e.ID == event.ID {
			found = e
			break
		}
	}
	require.NotNil(t, found, "written event must be readable back")
	assert.Equal(t, event.ChainID, found.ChainID)
	assert.Equal(t, event.BlockNumber, found.BlockNumber)
	assert.Equal(t, event.BlockHash.Hex(), found.BlockHash.Hex(), "block_hash must round-trip (I02)")
	assert.Equal(t, event.TransactionHash.Hex(), found.TransactionHash.Hex())
	assert.Equal(t, event.LogIndex, found.LogIndex)
	assert.Equal(t, event.EventName, found.EventName)
	assert.Equal(t, event.EventData, found.EventData, "event_data must round-trip")

	// GetEventByHash (the `id` column holds the canonical hash).
	byHash, err := pg.GetEventByHash(event.EventHash)
	require.NoError(t, err, "GetEventByHash")
	require.NotNil(t, byHash, "GetEventByHash must find the event")
	assert.Equal(t, event.BlockHash.Hex(), byHash.BlockHash.Hex())
	assert.Equal(t, event.ChainID, byHash.ChainID)

	// GetAllEvents.
	all, err := pg.GetAllEvents(ctx)
	require.NoError(t, err, "GetAllEvents")
	assert.NotEmpty(t, all, "GetAllEvents must return the written event")

	// QueryEvents.
	res, err := pg.QueryEvents(&core.EventFilter{FromBlock: 100, ToBlock: 100})
	require.NoError(t, err, "QueryEvents")
	require.NotNil(t, res)

	// Block readers must not fail against the real schema (an empty table is
	// a valid result; a missing column would be an error).
	_, err = pg.GetAllBlocks(ctx)
	require.NoError(t, err, "GetAllBlocks")
	_, err = pg.GetBlock(ctx, 999999)
	require.NoError(t, err, "GetBlock")

	// Cleanup so repeated runs stay idempotent.
	require.NoError(t, pg.DeleteEvent(ctx, event.ID))
}
