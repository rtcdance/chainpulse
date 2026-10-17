package database

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

func TestAggregateFlightSharesOnlyConcurrentRevision(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var flights aggregateFlights
		var reads atomic.Int64
		entered, release := make(chan struct{}), make(chan struct{})
		load := func() ([]byte, error) {
			reads.Add(1)
			close(entered)
			<-release
			return []byte("42"), nil
		}
		var wg sync.WaitGroup
		wg.Go(func() {
			value, err := flights.read(context.Background(), "revision1", load)
			require.NoError(t, err)
			require.Equal(t, "42", string(value))
		})
		<-entered
		for range 31 {
			wg.Go(func() {
				value, err := flights.read(context.Background(), "revision1", load)
				require.NoError(t, err)
				require.Equal(t, "42", string(value))
			})
		}
		synctest.Wait()
		value, err := flights.read(context.Background(), "revision2", func() ([]byte, error) { return []byte("43"), nil })
		require.NoError(t, err)
		require.Equal(t, "43", string(value))
		close(release)
		wg.Wait()
		require.EqualValues(t, 1, reads.Load())
		require.Empty(t, flights.active)
	})
}

func TestAggregateFlightWaiterCancellationDoesNotCancelOwner(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var flights aggregateFlights
		entered, release := make(chan struct{}), make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			_, err := flights.read(context.Background(), "key", func() ([]byte, error) { close(entered); <-release; return []byte("ok"), nil })
			require.NoError(t, err)
		})
		<-entered
		ctx, cancel := context.WithCancel(context.Background())
		waiter := make(chan error, 1)
		wg.Go(func() {
			_, err := flights.read(ctx, "key", func() ([]byte, error) { panic("waiter cannot become owner") })
			waiter <- err
		})
		synctest.Wait()
		cancel()
		require.ErrorIs(t, <-waiter, context.Canceled)
		close(release)
		wg.Wait()
		require.Empty(t, flights.active)
	})
}

func TestAggregateFlightRetriesSharedOwnerCancellationOnce(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var flights aggregateFlights
		ownerCtx, cancel := context.WithCancel(context.Background())
		entered := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			_, err := flights.read(ownerCtx, "key", func() ([]byte, error) { close(entered); <-ownerCtx.Done(); return nil, ownerCtx.Err() })
			require.ErrorIs(t, err, context.Canceled)
		})
		<-entered
		var healthyReads atomic.Int64
		wg.Go(func() {
			value, err := flights.read(context.Background(), "key", func() ([]byte, error) { healthyReads.Add(1); return []byte("healthy"), nil })
			require.NoError(t, err)
			require.Equal(t, "healthy", string(value))
		})
		synctest.Wait()
		cancel()
		wg.Wait()
		require.EqualValues(t, 1, healthyReads.Load())
		require.Empty(t, flights.active)
	})
}

func TestAggregateFlightStateIsBounded(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var flights aggregateFlights
		release := make(chan struct{})
		var wg sync.WaitGroup
		for index := range 17 {
			wg.Go(func() {
				_, err := flights.read(context.Background(), fmt.Sprint(index), func() ([]byte, error) { <-release; return []byte("ok"), nil })
				require.NoError(t, err)
			})
		}
		synctest.Wait()
		flights.mu.Lock()
		require.Len(t, flights.active, 16)
		flights.mu.Unlock()
		close(release)
		wg.Wait()
		require.Empty(t, flights.active)
	})
}

func TestAggregateFlightPanicReleasesWaiters(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var flights aggregateFlights
		entered, release := make(chan struct{}), make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			defer func() { require.Equal(t, "origin panic", recover()) }()
			_, _ = flights.read(context.Background(), "key", func() ([]byte, error) { close(entered); <-release; panic("origin panic") })
		})
		<-entered
		wg.Go(func() {
			_, err := flights.read(context.Background(), "key", func() ([]byte, error) { panic("must share") })
			require.ErrorContains(t, err, "did not complete")
		})
		synctest.Wait()
		close(release)
		wg.Wait()
		require.Empty(t, flights.active)
	})
}

func TestAggregateFlightDoesNotHideSQLFailureOrCacheDisabled(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"", "revision"} {
		t.Run(fmt.Sprintf("key=%s", key), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				var flights aggregateFlights
				var reads atomic.Int64
				originErr := errors.New("origin unavailable")
				entered, release := make(chan struct{}), make(chan struct{})
				load := func() ([]byte, error) {
					if reads.Add(1) == 1 {
						close(entered)
					}
					<-release
					return nil, originErr
				}
				var wg sync.WaitGroup
				wg.Go(func() { _, err := flights.read(context.Background(), key, load); require.ErrorIs(t, err, originErr) })
				<-entered
				wg.Go(func() { _, err := flights.read(context.Background(), key, load); require.ErrorIs(t, err, originErr) })
				synctest.Wait()
				close(release)
				wg.Wait()
				want := int64(1)
				if key == "" {
					want = 2
				}
				require.Equal(t, want, reads.Load())
				require.Empty(t, flights.active)
			})
		})
	}
}
