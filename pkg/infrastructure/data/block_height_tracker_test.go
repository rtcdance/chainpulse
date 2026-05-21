package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBlockHeightTracker tests creating a new block height tracker
func TestNewBlockHeightTracker(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()

	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.records)
	assert.Equal(t, 0, len(tracker.records))
}

// TestInitializeBlockHeightTracker tests initializing block height tracking
func TestInitializeBlockHeightTracker(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)

	assert.NoError(t, err)
	record, _ := tracker.GetRecord(ctx, "ethereum")
	assert.Equal(t, "ethereum", record.ChainID)
	assert.Equal(t, uint64(1000), record.LastBlockHeight)
	assert.Equal(t, "initialized", record.Status)
}

// TestInitializeMultipleChains tests initializing multiple chains
func TestInitializeMultipleChains(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	err = tracker.Initialize(ctx, "polygon", 2000)
	require.NoError(t, err)
	err = tracker.Initialize(ctx, "arbitrum", 3000)
	require.NoError(t, err)

	eth, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)
	poly, err := tracker.GetBlockHeight(ctx, "polygon")
	require.NoError(t, err)
	arb, err := tracker.GetBlockHeight(ctx, "arbitrum")
	require.NoError(t, err)

	assert.Equal(t, uint64(1000), eth)
	assert.Equal(t, uint64(2000), poly)
	assert.Equal(t, uint64(3000), arb)
}

// TestInitializeDuplicate tests initializing duplicate chain
func TestInitializeDuplicate(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	err = tracker.Initialize(ctx, "ethereum", 2000)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already tracked")
}

// TestUpdateBlockHeight tests updating block height
func TestUpdateBlockHeight(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	err = tracker.UpdateBlockHeight(ctx, "ethereum", 1100)

	assert.NoError(t, err)
	height, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, uint64(1100), height)
}

// TestUpdateBlockHeightNotInitialized tests updating non-existent chain
func TestUpdateBlockHeightNotInitialized(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.UpdateBlockHeight(ctx, "ethereum", 1100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not tracked")
}

// TestGetBlockHeight tests getting block height
func TestGetBlockHeight(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	_ = tracker.Initialize(ctx, "ethereum", 1000)
	height, err := tracker.GetBlockHeight(ctx, "ethereum")

	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), height)
}

// TestGetBlockHeightNotFound tests getting non-existent chain
func TestGetBlockHeightNotFound(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	_, err := tracker.GetBlockHeight(ctx, "ethereum")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not tracked")
}

// TestGetRecord tests getting block height record
func TestGetRecord(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	_ = tracker.Initialize(ctx, "ethereum", 1000)
	record, err := tracker.GetRecord(ctx, "ethereum")

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "ethereum", record.ChainID)
	assert.Equal(t, uint64(1000), record.LastBlockHeight)
}

// TestGetAllRecords tests getting all records
func TestGetAllRecords(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	err = tracker.Initialize(ctx, "polygon", 2000)
	require.NoError(t, err)

	records := tracker.GetAllRecords()

	assert.Equal(t, 2, len(records))
	assert.NotNil(t, records["ethereum"])
	assert.NotNil(t, records["polygon"])
}

// TestBlockHeightRecordStatus tests record status updates
func TestBlockHeightRecordStatus(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	record, err := tracker.GetRecord(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "initialized", record.Status)

	err = tracker.UpdateBlockHeight(ctx, "ethereum", 1100)
	require.NoError(t, err)
	record, err = tracker.GetRecord(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, "updated", record.Status)
}

// TestBlockHeightRecordTimestamp tests record timestamp updates
func TestBlockHeightRecordTimestamp(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	record1, err := tracker.GetRecord(ctx, "ethereum")
	require.NoError(t, err)
	time1 := record1.LastProcessedAt

	time.Sleep(10 * time.Millisecond)
	err = tracker.UpdateBlockHeight(ctx, "ethereum", 1100)
	require.NoError(t, err)
	record2, err := tracker.GetRecord(ctx, "ethereum")
	require.NoError(t, err)
	time2 := record2.LastProcessedAt

	assert.True(t, time2.After(time1))
}

// TestNewBlockHeightSynchronizer tests creating a new synchronizer
func TestNewBlockHeightSynchronizer(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	cache := &RedisClusterManager{}
	sync := NewBlockHeightSynchronizer(tracker, cache)

	assert.NotNil(t, sync)
	assert.Equal(t, tracker, sync.tracker)
	assert.Equal(t, cache, sync.cache)
}

// TestSyncBlockHeight tests syncing block height
func TestSyncBlockHeight(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	cache := &RedisClusterManager{}
	sync := NewBlockHeightSynchronizer(tracker, cache)
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	err = sync.SyncBlockHeight(ctx, "ethereum", 1100)

	assert.NoError(t, err)
	height, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, uint64(1100), height)
}

// TestGetSyncedBlockHeight tests getting synced block height
func TestGetSyncedBlockHeight(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	cache := &RedisClusterManager{}
	sync := NewBlockHeightSynchronizer(tracker, cache)
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	height, err := sync.GetSyncedBlockHeight(ctx, "ethereum")

	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), height)
}

// TestNewRecoveryManager tests creating a new recovery manager
func TestNewRecoveryManager(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	manager := NewRecoveryManager(tracker)

	assert.NotNil(t, manager)
	assert.Equal(t, tracker, manager.tracker)
}

// TestRecoverFromLastCheckpoint tests recovering from last checkpoint
func TestRecoverFromLastCheckpoint(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	manager := NewRecoveryManager(tracker)
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	height, err := manager.RecoverFromLastCheckpoint(ctx, "ethereum")

	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), height)
}

// TestCreateCheckpoint tests creating a checkpoint
func TestCreateCheckpoint(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	manager := NewRecoveryManager(tracker)
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)
	err = manager.CreateCheckpoint(ctx, "ethereum", 1500)

	assert.NoError(t, err)
	height, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, uint64(1500), height)
}

// TestValidateCheckpoint tests validating a checkpoint
func TestValidateCheckpoint(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	manager := NewRecoveryManager(tracker)
	ctx := context.Background()

	_ = tracker.Initialize(ctx, "ethereum", 1000)
	valid, err := manager.ValidateCheckpoint(ctx, "ethereum", 1000)

	assert.NoError(t, err)
	assert.True(t, valid)
}

// TestValidateCheckpointInvalid tests validating an invalid checkpoint
func TestValidateCheckpointInvalid(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	manager := NewRecoveryManager(tracker)
	ctx := context.Background()

	_ = tracker.Initialize(ctx, "ethereum", 1000)
	valid, err := manager.ValidateCheckpoint(ctx, "ethereum", 2000)

	assert.NoError(t, err)
	assert.False(t, valid)
}

// TestBlockHeightTrackerConcurrentUpdates tests concurrent updates
func TestBlockHeightTrackerConcurrentUpdates(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 1000)
	require.NoError(t, err)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			_ = tracker.UpdateBlockHeight(ctx, "ethereum", uint64(1000+index))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	height, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, height, uint64(1000))
	assert.LessOrEqual(t, height, uint64(1009))
}

// TestBlockHeightTrackerLargeBlockNumbers tests with large block numbers
func TestBlockHeightTrackerLargeBlockNumbers(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	largeBlock := uint64(18446744073709551615) // Max uint64
	err := tracker.Initialize(ctx, "ethereum", largeBlock)
	require.NoError(t, err)
	height, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)

	assert.Equal(t, largeBlock, height)
}

// TestBlockHeightTrackerZeroBlock tests with zero block height
func TestBlockHeightTrackerZeroBlock(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	err := tracker.Initialize(ctx, "ethereum", 0)
	require.NoError(t, err)
	height, err := tracker.GetBlockHeight(ctx, "ethereum")
	require.NoError(t, err)

	assert.Equal(t, uint64(0), height)
}

// TestBlockHeightTrackerMultipleChainsConcurrent tests concurrent operations on multiple chains
func TestBlockHeightTrackerMultipleChainsConcurrent(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	chains := []string{"ethereum", "polygon", "arbitrum", "optimism"}
	for i, chain := range chains {
		err := tracker.Initialize(ctx, chain, uint64(1000+i*1000))
		require.NoError(t, err)
	}

	done := make(chan bool, len(chains)*5)
	for _, chain := range chains {
		for i := 0; i < 5; i++ {
			go func(c string, idx int) {
				_ = tracker.UpdateBlockHeight(ctx, c, uint64(2000+idx))
				done <- true
			}(chain, i)
		}
	}

	for i := 0; i < len(chains)*5; i++ {
		<-done
	}

	records := tracker.GetAllRecords()
	assert.Equal(t, len(chains), len(records))
}

// TestBlockHeightRecordFields tests all fields of block height record
func TestBlockHeightRecordFields(t *testing.T) {
	t.Parallel()
	tracker := NewBlockHeightTracker()
	ctx := context.Background()

	_ = tracker.Initialize(ctx, "ethereum", 1000)
	record, _ := tracker.GetRecord(ctx, "ethereum")

	assert.NotEmpty(t, record.ChainID)
	assert.NotZero(t, record.LastBlockHeight)
	assert.False(t, record.LastProcessedAt.IsZero())
	assert.NotEmpty(t, record.Status)
}
