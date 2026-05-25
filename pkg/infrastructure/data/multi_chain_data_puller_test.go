package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMultiChainDataPuller(t *testing.T) {
	t.Parallel()
	mcdp := NewMultiChainDataPuller()
	assert.NotNil(t, mcdp)
	assert.NotNil(t, mcdp.pullers)
	assert.Empty(t, mcdp.pullers)
}

func TestMultiChainDataPuller_AddPuller(t *testing.T) {
	t.Parallel()
	mcdp := NewMultiChainDataPuller()
	puller := NewDataPuller(DataPullerConfig{ChainID: "ethereum"})

	err := mcdp.AddPuller("ethereum", puller)
	require.NoError(t, err)

	err = mcdp.AddPuller("ethereum", puller)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMultiChainDataPuller_RemovePuller(t *testing.T) {
	t.Parallel()
	mcdp := NewMultiChainDataPuller()
	puller := NewDataPuller(DataPullerConfig{ChainID: "ethereum"})

	err := mcdp.RemovePuller("ethereum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_ = mcdp.AddPuller("ethereum", puller)
	err = mcdp.RemovePuller("ethereum")
	require.NoError(t, err)

	_, err = mcdp.GetPuller("ethereum")
	assert.Error(t, err)
}

func TestMultiChainDataPuller_GetPuller(t *testing.T) {
	t.Parallel()
	mcdp := NewMultiChainDataPuller()

	_, err := mcdp.GetPuller("ethereum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	puller := NewDataPuller(DataPullerConfig{ChainID: "ethereum"})
	_ = mcdp.AddPuller("ethereum", puller)

	found, err := mcdp.GetPuller("ethereum")
	require.NoError(t, err)
	assert.Equal(t, puller, found)
}

func TestMultiChainDataPuller_StartAll(t *testing.T) {
	mcdp := NewMultiChainDataPuller()
	ethPuller := NewDataPuller(DataPullerConfig{ChainID: "ethereum", PollInterval: time.Hour})
	polyPuller := NewDataPuller(DataPullerConfig{ChainID: "polygon", PollInterval: time.Hour})

	require.NoError(t, mcdp.AddPuller("ethereum", ethPuller))
	require.NoError(t, mcdp.AddPuller("polygon", polyPuller))

	ctx := context.Background()
	err := mcdp.StartAll(ctx)
	require.NoError(t, err)

	assert.True(t, ethPuller.running)
	assert.True(t, polyPuller.running)

	_ = mcdp.StopAll()
}

func TestMultiChainDataPuller_StopAll(t *testing.T) {
	mcdp := NewMultiChainDataPuller()
	puller := NewDataPuller(DataPullerConfig{ChainID: "ethereum", PollInterval: time.Hour})

	require.NoError(t, mcdp.AddPuller("ethereum", puller))
	require.NoError(t, mcdp.StartAll(context.Background()))
	err := mcdp.StopAll()
	require.NoError(t, err)
	assert.False(t, puller.running)
}

func TestMultiChainDataPuller_GetAllMetrics(t *testing.T) {
	mcdp := NewMultiChainDataPuller()
	puller := NewDataPuller(DataPullerConfig{ChainID: "ethereum"})
	puller.metrics.RecordEventPulled()
	puller.metrics.RecordEventPulled()
	puller.metrics.RecordError()

	require.NoError(t, mcdp.AddPuller("ethereum", puller))

	metrics := mcdp.GetAllMetrics()
	assert.Contains(t, metrics, "ethereum")
	assert.Equal(t, int64(2), metrics["ethereum"]["events_pulled"])
	assert.Equal(t, int64(1), metrics["ethereum"]["errors"])
}

func TestMultiChainDataPuller_HealthAll(t *testing.T) {
	mcdp := NewMultiChainDataPuller()
	ethPuller := NewDataPuller(DataPullerConfig{ChainID: "ethereum"})
	polyPuller := NewDataPuller(DataPullerConfig{ChainID: "polygon"})

	ethPuller.mutex.Lock()
	ethPuller.running = true
	ethPuller.mutex.Unlock()

	require.NoError(t, mcdp.AddPuller("ethereum", ethPuller))
	require.NoError(t, mcdp.AddPuller("polygon", polyPuller))

	health := mcdp.HealthAll()
	assert.Equal(t, "healthy", health["ethereum"].Status)
	assert.Equal(t, "unhealthy", health["polygon"].Status)
}

func TestMultiChainDataPuller_ConcurrentAddRemove(t *testing.T) {
	mcdp := NewMultiChainDataPuller()
	done := make(chan struct{})

	addPuller := func(id string) {
		p := NewDataPuller(DataPullerConfig{ChainID: id})
		_ = mcdp.AddPuller(id, p)
		done <- struct{}{}
	}

	for _, id := range []string{"a", "b", "c", "d", "e"} {
		go addPuller(id)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	assert.Len(t, mcdp.pullers, 5)
}
