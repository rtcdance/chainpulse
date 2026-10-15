package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type completedRangePublisher struct {
	messages []core.MessageQueueMessage
	failAt   int
}

func (p *completedRangePublisher) Publish(context.Context, string, []byte) error {
	return errors.New("unkeyed publication forbidden")
}

func (p *completedRangePublisher) PublishMessage(_ context.Context, message core.MessageQueueMessage) error {
	p.messages = append(p.messages, message)
	if len(p.messages) == p.failAt {
		return errors.New("broker failed")
	}
	return nil
}

func TestPullerPublishesWholeBlocksIncludingEmptyBlockAndRetriesFailedRange(t *testing.T) {
	t.Parallel()
	logger := testhelpers.NewTestLogger()
	h1 := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(0), Time: 1700000000}
	h2 := &types.Header{Number: big.NewInt(2), Difficulty: big.NewInt(0), Time: 1700000001, ParentHash: h1.Hash()}
	source := &pullerExecutionTestPlugin{name: "31337", latestBlock: 2, headers: map[uint64]*types.Header{1: h1, 2: h2}, events: []blockchain.BlockchainEvent{
		{ChainID: "31337", BlockNumber: 1, BlockHash: h1.Hash(), LogIndex: 0},
		{ChainID: "31337", BlockNumber: 1, BlockHash: h1.Hash(), LogIndex: 1},
	}}
	multi := pullers.NewMultiChainDataPuller(logger)
	require.NoError(t, multi.RegisterPuller("31337", source))
	publisher := &completedRangePublisher{failAt: 2}
	r := newPullerExecutionRuntime(logger, testhelpers.NewTestMetricsCollector(), publisher, []string{"chainpulse.blocks.v1"})
	// This unit isolates Kafka range publication. The deployed acceptance test
	// exercises the real PostgreSQL reconciler; no production fallback is used.
	r.canonicalIntake = func(context.Context, *pullers.MultiChainDataPuller, string, uint64) error { return nil }
	cfg := PullerConfig{WorkerThreads: 1, DurableBlocks: true, BatchSize: 100, ReorgDetectionDepth: 12}
	require.Error(t, r.Poll(context.Background(), multi, cfg))
	require.Zero(t, source.lastBlock)
	publisher.failAt = 0
	require.NoError(t, r.Poll(context.Background(), multi, cfg))
	require.Equal(t, uint64(2), source.lastBlock)
	require.Len(t, publisher.messages, 4)
	for i, message := range publisher.messages {
		var frame appindexing.CompletedBlock
		require.NoError(t, json.Unmarshal(message.Payload, &frame))
		require.NoError(t, frame.Normalize())
		require.Equal(t, "31337", message.PartitionKey)
		if i%2 == 0 {
			require.Len(t, frame.Events, 2)
		} else {
			require.Empty(t, frame.Events)
		}
	}
}

func TestDurablePullerCannotRunWithoutCanonicalSupervision(t *testing.T) {
	t.Parallel()
	logger := testhelpers.NewTestLogger()
	multi := pullers.NewMultiChainDataPuller(logger)
	require.NoError(t, multi.RegisterPuller("31337", &pullerExecutionTestPlugin{name: "31337", latestBlock: 2}))
	runtime := newPullerExecutionRuntime(logger, testhelpers.NewTestMetricsCollector(), &completedRangePublisher{}, []string{"blocks"})
	require.ErrorContains(t, runtime.Poll(context.Background(), multi, PullerConfig{DurableBlocks: true, ReorgDetectionDepth: 12}), "canonical PostgreSQL supervision")
}
