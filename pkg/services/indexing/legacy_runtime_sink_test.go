package indexing

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLegacyRuntimeSinkRequiresDatabase(t *testing.T) {
	t.Parallel()
	sink, err := NewLegacyRuntimeSink(nil, nil, NewMockLogger())
	require.Error(t, err)
	assert.Nil(t, sink)
}

func TestLegacyRuntimeSinkPersistStoresEventAndCache(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()

	sink, err := NewLegacyRuntimeSink(db, cache, logger)
	require.NoError(t, err)

	event := &blockchain.BlockchainEvent{
		ID:              "event1",
		ChainID:         "ethereum",
		BlockNumber:     100,
		LogIndex:        3,
		CreatedAt:       time.Unix(1710000000, 0),
		TransactionHash: common.HexToHash("0x1234"),
	}

	err = sink.Persist(context.Background(), []core.EventEnvelope{
		toEventEnvelope(event),
	})
	require.NoError(t, err)

	stored, getErr := db.GetEvent(context.Background(), "event1")
	require.NoError(t, getErr)
	require.NotNil(t, stored)
	assert.False(t, stored.IndexedAt.IsZero())
	assert.False(t, stored.ProcessedAt.IsZero())

	cacheValue, cacheErr := cache.Get(context.Background(), cacheKeyForEvent("ethereum", event))
	require.NoError(t, cacheErr)
	assert.Equal(t, []byte("event1"), cacheValue)
}

func TestLegacyRuntimeSinkPersistAllowsNilCache(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	logger := NewMockLogger()

	sink, err := NewLegacyRuntimeSink(db, nil, logger)
	require.NoError(t, err)

	event := &blockchain.BlockchainEvent{
		ID:              "event1",
		ChainID:         "ethereum",
		BlockNumber:     100,
		LogIndex:        0,
		TransactionHash: common.HexToHash("0x1234"),
	}

	err = sink.Persist(context.Background(), []core.EventEnvelope{
		toEventEnvelope(event),
	})
	require.NoError(t, err)
}

func TestLegacyRuntimeSinkPersistRejectsInvalidPayload(t *testing.T) {
	t.Parallel()
	db := NewMockDatabasePlugin()
	cache := NewMockCachePlugin()
	logger := NewMockLogger()

	sink, err := NewLegacyRuntimeSink(db, cache, logger)
	require.NoError(t, err)

	err = sink.Persist(context.Background(), []core.EventEnvelope{
		{
			EventKey: "bad",
			ChainID:  "ethereum",
			Payload:  "not-an-event",
		},
	})
	require.Error(t, err)
}
