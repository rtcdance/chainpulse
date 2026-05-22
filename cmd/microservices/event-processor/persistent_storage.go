package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/services/query"
)

type persistentEventProcessorStorage struct {
	eventStore    query.EventStore
	metadataStore query.EventMetadataStore
	timeout       time.Duration
}

func newPersistentEventProcessorStorage(
	eventStore query.EventStore,
	metadataStore query.EventMetadataStore,
) *persistentEventProcessorStorage {
	return &persistentEventProcessorStorage{
		eventStore:    eventStore,
		metadataStore: metadataStore,
		timeout:       10 * time.Second,
	}
}

func (s *persistentEventProcessorStorage) WriteEvent(ctx context.Context, event *core.BlockchainEvent) error {
	if s == nil {
		return fmt.Errorf("persistent storage is required")
	}
	if s.eventStore == nil {
		return fmt.Errorf("event store is required")
	}
	if event == nil {
		return fmt.Errorf("event is required")
	}

	now := time.Now()
	if event.ProcessedAt.IsZero() {
		event.ProcessedAt = now
	}
	if event.IndexedAt.IsZero() {
		event.IndexedAt = now
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.eventStore.InsertEvent(ctx, event); err != nil && !isPersistentDuplicate(err) {
		return err
	}

	if s.metadataStore != nil {
		metadata := &query.EventMetadata{
			EventID:          event.ID,
			ChainID:          chainid.ResolveChainID(event.ChainID),
			BlockNumber:      int64(event.BlockNumber),
			TransactionHash:  event.TransactionHash.Hex(),
			LogIndex:         int64(event.LogIndex),
			ContractAddress:  event.ContractAddress.Hex(),
			EventName:        event.EventName,
			ProcessedAt:      event.ProcessedAt,
			ProcessingStatus: string(core.EventStatusConfirmed),
		}
		if err := s.metadataStore.InsertMetadata(ctx, metadata); err != nil && !isPersistentDuplicate(err) {
			return err
		}
	}

	return nil
}

// WriteBatch writes multiple events using InsertEventBatch for atomic storage.
func (s *persistentEventProcessorStorage) WriteBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	if s == nil {
		return fmt.Errorf("persistent storage is required")
	}
	if s.eventStore == nil {
		return fmt.Errorf("event store is required")
	}
	if len(events) == 0 {
		return nil
	}

	now := time.Now()
	for _, event := range events {
		if event == nil {
			continue
		}
		if event.ProcessedAt.IsZero() {
			event.ProcessedAt = now
		}
		if event.IndexedAt.IsZero() {
			event.IndexedAt = now
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.eventStore.InsertEventBatch(ctx, events); err != nil && !isPersistentDuplicate(err) {
		return err
	}

	if s.metadataStore != nil {
		// Metadata is written individually since the batch API doesn't guarantee
		// metadata atomicity across both stores. Duplicates are handled by the
		// unique constraint and on-conflict logic.
		for _, event := range events {
			if event == nil {
				continue
			}
			metadata := &query.EventMetadata{
				EventID:          event.ID,
				ChainID:          chainid.ResolveChainID(event.ChainID),
				BlockNumber:      int64(event.BlockNumber),
				TransactionHash:  event.TransactionHash.Hex(),
				LogIndex:         int64(event.LogIndex),
				ContractAddress:  event.ContractAddress.Hex(),
				EventName:        event.EventName,
				ProcessedAt:      event.ProcessedAt,
				ProcessingStatus: string(core.EventStatusConfirmed),
			}
			if err := s.metadataStore.InsertMetadata(ctx, metadata); err != nil && !isPersistentDuplicate(err) {
				return err
			}
		}
	}

	return nil
}

func (s *persistentEventProcessorStorage) DeleteEvent(ctx context.Context, eventID string) error {
	if d, ok := s.eventStore.(interface {
		DeleteEvent(context.Context, string) error
	}); ok {
		return d.DeleteEvent(ctx, eventID)
	}
	return fmt.Errorf("delete not supported by underlying event store")
}

func isPersistentDuplicate(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "E11000 duplicate key error") ||
		strings.Contains(message, "duplicate key value violates unique constraint")
}
