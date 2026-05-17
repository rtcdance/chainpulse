package bootstrap

import (
	"context"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonolithicMemoryDatabaseLifecycleAndStore(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(core.NewDefaultLogger(core.LogLevelInfo))

	require.NoError(t, db.Initialize(core.Config{}))
	require.NoError(t, db.Start())

	event := &core.BlockchainEvent{
		ID:              "event1",
		ChainID:         "ethereum",
		BlockNumber:     100,
		TransactionHash: common.HexToHash("0x1234"),
	}
	require.NoError(t, db.StoreEvent(context.Background(), event))

	stored, err := db.GetEvent(context.Background(), "event1")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, uint64(100), stored.BlockNumber)

	latest, err := db.GetLatestBlock(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(100), latest)
}

func TestMonolithicMemoryDatabaseStoreBlockSnapshot(t *testing.T) {
	t.Parallel()
	db := NewMonolithicMemoryDatabase(core.NewDefaultLogger(core.LogLevelInfo))

	require.NoError(t, db.Initialize(core.Config{}))
	require.NoError(t, db.Start())

	block := &core.Block{
		Number: 88,
		Hash:   common.HexToHash("0xabcd"),
	}
	require.NoError(t, db.StoreBlockSnapshot(context.Background(), block))

	stored, err := db.GetBlock(context.Background(), 88)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, common.HexToHash("0xabcd"), stored.Hash)

	latest, err := db.GetLatestBlock(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(88), latest)
}

func TestMonolithicMemoryCacheLifecycleAndSetGet(t *testing.T) {
	t.Parallel()
	cache := NewMonolithicMemoryCache()

	require.NoError(t, cache.Initialize(core.Config{}))
	require.NoError(t, cache.Start())
	require.NoError(t, cache.Set(context.Background(), "event:key", []byte("value"), 60))

	value, err := cache.Get(context.Background(), "event:key")
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), value)
}
