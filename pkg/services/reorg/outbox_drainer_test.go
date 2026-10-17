package reorg

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// fakeOutbox is an in-memory OutboxDrainer.
type fakeOutbox struct {
	mu        sync.Mutex
	pending   []Notification
	published []int64
	failures  []int64
	loadErr   error
	markErr   error
}

func (o *fakeOutbox) PendingNotifications(_ context.Context, limit int) ([]Notification, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.loadErr != nil {
		return nil, o.loadErr
	}
	if limit > len(o.pending) {
		limit = len(o.pending)
	}
	return append([]Notification(nil), o.pending[:limit]...), nil
}

func (o *fakeOutbox) MarkPublished(_ context.Context, id int64) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.markErr != nil {
		return o.markErr
	}
	o.published = append(o.published, id)
	for i, n := range o.pending {
		if n.ID == id {
			o.pending = append(o.pending[:i], o.pending[i+1:]...)
			break
		}
	}
	return nil
}

func (o *fakeOutbox) RecordFailure(_ context.Context, id int64) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.failures = append(o.failures, id)
	return nil
}

func (o *fakeOutbox) pendingCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.pending)
}

// fakePublisher records published payloads and can fail selected topics.
type fakePublisher struct {
	mu        sync.Mutex
	published []string
	failTopic string
}

func (p *fakePublisher) Publish(_ context.Context, topic string, _ []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failTopic != "" && topic == p.failTopic {
		return errors.New("publish failed")
	}
	p.published = append(p.published, topic)
	return nil
}

func newDrainer(t *testing.T, outbox OutboxDrainer, publisher NotificationPublisher) *OutboxDrainerLoop {
	t.Helper()
	d, err := NewOutboxDrainerLoop(outbox, publisher, core.NewDefaultLogger(core.LogLevelError))
	require.NoError(t, err)
	return d
}

func TestOutboxDrainerPublishesAndMarks(t *testing.T) {
	t.Parallel()

	outbox := &fakeOutbox{pending: []Notification{
		{ID: 1, Topic: "reorg.detected", Payload: []byte(`{"a":1}`)},
		{ID: 2, Topic: "reorg.rollback", Payload: []byte(`{"b":2}`)},
	}}
	publisher := &fakePublisher{}

	published, err := newDrainer(t, outbox, publisher).DrainOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, published)
	assert.Equal(t, []string{"reorg.detected", "reorg.rollback"}, publisher.published)
	assert.Equal(t, 0, outbox.pendingCount(), "delivered notifications must be marked")
}

func TestOutboxDrainerRecordsFailureAndContinues(t *testing.T) {
	t.Parallel()

	outbox := &fakeOutbox{pending: []Notification{
		{ID: 1, Topic: "poison", Payload: []byte(`{}`)},
		{ID: 2, Topic: "reorg.rollback", Payload: []byte(`{}`)},
	}}
	publisher := &fakePublisher{failTopic: "poison"}

	published, err := newDrainer(t, outbox, publisher).DrainOnce(context.Background())
	require.Error(t, err, "the first failure must be reported")
	assert.Equal(t, 1, published, "one poison notification must not block the rest")
	assert.Equal(t, []int64{1}, outbox.failures)
	assert.Equal(t, 1, outbox.pendingCount(), "the failed notification stays pending for a retry")
}

// TestOutboxDrainerIsAtLeastOnce pins the ordering: publish first, mark second.
// A mark failure must leave the notification pending, never dropped.
func TestOutboxDrainerIsAtLeastOnce(t *testing.T) {
	t.Parallel()

	outbox := &fakeOutbox{
		pending: []Notification{{ID: 1, Topic: "reorg.detected", Payload: []byte(`{}`)}},
		markErr: errors.New("mark failed"),
	}
	publisher := &fakePublisher{}

	published, err := newDrainer(t, outbox, publisher).DrainOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, published, "nothing counts as delivered until it is marked")
	assert.Equal(t, []string{"reorg.detected"}, publisher.published, "it was published...")
	assert.Equal(t, 1, outbox.pendingCount(), "...but it must remain pending, so it is re-published")
}

func TestOutboxDrainerEmptyOutboxIsNoop(t *testing.T) {
	t.Parallel()

	outbox := &fakeOutbox{}
	published, err := newDrainer(t, outbox, &fakePublisher{}).DrainOnce(context.Background())
	require.NoError(t, err)
	assert.Zero(t, published)
}

func TestOutboxDrainerRespectsBatchSize(t *testing.T) {
	t.Parallel()

	outbox := &fakeOutbox{pending: []Notification{
		{ID: 1, Topic: "t1"}, {ID: 2, Topic: "t2"}, {ID: 3, Topic: "t3"},
	}}
	d := newDrainer(t, outbox, &fakePublisher{}).WithBatchSize(2)

	published, err := d.DrainOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, published)
	assert.Equal(t, 1, outbox.pendingCount(), "the remainder waits for the next pass")
}

func TestOutboxDrainerRunStopsOnStop(t *testing.T) {
	t.Parallel()

	outbox := &fakeOutbox{pending: []Notification{{ID: 1, Topic: "t"}}}
	d := newDrainer(t, outbox, &fakePublisher{}).WithInterval(5 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- d.Run(context.Background()) }()

	require.Eventually(t, func() bool { return outbox.pendingCount() == 0 }, time.Second, 5*time.Millisecond,
		"the loop must drain the outbox")

	d.Stop()
	select {
	case err := <-done:
		assert.NoError(t, err, "Stop is a graceful shutdown, not an error")
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

func TestOutboxDrainerRunReturnsOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	d := newDrainer(t, &fakeOutbox{}, &fakePublisher{}).WithInterval(5 * time.Millisecond)

	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}
