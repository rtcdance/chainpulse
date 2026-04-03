package bootstrap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
	serviceindexing "chainpulse/pkg/services/indexing"
)

type monolithicIndexingRuntimeDeps struct {
	newRuntime func(deps appindexing.RuntimeDeps) (*appindexing.SharedRuntime, error)
	newSink    func(database core.DatabasePlugin, cache core.CachePlugin, logger core.Logger) (appindexing.EventSink, error)
}

func defaultMonolithicIndexingRuntimeDeps() monolithicIndexingRuntimeDeps {
	return monolithicIndexingRuntimeDeps{
		newRuntime: appindexing.NewSharedRuntime,
		newSink: func(database core.DatabasePlugin, cache core.CachePlugin, logger core.Logger) (appindexing.EventSink, error) {
			return serviceindexing.NewLegacyRuntimeSink(database, cache, logger)
		},
	}
}

// BuildMonolithicIndexingRuntime creates additive shared runtime wiring for
// monolithic mode without changing existing legacy indexing behavior.
func BuildMonolithicIndexingRuntime(
	logger core.Logger,
	database core.DatabasePlugin,
	cache core.CachePlugin,
	chains []string,
) (*appindexing.SharedRuntime, error) {
	return buildMonolithicIndexingRuntimeWithDeps(logger, database, cache, chains, defaultMonolithicIndexingRuntimeDeps())
}

// BuildInMemoryIndexingRuntime creates an additive shared runtime backed by
// in-memory checkpoint/idempotency/failure/replay ports. Callers retain
// ownership of the persistence sink semantics.
func BuildInMemoryIndexingRuntime(
	logger core.Logger,
	sink appindexing.EventSink,
	chains []string,
) (*appindexing.SharedRuntime, error) {
	return buildInMemoryIndexingRuntimeWithDeps(logger, sink, chains, appindexing.NewSharedRuntime)
}

func buildMonolithicIndexingRuntimeWithDeps(
	logger core.Logger,
	database core.DatabasePlugin,
	cache core.CachePlugin,
	chains []string,
	deps monolithicIndexingRuntimeDeps,
) (*appindexing.SharedRuntime, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if database == nil {
		return nil, fmt.Errorf("database plugin is required")
	}

	normalizedChains := normalizeChainList(chains)
	if len(normalizedChains) == 0 {
		return nil, fmt.Errorf("at least one chain is required")
	}

	sink, err := deps.newSink(database, cache, logger)
	if err != nil {
		return nil, fmt.Errorf("build runtime sink: %w", err)
	}

	return buildInMemoryIndexingRuntimeWithDeps(logger, sink, normalizedChains, deps.newRuntime)
}

func buildInMemoryIndexingRuntimeWithDeps(
	logger core.Logger,
	sink appindexing.EventSink,
	chains []string,
	newRuntime func(deps appindexing.RuntimeDeps) (*appindexing.SharedRuntime, error),
) (*appindexing.SharedRuntime, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if sink == nil {
		return nil, fmt.Errorf("event sink is required")
	}

	normalizedChains := normalizeChainList(chains)
	if len(normalizedChains) == 0 {
		return nil, fmt.Errorf("at least one chain is required")
	}

	failures := newMonolithicMemoryFailureJournal()

	return newRuntime(appindexing.RuntimeDeps{
		Logger:          logger,
		Source:          monolithicNoopEventSource{},
		Sink:            sink,
		CheckpointStore: newMonolithicMemoryCheckpointStore(normalizedChains),
		Idempotency:     newMonolithicMemoryIdempotencyStore(),
		FailureRouter:   failures,
		ReplaySource:    failures,
		Chains:          normalizedChains,
	})
}

func normalizeChainList(chains []string) []string {
	normalized := make([]string, 0, len(chains))
	for _, chainID := range chains {
		trimmed := strings.TrimSpace(chainID)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

type monolithicNoopEventSource struct{}

func (monolithicNoopEventSource) Fetch(ctx context.Context, chainID string) ([]appindexing.EventEnvelope, error) {
	return nil, nil
}

type monolithicMemoryCheckpointStore struct {
	mu          sync.RWMutex
	checkpoints map[string]appindexing.Checkpoint
}

func newMonolithicMemoryCheckpointStore(chains []string) *monolithicMemoryCheckpointStore {
	checkpoints := make(map[string]appindexing.Checkpoint, len(chains))
	for _, chainID := range chains {
		checkpoints[chainID] = appindexing.Checkpoint{ChainID: chainID}
	}
	return &monolithicMemoryCheckpointStore{checkpoints: checkpoints}
}

func (s *monolithicMemoryCheckpointStore) Load(ctx context.Context, chainID string) (appindexing.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checkpoint, ok := s.checkpoints[chainID]
	if !ok {
		return appindexing.Checkpoint{ChainID: chainID}, nil
	}
	return checkpoint, nil
}

func (s *monolithicMemoryCheckpointStore) Save(ctx context.Context, checkpoint appindexing.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkpoints[checkpoint.ChainID] = checkpoint
	return nil
}

type monolithicMemoryIdempotencyStore struct {
	mu        sync.RWMutex
	processed map[string]struct{}
}

func newMonolithicMemoryIdempotencyStore() *monolithicMemoryIdempotencyStore {
	return &monolithicMemoryIdempotencyStore{
		processed: make(map[string]struct{}),
	}
}

func (s *monolithicMemoryIdempotencyStore) IsProcessed(ctx context.Context, eventKey string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.processed[eventKey]
	return ok, nil
}

func (s *monolithicMemoryIdempotencyStore) MarkProcessed(ctx context.Context, eventKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processed[eventKey] = struct{}{}
	return nil
}

type monolithicMemoryFailureJournal struct {
	mu      sync.RWMutex
	entries map[string][]appindexing.EventEnvelope
}

func newMonolithicMemoryFailureJournal() *monolithicMemoryFailureJournal {
	return &monolithicMemoryFailureJournal{
		entries: make(map[string][]appindexing.EventEnvelope),
	}
}

func (s *monolithicMemoryFailureJournal) Route(
	ctx context.Context,
	failure appindexing.ProcessingFailure,
	event appindexing.EventEnvelope,
) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	chainID := event.ChainID
	if chainID == "" {
		chainID = failure.ChainID
	}
	if chainID == "" {
		chainID = "unknown"
	}
	event.ChainID = chainID
	s.entries[chainID] = append(s.entries[chainID], event)
	return nil
}

func (s *monolithicMemoryFailureJournal) Replay(
	ctx context.Context,
	chainID string,
	from appindexing.Checkpoint,
) ([]appindexing.EventEnvelope, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := append([]appindexing.EventEnvelope(nil), s.entries[chainID]...)
	if len(events) == 0 {
		return nil, nil
	}

	replayed := make([]appindexing.EventEnvelope, 0, len(events))
	for _, event := range events {
		if !shouldReplayMonolithicFailureEvent(event, from) {
			continue
		}
		replayed = append(replayed, event)
	}

	sort.SliceStable(replayed, func(i, j int) bool {
		if replayed[i].BlockNumber != replayed[j].BlockNumber {
			return replayed[i].BlockNumber < replayed[j].BlockNumber
		}
		return replayed[i].CheckpointCursor < replayed[j].CheckpointCursor
	})
	return replayed, nil
}

func shouldReplayMonolithicFailureEvent(
	event appindexing.EventEnvelope,
	from appindexing.Checkpoint,
) bool {
	if from.BlockNumber == 0 && from.Cursor == "" {
		return true
	}
	if event.BlockNumber > from.BlockNumber {
		return true
	}
	if event.BlockNumber < from.BlockNumber {
		return false
	}
	if from.Cursor == "" {
		return true
	}
	if event.CheckpointCursor == "" {
		return true
	}
	return event.CheckpointCursor >= from.Cursor
}

func (s *monolithicMemoryFailureJournal) Size(chainID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries[chainID])
}
