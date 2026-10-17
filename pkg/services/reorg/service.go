package reorg

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	portcontract "github.com/rtcdance/chainpulse/pkg/ports"

	"github.com/rtcdance/chainpulse/pkg/logkeys"
)

// Service runs the two background loops of the reorg repair pipeline: resume
// unfinished repair jobs, and drain the notification outbox to the publisher.
//
// Before it existed, Repair and OutboxDrainerLoop were only ever constructed
// inside tests — no production caller started them, so a committed rollback's
// job sat pending forever and its notifications were never delivered. Service
// is the single entry point a composition root needs.
type Service struct {
	repair  *Repair
	drainer *OutboxDrainerLoop
	logger  portcontract.Logger

	interval    time.Duration
	resumeLimit int

	stopOnce sync.Once
	stopped  chan struct{}
}

// NewService wires a repair runner and an outbox drainer into a runnable
// service. Both are required: repairing without draining strands the
// notifications, draining without repairing drains nothing.
func NewService(repair *Repair, drainer *OutboxDrainerLoop, logger portcontract.Logger) (*Service, error) {
	if repair == nil {
		return nil, fmt.Errorf("repair runner is required")
	}
	if drainer == nil {
		return nil, fmt.Errorf("outbox drainer is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &Service{
		repair:      repair,
		drainer:     drainer,
		logger:      logger,
		interval:    time.Second,
		resumeLimit: 100,
		stopped:     make(chan struct{}),
	}, nil
}

// WithInterval sets how often the service resumes jobs and drains the outbox
// (default 1s).
func (s *Service) WithInterval(interval time.Duration) *Service {
	if interval > 0 {
		s.interval = interval
	}
	return s
}

// Start launches Run in its own goroutine and returns immediately. This is the
// one call a composition root makes; Stop (or ctx cancellation) ends it.
//
// A loop failure is logged, not returned: the service must outlive individual
// repair or publish failures, because the pending job / undelivered
// notification it left behind is exactly the state the next pass retries.
func (s *Service) Start(ctx context.Context) {
	go func() {
		if err := s.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Error("Reorg service stopped with error", logkeys.LogKeyError, err)
		}
	}()
}

// Run resumes pending jobs and drains the outbox on a ticker until ctx is
// cancelled or Stop is called. One resume+drain pass runs immediately at
// startup, so a restarted process continues interrupted work without waiting
// for the first tick.
func (s *Service) Run(ctx context.Context) error {
	s.pass(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopped:
			return nil
		case <-ticker.C:
			s.pass(ctx)
		}
	}
}

// Stop makes Run return nil and stops the drainer loop. It is safe to call
// more than once.
func (s *Service) Stop() {
	s.stopOnce.Do(func() { close(s.stopped) })
	s.drainer.Stop()
}

func (s *Service) pass(ctx context.Context) {
	if _, err := s.repair.Resume(ctx, "", s.resumeLimit); err != nil {
		s.logger.Error("Reorg repair pass failed", logkeys.LogKeyError, err)
	}
	if _, err := s.drainer.DrainOnce(ctx); err != nil {
		s.logger.Error("Reorg outbox drain failed", logkeys.LogKeyError, err)
	}
}
