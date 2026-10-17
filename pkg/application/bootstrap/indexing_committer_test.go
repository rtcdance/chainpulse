package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/blockchain"

	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

type fakeTx struct {
	mu          sync.Mutex
	events      []any
	checkpoints []string
	committed   bool
	rolledBack  bool
	storeErr    error
	saveErr     error
	commitErr   error
}

func (t *fakeTx) StoreEvent(_ context.Context, event any) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.storeErr != nil {
		return t.storeErr
	}
	t.events = append(t.events, event)
	return nil
}

func (t *fakeTx) SaveCheckpoint(_ context.Context, chainID string, blockNumber uint64, blockHash string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.saveErr != nil {
		return t.saveErr
	}
	t.checkpoints = append(t.checkpoints, fmt.Sprintf("%s:%d:%s", chainID, blockNumber, blockHash))
	return nil
}

func (t *fakeTx) Commit(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.commitErr != nil {
		return t.commitErr
	}
	t.committed = true
	return nil
}

func (t *fakeTx) Rollback(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rolledBack = true
	return nil
}

type fakeFencedStore struct {
	portcontract.DatabasePlugin
	mu       sync.Mutex
	acquired []string
	released []string
	tx       *fakeTx
}

func (s *fakeFencedStore) AcquireLease(_ context.Context, chainID, _ string, _ time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acquired = append(s.acquired, chainID)
	return 1, nil
}

func (s *fakeFencedStore) ReleaseLease(_ context.Context, chainID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.released = append(s.released, chainID)
	return nil
}

func (s *fakeFencedStore) BeginFencedTx(_ context.Context, _, _ string) (portcontract.Tx, error) {
	return s.tx, nil
}

type unwrapWrapper struct {
	portcontract.DatabasePlugin
	inner portcontract.DatabasePlugin
}

func (w *unwrapWrapper) Unwrap() portcontract.DatabasePlugin {
	return w.inner
}

func newTestChainEvent(chainID string, blockNumber uint64) *blockchain.BlockchainEvent {
	return &blockchain.BlockchainEvent{ChainID: chainID, BlockNumber: blockNumber}
}

func TestBuildMonolithicBatchCommitterNonFenceableReturnsNil(t *testing.T) {
	t.Parallel()
	memoryDB := NewMonolithicMemoryDatabase(testhelpers.NewTestLogger())
	committer, err := BuildMonolithicBatchCommitter(memoryDB, NewDrainController(), []string{"ethereum"})
	if err != nil {
		t.Fatalf("BuildMonolithicBatchCommitter: %v", err)
	}
	if committer != nil {
		t.Fatal("expected nil committer for non-fenceable store")
	}
}

func TestBuildMonolithicBatchCommitterRequiresInputs(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	cases := []struct {
		name     string
		database portcontract.DatabasePlugin
		chains   []string
	}{
		{name: "nil database", database: nil, chains: []string{"ethereum"}},
		{name: "no chains", database: store, chains: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := BuildMonolithicBatchCommitter(tc.database, NewDrainController(), tc.chains); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestMonolithicBatchCommitterRoutesByChain(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	wrapper := &unwrapWrapper{inner: store}
	committer, err := BuildMonolithicBatchCommitter(wrapper, NewDrainController(), []string{"ethereum", "polygon"})
	if err != nil {
		t.Fatalf("BuildMonolithicBatchCommitter: %v", err)
	}
	if committer == nil {
		t.Fatal("expected committer, got nil")
	}

	events := []*blockchain.BlockchainEvent{
		newTestChainEvent("polygon", 42),
		{ChainID: "polygon", BlockNumber: 42},
	}
	if err := committer.CommitBlock(context.Background(), 42, "0xhash", events); err != nil {
		t.Fatalf("CommitBlock: %v", err)
	}
	if !store.tx.committed {
		t.Fatal("expected transaction to be committed")
	}
	wantCheckpoint := "polygon:42:0xhash"
	if len(store.tx.checkpoints) != 1 || store.tx.checkpoints[0] != wantCheckpoint {
		t.Fatalf("checkpoints = %v, want [%s]", store.tx.checkpoints, wantCheckpoint)
	}
	if len(store.tx.events) != 2 {
		t.Fatalf("stored events = %d, want 2", len(store.tx.events))
	}
	if len(store.acquired) == 0 || store.acquired[0] != "polygon" {
		t.Fatalf("acquired leases = %v, want polygon first", store.acquired)
	}
}

func TestMonolithicBatchCommitterUnknownChainRejected(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	committer, err := BuildMonolithicBatchCommitter(store, NewDrainController(), []string{"ethereum"})
	if err != nil {
		t.Fatalf("BuildMonolithicBatchCommitter: %v", err)
	}
	err = committer.CommitBlock(context.Background(), 1, "0xh", []*blockchain.BlockchainEvent{newTestChainEvent("polygon", 1)})
	if err == nil || !strings.Contains(err.Error(), "no batch committer") {
		t.Fatalf("CommitBlock = %v, want unknown-chain error", err)
	}
}

func TestMonolithicBatchCommitterRejectsUnroutableEvents(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	committer, err := BuildMonolithicBatchCommitter(store, NewDrainController(), []string{"ethereum"})
	if err != nil {
		t.Fatalf("BuildMonolithicBatchCommitter: %v", err)
	}
	cases := []struct {
		name   string
		events []*blockchain.BlockchainEvent
	}{
		{name: "nil events", events: nil},
		{name: "events without chain", events: []*blockchain.BlockchainEvent{{BlockNumber: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := committer.CommitBlock(context.Background(), 1, "0xh", tc.events)
			if err == nil || !strings.Contains(err.Error(), "chain ID") {
				t.Fatalf("CommitBlock = %v, want unroutable-events error", err)
			}
		})
	}
}

func TestMonolithicBatchCommitterHonorsDrainGate(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	drain := NewDrainController()
	committer, err := BuildMonolithicBatchCommitter(store, drain, []string{"ethereum"})
	if err != nil {
		t.Fatalf("BuildMonolithicBatchCommitter: %v", err)
	}
	if drained := drain.Drain(time.Second); !drained {
		t.Fatal("expected clean drain with no in-flight work")
	}
	err = committer.CommitBlock(context.Background(), 1, "0xh", []*blockchain.BlockchainEvent{newTestChainEvent("ethereum", 1)})
	if !errors.Is(err, processor.ErrDraining) {
		t.Fatalf("CommitBlock = %v, want ErrDraining", err)
	}
}

func TestMonolithicBatchCommitterReleaseAll(t *testing.T) {
	t.Parallel()
	store := &fakeFencedStore{tx: &fakeTx{}}
	committer, err := BuildMonolithicBatchCommitter(store, NewDrainController(), []string{"ethereum", "polygon"})
	if err != nil {
		t.Fatalf("BuildMonolithicBatchCommitter: %v", err)
	}
	if err := committer.ReleaseAll(context.Background()); err != nil {
		t.Fatalf("ReleaseAll: %v", err)
	}
	if len(store.released) != 2 {
		t.Fatalf("released leases = %v, want 2 chains", store.released)
	}
}

func TestNewLeaseOwnerToken(t *testing.T) {
	t.Parallel()
	first, err := newLeaseOwnerToken()
	if err != nil {
		t.Fatalf("newLeaseOwnerToken: %v", err)
	}
	if !strings.HasPrefix(first, "chainpulse-monolithic-") {
		t.Fatalf("token %q missing prefix", first)
	}
	second, err := newLeaseOwnerToken()
	if err != nil {
		t.Fatalf("newLeaseOwnerToken: %v", err)
	}
	if first == second {
		t.Fatal("expected distinct owner tokens")
	}
}
