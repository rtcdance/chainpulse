package indexing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

type commitCall struct {
	block  uint64
	hash   string
	events int
}

// recordingBlockCommitter records what the runtime asked it to commit.
type recordingBlockCommitter struct {
	calls  []commitCall
	err    error
	failAt uint64 // when non-zero, fail while committing this block
}

func (c *recordingBlockCommitter) CommitBlock(_ context.Context, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent) error {
	if c.err != nil {
		return c.err
	}
	if c.failAt != 0 && blockNumber == c.failAt {
		return errors.New("commit failed")
	}
	c.calls = append(c.calls, commitCall{block: blockNumber, hash: blockHash, events: len(events)})
	return nil
}

func startedAtomicRuntime(t *testing.T, c AtomicBatchCommitter) *SharedRuntime {
	t.Helper()

	deps := validRuntimeDeps()
	deps.Committer = c
	rt, err := NewSharedRuntime(deps)
	require.NoError(t, err)
	require.NoError(t, rt.Initialize(context.Background()))
	require.NoError(t, rt.Start(context.Background()))
	return rt
}

func atomicEnvelope(key string, block uint64) EventEnvelope {
	return EventEnvelope{
		EventKey:    key,
		ChainID:     "ethereum",
		BlockNumber: block,
		Payload: &blockchain.BlockchainEvent{
			ID:          key,
			ChainID:     "ethereum",
			BlockNumber: block,
		},
	}
}

func TestProcessBatchAtomicRequiresCommitter(t *testing.T) {
	t.Parallel()

	rt := startedAtomicRuntime(t, nil)
	err := rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{atomicEnvelope("a", 1)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no atomic committer")
}

func TestProcessBatchAtomicGroupsByBlock(t *testing.T) {
	t.Parallel()

	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)

	// Two events in block 10, one in block 11: two commits, in block order.
	require.NoError(t, rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{
		atomicEnvelope("c", 11),
		atomicEnvelope("a", 10),
		atomicEnvelope("b", 10),
	}))

	require.Len(t, c.calls, 2)
	assert.Equal(t, uint64(10), c.calls[0].block)
	assert.Equal(t, 2, c.calls[0].events)
	assert.Equal(t, uint64(11), c.calls[1].block)
	assert.Equal(t, 1, c.calls[1].events)
}

func TestProcessBatchAtomicRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)

	// A payload that is not a *blockchain.BlockchainEvent must fail the batch
	// before anything is committed.
	err := rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{
		atomicEnvelope("good", 10),
		{EventKey: "bad", ChainID: "ethereum", BlockNumber: 10, Payload: "not an event"},
	})
	require.Error(t, err)
	assert.Empty(t, c.calls, "a malformed envelope must not reach the committer")
}

func TestProcessBatchAtomicRejectsChainMismatch(t *testing.T) {
	t.Parallel()

	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)

	err := rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{
		{EventKey: "x", ChainID: "polygon", BlockNumber: 1, Payload: &blockchain.BlockchainEvent{ChainID: "polygon"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain ID mismatch")
	assert.Empty(t, c.calls)
}

func TestProcessBatchAtomicEmptyBatchIsNoop(t *testing.T) {
	t.Parallel()

	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)

	require.NoError(t, rt.ProcessBatchAtomic(context.Background(), "ethereum", nil))
	assert.Empty(t, c.calls)
}

func TestProcessBatchAtomicStopsOnBlockBoundary(t *testing.T) {
	t.Parallel()

	c := &recordingBlockCommitter{failAt: 11}
	rt := startedAtomicRuntime(t, c)

	// Block 10 commits, block 11 fails: the checkpoint stays at block 10, a
	// valid resume point.
	err := rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{
		atomicEnvelope("a", 10),
		atomicEnvelope("b", 11),
	})
	require.Error(t, err)

	require.Len(t, c.calls, 1)
	assert.Equal(t, uint64(10), c.calls[0].block)

	status := rt.Status()
	assert.Equal(t, int64(1), status.ProcessedEvents, "only the committed block counts")
	assert.Equal(t, "ethereum", status.LastCheckpointChainID)
	assert.Equal(t, "block:10", status.LastCheckpointCursor, "the checkpoint must name the last fully committed block")
}

// TestProcessBatchAtomicDoesNotPreCheckDuplicates documents the migration away
// from the four-step path: the same block committed twice reaches the committer
// twice, because dedup is the store's job, not a pre-check.
func TestProcessBatchAtomicDoesNotPreCheckDuplicates(t *testing.T) {
	t.Parallel()

	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)

	envelope := atomicEnvelope("dup", 5)
	require.NoError(t, rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{envelope}))
	require.NoError(t, rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{envelope}))

	assert.Len(t, c.calls, 2, "the atomic path must delegate dedup to the store")
	assert.Equal(t, int64(0), rt.Status().SkippedDuplicates, "it must not consult the idempotency store")
}
