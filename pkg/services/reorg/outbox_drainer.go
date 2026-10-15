package reorg

import (
	"context"
	"fmt"
	"sync"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/logkeys"
)

// NotificationPublisher delivers one notification payload to a topic. An
// event-bus or message-queue adapter satisfies it.
type NotificationPublisher interface {
	Publish(ctx context.Context, topic string, payload []byte) error
}

// OutboxDrainerLoop moves notifications from the outbox to the publisher.
//
// Delivery is AT-LEAST-ONCE: a notification is marked published only AFTER the
// publish succeeds, so a crash in between re-publishes it. That is deliberate —
// marking first would turn a crash into a lost notification, which is exactly
// the failure F07 describes. Consumers deduplicate on (job id, generation),
// which every notification carries.
type OutboxDrainerLoop struct {
	store     OutboxDrainer
	publisher NotificationPublisher
	logger    portcontract.Logger

	interval  time.Duration
	batchSize int

	stopOnce sync.Once
	stopped  chan struct{}
}

// NewOutboxDrainerLoop creates a drainer. It does not start anything; call
// DrainOnce for one pass or Run for the loop.
func NewOutboxDrainerLoop(store OutboxDrainer, publisher NotificationPublisher, logger portcontract.Logger) (*OutboxDrainerLoop, error) {
	if store == nil {
		return nil, fmt.Errorf("outbox store is required")
	}
	if publisher == nil {
		return nil, fmt.Errorf("publisher is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &OutboxDrainerLoop{
		store:     store,
		publisher: publisher,
		logger:    logger,
		interval:  time.Second,
		batchSize: 100,
		stopped:   make(chan struct{}),
	}, nil
}

// WithInterval sets the poll interval (default 1s).
func (d *OutboxDrainerLoop) WithInterval(interval time.Duration) *OutboxDrainerLoop {
	if interval > 0 {
		d.interval = interval
	}
	return d
}

// WithBatchSize caps how many notifications one pass handles (default 100).
func (d *OutboxDrainerLoop) WithBatchSize(n int) *OutboxDrainerLoop {
	if n > 0 {
		d.batchSize = n
	}
	return d
}

// DrainOnce publishes up to one batch of pending notifications and returns how
// many were delivered.
//
// A failed publish is recorded against the notification and the pass continues,
// so one poison message does not block the rest of the outbox. The first error
// is returned for the caller to log or alert on; the loop keeps going.
func (d *OutboxDrainerLoop) DrainOnce(ctx context.Context) (int, error) {
	pending, err := d.store.PendingNotifications(ctx, d.batchSize)
	if err != nil {
		return 0, fmt.Errorf("load pending notifications: %w", err)
	}

	published := 0
	var firstErr error

	for _, n := range pending {
		if err := ctx.Err(); err != nil {
			return published, err
		}

		if err := d.publisher.Publish(ctx, n.Topic, n.Payload); err != nil {
			if ferr := d.store.RecordFailure(ctx, n.ID); ferr != nil {
				d.logger.Error("Failed to record reorg notification failure",
					logkeys.LogKeyError, ferr, "notification_id", n.ID)
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("publish notification %d (%s): %w", n.ID, n.Topic, err)
			}
			continue
		}

		// Marked only after a successful publish: at-least-once.
		if err := d.store.MarkPublished(ctx, n.ID); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("mark notification %d published: %w", n.ID, err)
			}
			continue
		}
		published++
	}

	return published, firstErr
}

// Run drains on a ticker until ctx is cancelled or Stop is called. It returns
// ctx.Err() on cancellation and nil on Stop, so a graceful shutdown is not
// reported as a failure.
func (d *OutboxDrainerLoop) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.stopped:
			return nil
		case <-ticker.C:
			if _, err := d.DrainOnce(ctx); err != nil {
				d.logger.Error("Reorg outbox drain failed", logkeys.LogKeyError, err)
			}
		}
	}
}

// Stop makes Run return nil. It is safe to call more than once.
func (d *OutboxDrainerLoop) Stop() {
	d.stopOnce.Do(func() { close(d.stopped) })
}
