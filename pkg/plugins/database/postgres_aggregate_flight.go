package database

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type aggregateFlight struct {
	done chan struct{}
	data []byte
	err  error
}

// Owners run on the requesting goroutine. The map bounds shared state, and
// each owner's existing SQL deadline bounds its lifetime. Empty keys bypass it.
type aggregateFlights struct {
	mu     sync.Mutex
	active map[string]*aggregateFlight
}

func (f *aggregateFlights) read(ctx context.Context, key string, load func() ([]byte, error)) ([]byte, error) {
	data, err, shared := f.readOnce(ctx, key, load)
	if shared && ctx.Err() == nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		// A different request's canceled owner must not poison healthy callers.
		// One new shared attempt is the entire retry budget.
		data, err, _ = f.readOnce(ctx, key, load)
	}
	return data, err
}

func (f *aggregateFlights) readOnce(ctx context.Context, key string, load func() ([]byte, error)) ([]byte, error, bool) {
	if err := ctx.Err(); err != nil {
		return nil, err, false
	}
	if key == "" {
		data, err := load()
		return data, err, false
	}
	f.mu.Lock()
	if pending := f.active[key]; pending != nil {
		f.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err(), true
		case <-pending.done:
			if err := ctx.Err(); err != nil {
				return nil, err, true
			}
			return append([]byte(nil), pending.data...), pending.err, true
		}
	}
	if len(f.active) >= 16 {
		f.mu.Unlock()
		data, err := load()
		return data, err, false
	}
	if f.active == nil {
		f.active = make(map[string]*aggregateFlight)
	}
	pending := &aggregateFlight{done: make(chan struct{}), err: fmt.Errorf("aggregate source did not complete")}
	f.active[key] = pending
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		delete(f.active, key)
		close(pending.done)
		f.mu.Unlock()
	}()
	// The deferred release also runs on panic; the panic itself still propagates.
	pending.data, pending.err = load()
	if pending.err == nil {
		pending.err = ctx.Err()
	}
	return append([]byte(nil), pending.data...), pending.err, false
}
