package reorg

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// failingGenerationStore refuses to bump, simulating a generation-store outage
// after the rollback has already committed.
type failingGenerationStore struct{ err error }

func (s failingGenerationStore) BumpGeneration(context.Context, string) (int64, error) {
	return 0, s.err
}

func (s failingGenerationStore) CurrentGeneration(context.Context, string) (int64, error) {
	return 0, s.err
}

func TestMemGenerationStoreBumpsPerChain(t *testing.T) {
	t.Parallel()

	store := NewMemGenerationStore()
	ctx := context.Background()

	gen, err := store.BumpGeneration(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, int64(1), gen)

	gen, err = store.BumpGeneration(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, int64(2), gen)

	current, err := store.CurrentGeneration(ctx, "ethereum")
	require.NoError(t, err)
	assert.Equal(t, int64(2), current, "readers must observe the bumped generation")

	other, err := store.CurrentGeneration(ctx, "polygon")
	require.NoError(t, err)
	assert.Zero(t, other, "a chain that never reorged is generation 0")

	_, err = store.BumpGeneration(ctx, "")
	require.Error(t, err)
}

// TestHandleReorgBumpsGenerationAndStampsJob is the T06 generation-producer
// acceptance: one reorg advances the durable generation, and the repair job
// plus every notification it owes record that generation, so consumers and
// cache readers can fence pre-reorg state.
func TestHandleReorgBumpsGenerationAndStampsJob(t *testing.T) {
	t.Parallel()

	store := NewMemGenerationStore()
	jobs := &fakeJobStore{}
	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithJobStore(jobs).
		WithGenerationStore(store)

	require.NoError(t, handler.HandleReorg(context.Background(), 100))

	require.Len(t, jobs.jobs, 1)
	assert.Equal(t, int64(1), jobs.jobs[0].Generation)
	for _, n := range jobs.notifications {
		assert.Equal(t, int64(1), n.Generation, "notifications must carry the generation they were created under")
	}

	current, err := store.CurrentGeneration(context.Background(), "ethereum")
	require.NoError(t, err)
	assert.Equal(t, int64(1), current)
}

// TestHandleReorgBumpsGenerationPerReorg pins that consecutive reorgs advance
// the fence monotonically.
func TestHandleReorgBumpsGenerationPerReorg(t *testing.T) {
	t.Parallel()

	store := NewMemGenerationStore()
	jobs := &fakeJobStore{}
	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithJobStore(jobs).
		WithGenerationStore(store)

	require.NoError(t, handler.HandleReorg(context.Background(), 100))
	require.NoError(t, handler.HandleReorg(context.Background(), 103))

	require.Len(t, jobs.jobs, 2)
	assert.Equal(t, int64(1), jobs.jobs[0].Generation)
	assert.Equal(t, int64(2), jobs.jobs[1].Generation)
}

// TestHandleReorgSurfacesGenerationBumpFailure: the rollback has already
// committed, so a bump failure must be reported for retry, exactly like an
// enqueue failure, and no job may be recorded under a stale generation.
func TestHandleReorgSurfacesGenerationBumpFailure(t *testing.T) {
	t.Parallel()

	jobs := &fakeJobStore{}
	handler := NewReorgHandler(reorgDB(t), core.NewDefaultLogger(core.LogLevelError), 12, 120).
		WithChainID("ethereum").
		WithJobStore(jobs).
		WithGenerationStore(failingGenerationStore{err: errors.New("generation store unavailable")})

	err := handler.HandleReorg(context.Background(), 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to bump reorg generation")
	assert.Empty(t, jobs.jobs, "no job may be enqueued without its generation fence")
}
