package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

type recoveringOwnedConsumer struct {
	calls    int
	assigned func()
}

func (c *recoveringOwnedConsumer) ConsumeMessages(context.Context, string, func(core.MessageQueueMessage) error) error {
	return errors.New("unexpected legacy path")
}

func (c *recoveringOwnedConsumer) ConsumeOwnedMessages(ctx context.Context, _ string, _ func(context.Context, core.MessageQueueMessage) error, hooks mq.ConsumerOwnershipHooks) error {
	c.calls++
	if c.calls == 1 {
		return errors.New("coordinator unavailable")
	}
	if err := hooks.Assigned(ctx, "recovered-generation"); err != nil {
		return err
	}
	c.assigned()
	<-ctx.Done()
	return errors.Join(ctx.Err(), hooks.Revoked(context.Background()))
}

func TestDurableReadinessRecoversTransportWithoutNewRecord(t *testing.T) {
	t.Parallel()
	for _, unresolvedEffect := range []bool{false, true} {
		t.Run(map[bool]string{false: "transport only", true: "effect failure remains"}[unresolvedEffect], func(t *testing.T) {
			t.Parallel()
			assigned := make(chan eventProcessorConsumeLoopSnapshot, 1)
			consumer := &recoveringOwnedConsumer{}
			runtime := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), consumer, &dispatchOwnedProcessor{}, nil, []string{"blocks"}, nil, nil)
			consumer.assigned = func() { assigned <- runtime.Snapshot() }
			if unresolvedEffect {
				runtime.recordError(errors.New("durable effect unresolved"))
			}
			ctx, cancel := context.WithCancel(context.Background())
			var workers sync.WaitGroup
			t.Cleanup(func() { cancel(); workers.Wait() })
			runtime.Start(ctx, &workers)
			select {
			case state := <-assigned:
				require.True(t, state.OwnershipRequired)
				require.Equal(t, 1, state.OwnedTopics)
				if unresolvedEffect {
					require.Contains(t, state.LastError, "durable effect unresolved")
				} else {
					require.Empty(t, state.LastError, "a recovered coordinator must not require a new record to clear readiness")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("consumer did not recover its generation")
			}
			cancel()
			workers.Wait()
			require.Zero(t, runtime.Snapshot().OwnedTopics, "revoked generation cannot remain ready")
		})
	}
}

func TestOwnedRecoveryDoesNotClearAnotherTopicFailure(t *testing.T) {
	t.Parallel()
	runtime := newEventProcessorConsumeRuntime(testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, &dispatchOwnedProcessor{}, nil, []string{"a", "b"}, nil, nil)
	runtime.recordTransportError("a", errors.New("a unavailable"))
	runtime.recordTransportError("b", errors.New("b unavailable"))
	runtime.setOwnership("a", true)
	require.Equal(t, "b unavailable", runtime.Snapshot().LastError)
	runtime.setOwnership("b", true)
	require.Empty(t, runtime.Snapshot().LastError)
	runtime.recordError(errors.New("unresolved durable effects"))
	runtime.setOwnership("a", false)
	runtime.setOwnership("a", true)
	require.Equal(t, "unresolved durable effects", runtime.Snapshot().LastError)
}
