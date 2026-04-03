package indexing

import (
	"sync"

	"chainpulse/pkg/core"
)

var shadowWriteTracker = newRuntimeShadowWriteTracker()

type runtimeShadowWriteTracker struct {
	mu     sync.Mutex
	events map[*core.BlockchainEvent]struct{}
}

func newRuntimeShadowWriteTracker() *runtimeShadowWriteTracker {
	return &runtimeShadowWriteTracker{
		events: make(map[*core.BlockchainEvent]struct{}),
	}
}

func (t *runtimeShadowWriteTracker) mark(event *core.BlockchainEvent) {
	if event == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.events[event] = struct{}{}
}

func (t *runtimeShadowWriteTracker) consume(event *core.BlockchainEvent) bool {
	if event == nil {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.events[event]
	if ok {
		delete(t.events, event)
	}
	return ok
}
