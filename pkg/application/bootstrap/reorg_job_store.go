package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/core/topics"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/services/reorg"
)

// PostgresReorgJobStore adapts the Postgres transactional outbox to the
// reorg.JobStore port.
//
// It lives in the composition layer rather than in either package, so the reorg
// service does not depend on a database plugin and the plugin does not depend
// on a service.
type PostgresReorgJobStore struct {
	adapter *database.PostgreSQLPluginAdapter
}

// PostgresReorgJobStore must satisfy the port.
var _ reorg.JobStore = (*PostgresReorgJobStore)(nil)

// It also drains the outbox, so one adapter serves both halves.
var _ reorg.OutboxDrainer = (*PostgresReorgJobStore)(nil)

// NewPostgresReorgJobStore creates the adapter.
func NewPostgresReorgJobStore(adapter *database.PostgreSQLPluginAdapter) *PostgresReorgJobStore {
	return &PostgresReorgJobStore{adapter: adapter}
}

// BuildReorgStore unwraps lifecycle decorators without replacing a shared
// adapter's chain scope. Memory deployments keep their existing local path.
func BuildReorgStore(db portcontract.DatabasePlugin) *PostgresReorgJobStore {
	for range 5 {
		if adapter, ok := db.(*database.PostgreSQLPluginAdapter); ok {
			return NewPostgresReorgJobStore(adapter)
		}
		wrapper, ok := db.(interface {
			Unwrap() portcontract.DatabasePlugin
		})
		if !ok {
			return nil
		}
		next := wrapper.Unwrap()
		if next == nil || next == db {
			return nil
		}
		db = next
	}
	return nil
}

func (s *PostgresReorgJobStore) ChainDatabase(chainID string) portcontract.DatabasePlugin {
	return s.adapter.ChainView(chainID)
}

func (s *PostgresReorgJobStore) CommitReorg(ctx context.Context, chainID string, from, to uint64) (int64, error) {
	return s.adapter.CommitReorg(ctx, chainID, from, to)
}

func (s *PostgresReorgJobStore) CompleteReorgsThrough(ctx context.Context, chainID string, height uint64) error {
	return s.adapter.CompleteReorgsThrough(ctx, chainID, height)
}

// EnqueueJob records the repair job and its notifications in ONE transaction.
func (s *PostgresReorgJobStore) EnqueueJob(ctx context.Context, job reorg.Job, notifications []reorg.Notification) error {
	dbNotifications := make([]database.ReorgNotification, 0, len(notifications))
	for _, n := range notifications {
		dbNotifications = append(dbNotifications, database.ReorgNotification{
			JobID:      n.JobID,
			ChainID:    n.ChainID,
			Generation: n.Generation,
			Topic:      n.Topic,
			Payload:    n.Payload,
		})
	}

	return s.adapter.EnqueueReorgJob(ctx, database.ReorgJob{
		ID:         job.ID,
		ChainID:    job.ChainID,
		Generation: job.Generation,
		FromBlock:  job.FromBlock,
		ToBlock:    job.ToBlock,
		Status:     job.Status,
	}, dbNotifications)
}

// PendingJobs returns unfinished repair work so a restarted process can resume.
func (s *PostgresReorgJobStore) PendingJobs(ctx context.Context, chainID string, limit int) ([]reorg.Job, error) {
	jobs, err := s.adapter.PendingReorgJobs(ctx, chainID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]reorg.Job, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, reorg.Job{
			ID:         j.ID,
			ChainID:    j.ChainID,
			Generation: j.Generation,
			FromBlock:  j.FromBlock,
			ToBlock:    j.ToBlock,
			Status:     j.Status,
			CreatedAt:  j.CreatedAt,
			UpdatedAt:  j.UpdatedAt,
		})
	}
	return out, nil
}

// CompleteJob marks a repair job done.
func (s *PostgresReorgJobStore) CompleteJob(ctx context.Context, jobID string) error {
	return s.adapter.CompleteReorgJob(ctx, jobID)
}

// PendingNotifications returns notifications that still owe delivery.
func (s *PostgresReorgJobStore) PendingNotifications(ctx context.Context, limit int) ([]reorg.Notification, error) {
	rows, err := s.adapter.PendingReorgNotifications(ctx, limit)
	if err != nil {
		return nil, err
	}

	out := make([]reorg.Notification, 0, len(rows))
	for _, n := range rows {
		out = append(out, reorg.Notification{
			ID:         n.ID,
			JobID:      n.JobID,
			ChainID:    n.ChainID,
			Generation: n.Generation,
			Topic:      n.Topic,
			Payload:    n.Payload,
			Attempts:   n.Attempts,
		})
	}
	return out, nil
}

// MarkPublished records a successful delivery. It is idempotent.
func (s *PostgresReorgJobStore) MarkPublished(ctx context.Context, id int64) error {
	return s.adapter.MarkReorgNotificationPublished(ctx, id)
}

// RecordFailure bumps the attempt counter for a notification.
func (s *PostgresReorgJobStore) RecordFailure(ctx context.Context, id int64) error {
	return s.adapter.RecordReorgNotificationFailure(ctx, id)
}

// EventBusPublisher adapts portcontract.EventBus to reorg.NotificationPublisher.
type EventBusPublisher struct {
	bus portcontract.EventBus
}

var _ reorg.NotificationPublisher = (*EventBusPublisher)(nil)

// NewEventBusPublisher creates the adapter. It returns nil when bus is nil, so
// callers can treat "no bus configured" as "no publisher".
func NewEventBusPublisher(bus portcontract.EventBus) *EventBusPublisher {
	if bus == nil {
		return nil
	}
	return &EventBusPublisher{bus: bus}
}

// Publish reconstructs the typed rollback event expected by puller subscribers.
func (p *EventBusPublisher) Publish(ctx context.Context, topic string, payload []byte) error {
	if topic == topics.TopicReorgRollback {
		var event portcontract.ReorgRollbackEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode reorg rollback: %w", err)
		}
		return p.bus.Publish(ctx, topic, &event)
	}
	return p.bus.Publish(ctx, topic, payload)
}
