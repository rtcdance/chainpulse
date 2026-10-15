package indexing

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestProcessBatchUsesConfiguredAtomicCommitter(t *testing.T) {
	t.Parallel()
	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)
	require.NoError(t, rt.ProcessBatch(context.Background(), "ethereum", []EventEnvelope{atomicEnvelope("a", 10)}))
	require.Len(t, c.calls, 1, "the production and replay entry must use the configured transaction")
}

func TestAtomicBatchRejectsMixedBlockHashesBeforeCommit(t *testing.T) {
	t.Parallel()
	c := &recordingBlockCommitter{}
	rt := startedAtomicRuntime(t, c)
	a := atomicEnvelope("a", 10)
	b := atomicEnvelope("b", 11)
	d := atomicEnvelope("c", 11)
	b.Payload.(*blockchain.BlockchainEvent).BlockHash = common.HexToHash("0x01")
	d.Payload.(*blockchain.BlockchainEvent).BlockHash = common.HexToHash("0x02")
	require.Error(t, rt.ProcessBatchAtomic(context.Background(), "ethereum", []EventEnvelope{a, b, d}))
	require.Empty(t, c.calls, "a batch containing conflicting branches must fail validation before any commit")
}

type routedDeliveryCommitter struct {
	recordingBlockCommitter
	chain string
}

func (c *routedDeliveryCommitter) CommitChainBlock(ctx context.Context, chain string, height uint64, hash string, events []*blockchain.BlockchainEvent) error {
	c.chain = chain
	return c.CommitBlock(ctx, height, hash, events)
}

func TestProcessEmptyBlockPersistsCheckpoint(t *testing.T) {
	t.Parallel()
	c := &routedDeliveryCommitter{}
	rt := startedAtomicRuntime(t, c)
	require.NoError(t, rt.ProcessEmptyBlock(context.Background(), "ethereum", 10, "hash"))
	require.Equal(t, "ethereum", c.chain)
	require.Equal(t, []commitCall{{block: 10, hash: "hash", events: 0}}, c.calls)
	require.Equal(t, uint64(10), rt.Status().LastCheckpointBlock)
}

func TestAtomicBatchRejectsPayloadIdentityMismatch(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"chain", "block"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			c := &recordingBlockCommitter{}
			rt := startedAtomicRuntime(t, c)
			a := atomicEnvelope("a", 10)
			if field == "chain" {
				a.Payload.(*blockchain.BlockchainEvent).ChainID = "polygon"
			} else {
				a.Payload.(*blockchain.BlockchainEvent).BlockNumber = 11
			}
			require.Error(t, rt.ProcessBatch(context.Background(), "ethereum", []EventEnvelope{a}))
			require.Empty(t, c.calls)
		})
	}
}
