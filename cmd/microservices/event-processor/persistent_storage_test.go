package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/query"
)

type persistentStorageEventStoreStub struct {
	inserted []*core.BlockchainEvent
	err      error
}

func (s *persistentStorageEventStoreStub) Initialize(ctx context.Context) error { return nil }
func (s *persistentStorageEventStoreStub) InsertEvent(ctx context.Context, event *core.BlockchainEvent) error {
	s.inserted = append(s.inserted, event)
	return s.err
}

func (s *persistentStorageEventStoreStub) InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error {
	return nil
}

func (s *persistentStorageEventStoreStub) GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error) {
	return nil, nil
}

func (s *persistentStorageEventStoreStub) GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error) {
	return nil, false, nil
}

func (s *persistentStorageEventStoreStub) CountEvents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *persistentStorageEventStoreStub) DeleteExpiredEvents(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *persistentStorageEventStoreStub) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy"}
}
func (s *persistentStorageEventStoreStub) Close(ctx context.Context) error { return nil }

type persistentStorageMetadataStoreStub struct {
	inserted []*query.EventMetadata
	err      error
}

func (s *persistentStorageMetadataStoreStub) Initialize(ctx context.Context) error { return nil }
func (s *persistentStorageMetadataStoreStub) InsertMetadata(ctx context.Context, metadata *query.EventMetadata) error {
	s.inserted = append(s.inserted, metadata)
	return s.err
}

func (s *persistentStorageMetadataStoreStub) InsertMetadataBatch(ctx context.Context, metadataList []*query.EventMetadata) error {
	return nil
}

func (s *persistentStorageMetadataStoreStub) GetMetadata(ctx context.Context, eventID string) (*query.EventMetadata, error) {
	return nil, nil
}

func (s *persistentStorageMetadataStoreStub) GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*query.EventMetadata, error) {
	return nil, nil
}

func (s *persistentStorageMetadataStoreStub) UpdateMetadata(ctx context.Context, metadata *query.EventMetadata) error {
	return nil
}

func (s *persistentStorageMetadataStoreStub) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{Status: "healthy"}
}
func (s *persistentStorageMetadataStoreStub) Close(ctx context.Context) error { return nil }

func TestPersistentEventProcessorStorageWritesEventAndMetadata(t *testing.T) {
	eventStore := &persistentStorageEventStoreStub{}
	metadataStore := &persistentStorageMetadataStoreStub{}
	storage := newPersistentEventProcessorStorage(eventStore, metadataStore)

	event := &core.BlockchainEvent{
		ID:              "evt-1",
		ChainID:         "31337",
		BlockNumber:     7,
		TransactionHash: common.HexToHash("0x1"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		EventName:       "Ping",
		LogIndex:        2,
	}

	if err := storage.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("write event: %v", err)
	}
	if len(eventStore.inserted) != 1 {
		t.Fatalf("expected 1 inserted event, got %d", len(eventStore.inserted))
	}
	if len(metadataStore.inserted) != 1 {
		t.Fatalf("expected 1 inserted metadata, got %d", len(metadataStore.inserted))
	}
	if got := metadataStore.inserted[0].ChainID; got != 31337 {
		t.Fatalf("expected metadata chain id 31337, got %d", got)
	}
	if event.ProcessedAt.IsZero() || event.IndexedAt.IsZero() || event.CreatedAt.IsZero() {
		t.Fatal("expected timestamps to be populated")
	}
}

func TestPersistentEventProcessorStorageResolvesNamedChainID(t *testing.T) {
	eventStore := &persistentStorageEventStoreStub{}
	metadataStore := &persistentStorageMetadataStoreStub{}
	storage := newPersistentEventProcessorStorage(eventStore, metadataStore)

	event := &core.BlockchainEvent{
		ID:              "evt-eth",
		ChainID:         "ethereum",
		BlockNumber:     8,
		TransactionHash: common.HexToHash("0x3"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000003"),
		EventName:       "Ping",
	}

	if err := storage.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("write event: %v", err)
	}
	if got := metadataStore.inserted[0].ChainID; got != 1 {
		t.Fatalf("expected named chain ethereum to map to 1, got %d", got)
	}
}

func TestPersistentEventProcessorStorageIgnoresDuplicateInsert(t *testing.T) {
	eventStore := &persistentStorageEventStoreStub{err: errors.New("E11000 duplicate key error")}
	metadataStore := &persistentStorageMetadataStoreStub{err: errors.New("duplicate key value violates unique constraint")}
	storage := newPersistentEventProcessorStorage(eventStore, metadataStore)

	event := &core.BlockchainEvent{
		ID:              "evt-dup",
		ChainID:         "1",
		BlockNumber:     9,
		TransactionHash: common.HexToHash("0x2"),
		ContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000002"),
		EventName:       "Ping",
		ProcessedAt:     time.Now(),
	}

	if err := storage.WriteEvent(context.Background(), event); err != nil {
		t.Fatalf("expected duplicate inserts to be tolerated, got %v", err)
	}
}
