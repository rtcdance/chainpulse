//go:build integration

package acceptance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestProductionKafkaKeepsEachChainOnOnePartition(t *testing.T) {
	brokers := kafkaBrokers()
	if brokers == nil {
		t.Skip("set CHAINPULSE_RUN_KAFKA_INTEGRATION=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	topic := fmt.Sprintf("chainpulse-chain-order-%d", time.Now().UnixNano())
	client := &kafka.Client{Addr: kafka.TCP(brokers...)}
	created, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: []kafka.TopicConfig{{Topic: topic, NumPartitions: 3, ReplicationFactor: 1}}})
	require.NoError(t, err)
	require.NoError(t, created.Errors[topic])
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		conn, err := kafka.DialContext(cleanupCtx, "tcp", brokers[0])
		require.NoError(t, err)
		defer conn.Close()
		require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
		require.NoError(t, conn.DeleteTopics(topic))
	})
	p := mq.NewKafkaMQPlugin("chain-order", "1", &core.Config{}, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, brokers, topic)
	require.NoError(t, p.Initialize(ctx, core.Config{}))
	require.NoError(t, p.Start(ctx))
	t.Cleanup(func() { require.NoError(t, p.Stop(context.Background())) })
	for sequence := range 12 {
		for _, chain := range []string{"31337", "31338"} {
			require.NoError(t, p.PublishMessage(ctx, mq.NewChainMessage(chain, topic, fmt.Appendf(nil, "%s:%d", chain, sequence))))
		}
	}
	// Wait for actual broker visibility even when a producer uses acks=0.
	ends := make([]int64, 3)
	require.Eventually(t, func() bool {
		var count int64
		for partition := range 3 {
			conn, err := kafka.DialLeader(ctx, "tcp", brokers[0], topic, partition)
			if err != nil {
				return false
			}
			_ = conn.SetDeadline(time.Now().Add(time.Second))
			end, err := conn.ReadLastOffset()
			_ = conn.Close()
			if err != nil {
				return false
			}
			ends[partition] = end
			count += end
		}
		return count == 24
	}, 10*time.Second, 50*time.Millisecond)
	chainPartitions := map[string]int{}
	chainSequences := map[string]int{}
	total := 0
	for partition := range 3 {
		reader := kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, Topic: topic, Partition: partition, MinBytes: 1, MaxBytes: 1 << 20})
		for offset := int64(0); offset < ends[partition]; offset++ {
			message, err := reader.ReadMessage(ctx)
			require.NoError(t, err)
			chain := string(message.Key)
			previous, exists := chainPartitions[chain]
			if exists {
				require.Equal(t, previous, partition, "one chain must never span partitions")
			}
			chainPartitions[chain] = partition
			require.Equal(t, fmt.Sprintf("%s:%d", chain, chainSequences[chain]), string(message.Value), "chain offset order must preserve publication order")
			chainSequences[chain]++
			total++
		}
		require.NoError(t, reader.Close())
	}
	require.Equal(t, 24, total)
	require.Equal(t, map[string]int{"31337": 12, "31338": 12}, chainSequences)
}
