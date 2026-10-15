//go:build integration

package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
)

// kafkaBrokers resolves the broker list from the environment. The address is
// fully env-driven (decision D21(a)); CI must point one of these at the
// service-container address:
//
//	CHAINPULSE_TEST_KAFKA_BROKERS - this package's CHAINPULSE_TEST_* naming
//	KAFKA_BROKERS                 - the compose / test/integration convention
//
// The localhost fallback matches the compose port mapping (9092:9092).
func kafkaBrokers() []string {
	if os.Getenv("CHAINPULSE_RUN_KAFKA_INTEGRATION") != "1" {
		return nil
	}
	for _, key := range []string{"CHAINPULSE_TEST_KAFKA_BROKERS", "KAFKA_BROKERS"} {
		if raw := os.Getenv(key); raw != "" {
			return strings.Split(raw, ",")
		}
	}
	return []string{"localhost:9092"}
}

// createTopic provisions the topic over the Kafka protocol with the same
// kafka-go library pkg/plugins/mq uses (decision D21(a)). docker exec is not
// an option: a GitHub Actions service container is not reachable from the job
// shell. The plugin sets AllowAutoTopicCreation=false, so a topic that does
// not exist would make the rehearsal fail for the wrong reason.
//
// Every skip lives in this Setup phase and fires BEFORE the topic exists:
//   - bootstrap connection refused -> environment problem, skip;
//   - advertised broker address unreachable from this host (the broker
//     broadcasts kafka:9092 while only localhost:9092 is routable) -> skip.
//
// Once this function returns, the topic exists and every later error is a
// real failure - no skip may fire mid-test.
func createTopic(t *testing.T, brokers []string, topic string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Bootstrap reachability: a refused connection is an environment
	//    problem, not a rehearsal failure.
	{
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", brokers[0])
		if err != nil {
			t.Skipf("kafka broker %s is not reachable from this host (%v); "+
				"set CHAINPULSE_TEST_KAFKA_BROKERS/KAFKA_BROKERS to a reachable "+
				"broker (in CI: the service-container address)", brokers[0], err)
		}
		_ = conn.Close()
	}

	client := &kafka.Client{Addr: kafka.TCP(brokers[0])}

	// 2. Follow the advertised broker list exactly as kafka-go's writer will.
	//    An open bootstrap port does not prove the advertised listener
	//    resolves from here.
	md, err := client.Metadata(ctx, &kafka.MetadataRequest{})
	require.NoError(t, err, "fetching kafka metadata from %s", brokers[0])
	for _, b := range md.Brokers {
		addr := net.JoinHostPort(b.Host, fmt.Sprintf("%d", b.Port))
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			t.Skipf("kafka advertises %s which this host cannot reach (%v); "+
				"either make that address resolvable (e.g. /etc/hosts) or run "+
				"the broker with a host-reachable advertised listener", addr, err)
		}
		_ = conn.Close()
	}

	// 3. 1 partition, replication factor 1 - the shape E09 needs.
	res, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}},
	})
	require.NoError(t, err, "creating topic %s via the kafka protocol", topic)
	if e := res.Errors[topic]; e != nil && !errors.Is(e, kafka.TopicAlreadyExists) {
		require.NoError(t, e, "creating topic %s", topic)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c := &kafka.Client{Addr: kafka.TCP(brokers[0])}
		_, _ = c.DeleteTopics(ctx, &kafka.DeleteTopicsRequest{Topics: []string{topic}})
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
	createTopic(t, brokers, topic)

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

		// Reachability was decided in createTopic (Setup). Any failure here is
		// a real publish failure - it must fail the test, not skip it.
		err = publisher.Publish(ctx, topic, payload)
		require.NoError(t, err)
		events = append(events, event)
	}

	// 2. First delivery: consume and commit.
	first := collectPayloads(t, brokers, topic+"-first", topic, batchSize, 30*time.Second)
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
	second := collectPayloads(t, brokers, topic+"-second", topic, batchSize, 30*time.Second)
	require.Len(t, second, batchSize, "the redelivery must carry the same batch")

	// 4. Committing the redelivered batch must not change the effect.
	commitBatch(second)
	afterRedelivery := countEventsFor(t, dsn, chainID)

	assert.Equal(t, afterFirst, afterRedelivery,
		"a redelivered batch must not be applied twice (at-least-once delivery, single effect)")
	assert.Equal(t, int64(batchSize), afterRedelivery)
}
