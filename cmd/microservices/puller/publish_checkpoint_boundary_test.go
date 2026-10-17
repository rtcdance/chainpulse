package main

import (
	"context"
	"errors"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type boundaryFailPublisher struct{ calls, failAt int }

func (p *boundaryFailPublisher) Publish(context.Context, string, []byte) error {
	p.calls++
	if p.calls == p.failAt {
		return errors.New("broker unavailable")
	}
	return nil
}

func TestPublishFailureNeverAdvancesPastIncompleteBlock(t *testing.T) {
	t.Parallel()
	for _, failAt := range []int{1, 2} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			t.Parallel()
			logger := testhelpers.NewTestLogger()
			multi := pullers.NewMultiChainDataPuller(logger)
			source := &pullerExecutionTestPlugin{name: "ethereum", latestBlock: 1, events: []blockchain.BlockchainEvent{
				{ID: "log-0", ChainID: "ethereum", BlockNumber: 1, LogIndex: 0}, {ID: "log-1", ChainID: "ethereum", BlockNumber: 1, LogIndex: 1},
			}}
			require.NoError(t, multi.RegisterPuller("ethereum", source))
			publisher := &boundaryFailPublisher{failAt: failAt}
			r := newPullerExecutionRuntime(logger, testhelpers.NewTestMetricsCollector(), publisher, []string{"raw-events"})
			err := r.Poll(context.Background(), multi, PullerConfig{WorkerThreads: 1})
			require.Error(t, err)
			require.Zero(t, source.lastBlock, "an incomplete block must be retried in full")
			publisher.failAt = 0
			require.NoError(t, r.Poll(context.Background(), multi, PullerConfig{WorkerThreads: 1}))
			require.Equal(t, uint64(1), source.lastBlock, "a retry must republish the complete block")
		})
	}
}

type boundaryKeyedPublisher struct{ messages []core.MessageQueueMessage }

func (p *boundaryKeyedPublisher) Publish(context.Context, string, []byte) error {
	return errors.New("chain publication must use the keyed contract")
}

func (p *boundaryKeyedPublisher) PublishMessage(_ context.Context, message core.MessageQueueMessage) error {
	p.messages = append(p.messages, message)
	return nil
}

func TestPullerUsesChainPartitionKeyAndIncludesTrailingEmptyBlocks(t *testing.T) {
	t.Parallel()
	logger := testhelpers.NewTestLogger()
	source := &pullerExecutionTestPlugin{name: "31337", latestBlock: 2, events: []blockchain.BlockchainEvent{{ID: "log-0", ChainID: "31337", BlockNumber: 1}}}
	multi := pullers.NewMultiChainDataPuller(logger)
	require.NoError(t, multi.RegisterPuller("31337", source))
	publisher := &boundaryKeyedPublisher{}
	r := newPullerExecutionRuntime(logger, testhelpers.NewTestMetricsCollector(), publisher, []string{"raw-events"})
	require.NoError(t, r.Poll(context.Background(), multi, PullerConfig{WorkerThreads: 1}))
	require.Len(t, publisher.messages, 1)
	require.Equal(t, "31337", publisher.messages[0].PartitionKey)
	require.Equal(t, uint64(2), source.lastBlock)
}
