package reorg

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// fakeJobStore records what HandleReorg asked it to persist.
type fakeJobStore struct {
	jobs          []Job
	notifications []Notification
	err           error
}

func (s *fakeJobStore) EnqueueJob(_ context.Context, job Job, notifications []Notification) error {
	if s.err != nil {
		return s.err
	}
	s.jobs = append(s.jobs, job)
	s.notifications = append(s.notifications, notifications...)
	return nil
}

func (s *fakeJobStore) PendingJobs(context.Context, string, int) ([]Job, error) { return nil, nil }
func (s *fakeJobStore) CompleteJob(context.Context, string) error               { return nil }

func reorgDB(t *testing.T) *mockReorgDatabase {
	t.Helper()
	db := newMockReorgDatabase()
	db.addBlock(105, common.HexToHash("0x69"), common.HexToHash("0x68"))
	db.addEvent(100)
	db.deleteCount = 2
	return db
}

// TestHandleReorgEnqueuesJobAtomically is the T06 / F07 acceptance test: the
// repair job and the notifications it owes are handed to the store together,
// instead of being published best-effort after the rollback already committed.
func TestHandleReorgEnqueuesJobAtomically(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{}
	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithJobStore(store)

	require.NoError(t, handler.HandleReorg(context.Background(), 100))

	require.Len(t, store.jobs, 1, "the repair job must be recorded")
	job := store.jobs[0]
	assert.Equal(t, "ethereum:100:105", job.ID)
	assert.Equal(t, JobPending, job.Status)
	assert.Equal(t, uint64(100), job.FromBlock)
	assert.Equal(t, uint64(105), job.ToBlock)

	require.Len(t, store.notifications, 2, "both notifications must be recorded with the job")
	for _, n := range store.notifications {
		assert.Equal(t, job.ID, n.JobID)
		assert.Equal(t, "ethereum", n.ChainID)
		assert.NotEmpty(t, n.Payload)
	}
}

// TestHandleReorgSurfacesEnqueueFailure matters because the rollback has already
// committed: swallowing the failure would leave exactly the state F07 is about.
func TestHandleReorgSurfacesEnqueueFailure(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{err: errors.New("outbox unavailable")}
	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithJobStore(store)

	err := handler.HandleReorg(context.Background(), 100)
	require.Error(t, err, "an enqueue failure must not be swallowed")
	assert.Contains(t, err.Error(), "failed to enqueue reorg job")
}

// TestHandleReorgPrefersJobStoreOverEventBus pins the precedence: once a store
// is injected, the non-crash-safe publish path must not run at all.
func TestHandleReorgPrefersJobStoreOverEventBus(t *testing.T) {
	t.Parallel()

	published := make(chan struct{}, 2)
	bus := core.NewEventBus(core.NewDefaultLogger(core.LogLevelError))
	_, _ = bus.Subscribe(context.Background(), "reorg-detected", func(_ context.Context, _ any) error {
		published <- struct{}{}
		return nil
	})

	store := &fakeJobStore{}
	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithEventBus(bus).
		WithJobStore(store)

	require.NoError(t, handler.HandleReorg(context.Background(), 100))
	require.Len(t, store.jobs, 1)

	select {
	case <-published:
		t.Error("with a job store set, the legacy publish path must not run")
	case <-time.After(100 * time.Millisecond):
	}
}

// TestHandleReorgWithoutJobStoreKeepsLegacyPath documents backward
// compatibility: no store means the old best-effort publish behaviour.
func TestHandleReorgWithoutJobStoreKeepsLegacyPath(t *testing.T) {
	t.Parallel()

	published := make(chan *ReorgEvent, 1)
	bus := core.NewEventBus(core.NewDefaultLogger(core.LogLevelError))
	_, _ = bus.Subscribe(context.Background(), "reorg-detected", func(_ context.Context, payload any) error {
		published <- payload.(*ReorgEvent)
		return nil
	})

	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithEventBus(bus)

	require.NoError(t, handler.HandleReorg(context.Background(), 100))

	select {
	case <-published:
	case <-time.After(time.Second):
		t.Error("without a job store the legacy publish path must still run")
	}
}

func TestReorgJobIDIsStableAndChainScoped(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ReorgJobID("ethereum", 100, 105), ReorgJobID("ethereum", 100, 105),
		"the same reorg must derive the same job id, so a retry does not duplicate it")
	assert.NotEqual(t, ReorgJobID("ethereum", 100, 105), ReorgJobID("polygon", 100, 105),
		"chains must not share job ids")
	assert.NotEqual(t, ReorgJobID("ethereum", 100, 105), ReorgJobID("ethereum", 100, 106))
}
