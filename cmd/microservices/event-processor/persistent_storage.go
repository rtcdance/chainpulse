package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	if s.metadataStore == nil {
		return fmt.Errorf("metadata store is required")
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

	metadata := &query.EventMetadata{
		EventID:          event.ID,
		ChainID:          core.ResolveChainID(event.ChainID),
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

	return nil
}

func isPersistentDuplicate(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "E11000 duplicate key error") ||
		strings.Contains(message, "duplicate key value violates unique constraint")
}
