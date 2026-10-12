//go:build integration

package contracts

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// testEvent builds a well-formed event with a real physical identity (I02):
// distinct block_hash / transaction_hash / log_index per block, and an
// EventHash computed from them. Sparse fixtures (all-zero hashes) would collide
// on a real database's natural key, so the contract would not exercise the
// persistence path it claims to.
func testEvent(id, chain string, blockNumber, logIndex uint64) *blockchain.BlockchainEvent {
	e := &blockchain.BlockchainEvent{
		ID:              id,
		ChainID:         chain,
		BlockNumber:     blockNumber,
		BlockHash:       common.HexToHash(fmt.Sprintf("0x%064x", blockNumber)),
		TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", blockNumber*1000+logIndex)),
		LogIndex:        logIndex,
		ContractAddress: common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		EventName:       "Transfer",
		BlockTimestamp:  1700000000,
	}
	e.EventHash = core.ComputeEventHash(e)
	return e
}

// DatabaseContractTest defines the contract that all database implementations must satisfy
func DatabaseContractTest(t *testing.T, factory func(t *testing.T) core.DatabasePlugin) {
	t.Run("store_and_get_event", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		event := testEvent("test-event-1", "ethereum", 100, 0)

		err := db.StoreEvent(ctx, event)
		require.NoError(t, err)

		retrieved, err := db.GetEvent(ctx, "test-event-1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, event.ID, retrieved.ID)
		assert.Equal(t, event.ChainID, retrieved.ChainID)
		assert.Equal(t, event.BlockNumber, retrieved.BlockNumber)
	})

	t.Run("batch_store_events", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		events := []any{
			testEvent("batch-1", "ethereum", 1, 0),
			testEvent("batch-2", "ethereum", 2, 0),
			testEvent("batch-3", "ethereum", 3, 0),
		}

		err := db.BatchStoreEvents(ctx, events)
		require.NoError(t, err)

		// Verify all events stored
		for _, e := range events {
			event := e.(*blockchain.BlockchainEvent)
			retrieved, err := db.GetEvent(ctx, event.ID)
			require.NoError(t, err)
			require.NotNil(t, retrieved, "event %s must be readable back", event.ID)
			assert.Equal(t, event.ID, retrieved.ID)
		}
	})

	t.Run("get_nonexistent_event", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		// Convention: a missing entity is (nil, nil), not an error. The query
		// path (event_retrieval_service, GraphQL resolver) branches on
		// `event == nil`, and an error is reserved for real failures.
		event, err := db.GetEvent(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, event)
	})

	t.Run("delete_event", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		event := testEvent("delete-test", "ethereum", 100, 0)

		err := db.StoreEvent(ctx, event)
		require.NoError(t, err)

		err = db.DeleteEvent(ctx, "delete-test")
		require.NoError(t, err)

		deleted, err := db.GetEvent(ctx, "delete-test")
		require.NoError(t, err)
		assert.Nil(t, deleted)
	})

	t.Run("get_events_by_block_range", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		// Store events at different blocks (distinct identity per block).
		for i := uint64(10); i <= 20; i++ {
			event := testEvent(fmt.Sprintf("range-event-%d", i), "ethereum", i, 0)
			err := db.StoreEvent(ctx, event)
			require.NoError(t, err)
		}

		events, err := db.GetEventsByBlockRange(ctx, 12, 15)
		require.NoError(t, err)
		assert.Len(t, events, 4) // blocks 12, 13, 14, 15
	})
}
