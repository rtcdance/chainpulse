package batch

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestProcess(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		err := Process(context.Background(), []int{}, func(ctx context.Context, item int) error {
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
		err := Process(context.Background(), items, func(ctx context.Context, item int) error {
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
		err := Process(context.Background(), items, func(ctx context.Context, item int) error {
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
		err := Process(ctx, []int{1, 2, 3}, func(ctx context.Context, item int) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		}, 2)
		_ = err
	})

	t.Run("concurrency default", func(t *testing.T) {
		t.Parallel()
		err := Process(context.Background(), []int{1}, func(ctx context.Context, item int) error {
			return nil
		}, 0)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

func TestProcessWithResults(t *testing.T) {
	t.Parallel()

	t.Run("ordered results", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		results, err := ProcessWithResults(context.Background(), items,
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
		results, err := ProcessWithResults(context.Background(), items,
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
		results, err := ProcessWithResults[int, int](context.Background(), nil,
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
}

func TestProcessWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("all succeed first try", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		success, failed, err := ProcessWithRetry(context.Background(), items,
			func(ctx context.Context, item int) error {
				return nil
			}, 2, 1)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
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
		success, failed, _ := ProcessWithRetry(context.Background(), items,
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
		if failed[0] != 2 {
			t.Fatalf("expected failed item 2, got %d", failed[0])
		}
	})

	t.Run("retries exhausted", func(t *testing.T) {
		t.Parallel()
		items := []int{1}
		targetErr := errors.New("persistent")
		var attempts atomic.Int32
		success, failed, _ := ProcessWithRetry(context.Background(), items,
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
		if n := attempts.Load(); n != 3 {
			t.Fatalf("expected 3 attempts, got %d", n)
		}
	})

	t.Run("retry then succeed", func(t *testing.T) {
		t.Parallel()
		items := []int{1}
		var attempts atomic.Int32
		success, failed, _ := ProcessWithRetry(context.Background(), items,
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
		success, failed, err := ProcessWithRetry[int](context.Background(), nil,
			func(ctx context.Context, item int) error {
				return nil
			}, 2, 1)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if success != 0 {
			t.Fatalf("expected 0, got %d", success)
		}
		if len(failed) != 0 {
			t.Fatalf("expected 0 failed, got %d", len(failed))
		}
	})

	t.Run("concurrent safety with many items", func(t *testing.T) {
		t.Parallel()
		n := 100
		items := make([]int, n)
		for i := range items {
			items[i] = i
		}
		success, failed, _ := ProcessWithRetry(context.Background(), items,
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

func TestIndex(t *testing.T) {
	t.Parallel()

	t.Run("all succeed", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		var seen atomic.Int32
		err := Index(context.Background(), items, func(ctx context.Context, item int) error {
			seen.Add(int32(item))
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if seen.Load() != 6 {
			t.Fatalf("expected 6, got %d", seen.Load())
		}
	})

	t.Run("continues on error", func(t *testing.T) {
		t.Parallel()
		items := []int{1, 2, 3}
		var seen atomic.Int32
		err := Index(context.Background(), items, func(ctx context.Context, item int) error {
			seen.Add(1)
			if item == 2 {
				return errors.New("skip")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if seen.Load() != 3 {
			t.Fatalf("expected 3, got %d", seen.Load())
		}
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		err := Index(context.Background(), []int{}, func(ctx context.Context, item int) error {
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := Index(ctx, []int{1, 2, 3}, func(ctx context.Context, item int) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}