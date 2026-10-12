package processor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// --------------------------------------------------------------------------- //
// Fakes
// --------------------------------------------------------------------------- //

type fakeCheckpoint struct {
	chainID string
	block   uint64
	hash    string
}

type fakeTx struct {
	store *fakeStore

	events     []any
	checkpoint *fakeCheckpoint
	committed  bool
	rolledBack bool

	// failOn: "store" | "checkpoint" | "commit"
	failOn string
	// dedup makes StoreEvent behave like ON CONFLICT DO NOTHING: it succeeds
	// but records nothing (the event was already present).
	dedup bool
}

func (t *fakeTx) StoreEvent(_ context.Context, event any) error {
	if t.failOn == "store" {
		return errors.New("store failed")
	}
	if t.dedup {
		return nil
	}
	t.events = append(t.events, event)
	return nil
}

func (t *fakeTx) SaveCheckpoint(_ context.Context, chainID string, blockNumber uint64, blockHash string) error {
	if t.failOn == "checkpoint" {
		return errors.New("checkpoint failed")
	}
	t.checkpoint = &fakeCheckpoint{chainID: chainID, block: blockNumber, hash: blockHash}
	return nil
}

func (t *fakeTx) Commit(_ context.Context) error {
	if t.failOn == "commit" {
		return errors.New("commit failed")
	}
	t.committed = true
	t.store.recordCommit(t)
	return nil
}

func (t *fakeTx) Rollback(_ context.Context) error {
	if t.committed {
		return nil // no-op after commit, like the real pgTx
	}
	t.rolledBack = true
	return nil
}

type fakeStore struct {
	mu sync.Mutex

	owner      string
	generation int64

	txs       []*fakeTx
	committed []*fakeTx

	dedup  bool
	failOn string
}

func (s *fakeStore) AcquireLease(_ context.Context, _, ownerToken string, _ time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.owner != "" && s.owner != ownerToken {
		return 0, errors.New("lease is held by another owner")
	}
	s.owner = ownerToken
	s.generation++
	return s.generation, nil
}

func (s *fakeStore) ReleaseLease(_ context.Context, _, ownerToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner == ownerToken {
		s.owner = ""
	}
	return nil
}

func (s *fakeStore) BeginFencedTx(_ context.Context, _, ownerToken string) (core.Tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.owner != ownerToken {
		return nil, errors.New("lease is held by another owner")
	}
	tx := &fakeTx{store: s, failOn: s.failOn, dedup: s.dedup}
	s.txs = append(s.txs, tx)
	return tx, nil
}

func (s *fakeStore) recordCommit(t *fakeTx) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.committed = append(s.committed, t)
}

// stealLease simulates a Kafka rebalance handing the chain to another worker.
func (s *fakeStore) stealLease(newOwner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.owner = newOwner
}

type fakeGate struct {
	open  bool
	begun int
	ended int
}

func (g *fakeGate) Begin() bool {
	if !g.open {
		return false
	}
	g.begun++
	return true
}

func (g *fakeGate) End() { g.ended++ }

func testEvents(n int) []*blockchain.BlockchainEvent {
	out := make([]*blockchain.BlockchainEvent, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, &blockchain.BlockchainEvent{ChainID: "1", BlockNumber: 10, LogIndex: uint64(i)})
	}
	return out
}

func newCommitter(t *testing.T, store FencedStore, gate WorkGate) *BatchCommitter {
	t.Helper()
	c, err := NewBatchCommitter(store, gate, "1", "worker-A", time.Minute)
	require.NoError(t, err)
	return c
}

// --------------------------------------------------------------------------- //
// Tests
// --------------------------------------------------------------------------- //

func TestBatchCommitterCommitsEventsAndCheckpointTogether(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	gate := &fakeGate{open: true}
	c := newCommitter(t, store, gate)

	require.NoError(t, c.CommitBlock(context.Background(), 10, "0xabc", testEvents(3)))

	require.Len(t, store.committed, 1, "exactly one transaction must commit")
	tx := store.committed[0]
	assert.Len(t, tx.events, 3)
	require.NotNil(t, tx.checkpoint, "the checkpoint must be part of the same transaction")
	assert.Equal(t, uint64(10), tx.checkpoint.block)
	assert.Equal(t, "0xabc", tx.checkpoint.hash)
	assert.False(t, tx.rolledBack)
	assert.Equal(t, 1, gate.begun)
	assert.Equal(t, 1, gate.ended)
}

func TestBatchCommitterEmptyBlockStillAdvancesCheckpoint(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	c := newCommitter(t, store, nil)

	// An empty block must still move the checkpoint, or the chain stalls at
	// that height forever.
	require.NoError(t, c.CommitBlock(context.Background(), 77, "0xempty", nil))

	require.Len(t, store.committed, 1)
	assert.Empty(t, store.committed[0].events)
	require.NotNil(t, store.committed[0].checkpoint)
	assert.Equal(t, uint64(77), store.committed[0].checkpoint.block)
}

func TestBatchCommitterDuplicateOnlyBatchStillAdvancesCheckpoint(t *testing.T) {
	t.Parallel()

	// dedup makes every StoreEvent behave like ON CONFLICT DO NOTHING: the
	// batch is entirely duplicates, yet the checkpoint must still advance.
	store := &fakeStore{dedup: true}
	c := newCommitter(t, store, nil)

	require.NoError(t, c.CommitBlock(context.Background(), 12, "0xdup", testEvents(5)))

	require.Len(t, store.committed, 1)
	assert.Empty(t, store.committed[0].events, "duplicates are suppressed by the store, not pre-checked here")
	require.NotNil(t, store.committed[0].checkpoint)
	assert.Equal(t, uint64(12), store.committed[0].checkpoint.block)
}

func TestBatchCommitterDoesNotPreCheckDuplicates(t *testing.T) {
	t.Parallel()

	// The committer must not consult a separate duplicate check: dedup is the
	// store's natural-key constraint. A pre-check would reintroduce the
	// check-then-write window that F05 is about.
	store := &fakeStore{}
	c := newCommitter(t, store, nil)

	require.NoError(t, c.CommitBlock(context.Background(), 1, "0xa", testEvents(2)))
	require.Len(t, store.committed, 1)
	assert.Len(t, store.committed[0].events, 2, "every event is handed to the store; the store decides")
}

func TestBatchCommitterRollsBackOnCheckpointFailure(t *testing.T) {
	t.Parallel()

	store := &fakeStore{failOn: "checkpoint"}
	c := newCommitter(t, store, nil)

	err := c.CommitBlock(context.Background(), 10, "0xabc", testEvents(2))
	require.Error(t, err)

	assert.Empty(t, store.committed, "nothing may commit when the checkpoint fails")
	require.Len(t, store.txs, 1)
	assert.True(t, store.txs[0].rolledBack, "the transaction must be rolled back")
	assert.Len(t, store.txs[0].events, 2, "events were staged but must not persist")
}

func TestBatchCommitterRollsBackOnStoreFailure(t *testing.T) {
	t.Parallel()

	store := &fakeStore{failOn: "store"}
	c := newCommitter(t, store, nil)

	require.Error(t, c.CommitBlock(context.Background(), 10, "0xabc", testEvents(1)))
	assert.Empty(t, store.committed)
	assert.True(t, store.txs[0].rolledBack)
	assert.Nil(t, store.txs[0].checkpoint, "the checkpoint must not be reached")
}

func TestBatchCommitterRejectsNewWorkWhileDraining(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	gate := &fakeGate{open: false}
	c := newCommitter(t, store, gate)

	err := c.CommitBlock(context.Background(), 10, "0xabc", testEvents(1))
	require.ErrorIs(t, err, ErrDraining)
	assert.Empty(t, store.txs, "no transaction may start once draining")
}

func TestBatchCommitterSurfacesLeaseLoss(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	c := newCommitter(t, store, nil)

	// Another worker takes the chain (rebalance).
	store.stealLease("worker-B")

	err := c.CommitBlock(context.Background(), 10, "0xabc", testEvents(1))
	require.Error(t, err, "the fenced-out worker must not commit")
	assert.Empty(t, store.committed)
}

func TestBatchCommitterRenewsLeaseLazily(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	c := newCommitter(t, store, nil)

	require.NoError(t, c.CommitBlock(context.Background(), 1, "0xa", nil))
	first := c.Generation()

	// A second immediate batch must not re-acquire: the lease is still fresh.
	require.NoError(t, c.CommitBlock(context.Background(), 2, "0xb", nil))
	assert.Equal(t, first, c.Generation(), "the lease must not be churned on every batch")
}

// TestBatchCommitterAcceptsDrainControllerAsGate pins the wiring: the real
// shutdown gate must satisfy the interface the committer expects.
func TestBatchCommitterAcceptsDrainControllerAsGate(t *testing.T) {
	t.Parallel()

	var gate WorkGate = bootstrap.NewDrainController()
	c, err := NewBatchCommitter(&fakeStore{}, gate, "1", "worker-A", time.Minute)
	require.NoError(t, err)

	require.NoError(t, c.CommitBlock(context.Background(), 1, "0xa", nil))
	require.True(t, gate.(*bootstrap.DrainController).Drain(time.Second))
	require.ErrorIs(t, c.CommitBlock(context.Background(), 2, "0xb", nil), ErrDraining)
}
