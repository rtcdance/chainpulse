package main

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type dispatchOwnedProcessor struct {
	eventProcessorTestMessageProcessor
	assigned, revoked int
	observed          error
}

func (p *dispatchOwnedProcessor) AssignOwnership(context.Context, string) error {
	p.assigned++
	return nil
}
func (p *dispatchOwnedProcessor) RevokeOwnership(context.Context) error { p.revoked++; return nil }
func (p *dispatchOwnedProcessor) ProcessCompleteBlock(ctx context.Context, _ appindexing.CompletedBlock) error {
	p.observed = ctx.Err()
	return p.observed
}

type dispatchOwnedConsumer struct {
	legacy  bool
	message core.MessageQueueMessage
}

func (c *dispatchOwnedConsumer) ConsumeMessages(context.Context, string, func(core.MessageQueueMessage) error) error {
	c.legacy = true
	return nil
}

func (c *dispatchOwnedConsumer) ConsumeOwnedMessages(ctx context.Context, _ string, handler func(context.Context, core.MessageQueueMessage) error, hooks mq.ConsumerOwnershipHooks) error {
	if err := hooks.Assigned(ctx, "generation-token"); err != nil {
		return err
	}
	active, cancel := context.WithCancel(ctx)
	cancel()
	err := handler(active, c.message)
	if releaseErr := hooks.Revoked(context.Background()); releaseErr != nil {
		return releaseErr
	}
	return err
}

func TestDurableConsumerDispatchesRevocableContextAndOwnershipHooks(t *testing.T) {
	t.Parallel()
	frame := appindexing.CompletedBlock{Schema: appindexing.CompletedBlockSchema, ChainID: "31337", Header: &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(0)}}
	payload, err := json.Marshal(frame)
	require.NoError(t, err)
	consumer := &dispatchOwnedConsumer{message: core.MessageQueueMessage{Payload: payload, PartitionKey: frame.ChainID}}
	processor := &dispatchOwnedProcessor{}
	runtime := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), consumer, processor, nil, nil, nil, nil)
	err = runtime.consumeTopic(context.Background(), "blocks", func(core.MessageQueueMessage) error { t.Error("legacy path must not run"); return nil })
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, processor.observed, context.Canceled)
	require.Empty(t, runtime.Snapshot().LastError, "normal revocation must not leave readiness degraded")
	require.False(t, consumer.legacy)
	require.Equal(t, 1, processor.assigned)
	require.Equal(t, 1, processor.revoked)
}

func TestDurableConsumerRefusesLegacyBrokerFallback(t *testing.T) {
	t.Parallel()
	runtime := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), &ackBoundaryConsumer{}, &dispatchOwnedProcessor{}, nil, nil, nil, nil)
	err := runtime.consumeTopic(context.Background(), "blocks", func(core.MessageQueueMessage) error { return nil })
	require.ErrorContains(t, err, "generation-owned Kafka consumption")
}
