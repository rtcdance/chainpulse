package indexing

import (
	"context"
	"fmt"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

const eventCacheTTLSeconds = 24 * 3600

// LegacyRuntimeSink bridges shared runtime envelopes to the current indexing
// database/cache storage semantics.
type LegacyRuntimeSink struct {
	database core.DatabasePlugin
	cache    core.CachePlugin
	logger   core.Logger
}

// NewLegacyRuntimeSink builds a shared runtime sink backed by legacy storage
// plugins already used by the indexing layer.
func NewLegacyRuntimeSink(
	database core.DatabasePlugin,
	cache core.CachePlugin,
	logger core.Logger,
) (*LegacyRuntimeSink, error) {
	if database == nil {
		return nil, fmt.Errorf("database plugin is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &LegacyRuntimeSink{
		database: database,
		cache:    cache,
		logger:   logger,
	}, nil
}

// Persist stores payload-backed blockchain events and mirrors legacy cache
// writes when a cache plugin is configured.
func (s *LegacyRuntimeSink) Persist(ctx context.Context, events []core.EventEnvelope) error {
	for _, envelope := range events {
		event, err := eventFromEnvelope(envelope)
		if err != nil {
			return fmt.Errorf("convert envelope to event: %w", err)
		}

		if event.IndexedAt.IsZero() {
			event.IndexedAt = time.Now()
		}
		if event.ProcessedAt.IsZero() {
			event.ProcessedAt = event.IndexedAt
		}

		if err := s.database.StoreEvent(ctx, event); err != nil {
			return fmt.Errorf("store event %s: %w", event.ID, err)
		}
		markShadowWrite(event)

		if s.cache == nil {
			continue
		}

		if err := s.cache.Set(ctx, cacheKeyForEvent(envelope.ChainID, event), []byte(event.ID), eventCacheTTLSeconds); err != nil {
			s.logger.Warn("failed to cache event from runtime sink", "chain_id", envelope.ChainID, "event_id", event.ID, "error", err.Error())
		}
	}

	return nil
}

func eventFromEnvelope(envelope core.EventEnvelope) (*core.BlockchainEvent, error) {
	event, ok := envelope.Payload.(*core.BlockchainEvent)
	if !ok || event == nil {
		return nil, fmt.Errorf("event payload must be *core.BlockchainEvent")
	}
	return event, nil
}
