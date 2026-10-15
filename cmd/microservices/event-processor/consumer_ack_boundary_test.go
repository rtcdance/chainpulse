package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type ackBoundaryProcessor struct {
	eventProcessorTestMessageProcessor
	entered chan struct{}
	release chan struct{}
	result  error
	once    sync.Once
}

func (p *ackBoundaryProcessor) ProcessEvent(ctx context.Context, _ *blockchain.BlockchainEvent) error {
	p.once.Do(func() { close(p.entered) })
	select {
	case <-p.release:
		return p.result
	case <-ctx.Done():
		return ctx.Err()
	}
}

type ackBoundaryConsumer struct {
	message  core.MessageQueueMessage
	returned chan error
	once     sync.Once
}

func (c *ackBoundaryConsumer) ConsumeMessages(ctx context.Context, _ string, handler func(core.MessageQueueMessage) error) error {
	var result error
	c.once.Do(func() { result = handler(c.message); c.returned <- result })
	if result != nil {
		return result
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestConsumerAcknowledgementWaitsForDurableProcessing(t *testing.T) {
	t.Parallel()
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "failure"}[failure], func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(blockchain.BlockchainEvent{ID: "ack-boundary", ChainID: "ethereum", BlockNumber: 1})
			require.NoError(t, err)
			p := &ackBoundaryProcessor{entered: make(chan struct{}), release: make(chan struct{})}
			if failure {
				p.result = errors.New("durable write failed")
			}
			consumer := &ackBoundaryConsumer{message: core.MessageQueueMessage{ID: "message", Payload: payload}, returned: make(chan error, 1)}
			r := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), consumer, p, nil, []string{"raw-events"}, nil, nil)
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			var releaseOnce sync.Once
			t.Cleanup(func() { releaseOnce.Do(func() { close(p.release) }); cancel(); wg.Wait() })
			r.Start(ctx, &wg)
			select {
			case <-p.entered:
			case <-time.After(2 * time.Second):
				t.Fatal("processing never began")
			}
			select {
			case <-consumer.returned:
				t.Fatal("handler returned before durable processing finished")
			default:
			}
			releaseOnce.Do(func() { close(p.release) })
			select {
			case err := <-consumer.returned:
				if failure {
					require.ErrorContains(t, err, "durable write failed")
				} else {
					require.NoError(t, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not report processing outcome")
			}
		})
	}
}

func TestEveryPauseResumeCycleBroadcastsIntakeSignal(t *testing.T) {
	t.Parallel()
	r := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, nil, nil, nil, nil, nil)
	for range 3 {
		require.True(t, r.PauseIntake("maintenance").Paused)
		r.mu.RLock()
		waiting := r.waitCh
		r.mu.RUnlock()
		require.False(t, r.ResumeIntake("recovered").Paused)
		select {
		case <-waiting:
		default:
			t.Fatal("a resume failed to wake the paused intake")
		}
	}
}

type ackBoundaryFailedPublisher struct{}

func (ackBoundaryFailedPublisher) Publish(context.Context, string, []byte) error {
	return errors.New("output broker unavailable")
}

func TestOutputPublishFailurePreventsInputAcknowledgement(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(blockchain.BlockchainEvent{ID: "output-boundary", ChainID: "31337", BlockNumber: 1})
	require.NoError(t, err)
	consumer := &ackBoundaryConsumer{message: core.MessageQueueMessage{Payload: payload}, returned: make(chan error, 1)}
	r := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), consumer, &eventProcessorTestMessageProcessor{}, ackBoundaryFailedPublisher{}, []string{"raw-events"}, []string{"processed-events"}, nil)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	t.Cleanup(func() { cancel(); wg.Wait() })
	r.Start(ctx, &wg)
	select {
	case err := <-consumer.returned:
		require.ErrorContains(t, err, "output broker unavailable")
	case <-time.After(2 * time.Second):
		t.Fatal("publication failure was not returned to the input consumer")
	}
}
