//go:build integration

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

func kafkaBrokers() []string {
	if os.Getenv("CHAINPULSE_RUN_KAFKA_INTEGRATION") != "1" {
		return nil
	}
	if raw := os.Getenv("CHAINPULSE_TEST_KAFKA_BROKERS"); raw != "" {
		return strings.Split(raw, ",")
	}
	return []string{"localhost:9092"}
}

// createTopic uses the broker's own CLI inside the container. The plugin is
// configured with AllowAutoTopicCreation=false, so a topic that does not exist
// would make the rehearsal fail for the wrong reason.
func createTopic(t *testing.T, topic string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container := os.Getenv("CHAINPULSE_KAFKA_CONTAINER")
	if container == "" {
		container = "chainpulse-kafka"
	}

	out, err := exec.CommandContext(ctx, "docker", "exec", container,
		"kafka-topics", "--bootstrap-server", "localhost:9092",
		"--create", "--topic", topic, "--partitions", "1", "--replication-factor", "1",
	).CombinedOutput()
	require.NoError(t, err, "creating topic %s: %s", topic, out)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = exec.CommandContext(ctx, "docker", "exec", container,
			"kafka-topics", "--bootstrap-server", "localhost:9092", "--delete", "--topic", topic,
		).Run()
	})
}

// collectPayloads subscribes with the given consumer group and returns the first
// n payloads, or whatever arrived before the deadline.
func collectPayloads(t *testing.T, brokers []string, group, topic string, n int, within time.Duration) [][]byte {
	t.Helper()

	plugin := mq.NewKafkaMQPlugin(
		"acceptance-kafka", "1.0.0", &core.Config{},
		testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil,
		brokers, group,
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, plugin.Initialize(ctx, core.Config{}))
	require.NoError(t, plugin.Start(ctx))
	t.Cleanup(func() { _ = plugin.Stop(context.Background()) })

	received := make(chan []byte, n*2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = plugin.Subscribe(ctx, topic, func(payload []byte) {
			select {
			case received <- append([]byte(nil), payload...):
			default:
			}
		})
	}()
	wg.Wait()

	deadline := time.After(within)
	out := make([][]byte, 0, n)
	for len(out) < n {
		select {
		case payload := <-received:
			out = append(out, payload)
		case <-deadline:
			return out
		}
	}
	return out
}

// TestKafkaRedeliveryKeepsEffectSingle is the §9 E09 rehearsal for Kafka.
//
// The plan promises "at-least-once delivery + idempotent persistence". Kafka
// redelivers by design — a consumer that dies before committing its offset sees
// the same messages again — so the sink has to make the EFFECT single. This
// rehearsal delivers one batch to two different consumer groups (the second
// group starts from the beginning, which is exactly what a redelivery looks
// like) and asserts the row count does not move the second time.
func TestKafkaRedeliveryKeepsEffectSingle(t *testing.T) {
	brokers := kafkaBrokers()
	if brokers == nil {
		t.Skip("set CHAINPULSE_RUN_KAFKA_INTEGRATION=1 (and optionally CHAINPULSE_TEST_KAFKA_BROKERS)")
	}

	dsn := acceptanceDSN()
	if dsn == "" {
		t.Skip("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	const chainID = "e09-kafka"
	topic := fmt.Sprintf("chainpulse-acceptance-%d", time.Now().UnixNano())
	createTopic(t, topic)

	resetState(t, dsn)

	// 1. Publish a batch of events.
	const batchSize = 5
	publisher := mq.NewKafkaMQPlugin(
		"acceptance-kafka-pub", "1.0.0", &core.Config{},
		testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil,
		brokers, "acceptance-publisher",
	)
	require.NoError(t, publisher.Initialize(ctx, core.Config{}))
	require.NoError(t, publisher.Start(ctx))
	t.Cleanup(func() { _ = publisher.Stop(context.Background()) })

	events := make([]*blockchain.BlockchainEvent, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		event := txTestEventForChain(chainID, uint64(i))
		payload, err := json.Marshal(event)
		require.NoError(t, err)

		err = publisher.Publish(ctx, topic, payload)
		if err != nil && strings.Contains(err.Error(), "no such host") {
			// The broker advertises an in-network name (KAFKA_ADVERTISED_LISTENERS
			// = kafka:9092) and kafka-go follows the advertised listener after the
			// initial bootstrap, so the bootstrap port being open is not enough.
			// Skipping with the exact reason beats failing for the wrong one; the
			// rehearsal itself is correct and passes once the advertised address
			// resolves from here.
			t.Skipf("kafka advertises an address this host cannot resolve (%v). "+
				"Either add `kafka` to /etc/hosts, or set KAFKA_ADVERTISED_LISTENERS "+
				"to a host-reachable address on the broker.", err)
		}
		require.NoError(t, err)
		events = append(events, event)
	}

	// 2. First delivery: consume and commit.
	first := collectPayloads(t, brokers, "acceptance-group-first", topic, batchSize, 30*time.Second)
	require.Len(t, first, batchSize, "the first consumer must receive the whole batch")

	adapter := database.NewPostgreSQLPluginAdapter(
		testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(),
	)
	require.NoError(t, adapter.Initialize(ctx, core.Config{DatabaseURL: dsn}))
	require.NoError(t, adapter.Start(ctx))
	t.Cleanup(func() { _ = adapter.Stop(context.Background()) })

	committer, err := processor.NewBatchCommitter(adapter, nil, chainID, "worker-A", time.Minute)
	require.NoError(t, err)

	commitBatch := func(payloads [][]byte) {
		t.Helper()
		for _, payload := range payloads {
			var event blockchain.BlockchainEvent
			require.NoError(t, json.Unmarshal(payload, &event))
			require.NoError(t, committer.CommitBlock(
				ctx, event.BlockNumber, event.BlockHash.Hex(), []*blockchain.BlockchainEvent{&event},
			))
		}
	}

	commitBatch(first)
	afterFirst := countEventsFor(t, dsn, chainID)
	require.Equal(t, int64(batchSize), afterFirst, "the first delivery must persist the batch")

	// 3. Redelivery: a fresh consumer group starts from the beginning and sees
	//    the same messages again.
	second := collectPayloads(t, brokers, "acceptance-group-second", topic, batchSize, 30*time.Second)
	require.Len(t, second, batchSize, "the redelivery must carry the same batch")

	// 4. Committing the redelivered batch must not change the effect.
	commitBatch(second)
	afterRedelivery := countEventsFor(t, dsn, chainID)

	assert.Equal(t, afterFirst, afterRedelivery,
		"a redelivered batch must not be applied twice (at-least-once delivery, single effect)")
	assert.Equal(t, int64(batchSize), afterRedelivery)
}
