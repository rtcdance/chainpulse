//go:build integration

package acceptance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/rtcdance/chainpulse/pkg/testhelpers"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func TestKafkaGenerationRevokesDatabaseWriterBeforeReassignment(t *testing.T) {
	brokers, dsn := kafkaBrokers(), acceptanceDSN()
	if brokers == nil || dsn == "" {
		t.Skip("set PostgreSQL and Kafka integration flags and isolated targets")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	topic := fmt.Sprintf("chainpulse-generation-%d", time.Now().UnixNano())
	createTopic(t, brokers, topic)
	const chain = "t05-generation"
	db := openDB(t, dsn)
	for _, table := range []string{"blockchain_events", "blocks", "indexing_state", "chain_leases"} {
		_, err := db.ExecContext(ctx, "DELETE FROM "+table+" WHERE chain_id=$1", chain)
		require.NoError(t, err)
	}
	entered := make(chan string, 1)
	cancelled := make(chan error, 1)
	revoked := make(chan string, 16)
	committed := make(chan string, 4)
	allow := make(chan struct{})
	var allowOnce sync.Once
	var first atomic.Bool
	var workers sync.WaitGroup
	var plugins []*mq.KafkaMQPlugin
	defer func() {
		cancel()
		allowOnce.Do(func() { close(allow) })
		done := make(chan struct{})
		go func() { workers.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Error("owned consumers failed to stop")
		}
		for _, plugin := range plugins {
			require.NoError(t, plugin.Stop(context.Background()))
		}
	}()
	event := txTestEventForChain(chain, 1)
	start := func() {
		store := newAdapterForDigest(t, dsn)
		committer, err := processor.NewBatchCommitter(store, nil, chain, "initial", time.Minute)
		require.NoError(t, err)
		plugin := mq.NewKafkaMQPlugin("generation", "1", &core.Config{}, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, brokers, topic+"-group")
		require.NoError(t, plugin.Initialize(ctx, core.Config{}))
		require.NoError(t, plugin.Start(ctx))
		plugins = append(plugins, plugin)
		var owner string
		workers.Add(1)
		go func() {
			defer workers.Done()
			_ = plugin.ConsumeOwnedMessages(ctx, topic, func(active context.Context, _ core.MessageQueueMessage) error {
				if first.CompareAndSwap(false, true) {
					if err := committer.EnsureLease(active); err != nil {
						return err
					}
					tx, err := store.BeginFencedTx(active, chain, owner)
					if err != nil {
						return err
					}
					defer tx.Rollback(context.Background())
					if err := tx.StoreEvent(active, event); err != nil {
						return err
					}
					entered <- owner
					<-active.Done()
					err = tx.Commit(active)
					cancelled <- err
					return active.Err()
				}
				select {
				case <-active.Done():
					return active.Err()
				case <-allow:
				}
				if err := committer.CommitBlock(active, 1, event.BlockHash.Hex(), []*blockchain.BlockchainEvent{event}); err != nil {
					return err
				}
				committed <- owner
				return nil
			}, mq.ConsumerOwnershipHooks{Assigned: func(active context.Context, token string) error {
				owner = token
				return committer.RotateOwner(active, token)
			}, Revoked: func(cleanup context.Context) error {
				err := committer.Release(cleanup)
				select {
				case revoked <- owner:
				default:
				}
				return err
			}})
		}()
	}
	start()
	writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
	require.NoError(t, writer.WriteMessages(ctx, kafka.Message{Key: []byte(chain), Value: []byte("one-block")}))
	require.NoError(t, writer.Close())
	var oldOwner string
	select {
	case oldOwner = <-entered:
	case <-ctx.Done():
		t.Fatal("first generation never entered a transaction")
	}
	start() // A live second group member forces a real coordinator generation.
	select {
	case err := <-cancelled:
		require.Error(t, err, "revoked transaction must not commit")
	case <-ctx.Done():
		t.Fatal("broker revocation did not cancel old handler")
	}
	for {
		select {
		case owner := <-revoked:
			if owner == oldOwner {
				goto released
			}
		case <-ctx.Done():
			t.Fatal("old ownership was never released")
		}
	}
released:
	store := newAdapterForDigest(t, dsn)
	_, err := store.BeginFencedTx(ctx, chain, oldOwner)
	require.ErrorIs(t, err, database.ErrLeaseLost)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM blockchain_events WHERE chain_id=$1", chain).Scan(&count))
	require.Zero(t, count)
	client := &kafka.Client{Addr: kafka.TCP(brokers...)}
	readOffset := func() (int64, error) {
		response, err := client.OffsetFetch(ctx, &kafka.OffsetFetchRequest{GroupID: topic + "-group", Topics: map[string][]int{topic: {0}}})
		if err != nil {
			return 0, err
		}
		if response.Error != nil {
			return 0, response.Error
		}
		if len(response.Topics[topic]) != 1 {
			return 0, errors.New("missing partition offset")
		}
		entry := response.Topics[topic][0]
		return entry.CommittedOffset, entry.Error
	}
	offset, err := readOffset()
	require.NoError(t, err)
	require.LessOrEqual(t, offset, int64(0), "cancelled effect cannot acknowledge the record")
	allowOnce.Do(func() { close(allow) })
	select {
	case owner := <-committed:
		require.NotEqual(t, oldOwner, owner)
	case <-ctx.Done():
		t.Fatal("new generation did not replay the cancelled record")
	}
	require.Eventually(t, func() bool { offset, err := readOffset(); return err == nil && offset == 1 }, 10*time.Second, 50*time.Millisecond)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM blockchain_events WHERE chain_id=$1", chain).Scan(&count))
	require.Equal(t, 1, count)
}
