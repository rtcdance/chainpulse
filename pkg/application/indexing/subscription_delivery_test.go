package indexing

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/stretchr/testify/require"
)

type deliveryTestStore struct {
	block      func(context.Context, string, uint64, string) (ports.SubscriptionBlock, error)
	reorg      ports.SubscriptionReorg
	readErr    error
	quarantine error
	reads      atomic.Int64
	bad        atomic.Int64
}

func (s *deliveryTestStore) ReadSubscriptionBlock(ctx context.Context, chain string, height uint64, hash string) (ports.SubscriptionBlock, error) {
	s.reads.Add(1)
	if s.readErr != nil {
		return ports.SubscriptionBlock{}, s.readErr
	}
	return s.block(ctx, chain, height, hash)
}

func (s *deliveryTestStore) ReadSubscriptionReorg(context.Context, string, int64) (ports.SubscriptionReorg, error) {
	return s.reorg, s.readErr
}

func (s *deliveryTestStore) QuarantineSubscription(context.Context, ports.SubscriptionDeliveryMessage, string) error {
	s.bad.Add(1)
	return s.quarantine
}

type deliveryTestSink struct {
	mu     sync.Mutex
	calls  int
	failAt int
	events []string
	reorgs int
}

func (s *deliveryTestSink) BroadcastEvent(_ context.Context, e *blockchain.BlockchainEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.calls == s.failAt {
		return errors.New("fixture broadcast failure")
	}
	s.events = append(s.events, e.ID)
	return nil
}

func (s *deliveryTestSink) BroadcastReorg(context.Context, *ports.ReorgRollbackEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reorgs++
	return nil
}

func deliveryBlockFixture(t *testing.T, chain string) (CompletedBlock, ports.SubscriptionBlock, ports.SubscriptionDeliveryMessage) {
	t.Helper()
	frame := completedBlockFixture()
	frame.ChainID = chain
	for i := range frame.Events {
		frame.Events[i].ChainID = chain
	}
	require.NoError(t, frame.Normalize())
	record := ports.SubscriptionBlock{Canonical: true, Generation: 3}
	for i := range frame.Events {
		record.Events = append(record.Events, &frame.Events[i])
	}
	payload, err := json.Marshal(frame)
	require.NoError(t, err)
	return frame, record, ports.SubscriptionDeliveryMessage{Topic: "fixture-committed", PartitionKey: chain, Payload: payload}
}

func TestSubscriptionDeliveryResumesAndDeduplicatesLocalEffects(t *testing.T) {
	t.Parallel()
	_, record, message := deliveryBlockFixture(t, "31337")
	store := &deliveryTestStore{block: func(context.Context, string, uint64, string) (ports.SubscriptionBlock, error) { return record, nil }}
	sink := &deliveryTestSink{failAt: 2}
	delivery, err := NewSubscriptionDelivery(store, sink, []string{"31337"}, nil)
	require.NoError(t, err)
	require.ErrorContains(t, delivery.Handle(context.Background(), message), "broadcast")
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.Len(t, sink.events, 2)
	require.Equal(t, 3, sink.calls)
	record.Generation++
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.Len(t, sink.events, 4, "ABA canonical generation must not be hidden by physical-ID dedupe")
}

func TestSubscriptionDeliveryQuarantinesInvalidMessages(t *testing.T) {
	t.Parallel()
	_, record, message := deliveryBlockFixture(t, "31337")
	for _, kind := range []string{"malformed", "schema", "partition", "chain", "raw-mismatch", "incomplete"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			store := &deliveryTestStore{block: func(context.Context, string, uint64, string) (ports.SubscriptionBlock, error) { return record, nil }}
			sink := &deliveryTestSink{}
			delivery, err := NewSubscriptionDelivery(store, sink, []string{"31337"}, nil)
			require.NoError(t, err)
			input := message
			var mutated CompletedBlock
			require.NoError(t, json.Unmarshal(message.Payload, &mutated))
			switch kind {
			case "malformed":
				input.Payload = []byte("bad")
			case "schema":
				mutated.Schema = "unsupported"
			case "partition":
				input.PartitionKey = "31338"
			case "chain":
				_, _, input = deliveryBlockFixture(t, "31338")
			case "raw-mismatch":
				mutated.Events[0].EventData = []byte("different raw bytes")
			case "incomplete":
				mutated.Events = mutated.Events[:1]
			}
			if kind == "schema" || kind == "raw-mismatch" || kind == "incomplete" {
				input.Payload, err = json.Marshal(mutated)
				require.NoError(t, err)
			}
			require.NoError(t, delivery.Handle(context.Background(), input), "successful durable quarantine permits acknowledgement")
			require.EqualValues(t, 1, store.bad.Load())
			require.Empty(t, sink.events)
		})
	}
}

func TestSubscriptionDeliveryDependencyFailuresPreventAcknowledgement(t *testing.T) {
	t.Parallel()
	_, record, message := deliveryBlockFixture(t, "31337")
	store := &deliveryTestStore{readErr: context.DeadlineExceeded}
	sink := &deliveryTestSink{}
	delivery, err := NewSubscriptionDelivery(store, sink, []string{"31337"}, nil)
	require.NoError(t, err)
	require.ErrorIs(t, delivery.Handle(context.Background(), message), context.DeadlineExceeded)
	require.Zero(t, store.bad.Load())
	store.readErr = nil
	store.block = func(context.Context, string, uint64, string) (ports.SubscriptionBlock, error) {
		record.Canonical = false
		return record, nil
	}
	require.NoError(t, delivery.Handle(context.Background(), message), "orphan notification is not a new event")
	require.Empty(t, sink.events)
	store.quarantine = context.DeadlineExceeded
	message.Payload = []byte("bad")
	require.ErrorIs(t, delivery.Handle(context.Background(), message), context.DeadlineExceeded)
}

func TestSubscriptionDeliveryChainGatesHonorCancellation(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		_, a, first := deliveryBlockFixture(t, "31337")
		_, b, second := deliveryBlockFixture(t, "31338")
		entered, release := make(chan struct{}), make(chan struct{})
		store := &deliveryTestStore{block: func(_ context.Context, chain string, _ uint64, _ string) (ports.SubscriptionBlock, error) {
			if chain == "31337" {
				close(entered)
				<-release
				return a, nil
			}
			return b, nil
		}}
		sink := &deliveryTestSink{}
		delivery, err := NewSubscriptionDelivery(store, sink, []string{"31337", "31338"}, nil)
		require.NoError(t, err)
		var wg sync.WaitGroup
		wg.Go(func() { require.NoError(t, delivery.Handle(context.Background(), first)) })
		<-entered
		require.NoError(t, delivery.Handle(context.Background(), second), "chain A cannot block chain B")
		ctx, cancel := context.WithCancel(context.Background())
		waiter := make(chan error, 1)
		wg.Go(func() { waiter <- delivery.Handle(ctx, first) })
		synctest.Wait()
		cancel()
		require.ErrorIs(t, <-waiter, context.Canceled)
		close(release)
		wg.Wait()
		require.Len(t, sink.events, 4)
	})
}

func TestSubscriptionDeliveryReorgBindsDurableOutbox(t *testing.T) {
	t.Parallel()
	notice := reorgNoticeFixture(t)
	body, err := json.Marshal(notice)
	require.NoError(t, err)
	store := &deliveryTestStore{reorg: ports.SubscriptionReorg{Found: true, ChainID: notice.ChainID, JobID: notice.JobID, Kind: notice.Kind, Generation: notice.Generation, CurrentGeneration: notice.Generation, Payload: notice.Payload}}
	sink := &deliveryTestSink{}
	delivery, err := NewSubscriptionDelivery(store, sink, []string{"31337"}, nil)
	require.NoError(t, err)
	message := ports.SubscriptionDeliveryMessage{Topic: "fixture-reorg", PartitionKey: "31337", Payload: body}
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.Equal(t, 1, sink.reorgs)
	store.reorg.CurrentGeneration++
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.Equal(t, 1, sink.reorgs)
	store.reorg.Generation++
	require.NoError(t, delivery.Handle(context.Background(), message))
	require.EqualValues(t, 1, store.bad.Load(), "metadata mismatch must be quarantined even for a previously seen notice")
}
