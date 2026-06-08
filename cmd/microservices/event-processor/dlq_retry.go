package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
)

const (
	dlqRetryInterval   = 60 * time.Second
	dlqRetryMaxRetries = 5
	dlqRetryBatchSize  = 100
)

type dlqRetryService struct {
	logger    core.Logger
	metrics   core.MetricsCollector
	db        *sql.DB
	processor eventProcessorMessageProcessor

	mu        sync.RWMutex
	running   bool
	stopCh    chan struct{}
	closeOnce sync.Once
}

func newDLQRetryService(
	logger core.Logger,
	metrics core.MetricsCollector,
	db *sql.DB,
	processor eventProcessorMessageProcessor,
) *dlqRetryService {
	return &dlqRetryService{
		logger:    logger,
		metrics:   metrics,
		db:        db,
		processor: processor,
		stopCh:    make(chan struct{}),
	}
}

func (s *dlqRetryService) Start(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.db == nil || s.processor == nil {
		return
	}

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(dlqRetryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.retryPendingEvents(ctx)
			}
		}
	}()

	s.logger.Info("DLQ retry service started",
		"interval", dlqRetryInterval.String(),
		"max_retries", dlqRetryMaxRetries)
}

func (s *dlqRetryService) Stop() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		close(s.stopCh)
	})
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

func (s *dlqRetryService) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"running": s.running,
	}
}

func (s *dlqRetryService) retryPendingEvents(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chain_id, payload, retry_count, error_message
		 FROM dlq_events
		 WHERE status = 'pending' AND retry_count < $1
		 ORDER BY created_at ASC
		 LIMIT $2`,
		dlqRetryMaxRetries, dlqRetryBatchSize)
	if err != nil {
		s.logger.Warn("DLQ retry: failed to query pending events", "error", err.Error())
		return
	}
	defer rows.Close()

	var processed, failed, skipped int
	for rows.Next() {
		var id, chainID, payload, lastError string
		var retryCount int
		if err := rows.Scan(&id, &chainID, &payload, &retryCount, &lastError); err != nil {
			s.logger.Warn("DLQ retry: failed to scan row", "error", err.Error())
			skipped++
			continue
		}

		var event blockchain.BlockchainEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			s.logger.Warn("DLQ retry: failed to unmarshal event payload",
				"eventId", id, "error", err.Error())
			s.markFailed(ctx, id, fmt.Sprintf("payload unmarshal error: %v", err))
			failed++
			continue
		}

		if err := s.processor.ProcessEvent(ctx, &event); err != nil {
			newRetryCount := retryCount + 1
			if newRetryCount >= dlqRetryMaxRetries {
				s.markFailed(ctx, id, err.Error())
				failed++
			} else {
				s.updateRetryCount(ctx, id, err.Error(), newRetryCount)
				failed++
			}
			continue
		}

		s.markProcessed(ctx, id)
		processed++
	}

	if processed > 0 || failed > 0 {
		s.logger.Info("DLQ retry cycle completed",
			"processed", processed, "failed", failed, "skipped", skipped)
		if s.metrics != nil {
			s.metrics.RecordCounter("dlq_retry_processed", int64(processed), nil)
			s.metrics.RecordCounter("dlq_retry_failed", int64(failed), nil)
		}
	}
}

func (s *dlqRetryService) updateRetryCount(ctx context.Context, id, errMsg string, retryCount int) {
	_, err := s.db.ExecContext(ctx,
		`UPDATE dlq_events SET retry_count = $1, error_message = $2, updated_at = NOW() WHERE id = $3`,
		retryCount, errMsg, id)
	if err != nil {
		s.logger.Warn("DLQ retry: failed to update retry count", "eventId", id, "error", err.Error())
	}
}

func (s *dlqRetryService) markProcessed(ctx context.Context, id string) {
	_, err := s.db.ExecContext(ctx,
		`UPDATE dlq_events SET status = 'processed', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		s.logger.Warn("DLQ retry: failed to mark processed", "eventId", id, "error", err.Error())
	}
}

func (s *dlqRetryService) markFailed(ctx context.Context, id, errMsg string) {
	_, err := s.db.ExecContext(ctx,
		`UPDATE dlq_events SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
		errMsg, id)
	if err != nil {
		s.logger.Warn("DLQ retry: failed to mark failed", "eventId", id, "error", err.Error())
	}
}
