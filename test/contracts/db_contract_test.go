//go:build integration

package contracts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chainpulse/pkg/core"
)

// DatabaseContractTest defines the contract that all database implementations must satisfy
func DatabaseContractTest(t *testing.T, factory func(t *testing.T) core.DatabasePlugin) {
	t.Run("store_and_get_event", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		event := &core.BlockchainEvent{
			ID:        "test-event-1",
			ChainID:   "ethereum",
			BlockNumber: 100,
		}

		err := db.StoreEvent(ctx, event)
		require.NoError(t, err)

		retrieved, err := db.GetEvent(ctx, "test-event-1")
		require.NoError(t, err)
		assert.Equal(t, event.ID, retrieved.ID)
		assert.Equal(t, event.ChainID, retrieved.ChainID)
		assert.Equal(t, event.BlockNumber, retrieved.BlockNumber)
	})

	t.Run("batch_store_events", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		events := []interface{}{
			&core.BlockchainEvent{ID: "batch-1", ChainID: "ethereum", BlockNumber: 1},
			&core.BlockchainEvent{ID: "batch-2", ChainID: "ethereum", BlockNumber: 2},
			&core.BlockchainEvent{ID: "batch-3", ChainID: "ethereum", BlockNumber: 3},
		}

		err := db.BatchStoreEvents(ctx, events)
		require.NoError(t, err)

		// Verify all events stored
		for _, e := range events {
			event := e.(*core.BlockchainEvent)
			retrieved, err := db.GetEvent(ctx, event.ID)
			require.NoError(t, err)
			assert.Equal(t, event.ID, retrieved.ID)
		}
	})

	t.Run("get_nonexistent_event", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		_, err := db.GetEvent(ctx, "nonexistent")
		assert.Error(t, err)
	})

	t.Run("delete_event", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		event := &core.BlockchainEvent{
			ID:        "delete-test",
			ChainID:   "ethereum",
			BlockNumber: 100,
		}

		err := db.StoreEvent(ctx, event)
		require.NoError(t, err)

		err = db.DeleteEvent(ctx, "delete-test")
		require.NoError(t, err)

		_, err = db.GetEvent(ctx, "delete-test")
		assert.Error(t, err)
	})

	t.Run("get_events_by_block_range", func(t *testing.T) {
		db := factory(t)
		ctx := context.Background()

		// Store events at different blocks
		for i := uint64(10); i <= 20; i++ {
			event := &core.BlockchainEvent{
				ID:          "range-event-" + string(rune(i)),
				ChainID:     "ethereum",
				BlockNumber: i,
			}
			err := db.StoreEvent(ctx, event)
			require.NoError(t, err)
		}

		events, err := db.GetEventsByBlockRange(ctx, 12, 15)
		require.NoError(t, err)
		assert.Len(t, events, 4) // blocks 12, 13, 14, 15
	})
}
