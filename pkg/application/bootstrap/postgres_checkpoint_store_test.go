package bootstrap

import (
	"context"
	"errors"
	"testing"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
)

type fakeCheckpointStateStore struct {
	lastBlock    uint64
	lastHash     string
	saveCalls    int
	savedChainID string
	savedBlock   uint64
	savedHash    string
	savedCursor  string
	getErr       error
	saveErr      error
}

func (f *fakeCheckpointStateStore) GetLastIndexedBlock(context.Context, string) (uint64, string, error) {
	if f.getErr != nil {
		return 0, "", f.getErr
	}
	return f.lastBlock, f.lastHash, nil
}

func (f *fakeCheckpointStateStore) SaveCheckpoint(_ context.Context, chainID string, blockNumber uint64, blockHash string, cursor string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saveCalls++
	f.savedChainID = chainID
	f.savedBlock = blockNumber
	f.savedHash = blockHash
	f.savedCursor = cursor
	return nil
}

func TestPostgresCheckpointStoreLoadReturnsBlockPosition(t *testing.T) {
	t.Parallel()
	inner := &fakeCheckpointStateStore{lastBlock: 296, lastHash: "0xabc"}
	store := newPostgresCheckpointStore(inner)

	checkpoint, err := store.Load(context.Background(), "ethereum")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if checkpoint.ChainID != "ethereum" {
		t.Fatalf("expected chain id ethereum, got %q", checkpoint.ChainID)
	}
	if checkpoint.BlockNumber != 296 {
		t.Fatalf("expected block 296, got %d", checkpoint.BlockNumber)
	}
	// Load deliberately returns an empty cursor: after a restart the failure
	// journal is empty, so replay filtering must be block-level only.
	if checkpoint.Cursor != "" {
		t.Fatalf("expected empty cursor on load, got %q", checkpoint.Cursor)
	}
}

func TestPostgresCheckpointStoreSaveMapsCursor(t *testing.T) {
	t.Parallel()
	inner := &fakeCheckpointStateStore{}
	store := newPostgresCheckpointStore(inner)

	err := store.Save(context.Background(), appindexing.Checkpoint{
		ChainID:     "ethereum",
		BlockNumber: 296,
		Cursor:      "ethereum:296:0",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if inner.saveCalls != 1 {
		t.Fatalf("expected 1 save call, got %d", inner.saveCalls)
	}
	if inner.savedChainID != "ethereum" || inner.savedBlock != 296 {
		t.Fatalf("unexpected save mapping: chain=%q block=%d", inner.savedChainID, inner.savedBlock)
	}
	if inner.savedCursor != "ethereum:296:0" {
		t.Fatalf("expected cursor persisted, got %q", inner.savedCursor)
	}
}

func TestPostgresCheckpointStoreSaveRejectsEmptyChainID(t *testing.T) {
	t.Parallel()
	inner := &fakeCheckpointStateStore{}
	store := newPostgresCheckpointStore(inner)

	if err := store.Save(context.Background(), appindexing.Checkpoint{BlockNumber: 5}); err == nil {
		t.Fatal("expected empty chain id rejection")
	}
	if inner.saveCalls != 0 {
		t.Fatal("inner store must not be called for an invalid checkpoint")
	}
}

func TestPostgresCheckpointStoreWrapsInnerErrors(t *testing.T) {
	t.Parallel()
	store := newPostgresCheckpointStore(&fakeCheckpointStateStore{saveErr: errors.New("pg down")})
	if err := store.Save(context.Background(), appindexing.Checkpoint{ChainID: "ethereum"}); err == nil {
		t.Fatal("expected inner save error to propagate")
	}

	store2 := newPostgresCheckpointStore(&fakeCheckpointStateStore{getErr: errors.New("pg down")})
	if _, err := store2.Load(context.Background(), "ethereum"); err == nil {
		t.Fatal("expected inner load error to propagate")
	}
}

func TestResolveCheckpointStorePrefersConfigured(t *testing.T) {
	t.Parallel()
	configured := newPostgresCheckpointStore(&fakeCheckpointStateStore{})
	if got := resolveCheckpointStore(configured, []string{"ethereum"}); got != configured {
		t.Fatal("expected the configured store to win")
	}

	fallback := resolveCheckpointStore(nil, []string{"ethereum"})
	if fallback == nil {
		t.Fatal("expected in-memory fallback")
	}
	if _, ok := fallback.(*monolithicMemoryCheckpointStore); !ok {
		t.Fatalf("expected in-memory fallback type, got %T", fallback)
	}
}
