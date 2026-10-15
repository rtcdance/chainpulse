package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type orphanAckProcessor struct{ result error }

func (p orphanAckProcessor) ProcessCompleteBlock(context.Context, appindexing.CompletedBlock) error {
	return p.result
}

type orphanAckPublisher struct{ calls int }

func (p *orphanAckPublisher) Publish(context.Context, string, []byte) error { p.calls++; return nil }

func (p *orphanAckPublisher) PublishMessage(context.Context, core.MessageQueueMessage) error {
	p.calls++
	return nil
}

func TestOrphanAcknowledgementRequiresDurableAuditAndNeverPublishesCanonical(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"audited-orphan", "audit-failed", "canonical"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			frame := appindexing.CompletedBlock{Schema: appindexing.CompletedBlockSchema, ChainID: "31337", Header: &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(0)}}
			payload, err := json.Marshal(frame)
			require.NoError(t, err)
			publisher := &orphanAckPublisher{}
			runtime := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, nil, publisher, nil, []string{"committed"}, nil)
			processor := orphanAckProcessor{}
			if name == "audited-orphan" {
				processor.result = &appindexing.OrphanedBlockError{ChainID: frame.ChainID, Hash: frame.Header.Hash().Hex()}
			}
			if name == "audit-failed" {
				processor.result = errors.New("audit database unavailable")
			}
			err = runtime.processCompleteBlockMessage(context.Background(), core.MessageQueueMessage{PartitionKey: frame.ChainID, Payload: payload}, processor)
			if name == "audit-failed" {
				require.ErrorContains(t, err, "audit database unavailable")
			} else {
				require.NoError(t, err)
			}
			if name == "canonical" {
				require.Equal(t, 1, publisher.calls)
			} else {
				require.Zero(t, publisher.calls)
			}
		})
	}
}
