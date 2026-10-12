package pullers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blockRangeFetch returns the block numbers in the range, refusing any range
// wider than limit.
func blockRangeFetch(limit uint64, calls *[]string) func(context.Context, uint64, uint64) ([]uint64, error) {
	return func(_ context.Context, from, to uint64) ([]uint64, error) {
		if calls != nil {
			*calls = append(*calls, fmt.Sprintf("%d-%d", from, to))
		}
		if to-from+1 > limit {
			return nil, &ErrRangeTooWide{From: from, To: to, Reason: "range too wide"}
		}

		out := make([]uint64, 0, to-from+1)
		for b := from; b <= to; b++ {
			out = append(out, b)
		}
		return out, nil
	}
}

// TestSplitterCoversEveryBlockWithLowRangeLimit is the T10 / F25 acceptance
// test: a provider that only tolerates narrow ranges must still yield every
// block, with no gap and no duplicate.
func TestSplitterCoversEveryBlockWithLowRangeLimit(t *testing.T) {
	t.Parallel()

	s, err := NewRangeSplitter[uint64](blockRangeFetch(10, nil), SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	got, err := s.Fetch(context.Background(), 1, 100)
	require.NoError(t, err)

	require.Len(t, got, 100, "every block must be covered exactly once")
	for i, block := range got {
		assert.Equal(t, uint64(i+1), block, "blocks must be covered in order with no gap")
	}
	assert.Greater(t, s.Stats().Splits, 0, "the range must actually have been split")
	assert.LessOrEqual(t, s.Stats().LargestRange, uint64(10), "the provider never accepted more than its limit")
}

// TestSplitterHandlesResultCountLimit covers the other provider limit: a
// response truncated by result count must be split, not advanced past.
func TestSplitterHandlesResultCountLimit(t *testing.T) {
	t.Parallel()

	const maxResults = 5
	fetch := func(_ context.Context, from, to uint64) ([]uint64, error) {
		width := to - from + 1
		if width > maxResults {
			// The provider answered, but with a truncated set. This is the
			// signal that must trigger a split.
			return nil, TruncationError(from, to, maxResults)
		}

		out := make([]uint64, 0, width)
		for b := from; b <= to; b++ {
			out = append(out, b)
		}
		return out, nil
	}

	s, err := NewRangeSplitter[uint64](fetch, SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	got, err := s.Fetch(context.Background(), 1, 20)
	require.NoError(t, err)

	require.Len(t, got, 20)
	for i, block := range got {
		assert.Equal(t, uint64(i+1), block)
	}
}

// TestSplitterSurfacesErrorAtMinimumRange: when even the minimum range is
// refused, the error must be surfaced rather than retried forever.
func TestSplitterSurfacesErrorAtMinimumRange(t *testing.T) {
	t.Parallel()

	// The provider only accepts single-block ranges, but the caller requires 4.
	s, err := NewRangeSplitter[uint64](blockRangeFetch(1, nil), SplitConfig{MinRange: 4, MaxCalls: 4096})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum range")
}

// TestSplitterBoundsCallsWhenProviderAlwaysRefuses is the "no runaway retries"
// half of the acceptance: a provider that refuses everything must fail fast.
func TestSplitterBoundsCallsWhenProviderAlwaysRefuses(t *testing.T) {
	t.Parallel()

	fetch := func(_ context.Context, from, to uint64) ([]uint64, error) {
		return nil, &ErrRangeTooWide{From: from, To: to, Reason: "never"}
	}

	// MaxCalls is deliberately below the number of halvings needed to reach a
	// single-block range, so the call guard is what stops the recursion.
	s, err := NewRangeSplitter[uint64](fetch, SplitConfig{MinRange: 1, MaxCalls: 5})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 1, 1_000_000)
	require.Error(t, err)

	calls := s.Stats().Calls
	assert.LessOrEqual(t, calls, 6, "runaway retries must be bounded by MaxCalls, got %d", calls)
	assert.Contains(t, err.Error(), "exceeded", "the guard, not luck, must stop it")
}

// TestSplitterFailsFastAtMinimumRangeWithoutCallGuard covers the other bound:
// when MinRange stops the recursion first, the error still names the limit.
func TestSplitterFailsFastAtMinimumRangeWithoutCallGuard(t *testing.T) {
	t.Parallel()

	fetch := func(_ context.Context, from, to uint64) ([]uint64, error) {
		return nil, &ErrRangeTooWide{From: from, To: to, Reason: "never"}
	}

	s, err := NewRangeSplitter[uint64](fetch, SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 1, 1_000_000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum range")
	assert.LessOrEqual(t, s.Stats().Calls, 32, "halving to width 1 needs about 20 calls, not a million")
}

// TestSplitterDoesNotSplitOnOtherErrors: a transport error is fatal, not a
// reason to hammer the provider with more requests.
func TestSplitterDoesNotSplitOnOtherErrors(t *testing.T) {
	t.Parallel()

	calls := 0
	fetch := func(_ context.Context, _, _ uint64) ([]uint64, error) {
		calls++
		return nil, errors.New("connection reset by peer")
	}

	s, err := NewRangeSplitter[uint64](fetch, SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 1, 100)
	require.Error(t, err)
	assert.Equal(t, 1, calls, "a non-range error must not trigger splitting")
}

func TestSplitterStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s, err := NewRangeSplitter[uint64](blockRangeFetch(1, nil), SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	_, err = s.Fetch(ctx, 1, 100)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSplitterRecordsLargestAcceptedRange(t *testing.T) {
	t.Parallel()

	s, err := NewRangeSplitter[uint64](blockRangeFetch(8, nil), SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 1, 32)
	require.NoError(t, err)

	stats := s.Stats()
	assert.Greater(t, stats.LargestRange, uint64(0))
	assert.LessOrEqual(t, stats.LargestRange, uint64(8), "the learned limit must respect the provider's cap")
}

func TestSplitterRejectsInvertedRange(t *testing.T) {
	t.Parallel()

	s, err := NewRangeSplitter[uint64](blockRangeFetch(10, nil), SplitConfig{MinRange: 1, MaxCalls: 16})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 10, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inverted range")
}

// TestSplitterStatsResetPerFetch keeps the observability honest: Stats describes
// the most recent Fetch, not the lifetime of the splitter.
func TestSplitterStatsResetPerFetch(t *testing.T) {
	t.Parallel()

	s, err := NewRangeSplitter[uint64](blockRangeFetch(10, nil), SplitConfig{MinRange: 1, MaxCalls: 4096})
	require.NoError(t, err)

	_, err = s.Fetch(context.Background(), 1, 100)
	require.NoError(t, err)
	first := s.Stats().Calls

	_, err = s.Fetch(context.Background(), 1, 5)
	require.NoError(t, err)
	assert.Less(t, s.Stats().Calls, first, "stats must describe the latest fetch")
}
