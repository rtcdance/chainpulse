//go:build integration

package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
	mqplugin "github.com/rtcdance/chainpulse/pkg/plugins/mq"
)

// MQContractTest defines the contract that all MQ implementations must satisfy
func MQContractTest(t *testing.T, factory func(t *testing.T) core.MQPlugin) {
	t.Run("publish_and_subscribe", func(t *testing.T) {
		mq := factory(t)
		ctx := context.Background()
		received := make(chan []byte, 1)

		err := mq.Subscribe(ctx, "test-topic", func(msg []byte) {
			received <- msg
		})
		require.NoError(t, err)

		err = mq.Publish(ctx, "test-topic", []byte("hello"))
		require.NoError(t, err)

		select {
		case msg := <-received:
			assert.Equal(t, "hello", string(msg))
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("multiple_subscribers", func(t *testing.T) {
		mq := factory(t)
		ctx := context.Background()
		received1 := make(chan []byte, 1)
		received2 := make(chan []byte, 1)

		err := mq.Subscribe(ctx, "multi-topic", func(msg []byte) {
			received1 <- msg
		})
		require.NoError(t, err)

		err = mq.Subscribe(ctx, "multi-topic", func(msg []byte) {
			received2 <- msg
		})
		require.NoError(t, err)

		err = mq.Publish(ctx, "multi-topic", []byte("broadcast"))
		require.NoError(t, err)

		// Both subscribers should receive
		select {
		case msg := <-received1:
			assert.Equal(t, "broadcast", string(msg))
		case <-time.After(5 * time.Second):
			t.Fatal("subscriber 1 timeout")
		}

		select {
		case msg := <-received2:
			assert.Equal(t, "broadcast", string(msg))
		case <-time.After(5 * time.Second):
			t.Fatal("subscriber 2 timeout")
		}
	})

	t.Run("different_topics_isolated", func(t *testing.T) {
		mq := factory(t)
		ctx := context.Background()
		receivedA := make(chan []byte, 1)
		receivedB := make(chan []byte, 1)

		err := mq.Subscribe(ctx, "topic-a", func(msg []byte) {
			receivedA <- msg
		})
		require.NoError(t, err)

		err = mq.Subscribe(ctx, "topic-b", func(msg []byte) {
			receivedB <- msg
		})
		require.NoError(t, err)

		// Publish only to topic-a
		err = mq.Publish(ctx, "topic-a", []byte("message-a"))
		require.NoError(t, err)

		select {
		case msg := <-receivedA:
			assert.Equal(t, "message-a", string(msg))
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for topic-a message")
		}

		// topic-b should not receive
		select {
		case <-receivedB:
			t.Fatal("topic-b should not receive message from topic-a")
		case <-time.After(100 * time.Millisecond):
			// Expected
		}
	})

	t.Run("empty_message", func(t *testing.T) {
		mq := factory(t)
		ctx := context.Background()
		received := make(chan []byte, 1)

		err := mq.Subscribe(ctx, "empty-topic", func(msg []byte) {
			received <- msg
		})
		require.NoError(t, err)

		err = mq.Publish(ctx, "empty-topic", []byte{})
		require.NoError(t, err)

		select {
		case msg := <-received:
			assert.Empty(t, msg)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for empty message")
		}
	})
}

// TestMemoryMQContract tests the in-memory MQ implementation
func TestMemoryMQContract(t *testing.T) {
	MQContractTest(t, func(t *testing.T) core.MQPlugin {
		mq := mqplugin.NewMemoryMQ()
		require.NoError(t, mq.Initialize(core.Config{}))
		require.NoError(t, mq.Start())

		t.Cleanup(func() {
			mq.Stop()
		})

		return mq
	})
}
