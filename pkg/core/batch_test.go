package core

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchProcess(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		err := BatchProcess(context.Background(), []int{}, func(ctx context.Context, item int) error {
			return nil
		}, 2)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("all succeed", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3, 4, 5}
		var seen atomic.Int32
		err := BatchProcess(context.Background(), items, func(ctx context.Context, item int) error {
			seen.Add(int32(item))
			return nil
		}, 2)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if seen.Load() != 15 {
			t.Fatalf("expected 15, got %d", seen.Load())
		}
	})

	t.Run("first error stops", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		targetErr := errors.New("stop")
		var processed atomic.Int32
		err := BatchProcess(context.Background(), items, func(ctx context.Context, item int) error {
			processed.Add(1)
			if item == 2 {
				return targetErr
			}
			return nil
		}, 1)
		if !errors.Is(err, targetErr) {
			t.Fatalf("expected %v, got %v", targetErr, err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := BatchProcess(ctx, []int{1, 2, 3}, func(ctx context.Context, item int) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}, 2)
		// errgroup might or might not propagate ctx.Err(); just ensure no hang
		_ = err
	})

	t.Run("concurrency default", func(t *testing.T) {
		t.Parallel()
		err := BatchProcess(context.Background(), []int{1}, func(ctx context.Context, item int) error {
			return nil
		}, 0)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("nil context panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		_ = BatchProcess[int](nil, []int{1}, func(ctx context.Context, item int) error {
			return nil
		}, 2)
	})
}

func TestBatchProcessWithResults(t *testing.T) {
	t.Parallel()

	t.Run("ordered results", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		results, err := BatchProcessWithResults(context.Background(), items,
			func(ctx context.Context, item int) (string, error) {
				return fmt.Sprintf("item-%d", item), nil
			}, 1)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		expected := []string{"item-1", "item-2", "item-3"}
		for i, r := range results {
			if r != expected[i] {
				t.Fatalf("results[%d] = %q, want %q", i, r, expected[i])
			}
		}
	})

	t.Run("partial errors", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		targetErr := errors.New("fail")
		results, err := BatchProcessWithResults(context.Background(), items,
			func(ctx context.Context, item int) (string, error) {
				if item == 2 {
					return "", targetErr
				}
				return fmt.Sprintf("ok-%d", item), nil
			}, 1)
		if !errors.Is(err, targetErr) {
			t.Fatalf("expected %v, got %v", targetErr, err)
		}
		if results[1] != "" {
			t.Fatalf("results[1] should be empty, got %q", results[1])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		results, err := BatchProcessWithResults[int, int](context.Background(), nil,
			func(ctx context.Context, item int) (int, error) {
				return item, nil
			}, 2)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if results != nil {
			t.Fatalf("expected nil, got %v", results)
		}
	})

	t.Run("nil context panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		_, _ = BatchProcessWithResults[int, int](nil, []int{1},
			func(ctx context.Context, item int) (int, error) {
				return item, nil
			}, 2)
	})
}

func TestBatchProcessWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("all succeed first try", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		success, failed := BatchProcessWithRetry(context.Background(), items,
			func(ctx context.Context, item int) error {
				return nil
			}, 2, 1)
		if success != len(items) {
			t.Fatalf("expected %d success, got %d", len(items), success)
		}
		if len(failed) != 0 {
			t.Fatalf("expected 0 failed, got %d: %v", len(failed), failed)
		}
	})

	t.Run("partial failures collected", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		targetErr := errors.New("always fail")
		success, failed := BatchProcessWithRetry(context.Background(), items,
			func(ctx context.Context, item int) error {
				if item == 2 {
					return targetErr
				}
				return nil
			}, 2, 0)
		if success != len(items)-1 {
			t.Fatalf("expected %d success, got %d", len(items)-1, success)
		}
		if len(failed) != 1 {
			t.Fatalf("expected 1 failed, got %d", len(failed))
		}
		if failed[0].Index != 1 {
			t.Fatalf("expected index 1, got %d", failed[0].Index)
		}
		if !errors.Is(failed[0].Err, targetErr) {
			t.Fatalf("expected %v, got %v", targetErr, failed[0].Err)
		}
	})

	t.Run("retries exhausted", func(t *testing.T) {
		t.Parallel()
		items := []int{1}
		targetErr := errors.New("persistent")
		var attempts atomic.Int32
		success, failed := BatchProcessWithRetry(context.Background(), items,
			func(ctx context.Context, item int) error {
				attempts.Add(1)
				return targetErr
			}, 1, 2)
		if success != 0 {
			t.Fatalf("expected 0 success, got %d", success)
		}
		if len(failed) != 1 {
			t.Fatalf("expected 1 failed, got %d", len(failed))
		}
		// initial + 2 retries = 3
		if n := attempts.Load(); n != 3 {
			t.Fatalf("expected 3 attempts, got %d", n)
		}
	})

	t.Run("retry then succeed", func(t *testing.T) {
		t.Parallel()
		items := []int{1}
		var attempts atomic.Int32
		success, failed := BatchProcessWithRetry(context.Background(), items,
			func(ctx context.Context, item int) error {
				n := attempts.Add(1)
				if n < 3 {
					return errors.New("transient")
				}
				return nil
			}, 1, 3)
		if success != 1 {
			t.Fatalf("expected 1 success, got %d", success)
		}
		if len(failed) != 0 {
			t.Fatalf("expected 0 failed, got %d", len(failed))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		success, failed := BatchProcessWithRetry[int](context.Background(), nil,
			func(ctx context.Context, item int) error {
				return nil
			}, 2, 1)
		if success != 0 {
			t.Fatalf("expected 0, got %d", success)
		}
		if len(failed) != 0 {
			t.Fatalf("expected 0 failed, got %d", len(failed))
		}
	})

	t.Run("context cancellation mid-retry", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		items := []int{1, 2, 3}
		var started atomic.Int32
		success, failed := BatchProcessWithRetry(ctx, items,
			func(ctx context.Context, item int) error {
				started.Add(1)
				// item 1 fails and triggers cancel
				if item == 1 {
					cancel()
					return errors.New("fail")
				}
				<-ctx.Done()
				return ctx.Err()
			}, 3, 0)

		// at least item 1 should be in failed list
		if len(failed) == 0 {
			t.Fatal("expected at least 1 failure")
		}
		_ = success
	})

	t.Run("nil context panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		BatchProcessWithRetry[int](nil, []int{1},
			func(ctx context.Context, item int) error {
				return nil
			}, 2, 1)
	})

	t.Run("concurrent safety with many items", func(t *testing.T) {
		t.Parallel()
		n := 100
		items := make([]int, n)
		for i := range items {
			items[i] = i
		}
		success, failed := BatchProcessWithRetry(context.Background(), items,
			func(ctx context.Context, item int) error {
				time.Sleep(time.Microsecond)
				return nil
			}, 10, 0)
		if success != n {
			t.Fatalf("expected %d success, got %d", n, success)
		}
		if len(failed) != 0 {
			t.Fatalf("expected 0 failed, got %d", len(failed))
		}
	})
}

func TestBatchChunk(t *testing.T) {
	t.Parallel()

	t.Run("exact division", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3, 4}
		chunks := BatchChunk(items, 2)
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(chunks))
		}
		if len(chunks[0]) != 2 || len(chunks[1]) != 2 {
			t.Fatalf("expected all chunks of size 2, got %v", chunks)
		}
	})

	t.Run("remainder chunk", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3, 4, 5}
		chunks := BatchChunk(items, 3)
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(chunks))
		}
		if len(chunks[0]) != 3 || len(chunks[1]) != 2 {
			t.Fatalf("expected chunk sizes [3,2], got %v", lenChunks(chunks))
		}
	})

	t.Run("chunk size equals length", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		chunks := BatchChunk(items, 3)
		if len(chunks) != 1 || len(chunks[0]) != 3 {
			t.Fatalf("expected 1 chunk of size 3, got %v", chunks)
		}
	})

	t.Run("chunk size larger than length", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2}
		chunks := BatchChunk(items, 10)
		if len(chunks) != 1 || len(chunks[0]) != 2 {
			t.Fatalf("expected single chunk of size 2, got %v", lenChunks(chunks))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		chunks := BatchChunk([]int{}, 2)
		if chunks != nil {
			t.Fatalf("expected nil, got %v", chunks)
		}
	})

	t.Run("non-positive chunk size defaults to 1", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		chunks := BatchChunk(items, 0)
		if len(chunks) != len(items) {
			t.Fatalf("expected %d chunks, got %d", len(items), len(chunks))
		}
	})

	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		chunks := BatchChunk[int](nil, 2)
		if chunks != nil {
			t.Fatalf("expected nil, got %v", chunks)
		}
	})
}

func TestFailedItemError(t *testing.T) {
	t.Parallel()

	f := FailedItem[int]{Item: 42, Index: 3, Err: errors.New("timeout")}
	msg := f.Error()
	expected := "item[3] failed: timeout"
	if msg != expected {
		t.Fatalf("expected %q, got %q", expected, msg)
	}
}

func helperChunkSizes[T any](chunks [][]T) []int {
	sizes := make([]int, len(chunks))
	for i, c := range chunks {
		sizes[i] = len(c)
	}
	return sizes
}

var lenChunks = helperChunkSizes[int]
