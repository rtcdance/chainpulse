package mq

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestOwnedGenerationContextRejectsRevocationBeforeAsyncCancellation(t *testing.T) {
	t.Parallel()
	generation, cancel := context.WithCancel(context.Background())
	ctx := ownedGenerationContext{Context: context.Background(), generation: generation}
	require.NoError(t, ctx.Err())
	cancel()
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}

func TestOwnedQueueMessagePreservesBrokerIdentityAndRejectsForgedHeader(t *testing.T) {
	t.Parallel()
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "consistent", false: "forged"}[valid], func(t *testing.T) {
			t.Parallel()
			key := "31337"
			if !valid {
				key = "31338"
			}
			message := kafka.Message{Key: []byte("31337"), Offset: 7, Value: []byte("frame"), Headers: []kafka.Header{{Key: "partition_key", Value: []byte(key)}, {Key: "message_id", Value: []byte("identity")}}}
			queued, err := ownedQueueMessage("blocks", message)
			if !valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "31337", queued.PartitionKey)
			require.Equal(t, int64(7), queued.Offset)
			require.Equal(t, "identity", queued.ID)
		})
	}
}
