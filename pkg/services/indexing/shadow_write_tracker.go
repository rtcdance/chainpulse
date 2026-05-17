package indexing

import (
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// ShadowWriteTracker tracks events that have been written to detect
// duplicate writes (shadow reads) during re-indexing or reorg recovery.
type ShadowWriteTracker interface {
	Mark(event *core.BlockchainEvent)
	Consume(event *core.BlockchainEvent) bool
}

// runtimeShadowWriteTracker is the production implementation.
type runtimeShadowWriteTracker struct {
	mu     sync.Mutex
	events map[*core.BlockchainEvent]struct{}
}

// NewShadowWriteTracker creates a new ShadowWriteTracker.
func NewShadowWriteTracker() ShadowWriteTracker {
	return &runtimeShadowWriteTracker{
		events: make(map[*core.BlockchainEvent]struct{}),
	}
}

func (t *runtimeShadowWriteTracker) Mark(event *core.BlockchainEvent) {
	if event == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.events[event] = struct{}{}
}

func (t *runtimeShadowWriteTracker) Consume(event *core.BlockchainEvent) bool {
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

// defaultTracker is the package-level default for backward compatibility.
// Production code should inject ShadowWriteTracker via constructor.
var defaultTracker ShadowWriteTracker = NewShadowWriteTracker()

// markShadowWrite marks an event via the default tracker.
// Deprecated: use an injected ShadowWriteTracker instead.
func markShadowWrite(event *core.BlockchainEvent) {
	defaultTracker.Mark(event)
}

// consumeShadowWrite checks and removes an event via the default tracker.
// Deprecated: use an injected ShadowWriteTracker instead.
func consumeShadowWrite(event *core.BlockchainEvent) bool {
	return defaultTracker.Consume(event)
}
