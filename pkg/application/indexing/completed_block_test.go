package indexing

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/stretchr/testify/require"
)

func completedBlockFixture() CompletedBlock {
	header := &types.Header{Number: big.NewInt(7), Time: 1700000000, Difficulty: big.NewInt(0)}
	return CompletedBlock{Schema: CompletedBlockSchema, ChainID: "31337", Header: header, Events: []blockchain.BlockchainEvent{
		{ChainID: "31337", BlockNumber: 7, BlockHash: header.Hash(), TransactionHash: common.HexToHash("0x11"), TransactionIndex: 1, LogIndex: 2, EventData: []byte{0, 255, 1}},
		{ChainID: "31337", BlockNumber: 7, BlockHash: header.Hash(), TransactionHash: common.HexToHash("0x10"), TransactionIndex: 0, LogIndex: 1},
	}}
}

type completeHeaderRecorder struct {
	recordingBlockCommitter
	headers []*blockchain.Block
}

func (c *completeHeaderRecorder) CommitChainCompleteBlock(ctx context.Context, _ string, block *blockchain.Block, events []*blockchain.BlockchainEvent) error {
	if err := c.CommitBlock(ctx, block.Number, block.Hash.Hex(), events); err != nil {
		return err
	}
	c.headers = append(c.headers, block)
	return nil
}

func TestCompleteBlockUsesDurableResumePointBeforeWriting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		height    uint64
		wantError bool
	}{{"next", 9, false}, {"gap", 10, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writer := &completeHeaderRecorder{}
			runtime := startedAtomicRuntime(t, writer)
			header := &types.Header{Number: new(big.Int).SetUint64(tc.height), Time: 1700000000, ParentHash: common.HexToHash("0x888")}
			err := runtime.ProcessCompleteBlock(context.Background(), CompletedBlock{Schema: CompletedBlockSchema, ChainID: "ethereum", Header: header})
			if tc.wantError {
				var gap *BlockGapError
				require.ErrorAs(t, err, &gap)
				require.Equal(t, tc.height, gap.Block)
				require.Equal(t, uint64(8), gap.Checkpoint)
				require.Error(t, err)
				require.Empty(t, writer.calls)
				require.Empty(t, writer.headers)
			} else {
				require.NoError(t, err)
				require.Len(t, writer.headers, 1)
				require.Equal(t, header.ParentHash, writer.headers[0].ParentHash)
				require.Equal(t, tc.height, runtime.Status().LastCheckpointBlock)
			}
		})
	}
}

func TestCompletedBlockRoundTripPreservesRawDataAndOrdersLogs(t *testing.T) {
	t.Parallel()
	block := completedBlockFixture()
	block.Events[0].DecodedData = map[string]any{"value": new(big.Int).Lsh(big.NewInt(1), 200)}
	require.NoError(t, block.Normalize())
	payload, err := json.Marshal(block)
	require.NoError(t, err)
	var decoded CompletedBlock
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.NoError(t, decoded.Normalize())
	require.Equal(t, block.Header.Hash(), decoded.Header.Hash())
	require.Equal(t, uint64(1), decoded.Events[0].LogIndex)
	require.Equal(t, []byte{0, 255, 1}, decoded.Events[1].EventData)
	require.Equal(t, json.Number(new(big.Int).Lsh(big.NewInt(1), 200).String()), decoded.Events[1].DecodedData["value"])
	require.NotEmpty(t, decoded.Events[0].ID)
	require.Equal(t, decoded.Events[0].ID, decoded.Events[0].EventHash)
}

func TestCompletedBlockRejectsUntrustworthyFrames(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*CompletedBlock)
	}{
		{"schema", func(b *CompletedBlock) { b.Schema = "chainpulse.block.v999" }},
		{"chain", func(b *CompletedBlock) { b.Events[0].ChainID = "31338" }},
		{"branch", func(b *CompletedBlock) { b.Events[0].BlockHash = common.HexToHash("0x99") }},
		{"height", func(b *CompletedBlock) { b.Events[0].BlockNumber++ }},
		{"removed", func(b *CompletedBlock) { b.Events[0].Removed = true }},
		{"duplicate-log-index", func(b *CompletedBlock) { b.Events[1].LogIndex = b.Events[0].LogIndex }},
		{"nil-header", func(b *CompletedBlock) { b.Header = nil }},
		{"overlarge-height", func(b *CompletedBlock) { b.Header.Number = new(big.Int).Lsh(big.NewInt(1), 65) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			block := completedBlockFixture()
			tc.mutate(&block)
			require.Error(t, block.Normalize())
		})
	}
}

func TestCompletedBlockSupportsZeroLogs(t *testing.T) {
	t.Parallel()
	block := completedBlockFixture()
	block.Events = nil
	require.NoError(t, block.Normalize())
}
