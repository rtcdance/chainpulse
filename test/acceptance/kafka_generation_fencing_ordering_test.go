//go:build integration

package acceptance

import (
	"context"
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

// TestKafkaGenerationRevokedBeforeNextGenerationAssigned pins the rebalance
// ordering contract of delivery-plan T05 against a real coordinator:
//
//  1. The old owner's Revoked hook (Release → chain_leases soft-expiry) runs
//     to completion BEFORE any member of the next generation receives
//     Assigned — the window in which a rebalanced-away worker could still
//     write must not exist.
//  2. After revocation the chain_leases row still EXISTS with
//     expires_at <= clock_timestamp() under the old owner, and that owner is
//     rejected with ErrLeaseLost when it tries to open a fenced transaction.
//  3. The next generation takes the lease over (generation bumped) and its
//     fenced commit persists exactly once, while the old token stays
//     rejected afterwards.
//
// Unlike the neighbouring tests this one FAILS (it does not skip) when the
// integration environment is missing: it is the acceptance gate for the
// ordering invariant, and a silent skip would make the gate meaningless.
func TestKafkaGenerationRevokedBeforeNextGenerationAssigned(t *testing.T) {
	brokers, dsn := kafkaBrokers(), acceptanceDSN()
	if brokers == nil {
		t.Fatalf("set CHAINPULSE_RUN_KAFKA_INTEGRATION=1 (optionally CHAINPULSE_TEST_KAFKA_BROKERS); the T05 ordering gate must not skip")
	}
	if dsn == "" {
		t.Fatalf("set CHAINPULSE_RUN_POSTGRES_INTEGRATION=1 and CHAINPULSE_TEST_DATABASE_URL; the T05 ordering gate must not skip")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	topic := fmt.Sprintf("chainpulse-genorder-%d", time.Now().UnixNano())
	createTopic(t, brokers, topic)
	const chain = "t05-genorder"

	db := openDB(t, dsn)
	for _, table := range []string{"blockchain_events", "blocks", "indexing_state", "chain_leases"} {
		_, err := db.ExecContext(ctx, "DELETE FROM "+table+" WHERE chain_id=$1", chain)
		require.NoError(t, err)
	}

	// Shared, ordered hook log: every member appends to the same sequence so
	// cross-member ordering can be asserted exactly.
	type hookEvent struct {
		seq    int
		member string
		kind   string // "assigned" | "revoked"
		token  string
	}
	var recMu sync.Mutex
	var hookLog []hookEvent
	record := func(member, kind, token string) {
		recMu.Lock()
		defer recMu.Unlock()
		hookLog = append(hookLog, hookEvent{seq: len(hookLog), member: member, kind: kind, token: token})
	}
	snapshot := func() []hookEvent {
		recMu.Lock()
		defer recMu.Unlock()
		return append([]hookEvent(nil), hookLog...)
	}

	committed := make(chan string, 4) // owner tokens whose fenced commit succeeded

	var workers sync.WaitGroup
	var plugins []*mq.KafkaMQPlugin
	defer func() {
		cancel()
		done := make(chan struct{})
		go func() { workers.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Error("owned consumers failed to stop")
		}
		for _, plugin := range plugins {
			_ = plugin.Stop(context.Background())
		}
	}()

	event := txTestEventForChain(chain, 1)

	// startMember joins the consumer group. ensureFirstGen makes the member's
	// FIRST generation acquire the chain lease; only member A runs with true,
	// so two members can never race for the lease inside one generation (and
	// exactly one member owns the partition when the record is published).
	startMember := func(label string, ensureFirstGen bool) {
		store := newAdapterForDigest(t, dsn)
		committer, err := processor.NewBatchCommitter(store, nil, chain, label+"-initial", time.Minute)
		require.NoError(t, err)
		plugin := mq.NewKafkaMQPlugin(label, "1", &core.Config{}, testhelpers.NewTestLogger(), testhelpers.NewTestMetricsCollector(), nil, brokers, topic+"-group")
		require.NoError(t, plugin.Initialize(ctx, core.Config{}))
		require.NoError(t, plugin.Start(ctx))
		plugins = append(plugins, plugin)

		var owner atomic.Value
		owner.Store("")
		firstGen := true // touched only from this member's sequential hooks

		workers.Add(1)
		go func() {
			defer workers.Done()
			_ = plugin.ConsumeOwnedMessages(ctx, topic, func(active context.Context, _ core.MessageQueueMessage) error {
				tok, _ := owner.Load().(string)
				if err := committer.CommitBlock(active, 1, event.BlockHash.Hex(), []*blockchain.BlockchainEvent{event}); err != nil {
					return err
				}
				committed <- tok
				return nil
			}, mq.ConsumerOwnershipHooks{
				Assigned: func(active context.Context, token string) error {
					if err := committer.RotateOwner(active, token); err != nil {
						return err
					}
					if ensureFirstGen && firstGen {
						firstGen = false
						if err := committer.EnsureLease(active); err != nil {
							return err
						}
					}
					owner.Store(token)
					record(label, "assigned", token)
					return nil
				},
				Revoked: func(cleanup context.Context) error {
					err := committer.Release(cleanup)
					// Record only after the revoke effect has finished: the
					// ordering assertion below compares this position against
					// the next generation's assigned events.
					tok, _ := owner.Load().(string)
					record(label, "revoked", tok)
					return err
				},
			})
		}()
	}

	startMember("A", true)

	// 1. Generation 1 exists and its lease is live under its own token.
	require.Eventually(t, func() bool {
		for _, e := range snapshot() {
			if e.member == "A" && e.kind == "assigned" {
				return true
			}
		}
		return false
	}, 30*time.Second, 50*time.Millisecond, "member A never received its first generation")
	var tokenA string
	for _, e := range snapshot() {
		if e.member == "A" && e.kind == "assigned" {
			tokenA = e.token
			break
		}
	}
	require.NotEmpty(t, tokenA)

	var generation1 int64
	require.Eventually(t, func() bool {
		err := db.QueryRowContext(ctx,
			"SELECT generation FROM chain_leases WHERE chain_id=$1 AND owner_token=$2 AND expires_at > clock_timestamp()",
			chain, tokenA).Scan(&generation1)
		return err == nil
	}, 10*time.Second, 50*time.Millisecond, "generation 1 never acquired a live lease under its own token")

	// 2. A second live group member forces a real coordinator rebalance — the
	// generation ends WITHOUT a SIGTERM, which is the case T05 is about.
	startMember("B", false)

	require.Eventually(t, func() bool {
		sawRevoke, sawReassignA, sawAssignB := false, false, false
		for _, e := range snapshot() {
			switch {
			case e.kind == "revoked" && e.token == tokenA:
				sawRevoke = true
			case e.kind == "assigned" && e.member == "A" && e.token != tokenA:
				sawReassignA = true
			case e.kind == "assigned" && e.member == "B":
				sawAssignB = true
			}
		}
		return sawRevoke && sawReassignA && sawAssignB
	}, 30*time.Second, 50*time.Millisecond, "rebalance never produced revoke(A gen1) + reassign(A) + join(B)")

	// Ordering invariant: every next-generation assignment — intra- and
	// cross-member — must come strictly after the old owner's Revoked
	// completed (Revoked is recorded after Release returns).
	snap := snapshot()
	revokeSeq := -1
	for _, e := range snap {
		if e.kind == "revoked" && e.token == tokenA {
			revokeSeq = e.seq
			break
		}
	}
	require.GreaterOrEqual(t, revokeSeq, 0, "generation 1 revocation was never recorded")
	for _, e := range snap {
		if e.kind == "assigned" && e.token != tokenA {
			require.Greaterf(t, e.seq, revokeSeq,
				"member %s received generation token %s (seq %d) before the old owner's Revoked completed (seq %d)",
				e.member, e.token, e.seq, revokeSeq)
		}
	}

	// 3. Post-revocation database state: the row EXISTS, still attributed to
	// the old owner, and is soft-expired (expires_at <= clock_timestamp()).
	var stale int
	require.Eventually(t, func() bool {
		err := db.QueryRowContext(ctx,
			"SELECT count(*) FROM chain_leases WHERE chain_id=$1 AND owner_token=$2 AND expires_at <= clock_timestamp()",
			chain, tokenA).Scan(&stale)
		return err == nil && stale == 1
	}, 10*time.Second, 50*time.Millisecond,
		"revocation must leave the chain_leases row present and soft-expired under the old owner")

	// 4. The old owner is fenced out.
	storeOld := newAdapterForDigest(t, dsn)
	_, err := storeOld.BeginFencedTx(ctx, chain, tokenA)
	require.ErrorIs(t, err, database.ErrLeaseLost, "a revoked owner must not open a fenced transaction")

	// 5. The next generation takes the lease over and commits exactly once.
	writer := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic, Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll}
	require.NoError(t, writer.WriteMessages(ctx, kafka.Message{Key: []byte(chain), Value: []byte("next-generation-block")}))
	require.NoError(t, writer.Close())

	var newOwner string
	select {
	case newOwner = <-committed:
		require.NotEqual(t, tokenA, newOwner, "the committing worker must belong to the new generation")
	case <-ctx.Done():
		t.Fatal("the next generation never committed the record")
	}

	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count(*) FROM blockchain_events WHERE chain_id=$1", chain).Scan(&count))
	require.Equal(t, 1, count, "the block must persist exactly once")

	var generation2 int64
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT generation FROM chain_leases WHERE chain_id=$1 AND owner_token=$2", chain, newOwner).Scan(&generation2))
	require.Greater(t, generation2, generation1, "takeover must bump the fencing generation")

	// The old token stays rejected after the takeover.
	_, err = storeOld.BeginFencedTx(ctx, chain, tokenA)
	require.ErrorIs(t, err, database.ErrLeaseLost, "the old owner must stay fenced out after takeover")
}
