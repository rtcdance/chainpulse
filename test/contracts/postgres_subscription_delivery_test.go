//go:build integration

package contracts

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/rtcdance/chainpulse/pkg/services/processor"
	"github.com/stretchr/testify/require"
)

func TestPostgresSubscriptionDeliveryUsesCommittedAuthority(t *testing.T) {
	t.Parallel()
	dsn := postgresDSN()
	if dsn == "" {
		t.Skip("set PostgreSQL integration gate with isolated endpoint")
	}
	const chain = "subscription-delivery"
	ctx := context.Background()
	db := openContractDBForReturn(t, dsn)
	for _, query := range []string{
		`DELETE FROM reorg_outbox WHERE job_id IN (SELECT id FROM reorg_jobs WHERE chain_id=$1)`,
		`DELETE FROM reorg_jobs WHERE chain_id=$1`, `DELETE FROM blockchain_events WHERE chain_id=$1`,
		`DELETE FROM blocks WHERE chain_id=$1`, `DELETE FROM indexing_state WHERE chain_id=$1`,
		`DELETE FROM reorg_generations WHERE chain_id=$1`, `DELETE FROM chain_leases WHERE chain_id=$1`,
		`DELETE FROM dlq_events WHERE chain_id=$1`,
	} {
		_, err := db.ExecContext(ctx, query, chain)
		require.NoError(t, err)
	}
	store := newAdapter(t, dsn).WithChainScope(chain)
	_, err := store.ReadSubscriptionBlock(ctx, chain, 1, common.HexToHash("0x01").Hex())
	require.ErrorContains(t, err, "precedes durable checkpoint", "a notification ahead of the checkpoint must not be acknowledged")
	committer, err := processor.NewBatchCommitter(store, nil, chain, "subscription-contract-owner", time.Minute)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, committer.Release(context.Background())) })
	block := &blockchain.Block{Number: 1, Hash: common.HexToHash("0xabc"), ParentHash: common.HexToHash("0x123"), Timestamp: 1700000000}
	event := txTestEvent(chain, 1)
	event.BlockHash = block.Hash
	event.ID = core.ComputeEventHash(event)
	event.EventHash = event.ID
	event.EventData = []byte{0, 255, 0, 128}
	event.EventTopic = []common.Hash{common.HexToHash("0x11")}
	amount := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()
	event.DecodedData = map[string]any{"value": json.Number(amount)}
	require.NoError(t, committer.CommitCompleteBlock(ctx, block, []*blockchain.BlockchainEvent{event}))
	result, err := store.ReadSubscriptionBlock(ctx, chain, 1, block.Hash.Hex())
	require.NoError(t, err)
	require.True(t, result.Canonical)
	require.Zero(t, result.Generation)
	require.Len(t, result.Events, 1)
	require.Equal(t, event.ID, result.Events[0].ID)
	require.Equal(t, event.EventData, result.Events[0].EventData)
	require.Equal(t, event.EventTopic, result.Events[0].EventTopic)
	require.Equal(t, json.Number(amount), result.Events[0].DecodedData["value"], "subscription projection cannot round stored numeric lexemes")
	result, err = store.ReadSubscriptionBlock(ctx, chain, 1, common.HexToHash("0xdef").Hex())
	require.NoError(t, err)
	require.False(t, result.Canonical)
	_, err = store.ReadSubscriptionBlock(ctx, "other-chain", 1, block.Hash.Hex())
	require.Error(t, err)
	_, err = store.CommitReorg(ctx, chain, 1, 1)
	require.NoError(t, err)
	result, err = store.ReadSubscriptionBlock(ctx, chain, 1, block.Hash.Hex())
	require.NoError(t, err)
	require.False(t, result.Canonical, "old branch queued after rewind is discarded, including before replay catches up")
	var noticeID int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT o.id FROM reorg_outbox o JOIN reorg_jobs j ON j.id=o.job_id WHERE j.chain_id=$1 AND o.topic=$2`, chain, topics.TopicReorgRollback).Scan(&noticeID))
	notice, err := store.ReadSubscriptionReorg(ctx, chain, noticeID)
	require.NoError(t, err)
	require.True(t, notice.Found)
	require.EqualValues(t, 1, notice.Generation)
	require.Equal(t, notice.Generation, notice.CurrentGeneration)
	var correction ports.ReorgRollbackEvent
	require.NoError(t, json.Unmarshal(notice.Payload, &correction))
	require.Equal(t, chain, correction.ChainID)
	require.EqualValues(t, 1, correction.FromBlock)
	message := ports.SubscriptionDeliveryMessage{ID: "fixture-message", Topic: "fixture-committed", PartitionKey: chain, Offset: 17, Payload: []byte{0, 255, 'x'}}
	require.NoError(t, store.QuarantineSubscription(ctx, message, "invalid-envelope"))
	require.NoError(t, store.QuarantineSubscription(ctx, message, "invalid-envelope"))
	var count int
	var payload string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*),min(payload) FROM dlq_events WHERE chain_id=$1 AND status='subscription_pending'`, chain).Scan(&count, &payload))
	require.Equal(t, 1, count, "quarantine survives retries without duplicate rows")
	var persisted ports.SubscriptionDeliveryMessage
	require.NoError(t, json.Unmarshal([]byte(payload), &persisted))
	require.Equal(t, message, persisted)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = store.ReadSubscriptionReorg(canceled, chain, noticeID)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, store.QuarantineSubscription(canceled, message, "invalid-envelope"), context.Canceled)
}
