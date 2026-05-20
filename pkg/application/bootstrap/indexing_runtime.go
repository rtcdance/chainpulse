package bootstrap

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	serviceindexing "github.com/rtcdance/chainpulse/pkg/services/indexing"
)

type monolithicIndexingRuntimeDeps struct {
	newRuntime func(deps appindexing.RuntimeDeps) (*appindexing.SharedRuntime, error)
	newSink    func(database core.DatabasePlugin, cache core.CachePlugin, logger core.Logger) (appindexing.EventSink, error)
}

type InMemoryIndexingRuntimeOptions struct {
	DLQRetention time.Duration
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
	return BuildMonolithicIndexingRuntimeWithOptions(
		logger,
		database,
		cache,
		chains,
		InMemoryIndexingRuntimeOptions{},
	)
}

func BuildMonolithicIndexingRuntimeWithOptions(
	logger core.Logger,
	database core.DatabasePlugin,
	cache core.CachePlugin,
	chains []string,
	options InMemoryIndexingRuntimeOptions,
) (*appindexing.SharedRuntime, error) {
	return buildMonolithicIndexingRuntimeWithDeps(
		logger,
		database,
		cache,
		chains,
		options,
		defaultMonolithicIndexingRuntimeDeps(),
	)
}

// BuildInMemoryIndexingRuntime creates an additive shared runtime backed by
// in-memory checkpoint/idempotency/failure/replay ports. Callers retain
// ownership of the persistence sink semantics.
func BuildInMemoryIndexingRuntime(
	logger core.Logger,
	sink appindexing.EventSink,
	chains []string,
) (*appindexing.SharedRuntime, error) {
	return BuildInMemoryIndexingRuntimeWithOptions(logger, sink, chains, InMemoryIndexingRuntimeOptions{})
}

func BuildInMemoryIndexingRuntimeWithOptions(
	logger core.Logger,
	sink appindexing.EventSink,
	chains []string,
	options InMemoryIndexingRuntimeOptions,
) (*appindexing.SharedRuntime, error) {
	return buildInMemoryIndexingRuntimeWithDeps(logger, sink, chains, options, appindexing.NewSharedRuntime)
}

func buildMonolithicIndexingRuntimeWithDeps(
	logger core.Logger,
	database core.DatabasePlugin,
	cache core.CachePlugin,
	chains []string,
	options InMemoryIndexingRuntimeOptions,
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

	return buildInMemoryIndexingRuntimeWithDeps(logger, sink, normalizedChains, options, deps.newRuntime)
}

func buildInMemoryIndexingRuntimeWithDeps(
	logger core.Logger,
	sink appindexing.EventSink,
	chains []string,
	options InMemoryIndexingRuntimeOptions,
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

	failures := newMonolithicMemoryFailureJournal(options.DLQRetention)

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
	mu        sync.RWMutex
	retention time.Duration
	entries   map[string][]monolithicFailureRecord
}

type monolithicFailureRecord struct {
	failure    appindexing.ProcessingFailure
	event      appindexing.EventEnvelope
	recordedAt time.Time
}

func newMonolithicMemoryFailureJournal(retention time.Duration) *monolithicMemoryFailureJournal {
	return &monolithicMemoryFailureJournal{
		retention: retention,
		entries:   make(map[string][]monolithicFailureRecord),
	}
}

func (s *monolithicMemoryFailureJournal) Route(
	_ context.Context,
	failure appindexing.ProcessingFailure,
	event appindexing.EventEnvelope,
) error {
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
	recordedAt := time.Now().UTC()

	if failure.OccurredAt.IsZero() {
		failure.OccurredAt = recordedAt
	}

	s.cleanupExpiredLocked(recordedAt)
	s.entries[chainID] = append(s.entries[chainID], monolithicFailureRecord{
		failure:    failure,
		event:      event,
		recordedAt: recordedAt,
	})
	return nil
}

func (s *monolithicMemoryFailureJournal) Replay(
	_ context.Context,
	chainID string,
	from appindexing.Checkpoint,
) ([]appindexing.EventEnvelope, error) {
	s.mu.Lock()
	s.cleanupExpiredLocked(time.Now().UTC())
	records := append([]monolithicFailureRecord(nil), s.entries[chainID]...)
	s.mu.Unlock()

	if len(records) == 0 {
		return nil, nil
	}

	replayed := make([]appindexing.EventEnvelope, 0, len(records))

	for _, record := range records {
		event := record.event

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

func (s *monolithicMemoryFailureJournal) ReplayRange(
	ctx context.Context,
	chainID string,
	from, to appindexing.Checkpoint,
	limit int,
) ([]appindexing.EventEnvelope, error) {
	replayed, err := s.Replay(ctx, chainID, from)
	if err != nil {
		return nil, err
	}

	filtered := make([]appindexing.EventEnvelope, 0, len(replayed))

	for _, event := range replayed {
		if !monolithicFailureWithinRange(event, to) {
			continue
		}

		filtered = append(filtered, event)

		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

func (s *monolithicMemoryFailureJournal) AcknowledgeReplay(
	_ context.Context,
	chainID string,
	events []appindexing.EventEnvelope,
) error {
	if len(events) == 0 {
		return nil
	}

	keys := make(map[string]struct{}, len(events))

	for _, event := range events {
		if event.EventKey == "" {
			continue
		}

		keys[event.EventKey] = struct{}{}
	}

	if len(keys) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(time.Now().UTC())

	records := s.entries[chainID]
	if len(records) == 0 {
		return nil
	}

	filtered := records[:0]

	for _, record := range records {
		if _, ok := keys[record.event.EventKey]; ok {
			continue
		}

		filtered = append(filtered, record)
	}

	if len(filtered) == 0 {
		delete(s.entries, chainID)

		return nil
	}

	s.entries[chainID] = append([]monolithicFailureRecord(nil), filtered...)

	return nil
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(time.Now().UTC())
	return len(s.entries[chainID])
}

func monolithicFailureWithinRange(event appindexing.EventEnvelope, to appindexing.Checkpoint) bool {
	if to.BlockNumber == 0 && to.Cursor == "" {
		return true
	}

	if event.BlockNumber < to.BlockNumber {
		return true
	}

	if event.BlockNumber > to.BlockNumber {
		return false
	}

	if to.Cursor == "" || event.CheckpointCursor == "" {
		return true
	}

	return event.CheckpointCursor <= to.Cursor
}

func (s *monolithicMemoryFailureJournal) cleanupExpiredLocked(now time.Time) {
	if s.retention <= 0 {
		return
	}

	for chainID, records := range s.entries {
		filtered := records[:0]

		for _, record := range records {
			if now.Sub(record.recordedAt) >= s.retention {
				continue
			}

			filtered = append(filtered, record)
		}

		if len(filtered) == 0 {
			delete(s.entries, chainID)

			continue
		}

		s.entries[chainID] = append([]monolithicFailureRecord(nil), filtered...)
	}
}
