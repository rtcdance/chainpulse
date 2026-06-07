package indexing

import (
	"sync"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// ShadowWriteTracker tracks events that have been written to detect
// duplicate writes (shadow reads) during re-indexing or reorg recovery.
type ShadowWriteTracker interface {
	Mark(event *blockchain.BlockchainEvent)
	Consume(event *blockchain.BlockchainEvent) bool
}

// runtimeShadowWriteTracker is the production implementation.
type runtimeShadowWriteTracker struct {
	mu     sync.Mutex
	events map[*blockchain.BlockchainEvent]struct{}
}

// NewShadowWriteTracker creates a new ShadowWriteTracker.
func NewShadowWriteTracker() ShadowWriteTracker {
	return &runtimeShadowWriteTracker{
		events: make(map[*blockchain.BlockchainEvent]struct{}),
	}
}

func (t *runtimeShadowWriteTracker) Mark(event *blockchain.BlockchainEvent) {
	if event == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.events[event] = struct{}{}
}

func (t *runtimeShadowWriteTracker) Consume(event *blockchain.BlockchainEvent) bool {
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
func markShadowWrite(event *blockchain.BlockchainEvent) {
	defaultTracker.Mark(event)
}

// consumeShadowWrite checks and removes an event via the default tracker.
// Deprecated: use an injected ShadowWriteTracker instead.
func consumeShadowWrite(event *blockchain.BlockchainEvent) bool {
	return defaultTracker.Consume(event)
}
