package bootstrap

import (
	"sync"
	"testing"
	"testing/synctest"
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
	synctest.Test(t, func(t *testing.T) {
		d := NewDrainController()
		require.True(t, d.Begin())
		result := make(chan bool, 1)
		go func() { result <- d.Drain(5 * time.Second) }()
		// All drain goroutines have reached their wait points. No real sleeps or
		// assumption that a statement after End executes before Drain returns.
		synctest.Wait()
		assert.True(t, d.Draining())
		assert.Len(t, result, 0, "drain must remain blocked while work is registered")
		assert.Equal(t, int64(1), d.InFlight())
		d.End()
		synctest.Wait()
		require.True(t, <-result, "drain completes after End releases the work")
		assert.Equal(t, int64(0), d.InFlight())
	})
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
