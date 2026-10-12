package processor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// recordingCommitter captures what the processor asked it to commit.
type recordingCommitter struct {
	calls     int
	block     uint64
	blockHash string
	events    []*blockchain.BlockchainEvent
	err       error
}

func (c *recordingCommitter) CommitBlock(_ context.Context, blockNumber uint64, blockHash string, events []*blockchain.BlockchainEvent) error {
	c.calls++
	c.block = blockNumber
	c.blockHash = blockHash
	c.events = events
	return c.err
}

func startedProcessor(t *testing.T) *DefaultEventProcessor {
	t.Helper()
	p := newTestProcessor()
	require.NoError(t, p.Initialize(&core.Config{ServiceName: "test"}))
	require.NoError(t, p.Start())
	t.Cleanup(func() { _ = p.Stop() })
	return p
}

func TestProcessBlockRequiresCommitter(t *testing.T) {
	t.Parallel()

	p := startedProcessor(t)
	err := p.ProcessBlock(context.Background(), 1, "0xa", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no block committer")
}

func TestProcessBlockRequiresRunning(t *testing.T) {
	t.Parallel()

	p := newTestProcessor()
	require.NoError(t, p.Initialize(&core.Config{ServiceName: "test"}))
	c := &recordingCommitter{}
	p.WithCommitter(c)

	require.Error(t, p.ProcessBlock(context.Background(), 1, "0xa", nil))
	assert.Zero(t, c.calls, "a stopped processor must not reach the committer")
}

func TestProcessBlockDelegatesToCommitter(t *testing.T) {
	t.Parallel()

	p := startedProcessor(t)
	c := &recordingCommitter{}
	p.WithCommitter(c)

	require.NoError(t, p.ProcessBlock(context.Background(), 42, "0xblock42", []*blockchain.BlockchainEvent{makeValidEvent()}))

	assert.Equal(t, 1, c.calls)
	assert.Equal(t, uint64(42), c.block)
	assert.Equal(t, "0xblock42", c.blockHash)
	assert.Len(t, c.events, 1)
	assert.Equal(t, int64(1), p.GetProcessedCount())
	assert.Equal(t, int64(0), p.GetFailedCount())
}

func TestProcessBlockEmptyBlockIsValid(t *testing.T) {
	t.Parallel()

	p := startedProcessor(t)
	c := &recordingCommitter{}
	p.WithCommitter(c)

	// An empty block must still be committed, because committing it is what
	// advances the checkpoint past that height.
	require.NoError(t, p.ProcessBlock(context.Background(), 7, "0xempty", nil))
	assert.Equal(t, 1, c.calls)
}

func TestProcessBlockValidatesBeforeCommitting(t *testing.T) {
	t.Parallel()

	p := startedProcessor(t)
	c := &recordingCommitter{}
	p.WithCommitter(c)

	// A zero event fails validation, so the block must not open a transaction.
	require.Error(t, p.ProcessBlock(context.Background(), 9, "0xb", []*blockchain.BlockchainEvent{{}}))
	assert.Zero(t, c.calls, "an invalid block must not reach the committer")
	assert.Equal(t, int64(1), p.GetFailedCount())
}

func TestProcessBlockSurfacesCommitError(t *testing.T) {
	t.Parallel()

	p := startedProcessor(t)
	c := &recordingCommitter{err: errors.New("commit boom")}
	p.WithCommitter(c)

	require.Error(t, p.ProcessBlock(context.Background(), 3, "0xc", []*blockchain.BlockchainEvent{makeValidEvent()}))
	assert.Equal(t, int64(0), p.GetProcessedCount(), "a failed commit must not count as processed")
	assert.Equal(t, int64(1), p.GetFailedCount())
}

// TestProcessBlockReplacesThePerEventPath documents the intended migration: the
// block path must not consult the in-memory duplicate check at all.
func TestProcessBlockDoesNotUseIdempotencyService(t *testing.T) {
	t.Parallel()

	p := startedProcessor(t)
	c := &recordingCommitter{}
	p.WithCommitter(c)

	// Commit the same block twice: the committer is called both times, because
	// deduplication is the database's job, not a pre-check.
	event := makeValidEvent()
	require.NoError(t, p.ProcessBlock(context.Background(), 5, "0xd", []*blockchain.BlockchainEvent{event}))
	require.NoError(t, p.ProcessBlock(context.Background(), 5, "0xd", []*blockchain.BlockchainEvent{event}))

	assert.Equal(t, 2, c.calls, "the block path must delegate dedup to the store, not pre-check")
	assert.Equal(t, int64(0), p.GetDuplicateCount(), "the block path must not touch the idempotency service")
}
