package core

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/core/batch"
)

func BatchProcess[T any](ctx context.Context, items []T, fn func(ctx context.Context, item T) error, concurrency int) error {
	return batch.Process(ctx, items, fn, concurrency)
}

func BatchProcessWithResults[T any, R any](ctx context.Context, items []T, fn func(ctx context.Context, item T) (R, error), concurrency int) ([]R, error) {
	return batch.ProcessWithResults(ctx, items, fn, concurrency)
}

func BatchProcessWithRetry[T any](ctx context.Context, items []T, fn func(ctx context.Context, item T) error, concurrency, maxRetries int) (int, []T, error) {
	return batch.ProcessWithRetry(ctx, items, fn, concurrency, maxRetries)
}

func BatchIndex[T any](ctx context.Context, items []T, fn func(context.Context, T) error) error {
	return batch.Index(ctx, items, fn)
}