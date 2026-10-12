package bootstrap

import (
	"context"

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

// NewPostgresReorgJobStore creates the adapter.
func NewPostgresReorgJobStore(adapter *database.PostgreSQLPluginAdapter) *PostgresReorgJobStore {
	return &PostgresReorgJobStore{adapter: adapter}
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
