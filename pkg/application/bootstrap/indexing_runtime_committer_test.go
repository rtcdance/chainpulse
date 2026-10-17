package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

// recordingCommitter records the blocks the runtime asked it to commit.
type recordingCommitter struct {
	blocks []uint64
}

func (c *recordingCommitter) CommitBlock(_ context.Context, blockNumber uint64, _ string, _ []*blockchain.BlockchainEvent) error {
	c.blocks = append(c.blocks, blockNumber)
	return nil
}

func buildRuntimeWithOptions(t *testing.T, options InMemoryIndexingRuntimeOptions) *appindexing.SharedRuntime {
	t.Helper()

	rt, err := buildMonolithicIndexingRuntimeWithDeps(
		core.NewDefaultLogger(core.LogLevelError),
		&runtimeTestDatabasePlugin{},
		&runtimeTestCachePlugin{},
		[]string{"ethereum"},
		options,
		defaultMonolithicIndexingRuntimeDeps(),
	)
	require.NoError(t, err)
	require.NotNil(t, rt)

	ctx := context.Background()
	require.NoError(t, rt.Initialize(ctx))
	require.NoError(t, rt.Start(ctx))
	return rt
}

// TestMonolithicIndexingRuntimeWiresCommitter closes the T05 wiring loop: a
// committer handed in through the options must reach the runtime, so the atomic
// path is actually reachable from the real construction path rather than only
// from a hand-built runtime in a test.
func TestMonolithicIndexingRuntimeWiresCommitter(t *testing.T) {
	t.Parallel()

	committer := &recordingCommitter{}
	rt := buildRuntimeWithOptions(t, InMemoryIndexingRuntimeOptions{Committer: committer})

	err := rt.ProcessBatchAtomic(context.Background(), "ethereum", []appindexing.EventEnvelope{{
		EventKey:    "evt-1",
		ChainID:     "ethereum",
		BlockNumber: 42,
		Payload:     &blockchain.BlockchainEvent{ID: "evt-1", ChainID: "ethereum", BlockNumber: 42},
	}})
	require.NoError(t, err)
	assert.Equal(t, []uint64{42}, committer.blocks, "the injected committer must be reached")
}

// TestMonolithicIndexingRuntimeWithoutCommitterHasNoAtomicPath pins the
// opt-in behaviour: without a committer the runtime keeps the four-step path
// and ProcessBatchAtomic reports why it is unavailable.
func TestMonolithicIndexingRuntimeWithoutCommitterHasNoAtomicPath(t *testing.T) {
	t.Parallel()

	rt := buildRuntimeWithOptions(t, InMemoryIndexingRuntimeOptions{})

	err := rt.ProcessBatchAtomic(context.Background(), "ethereum", []appindexing.EventEnvelope{{
		EventKey:    "evt-1",
		ChainID:     "ethereum",
		BlockNumber: 1,
		Payload:     &blockchain.BlockchainEvent{ChainID: "ethereum", BlockNumber: 1},
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no atomic committer")
}
