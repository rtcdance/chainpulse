//go:build integration

package acceptance

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func TestKafkaHandlerFailureIsRedeliveredBeforeLaterOffsets(t *testing.T) {
	brokers := kafkaBrokers()
	if brokers == nil {
		t.Skip("set CHAINPULSE_RUN_KAFKA_INTEGRATION=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	topic := fmt.Sprintf("chainpulse-ack-boundary-%d", time.Now().UnixNano())
	createTopic(t, brokers, topic)
	writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic, Balancer: &kafka.Hash{}}
	t.Cleanup(func() { require.NoError(t, writer.Close()) })
	require.NoError(t, writer.WriteMessages(ctx, kafka.Message{Value: []byte("first")}, kafka.Message{Value: []byte("second")}))
	group := topic + "-group"
	newConsumer := func() *mq.KafkaMQPlugin {
		p := mq.NewKafkaMQPlugin("ack-boundary", "1", &core.Config{}, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, brokers, group)
		require.NoError(t, p.Initialize(ctx, core.Config{}))
		require.NoError(t, p.Start(ctx))
		return p
	}
	first := newConsumer()
	failedCtx, stopFailed := context.WithCancel(ctx)
	var attempted string
	err := first.ConsumeMessages(failedCtx, topic, func(msg core.MessageQueueMessage) error {
		attempted = string(msg.Payload)
		stopFailed()
		return errors.New("durable write failed")
	})
	require.Error(t, err)
	require.Equal(t, "first", attempted)
	require.NoError(t, first.Stop(ctx))

	// Restart the SAME group; a new group would only test replay from offset 0.
	second := newConsumer()
	readCtx, stopRead := context.WithCancel(ctx)
	var replayed []string
	_ = second.ConsumeMessages(readCtx, topic, func(msg core.MessageQueueMessage) error {
		replayed = append(replayed, string(msg.Payload))
		if len(replayed) == 2 {
			stopRead()
		}
		return nil
	})
	require.NoError(t, second.Stop(ctx))
	require.Equal(t, []string{"first", "second"}, replayed, "a failed effect must not be acknowledged or overtaken")
}
