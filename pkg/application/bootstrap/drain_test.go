package bootstrap

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrainControllerAcceptsThenRejects(t *testing.T) {
	t.Parallel()

	d := NewDrainController()

	require.True(t, d.Begin(), "work is accepted before draining")
	d.End()
	assert.Equal(t, int64(0), d.InFlight())
	assert.Equal(t, int64(0), d.Rejected())
	assert.False(t, d.Draining())

	require.True(t, d.Drain(time.Second))
	assert.True(t, d.Draining())

	assert.False(t, d.Begin(), "no new work may start once draining")
	assert.False(t, d.Begin())
	assert.Equal(t, int64(2), d.Rejected())
}

func TestDrainControllerWaitsForInFlightWork(t *testing.T) {
	t.Parallel()

	d := NewDrainController()
	require.True(t, d.Begin())

	// A unit of work that finishes shortly after draining starts.
	finished := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		d.End()
		close(finished)
	}()

	// Drain must not return before the in-flight unit completes.
	require.True(t, d.Drain(5*time.Second), "drain must complete once in-flight work finishes")
	select {
	case <-finished:
	default:
		t.Fatal("Drain returned before the in-flight unit finished")
	}
	assert.Equal(t, int64(0), d.InFlight())
}

func TestDrainControllerReportsTimeout(t *testing.T) {
	t.Parallel()

	d := NewDrainController()
	require.True(t, d.Begin())
	// Never call End: the unit is stuck.

	start := time.Now()
	assert.False(t, d.Drain(50*time.Millisecond), "a stuck unit must be reported, not hidden")
	assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond)
	assert.Equal(t, int64(1), d.InFlight())

	d.End() // release so the goroutine can exit
}

func TestDrainControllerConcurrentBeginsAreAccounted(t *testing.T) {
	t.Parallel()

	d := NewDrainController()
	const workers = 50

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if d.Begin() {
				d.End()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(0), d.InFlight(), "every Begin must be matched by an End")
	require.True(t, d.Drain(time.Second))
}
