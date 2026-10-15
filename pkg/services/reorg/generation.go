package reorg

import (
	"context"
	"fmt"
	"sync"
)

// GenerationStore persists the per-chain reorg generation.
//
// The generation is the fence that invalidates derived state: a reorg bumps it,
// and every value cached under an older generation becomes unreachable in one
// step (see cache.CacheScope). It must be durable, because a generation that
// resets on restart would let pre-reorg cached values be served again.
type GenerationStore interface {
	// BumpGeneration advances the chain's generation by one and returns the new
	// value. It must be atomic: two concurrent reorgs must not observe the same
	// generation.
	BumpGeneration(ctx context.Context, chainID string) (int64, error)
	// CurrentGeneration returns the generation in force for a chain. A chain
	// that has never reorged is generation 0.
	CurrentGeneration(ctx context.Context, chainID string) (int64, error)
}

// MemGenerationStore is the in-memory GenerationStore for deployments without
// a database (monolithic mode) and for tests.
type MemGenerationStore struct {
	mu     sync.Mutex
	bySeed map[string]int64
}

// NewMemGenerationStore creates an empty in-memory generation store.
func NewMemGenerationStore() *MemGenerationStore {
	return &MemGenerationStore{bySeed: make(map[string]int64)}
}

// BumpGeneration advances and returns the chain's generation.
func (s *MemGenerationStore) BumpGeneration(_ context.Context, chainID string) (int64, error) {
	if chainID == "" {
		return 0, fmt.Errorf("chain id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bySeed[chainID]++
	return s.bySeed[chainID], nil
}

// CurrentGeneration returns the chain's generation (0 when never bumped).
func (s *MemGenerationStore) CurrentGeneration(_ context.Context, chainID string) (int64, error) {
	if chainID == "" {
		return 0, fmt.Errorf("chain id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bySeed[chainID], nil
}
