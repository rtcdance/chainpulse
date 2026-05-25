package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDataPuller(t *testing.T) {
	t.Parallel()
	config := DataPullerConfig{
		ChainType:    EVM,
		ChainID:      "ethereum",
		StartBlock:   100,
		BatchSize:    50,
		PollInterval: time.Second,
	}
	dp := NewDataPuller(config)

	assert.NotNil(t, dp)
	assert.Equal(t, config, dp.config)
	assert.Equal(t, uint64(100), dp.currentBlock)
	assert.NotNil(t, dp.eventChan)
	assert.NotNil(t, dp.errorChan)
	assert.NotNil(t, dp.metrics)
	assert.False(t, dp.running)
}

func TestNewDataPuller_DefaultStartBlock(t *testing.T) {
	t.Parallel()
	config := DataPullerConfig{
		ChainType: EVM,
		ChainID:   "ethereum",
	}
	dp := NewDataPuller(config)
	assert.Equal(t, uint64(0), dp.currentBlock)
}

func TestDataPuller_StartAndStop(t *testing.T) {
	config := DataPullerConfig{
		ChainType:    EVM,
		ChainID:      "ethereum",
		StartBlock:   100,
		BatchSize:    10,
		PollInterval: time.Hour,
	}
	dp := NewDataPuller(config)
	ctx := context.Background()

	err := dp.Start(ctx)
	require.NoError(t, err)
	assert.True(t, dp.running)

	err = dp.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	err = dp.Stop()
	require.NoError(t, err)
	assert.False(t, dp.running)
}

func TestDataPuller_StopIdempotent(t *testing.T) {
	config := DataPullerConfig{
		ChainType:    EVM,
		ChainID:      "ethereum",
		PollInterval: time.Hour,
	}
	dp := NewDataPuller(config)
	ctx := context.Background()

	err := dp.Start(ctx)
	require.NoError(t, err)

	err = dp.Stop()
	require.NoError(t, err)

	err = dp.Stop()
	require.NoError(t, err)
}

func TestDataPuller_GetLatestBlock(t *testing.T) {
	config := DataPullerConfig{
		ChainType: EVM,
		ChainID:   "ethereum",
	}
	dp := NewDataPuller(config)
	ctx := context.Background()

	_, err := dp.GetLatestBlock(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	dp.mutex.Lock()
	dp.running = true
	dp.currentBlock = 200
	dp.mutex.Unlock()
	defer func() {
		dp.mutex.Lock()
		dp.running = false
		dp.mutex.Unlock()
	}()

	block, err := dp.GetLatestBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(200), block)
}

func TestDataPuller_GetProcessedHeight(t *testing.T) {
	config := DataPullerConfig{
		ChainType:  EVM,
		ChainID:    "ethereum",
		StartBlock: 0,
	}
	dp := NewDataPuller(config)
	ctx := context.Background()

	_, err := dp.GetProcessedHeight(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	dp.mutex.Lock()
	dp.running = true
	dp.mutex.Unlock()
	defer func() {
		dp.mutex.Lock()
		dp.running = false
		dp.mutex.Unlock()
	}()

	height, err := dp.GetProcessedHeight(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), height)
}

func TestDataPuller_GetEventsAndErrors(t *testing.T) {
	config := DataPullerConfig{
		ChainType: EVM,
		ChainID:   "ethereum",
	}
	dp := NewDataPuller(config)

	eventChan := dp.GetEvents()
	assert.NotNil(t, eventChan)

	errChan := dp.GetErrors()
	assert.NotNil(t, errChan)
}

func TestDataPuller_Health(t *testing.T) {
	config := DataPullerConfig{
		ChainType: EVM,
		ChainID:   "ethereum",
	}
	dp := NewDataPuller(config)

	health := dp.Health()
	assert.Equal(t, "unhealthy", health.Status)

	dp.mutex.Lock()
	dp.running = true
	dp.mutex.Unlock()

	health = dp.Health()
	assert.Equal(t, "healthy", health.Status)
}

func TestDataPuller_simulatePullEvents(t *testing.T) {
	t.Parallel()
	config := DataPullerConfig{
		ChainType: EVM,
		ChainID:   "ethereum",
		BatchSize: 5,
	}
	dp := NewDataPuller(config)

	events := dp.simulatePullEvents(100)
	require.Len(t, events, 5)

	for i, ev := range events {
		assert.Equal(t, "ethereum", ev.ChainID)
		assert.Equal(t, "Transfer", ev.EventName)
		assert.Equal(t, uint64(100+i), ev.BlockNumber)
		assert.Equal(t, uint(i), ev.LogIndex)
		assert.Equal(t, "pending", ev.Status)
	}
}

func TestDataPuller_pullEvents(t *testing.T) {
	config := DataPullerConfig{
		ChainType: EVM,
		ChainID:   "ethereum",
		BatchSize: 3,
	}
	dp := NewDataPuller(config)

	dp.pullEvents(context.Background())
	assert.Equal(t, uint64(0), dp.currentBlock)

	dp.mutex.Lock()
	dp.running = true
	dp.currentBlock = 100
	dp.mutex.Unlock()

	dp.pullEvents(context.Background())

	dp.mutex.RLock()
	assert.GreaterOrEqual(t, dp.currentBlock, uint64(100))
	dp.mutex.RUnlock()
}

func TestDataPullerMetrics(t *testing.T) {
	t.Parallel()
	m := NewDataPullerMetrics()
	assert.NotNil(t, m)

	m.RecordEventPulled()
	m.RecordEventPulled()
	m.RecordEventDropped()
	m.RecordError()

	metrics := m.GetMetrics()
	assert.Equal(t, int64(2), metrics["events_pulled"])
	assert.Equal(t, int64(1), metrics["events_dropped"])
	assert.Equal(t, int64(1), metrics["errors"])
}

func TestDataPullerMetrics_Concurrent(t *testing.T) {
	m := NewDataPullerMetrics()
	done := make(chan struct{})

	for i := 0; i < 50; i++ {
		go func() {
			m.RecordEventPulled()
			m.RecordEventDropped()
			m.RecordError()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	metrics := m.GetMetrics()
	assert.Equal(t, int64(50), metrics["events_pulled"])
	assert.Equal(t, int64(50), metrics["events_dropped"])
	assert.Equal(t, int64(50), metrics["errors"])
}
