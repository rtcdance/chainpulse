// Package batch provides generic batch processing utilities with controlled
// concurrency and retry support.
package batch

import (
	"context"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"
)

func Process[T any](ctx context.Context, items []T, fn func(ctx context.Context, item T) error, concurrency int) error {
	if len(items) == 0 {
		return nil
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	for _, item := range items {
		item := item
		g.Go(func() error { return fn(gCtx, item) })
	}
	return g.Wait()
}

func ProcessWithResults[T any, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error), concurrency int) ([]R, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	results := make([]R, len(items))
	errs := make([]error, len(items))
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	for i, item := range items {
		i, item := i, item
		g.Go(func() error {
			results[i], errs[i] = fn(gCtx, item)
			return errs[i]
		})
	}
	err := g.Wait()
	if err != nil {
		return results, err
	}
	return results, nil
}

func ProcessWithRetry[T any](ctx context.Context, items []T, fn func(ctx context.Context, item T) error, concurrency, maxRetries int) (successCount int, failedItems []T, err error) {
	if len(items) == 0 {
		return 0, nil, nil
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	var mu sync.Mutex
	var success int
	var failed []T
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)
	for _, item := range items {
		item := item
		g.Go(func() error {
			var lastErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if err := fn(gCtx, item); err == nil {
					mu.Lock()
					success++
					mu.Unlock()
					return nil
				} else {
					lastErr = err
				}
			}
			mu.Lock()
			failed = append(failed, item)
			mu.Unlock()
			return lastErr
		})
	}
	if err = g.Wait(); err != nil {
		return success, failed, err
	}
	return success, failed, nil
}

func Index[T any](ctx context.Context, items []T, fn func(context.Context, T) error) error {
	if len(items) == 0 {
		return nil
	}
	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := fn(ctx, item); err != nil {
			slog.Error("batch index: failed to process item", "error", err)
		}
	}
	return nil
}
