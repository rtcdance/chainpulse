package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type completedAckProcessor struct{ ackBoundaryProcessor }

func (p *completedAckProcessor) ProcessCompleteBlock(ctx context.Context, _ appindexing.CompletedBlock) error {
	return p.ProcessEvent(ctx, nil)
}

func TestCompletedBlockConsumerWaitsForAtomicCommitBeforeAcknowledgement(t *testing.T) {
	t.Parallel()
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[failure], func(t *testing.T) {
			t.Parallel()
			frame := appindexing.CompletedBlock{Schema: appindexing.CompletedBlockSchema, ChainID: "31337", Header: &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(0)}}
			payload, err := json.Marshal(frame)
			require.NoError(t, err)
			processor := &completedAckProcessor{ackBoundaryProcessor: ackBoundaryProcessor{entered: make(chan struct{}), release: make(chan struct{})}}
			if failure {
				processor.result = errors.New("atomic transaction rolled back")
			}
			consumer := &ackBoundaryConsumer{message: core.MessageQueueMessage{Payload: payload, PartitionKey: "31337"}, returned: make(chan error, 1)}
			r := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), consumer, processor, nil, []string{"blocks.v1"}, nil, nil)
			ctx, cancel := context.WithCancel(context.Background())
			var workers sync.WaitGroup
			var release sync.Once
			t.Cleanup(func() { release.Do(func() { close(processor.release) }); cancel(); workers.Wait() })
			r.Start(ctx, &workers)
			select {
			case <-processor.entered:
			case <-time.After(2 * time.Second):
				t.Fatal("block commit never started")
			}
			select {
			case <-consumer.returned:
				t.Fatal("ack preceded the atomic commit")
			default:
			}
			release.Do(func() { close(processor.release) })
			select {
			case err := <-consumer.returned:
				if failure {
					require.ErrorContains(t, err, "rolled back")
				} else {
					require.NoError(t, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("commit result never returned")
			}
		})
	}
}
