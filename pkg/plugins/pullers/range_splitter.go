package pullers

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrRangeTooWide marks a fetch the provider refused or truncated.
//
// F25: providers differ in how wide a block range and how many results they
// tolerate, and geth/erigon/reth report the refusal differently — some return a
// JSON-RPC error, others return a silently truncated result set. Both must be
// turned into this error so the splitter reacts the same way.
type ErrRangeTooWide struct {
	From   uint64
	To     uint64
	Reason string
}

func (e *ErrRangeTooWide) Error() string {
	return fmt.Sprintf("provider refused block range %d..%d: %s", e.From, e.To, e.Reason)
}

// TooWideError reports whether err is (or wraps) an ErrRangeTooWide.
func TooWideError(err error) bool {
	var e *ErrRangeTooWide
	return errors.As(err, &e)
}

// TruncationError is the error a fetch callback should return when the provider
// returned exactly as many results as it is willing to return. That is the
// signal that the range must be SPLIT, not advanced past.
func TruncationError(from, to uint64, maxResults int) error {
	return &ErrRangeTooWide{
		From:   from,
		To:     to,
		Reason: fmt.Sprintf("result count reached the provider limit of %d", maxResults),
	}
}

// SplitConfig bounds the splitter.
type SplitConfig struct {
	// MinRange is the smallest range the splitter will request. Reaching it
	// means the provider cannot serve even that, so the error is surfaced
	// rather than retried forever.
	MinRange uint64
	// MaxCalls bounds the fetch calls for one Fetch, so a provider that rejects
	// everything cannot cause runaway retries.
	MaxCalls int
}

// SplitStats reports what one Fetch did.
type SplitStats struct {
	Calls        int
	Splits       int
	LargestRange uint64 // widest range the provider accepted
}

// RangeSplitter fetches a payload over an inclusive block range, bisecting the
// range when the provider refuses it.
//
// The invariant is that NO BLOCK IS SKIPPED: the sub-ranges visited are exactly
// a partition of [from, to]. That is the property the previous chunk loop did
// not have — on truncation it advanced past the returned tail, silently
// dropping every block between that tail and the end of the chunk.
type RangeSplitter[T any] struct {
	fetch func(ctx context.Context, from, to uint64) ([]T, error)
	cfg   SplitConfig

	mu    sync.Mutex
	stats SplitStats
}

// NewRangeSplitter creates a splitter over fetch.
//
// fetch must return an ErrRangeTooWide (see TruncationError) when the provider
// refuses the range or truncates the result set; any other error is treated as
// fatal and propagated.
func NewRangeSplitter[T any](fetch func(ctx context.Context, from, to uint64) ([]T, error), cfg SplitConfig) (*RangeSplitter[T], error) {
	if fetch == nil {
		return nil, fmt.Errorf("fetch function is required")
	}
	if cfg.MinRange == 0 {
		cfg.MinRange = 1
	}
	if cfg.MaxCalls <= 0 {
		cfg.MaxCalls = 1024
	}
	return &RangeSplitter[T]{fetch: fetch, cfg: cfg}, nil
}

// Fetch returns every item in [from, to], splitting when necessary.
func (s *RangeSplitter[T]) Fetch(ctx context.Context, from, to uint64) ([]T, error) {
	if to < from {
		return nil, fmt.Errorf("inverted range %d..%d", from, to)
	}

	s.mu.Lock()
	s.stats = SplitStats{}
	s.mu.Unlock()

	out := make([]T, 0)
	if err := s.fetchRange(ctx, from, to, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Stats returns the stats for the most recent Fetch.
func (s *RangeSplitter[T]) Stats() SplitStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func (s *RangeSplitter[T]) fetchRange(ctx context.Context, from, to uint64, out *[]T) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.stats.Calls++
	calls := s.stats.Calls
	s.mu.Unlock()

	if calls > s.cfg.MaxCalls {
		return fmt.Errorf("range splitter exceeded %d fetch calls at %d..%d", s.cfg.MaxCalls, from, to)
	}

	items, err := s.fetch(ctx, from, to)
	if err == nil {
		*out = append(*out, items...)

		s.mu.Lock()
		if width := to - from + 1; width > s.stats.LargestRange {
			s.stats.LargestRange = width
		}
		s.mu.Unlock()
		return nil
	}

	// Only a "too wide" refusal is a reason to split; anything else is fatal.
	if !TooWideError(err) {
		return fmt.Errorf("fetch %d..%d: %w", from, to, err)
	}

	width := to - from + 1
	if width <= s.cfg.MinRange {
		return fmt.Errorf("provider cannot serve %d block(s) at %d..%d, the minimum range: %w",
			width, from, to, err)
	}

	s.mu.Lock()
	s.stats.Splits++
	s.mu.Unlock()

	// Bisect into two non-empty halves that together cover [from, to] exactly.
	mid := from + width/2 - 1
	if err := s.fetchRange(ctx, from, mid, out); err != nil {
		return err
	}
	return s.fetchRange(ctx, mid+1, to, out)
}
